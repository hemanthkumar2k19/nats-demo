# Operational Baseline & Sizing Invariants

## 1. Fixed Architecture Constraints

* **Cluster Topology**: Strictly fixed at 3 Nodes.
* **Replication Factor**: $R = 3$ (Raft consensus quorum across all 3 nodes).
* **Storage Engine**: JetStream File Store (append-only sequential block files).
* **Ingress Tier**: HAProxy Layer 4 Load Balancer (round-robin / leastconn across nodes).

---

## 2. Cluster Operations

### 2.1 Messaging Operations (Write & Read)

| Type | Operation | Flow & Persistence | Throughput Formula |
|---|---|---|---|
| **Write** | **Core NATS Publish** | `Client -> Server [Mem]` (in-memory route, zero disk, no ACK) | $x_{\text{core}}$ (msgs/s) |
| **Write** | **JetStream Publish** | `Client -> Server -> 2x Peer [Raft] -> 3x Disk -> Client [PubAck]` | $x_{\text{js}}$ (msgs/s) |
| **Read** | **Core NATS Delivery** | `Server [Mem] -> Subscriber` (ephemeral push, zero cursor state) | $y_{\text{core}} = x_{\text{core}} \times c_{\text{core}}$ |
| **Read** | **JetStream Consume** | `Client [Batch Fetch] -> Server [Deliver] -> Client [+ACK]` (stateful cursor) | $y_{\text{js}} = x_{\text{js}} \times c_{\text{js}}$ |

### 2.2 Heartbeats & Keepalives

| Mechanism | Symbol | Default Interval | Configurability | Scope & Purpose |
|---|---|---|---|---|
| **Raft Heartbeats** | `T_RAFT` | 250 ms (4.0 ops/s) | Fixed (NATS Code) | Stream leader lease renewal to followers across Raft groups |
| **Cluster Route Pings** | `T_ROUTE` | 2 s (0.5 ops/s) | Configurable (Server) | Full-mesh inter-node TCP keepalive (`cluster.ping_interval`) |
| **Client TCP Pings** | `T_PING` | 60 s (1 op/min) | Configurable (Client/Server) | Client keepalive preventing cloud L4 LB / firewall idle pruning |
| **Consumer Heartbeats** | `T_HB` | 5 s (0.2 ops/s) | Configurable (Consumer) | Pull/push session alive check (`IdleHeartbeat`; idle only) |

---

## 3. Parameters

### 3.1 Workload Inputs

| Workload Parameter | Symbol | Unit | Sizing Role |
|---|---|---|---|
| **Client Connections** | `n_conn_avg` / `n_conn_peak` | count | Baseline idle RAM (~$n_{\text{conn}} / 3$ per node) vs peak K8s limits & OS `nofile` |
| **Payload Size** | `payload_avg` / `payload_peak` | bytes | Persistent disk storage volume vs peak network NIC bandwidth |
| **Message Headers** | `header` | bytes | Wire protocol tracing and metadata overhead |
| **Core Ingress Rate** | `x_core_avg` / `x_core_peak` | msgs/s | Baseline vs peak ephemeral routing CPU & network egress |
| **JetStream Ingress Rate**| `x_js_avg` / `x_js_peak` | msgs/s | Persistent disk volume growth over $t$ vs peak write CPU & disk IOPS |
| **Core Fan-Out** | `c_core` | count | Egress push delivery multiplier ($y_{\text{core}} = x_{\text{core}} \times c_{\text{core}}$) |
| **JetStream Fan-Out** | `c_js` | count | Consumer delivery & explicit ACK consensus multiplier ($y_{\text{js}} = x_{\text{js}} \times c_{\text{js}}$) |
| **Message Retention** | `t` | hours | Total persistent disk retention window |
| **Subject Cardinality** | `n_subj` | count | Sublist prefix trie memory (~940 B / subject cluster-wide) |
| **Stream Count** | `n_stream` | count | Multi-Raft state memory (~720 KiB / stream) & heartbeat overhead |
| **Durable Consumers** | `n_consumer_avg` / `n_consumer_peak` | count | Baseline cursor tracking vs peak Go heap progress state (~21.9 MB / consumer) |

