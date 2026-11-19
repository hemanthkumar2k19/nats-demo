# Persistent Disk Storage Capacity Planning and Sizing Model

## 1. Impact of Variables and Constants on Persistent Disk

In enterprise JetStream architecture ($R=3$, File Store), **Persistent Disk is a durable capacity and sequential write resource**. NATS persists messages sequentially into append-only segment files (`.blk`), committing writes to disk via synchronous or group `fsync()` operations, while relying on the Linux OS Page Cache for zero-disk reads.

### 1.1 Workload Input Variables (Audit of All 16 Variables)

| Input Variable | Category | Included in Disk? | Role in Disk Sizing | Physical Justification |
|---|---|:---:|---|---|
| **`x_js_avg`** | Throughput | **YES** | **Primary Volume Driver** | Average persistent ingress rate (msgs/s). Governs the steady-state cumulative byte accumulation over the retention window. |
| **`payload_avg`** | Message | **YES** | **Primary Volume Driver** | Average message payload size in bytes. Directly dictates raw data byte density stored sequentially in stream segment blocks. |
| **`t`** | Retention | **YES** | **Primary Volume Driver** | Message retention window in hours ($t \times 3,600\text{ s}$). Multiplies average write velocity to define total accumulated steady-state volume. |
| **`header`** | Message | **YES** | **Storage Volume Driver** | Message tracing headers and metadata bytes persisted alongside the payload in each segment block record. |
| **`x_js_peak`** | Throughput | **YES** | **Peak I/O & IOPS Driver** | Peak persistent ingress rate (msgs/s). Determines burst disk write bandwidth and physical disk `fsync()` commit queue depth. |
| **`payload_peak`** | Message | **YES** | **Peak I/O Bandwidth** | Peak burst payload size. Sizes maximum instantaneous write throughput to the storage controller. |
| **`n_stream`** | Resources | **YES** | **Partitioning & Descriptors** | Number of active streams. Each stream maintains an independent directory, segment block files (`.blk`), and consumer state journals. |
| **`c_js`** | Fan-Out | **NO** | Excluded (Write-Once) | Messages are written to disk exactly once upon ingress. All 10 or 20 consumers read from kernel Page Cache; fan-out generates 0 extra disk writes. |
| **`x_core_avg`** | Throughput | **NO** | Excluded (Ephemeral) | Core NATS routing is strictly in-memory. Messages are never written to physical disk. |
| **`x_core_peak`** | Throughput | **NO** | Excluded (Ephemeral) | Peak ephemeral routing consumes CPU cycles and network buffers, with zero disk persistence. |
| **`c_core`** | Fan-Out | **NO** | Excluded (Ephemeral) | Ephemeral broadcast fan-out occurs purely in memory. |
| **`n_conn_avg`** | Connections | **NO** | Excluded (Network) | TCP connections reside in kernel socket tables and consume zero disk storage. |
| **`n_conn_peak`** | Connections | **NO** | Excluded (Network) | Peak client sockets size network buffers and epoll loops, not disk storage. |
| **`n_subj`** | Resources | **NO** | Excluded (Namespace) | Subject hierarchy is maintained in memory (sublist prefix trie). Messages append sequentially to streams regardless of subject count. |
| **`n_consumer_avg`**| Resources | **NO** | Excluded (Negligible) | Consumer cursor state updates write periodic compact sequence journals (~few bytes), absorbed by the 30% headroom buffer. |
| **`n_consumer_peak`**| Resources | **NO** | Excluded (Negligible) | Consumer progress tracking state resides primarily in Go heap RAM; disk journaling overhead is negligible relative to message payloads. |

---

### 1.2 Standard System Constants (Audit of All 7 Constants)

