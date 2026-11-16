# NATS JetStream Capacity Evaluation and Sizing Model

## 1. Architecture and Fixed Constraints

1. **Cluster Topology**: Strictly Fixed at 3 Nodes.
2. **Replication Factor**: Fixed at Replication = 3 (R=3) for all streams.
3. **Storage Engine**: JetStream File Store (append-only sequential block files).
4. **Access Tier**: HAProxy Layer 4 Ingress Load Balancer (round-robin / leastconn across the 3 nodes).

---

## 2. Operations

### 2.1 Core Messaging Operations (Write & Read)

| Category | Operation | Trigger & Flow | Frequency / Rate |
|---|---|---|---|
| **Write** | **Core NATS Publish** | `Client -> Server [Mem Ingress]` (in-memory parse & route table match, zero disk, no server ack) | Driven by publisher rate (`x_core`) |
| **Write** | **JetStream Publish** | `Client -> Server -> 2x Peer [Raft] -> 3x Disk -> Client [PubAck]` (R=3 persistence) | Driven by publisher rate (`x_js`) |
| **Read** | **Core NATS Delivery** | `Server [Mem] -> Subscriber` (ephemeral push delivery, zero cursor state, no ack) | Driven by delivery rate (`y_core = c_core * x_core`) |
| **Read** | **JetStream Consume** | `Client [Batch Fetch] -> Server [Deliver] -> Client [+ACK]` (stateful read cursor) | Driven by delivery rate (`y_js = c_js * x_js`) |

---

### 2.2 Heartbeats and Keepalives

| Mechanism | Scope | Interval Symbol | Trigger & Scope | Purpose |
|---|---|---|---|---|
| **Raft Leader Heartbeats** | Server to Server | `t_raft` | Timer-driven per stream | Raft leaders renew lease to followers across stream & meta consensus groups |
| **Cluster Route Pings** | Server to Server | `t_route` | Full-mesh inter-node timer | Full-mesh TCP route keepalive detecting network partitions between the 3 nodes |
| **Client TCP Pings** | Client to Server | `t_ping` | Client TCP timer | Client-side TCP keepalive preventing firewall / NAT / LB idle connection pruning |
| **Consumer Idle Heartbeats** | Server to Client | `t_hb` | Timer-driven (idle only) | Informs pull/push consumers that connection is alive; suppressed during active traffic |

---

### 2.3 Monitoring and Telemetry (P2 - Ignored in Capacity Formulas)

| Mechanism | Channel / Endpoint | Interval | Purpose |
|---|---|---|---|
| **Prometheus Scraping** | HTTP `/metrics`, `/varz`, `/connz`, `/jsz` | Every 15 seconds | External metrics collector scrapes node, connection, and stream telemetry |
| **Kubernetes Health Probes** | HTTP `/healthz` | Every 10 seconds | Kubelet liveness and readiness probes checking node responsiveness |
| **System Advisory Events** | Internal `$SYS.*` account | Event-driven + 30s broadcast | Audit events for client connect/disconnect, stream leader election, and stats |

*Note on Priority 2 (P2)*: Monitoring, health probes, and system advisories generate low-frequency HTTP/in-memory traffic with negligible CPU (<0.5%) and zero disk impact. They are acknowledged for operational completeness but excluded from core capacity formulas.

---

### 2.4 Subscription Interest Gossip (P2 - Ignored in Capacity Formulas)

| Mechanism | Trigger | Frequency in Steady-State | Purpose |
|---|---|---|---|
| **Interest Graph Gossip** | Client `SUB`, `UNSUB`, or ephemeral `_INBOX.*` | **Virtually Zero (~0 ops/s)** | Informs remote nodes where subscribers exist so messages are routed correctly across the mesh |

*Note on Steady-State & Priority 2 (P2)*: Once subscribers connect and durable consumers are initialized, subject interest graph gossip across the cluster mesh is completely silent. It produces zero steady-state operations and is excluded from capacity formulas.

---

## 3. Notation