### 3.2 Standard System Constants

| Constant | Symbol | Default Value | Configurability | Operational Role & Scope |
|---|---|---|---|---|
| **Batch Size** | `BATCH` | 500 msgs | Configurable (Client SDK) | Amortizes pull request/reply frequency ($R_{\text{batch}} = y_{\text{js}} / \text{BATCH}$) |
| **Headroom Buffer** | `HEADROOM` | 30% (0.30) | Fixed (Model Standard) | Targets 70% peak utilization ($\frac{\text{Peak}}{0.70}$) for N-1 failover safety |
| **Protocol Overhead** | `S_PROTO` | 32 bytes | Fixed (NATS Code) | Fixed framing bytes per message (`PUB <subject> <reply> <bytes>\r\n`) |
| **Raft Heartbeat** | `T_RAFT` | 250 ms (4.0 ops/s) | Fixed (NATS Code) | Per-stream consensus leader lease timer |
| **Route Keepalive** | `T_ROUTE` | 2 s (0.5 ops/s) | Configurable (Server) | Full-mesh inter-node TCP route ping (`cluster.ping_interval`) |
| **Client Keepalive** | `T_PING` | 60 s (1 op/min) | Configurable (Client/Server) | Client-side TCP ping preventing idle LB/firewall drops |
| **Consumer Idle Ping** | `T_HB` | 5 s (0.2 ops/s) | Configurable (Consumer) | Inactive consumer session alive check (`IdleHeartbeat`) |

---

## 4. Derived System Invariants

| Invariant | Formula | Sizing Application |
|---|---|---|
| **Read Throughput** | $y_{\text{core}} = x_{\text{core}} \times c_{\text{core}}$<br>$y_{\text{js}} = x_{\text{js}} \times c_{\text{js}}$ | Derived egress delivery rates for Core and JetStream (evaluated for both Avg and Peak). |
| **Pull Request Rate** | $R_{\text{batch\_peak}} = \frac{y_{\text{js\_peak}}}{\text{BATCH}}$ | Batch pull request frequency; amortizes control-plane RPC overhead 500x. |
| **Effective Message Size** | $S_{\text{avg}} = \text{payload}_{\text{avg}} + \text{header} + S_{\text{PROTO}}$<br>$S_{\text{peak}} = \text{payload}_{\text{peak}} + \text{header} + S_{\text{PROTO}}$ | $S_{\text{avg}}$ sizes persistent disk storage volume; $S_{\text{peak}}$ sizes peak network NIC bandwidth & socket buffers. |
| **Persistent Storage ($R=3$)**| $D_{\text{node}} = x_{\text{js\_avg}} \times S_{\text{avg}} \times t$<br>$D_{\text{cluster}} = 3 \times D_{\text{node}}$ | Total retained persistent disk storage per node and across the 3-node cluster. |
| **Target Utilization** | $\text{Target} = 1 - \text{HEADROOM} = 0.70$<br>$\text{Provisioned} = \frac{\text{Peak}}{0.70}$ | Sizes K8s compute limits to guarantee single-node failover survivability ($N-1$). |

---

## 5. Ignored Operations (Excluded from Sizing)

| Mechanism | Trigger / Frequency | Scope | Exclusion Rationale |
|---|---|---|---|
| **Prometheus Scraping** | Every 15 seconds | HTTP `/metrics`, `/varz` | Negligible CPU (<0.5%), zero disk writes, absorbed by 30% headroom. |
| **K8s Health Probes** | Every 10 seconds | HTTP `/healthz` | Lightweight in-memory liveness/readiness checks (<0.1% CPU). |
| **System Advisories** | Event-driven + 30s | Internal `$SYS.*` | Transient event notifications running in-memory without persistent disk IO. |
| **Interest Graph Gossip**| On client connect / churn | Remote node routes | Inactive during steady state (~0 ops/s); routes only sync subscription deltas. |