| System Constant | Default Value | Included in Disk? | Role in Disk Sizing | Physical Justification |
|---|---|:---:|---|---|
| **`HEADROOM`** | **30%** (0.30) | **YES** | **Capacity Sizing Divisor** | Divides raw stored volume by $1 - \text{HEADROOM} = 0.70$. Guarantees disk utilization never exceeds 70%, preventing ext4/xfs fragmentation, absorbing segment roll-over padding, and covering Linux 5% root reservation. |
| **`S_PROTO`** | **32 bytes** | **YES** | **Block Record Overhead** | Wire and storage record framing overhead per message (magic bytes, sequence numbers, timestamp, subject length, and header length). |
| **`BATCH`** | **500 msgs** | **NO** | Excluded (CPU/RPC) | Amortizes pull request/reply frequency over the network; has zero impact on persistent disk storage. |
| **`T_RAFT`** | **250 ms** (4 ops/s) | **NO** | Excluded (CPU Rate) | Consensus leader lease heartbeat timer; consumes CPU and network, with zero disk writes. |
| **`T_ROUTE`** | **2 seconds** (0.5 ops/s) | **NO** | Excluded (Network) | Inter-node TCP cluster keepalive; zero disk persistence. |
| **`T_PING`** | **60 seconds** (1 op/min) | **NO** | Excluded (Network) | Client TCP ping keepalive; zero disk persistence. |
| **`T_HB`** | **5 seconds** (0.2 ops/s) | **NO** | Excluded (Network) | Consumer idle session check; zero disk persistence. |

---

## 2. Deduction of Mathematical Model

Persistent Disk planning requires sizing two distinct dimensions: **Storage Volume (Capacity in GB)** and **I/O Throughput & IOPS (Write Performance)**.

---

### 2.1 Persistent Storage Volume Formulation ($D_{\text{node}}$ and $D_{\text{cluster}}$)

#### 1. Effective Stored Message Size ($S_{\text{avg}}$)
Each persisted message record in a JetStream segment block includes payload bytes, metadata headers, and record protocol framing:

$$S_{\text{avg}} = \text{payload}_{\text{avg}} + \text{header} + S_{\text{PROTO}} \quad (\text{bytes})$$

*Example*: A 1,024-byte payload with 64-byte headers and 32-byte protocol framing yields $S_{\text{avg}} = 1,120\text{ bytes per message record}$.

#### 2. Raw Cumulative Storage per Node ($D_{\text{node}}$)
In a 3-node cluster ($R=3$, File storage), every node stores a complete, independent replica of the stream data. Total raw bytes accumulated over retention window $t$ (hours) is:

$$D_{\text{node}} = x_{\text{js\_avg}} \cdot S_{\text{avg}} \cdot (t \cdot 3,600) \quad (\text{bytes})$$

#### 3. Provisioned Disk Capacity with 30% Headroom ($D_{\text{provisioned, node}}$)
To maintain operational health, storage volumes must be provisioned targeting $\le 70\%$ peak utilization:

$$D_{\text{provisioned, node}} = \frac{D_{\text{node}}}{1 - \text{HEADROOM}} = \frac{D_{\text{node}}}{0.70}$$

The 30% Headroom buffer is an essential physical requirement for persistent storage:
1. **Filesystem Performance**: Standard Linux filesystems (`ext4`, `xfs`) experience severe allocation latency and block fragmentation when utilization exceeds 80% to 85%.
2. **Segment File Roll-Over Padding**: JetStream allocates messages into discrete block files (`.blk`, typically 8 MB or 16 MB). Partially filled blocks and roll-over boundaries require temporary storage padding.
3. **Retention Sweep Delay**: JetStream background retention cleanup sweeps delete expired segment blocks asynchronously, creating temporary storage spikes before disk blocks are reclaimed.
4. **OS Reserved Blocks**: The Linux kernel reserves 5% of filesystem blocks for the root user (`tune2fs`), reducing usable user space.

#### 4. Total Cluster Provisioned Storage ($D_{\text{provisioned, cluster}}$)
Across all 3 nodes ($R=3$):

$$D_{\text{provisioned, cluster}} = 3 \cdot D_{\text{provisioned, node}}$$