| Symbol | Meaning | Mode / Scope | Unit / Default | Description |
|---|---|---|---|---|
| **`x_js`** | JetStream Ingestion Rate | JetStream (Sending) | msgs/sec | Offered writes persisted to disk across R=3 with PubAck (Avg / Peak) |
| **`c_js`** | JetStream Fan-Out | JetStream (Receiving) | count (default 1) | Active consumers receiving and acknowledging each message |
| **`y_js`** | JetStream Egress Rate | JetStream (Receiving) | msgs/sec (`y_js = c_js * x_js`) | Total delivered messages acknowledged across JetStream consumers |
| **`x_core`** | Core NATS Ingestion Rate | Core NATS (Sending) | msgs/sec (default 0) | Ephemeral in-memory writes (zero disk, no server ack, Avg / Peak) |
| **`c_core`** | Core NATS Fan-Out | Core NATS (Receiving) | count (default 0) | Ephemeral push subscribers per Core message |
| **`y_core`** | Core NATS Egress Rate | Core NATS (Receiving) | msgs/sec (`y_core = c_core * x_core`) | Total ephemeral messages pushed to Core subscribers |
| **`payload`** | Application Payload Size | Message Spec | bytes (default 1,024 B) | Raw application payload bytes sent by the publisher |
| **`header`** | Custom NATS Headers | Message Spec | bytes (default 0 B) | Tracing headers (`traceparent`), message IDs, and custom metadata |
| **`t`** | Message Retention Time | JetStream Storage | seconds (default 72h) | Duration JetStream messages remain stored in disk block files |
| **`n_conn`** | Client TCP Connections | Topology | count (default 100) | Concurrent client TCP sockets connected to the cluster |
| **`n_subj`** | Subject Cardinality | Topology | count (default 100) | Total unique active subjects in sublist trie |
| **`n_stream`** | Stream Count | Topology | count (default 1) | Active JetStream streams (Raft replica consensus groups) |
| **`n_consumer`** | Durable Consumer Count | Topology | count (default 1) | Active durable consumers tracking stateful progress |
| **`batch`** | Pull Consumer Batch Size | Constant | count (default 500) | Messages requested per pull fetch (`$JS.API.CONSUMER.MSG.NEXT`) |
| **`w`** | Working Set Window | Constant | seconds (default 120s) | Duration messages stay in Linux OS Page Cache for zero-disk replay |
| **`headroom`** | Operational Headroom | Constant | ratio (default 0.30) | Headroom buffer targeting 70% peak utilization for N-1 resilience |
| **`S_proto`** | Protocol Wire Framing | Constant | bytes (default 32 B) | NATS protocol line framing (`PUB <subject> <reply> <bytes>\r\n`) |
| **`S_msg`** | Effective Message Size | Derived | bytes (`payload + header + S_proto`) | Total size per message across wire and disk storage |
| **`t_raft`** | Raft Heartbeat Interval | Constant | seconds (default 250 ms) | Internal consensus lease renewal timer |
| **`t_route`** | Cluster Route Ping Interval | Constant | seconds (default 2s) | Inter-node TCP cluster mesh keepalive timer |
| **`t_ping`** | Client TCP Ping Interval | Constant | seconds (default 60s) | Client-side TCP keepalive timer |
| **`t_hb`** | Consumer Idle Heartbeat | Constant | seconds (default 5s) | Consumer keepalive timer during traffic gaps |

---

## 4. Parameters

### 4.1 Workload Input Parameters (User Inputs)

| Category | Parameter | Symbol | Unit / Range | Default | Sizing Impact |
|---|---|---|---|---|---|
| **Connections** | Client TCP Connections | `n_conn` | count | 100 | HAProxy distributes evenly (~`n_conn / 3` sockets/node), driving socket RAM |
| **Message** | Application Payload Size | `payload` | bytes | 1,024 B (1 KiB) | Wire network bandwidth and disk storage volume |
| **Message** | Custom Headers Size | `header` | bytes | 0 B | Tracing context and metadata wire bandwidth |
| **Write Rate** | Core NATS Ingress (Avg) | `x_core_avg` | msgs/sec | 0 | Baseline ephemeral routing throughput |
| **Write Rate** | Core NATS Ingress (Peak) | `x_core_peak` | msgs/sec | 0 | Peak ephemeral routing CPU and egress network |
| **Write Rate** | JetStream Ingress (Avg) | `x_js_avg` | msgs/sec | 1,000 | Persistent disk volume growth over retention window `t` |
| **Write Rate** | JetStream Ingress (Peak) | `x_js_peak` | msgs/sec | 2,500 | Peak write CPU, Raft mesh network, and disk write IOPS |
| **Fan-Out** | Core NATS Fan-Out | `c_core` | count | 0 | Push delivery network egress multiplier |
| **Fan-Out** | JetStream Fan-Out | `c_js` | count | 1 | Consumer ACK consensus tax and egress multiplier |
| **Retention** | Message Retention Period | `t` | hours / days | 72 hours | Total persistent NVMe SSD disk volume |
| **Resources** | Subject Cardinality | `n_subj` | count | 100 | Sublist prefix trie memory (~550 B / subject) |
| **Resources** | Stream Count | `n_stream` | count | 1 | Raft replica memory (~312 KiB / stream) and heartbeats |
| **Resources** | Active Durable Consumers | `n_consumer` | count | 1 | Consumer progress cursor memory (~19 MB / consumer) |