*Comprehensive Sizing Example*:
* Workload: $x_{\text{js\_avg}} = 2,500\text{ msgs/s}$, $\text{payload}_{\text{avg}} = 1,024\text{ B}$, $\text{header} = 64\text{ B}$, $S_{\text{PROTO}} = 32\text{ B}$ ($S_{\text{avg}} = 1,120\text{ B}$), retention $t = 24\text{ hours}$ (86,400 seconds).
* Raw Storage per Node:
  $$D_{\text{node}} = 2,500 \cdot 1,120 \cdot 86,400 = 241,920,000,000\text{ bytes} \approx 241.9\text{ GB} \ (225.3\text{ GiB})$$
* Provisioned Storage per Node (with 30% Headroom):
  $$D_{\text{provisioned, node}} = \frac{241.9\text{ GB}}{0.70} \approx 345.6\text{ GB} \ (321.9\text{ GiB})$$
* Cluster Total Storage ($R=3$):
  $$D_{\text{provisioned, cluster}} = 3 \cdot 345.6\text{ GB} \approx 1,036.8\text{ GB} \ (1.04\text{ TB})$$

---

### 2.2 Persistent Disk I/O Performance & Bandwidth Formulation

#### 1. Sustained Peak Write Bandwidth ($\text{BW}_{\text{write}}$)
Peak write bandwidth requirements dictate disk controller throughput:

$$\text{BW}_{\text{write, node}} = x_{\text{js\_peak}} \cdot S_{\text{peak}} \quad (\text{bytes/sec})$$
$$\text{BW}_{\text{write, cluster}} = 3 \cdot \text{BW}_{\text{write, node}} \quad (\text{bytes/sec})$$

*Example*: At $x_{\text{js\_peak}} = 5,000\text{ msgs/s}$ and $S_{\text{peak}} = 1,120\text{ bytes}$:
$$\text{BW}_{\text{write, node}} = 5,000 \cdot 1,120 = 5,600,000\text{ B/s} \approx 5.60\text{ MB/s} \ (5.34\text{ MiB/s})$$
$$\text{BW}_{\text{write, cluster}} = 3 \cdot 5.60\text{ MB/s} = 16.80\text{ MB/s}$$

#### 2. Disk IOPS & Fsync Group Commit Dynamics
In JetStream replicated file storage, incoming writes are appended to the active segment block and committed to physical storage using `fsync()`:
* **Low Offered Load ($< 1,000\text{ msgs/s}$)**: Incoming messages arrive into an idle write queue, triggering individual `fsync()` calls ($1\text{ fsync/msg}$). Median latency reflects physical disk sync time (~3-4 ms).
* **High Offered Load ($2,500 - 5,000\text{ msgs/s}$)**: NATS automatically batches multiple concurrent write proposals into shared `fsync()` group commits. Effective sync latency per message drops below 1 ms.
* **Storage Sizing Recommendation**: Provision cloud persistent disks (e.g. AWS GP3, GCP Persistent Disk SSD, Azure Premium SSD) with a baseline of **$\ge 3,000$ write IOPS** and **$\ge 125\text{ MB/s}$ throughput** per node to eliminate disk controller bottlenecks.

#### 3. The Zero-Disk Read Replay Principle
Because live streaming consumers read directly from memory pages held in spare host RAM (Linux OS Page Cache), **real-time consumers generate zero physical disk read IOPS**. Disk storage is sized exclusively for write persistence and retention cleanup.

---

### 2.3 Operational Best Practices & Cross-References

1. **Mount Options**: Format persistent disk volumes with `ext4` or `xfs` using the `noatime` mount option to prevent access-time write amplification.
2. **Single-Stream Fsync Ceiling**: A single stream's persistent write throughput is physically capped at **$\le 7,000\text{ msgs/s}$** by physical drive fsync serialization. Workloads exceeding 7,000 msgs/s must be horizontally partitioned across multiple streams (see `docs/infra_calculator/guardrails.md` Guardrail 2).
3. **Cold Read Isolation**: When consumer lag volume exceeds spare host RAM, reads fall back to physical disk, causing disk contention. Direct long catch-up replays to dedicated off-peak windows (see `docs/infra_calculator/guardrails.md` Guardrail 4).