---

### 4.2 Standard System Constants (Assumed Defaults & Sizing Constants)

All operational constants and tuning baselines are consolidated here:

| Constant | Symbol | Default Value | Tuning Scope / NATS Config | Operational & Sizing Impact |
|---|---|---|---|---|
| **Pull Consumer Batch Size** | `batch` | **500 msgs** | Client SDK `Fetch(batch)` | Amortizes pull request/reply overhead ($\frac{y_{\text{js}}}{\text{batch}}$ requests/sec) |
| **Working Set Cache Window** | `w` | **120 seconds** (2 min) | OS Page Cache allocation | Retains recent writes in RAM for zero-disk consumption |
| **Operational Headroom Buffer** | `headroom` | **30%** (0.30) | FinOps / Target 70% | Ensures safety margin for unexpected spikes and single-node failover |
| **Protocol Wire Framing** | `S_proto` | **32 bytes** | Protocol constant | Wire overhead per message line (`PUB <subject> <reply> <bytes>\r\n`) |
| **Raft Heartbeat Interval** | `t_raft` | **250 ms** (4.0 ops/s) | NATS internal timer | Consensus lease renewal per stream Raft group |
| **Cluster Route Ping Interval** | `t_route` | **2 seconds** (0.5 ops/s) | `cluster { ping_interval: 2s }` | Inter-node TCP cluster mesh keepalive across the 3 nodes |
| **Client TCP Ping Interval** | `t_ping` | **60 seconds** (1 op/min) | Client SDK `Options.PingInterval` | Client keepalive preventing cloud L4 LB / firewall idle termination |
| **Consumer Idle Heartbeat** | `t_hb` | **5 seconds** (0.2 ops/s) | Consumer `IdleHeartbeat: 5s` | Keeps consumer session active during quiet periods (suppressed when active) |

---

## 5. System Invariants (Derived Relations)

Deterministic mathematical laws and invariant relationships governing capacity calculations:

### 5.1 Read Throughput Invariant (Point A)
Egress read throughput is strictly derived from ingress write throughput multiplied by fan-out:
$$y_{\text{core\_avg}} = x_{\text{core\_avg}} \times c_{\text{core}}$$
$$y_{\text{core\_peak}} = x_{\text{core\_peak}} \times c_{\text{core}}$$
$$y_{\text{js\_avg}} = x_{\text{js\_avg}} \times c_{\text{js}}$$
$$y_{\text{js\_peak}} = x_{\text{js\_peak}} \times c_{\text{js}}$$

### 5.2 Pull Consumer Request Invariant
Pull consumers request messages in discrete batches, keeping server request/reply overhead low:
$$R_{\text{batch\_peak}} = \frac{y_{\text{js\_peak}}}{\text{batch}} \quad \text{(fetch requests/sec)}$$

### 5.3 Message Size Invariant
Total effective message size across wire transmission and disk block storage framing:
$$S_{\text{msg}} = \text{payload} + \text{header} + S_{\text{proto}} \quad \text{(bytes/msg)}$$

### 5.4 Storage Replication Invariant
With fixed Replication R=3 across 3 nodes, every node persists 100% of JetStream ingestion:
$$D_{\text{node}} = x_{\text{js\_avg}} \times S_{\text{msg}} \times t \quad \text{(retained disk bytes per node)}$$
$$D_{\text{cluster}} = 3 \times D_{\text{node}} \quad \text{(total cluster persistent storage)}$$

### 5.5 Target Utilization & Headroom Invariant
Hardware resources are sized to maintain a 30% headroom buffer during peak load:
$$\text{Target Utilization} = 1 - \text{headroom} = 0.70$$
$$\text{Provisioned Capacity} = \frac{\text{Peak Resource Requirement}}{\text{Target Utilization}}$$

### 5.6 Working Set Cache Invariant (Point B)
Real-time consumers reading within the working set window ($w = 120\text{ s}$) are served directly from Linux OS Page Cache on each node without generating physical disk read IOPS:
$$M_{\text{cache, node}} = x_{\text{js\_avg}} \times S_{\text{msg}} \times w \quad \text{(RAM bytes per node for zero-disk replay)}$$
