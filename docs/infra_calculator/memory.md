# Memory (RAM) Capacity Planning and Sizing Model

## 1. Impact of Variables and Constants on Memory (RAM)

In enterprise message broker architecture, **Memory (RAM) is an allocation and state-tracking resource**. The NATS server manages all active routing and consensus structures inside the Go runtime process heap, while delegating file block caching to the underlying operating system.

### 1.1 Workload Input Variables (Audit of All 16 Variables)

| Input Variable | Category | Included in RAM? | Role in Memory Sizing | Physical Justification |
|---|---|:---:|---|---|
| **`n_consumer_peak`**| Resources | **YES** | **Primary Driver (Go Heap)** | Peak active durable consumers. Each consumer maintains stateful cursor tracking, pending ACK bitsets, and delivery sequence journals in Go heap (~21.9 MB cluster-wide, ~7.3 MB per node). |
| **`n_subj`** | Resources | **YES** | **Secondary Driver (Trie Heap)** | Total registered subject taxonomy cardinality. Each subject path registers a prefix trie node in the sublist (~940 B cluster-wide, ~313 B per node). |
| **`n_stream`** | Resources | **YES** | **Secondary Driver (Raft Heap)** | Number of active JetStream streams. Each stream maintains an independent Raft consensus state machine, log index slices, and leader leases (~702 KiB cluster-wide, ~234 KiB per node). |
| **`n_conn_peak`** | Connections | **YES** | **Secondary Driver (Socket RAM)** | Peak concurrent client TCP connections distributed across nodes ($\approx n_{\text{conn\_peak}} / 3$). Sizes kernel TCP read/write socket buffers (`SO_RCVBUF`, `SO_SNDBUF`, ~64 KiB per socket). |
| **`payload_peak`** | Message | **YES** | **Socket Buffer Ceiling** | Peak burst message size. Sizes maximum temporary socket read buffer allocations before frame dispatch. |
| **`header`** | Message | **NO** | Excluded (Wire Only) | Tracing headers and metadata are read and dispatched directly from socket buffers without accumulating in resident process memory. |
| **`payload_avg`** | Message | **NO** | Excluded (Disk Volume) | Average payload size governs accumulated persistent disk storage bytes ($D_{\text{node}}$). It does not affect resident process memory. |
| **`x_js_avg`** | Throughput | **NO** | Excluded (Disk Volume) | Average persistent ingress rate sizes accumulated storage volume over time ($t$). It does not expand the resident Go heap. |
| **`x_js_peak`** | Throughput | **NO** | Excluded (CPU Limit) | Peak write throughput drives CPU execution cycles and disk write IOPS, not permanent resident process memory. |
| **`c_js`** | Fan-Out | **NO** | Excluded (Throughput Driver) | Consumer fan-out multiplies CPU delivery and ACK consensus, while memory state is governed by the number of registered consumer cursors (`n_consumer_peak`). |
| **`x_core_avg`** | Throughput | **NO** | Excluded (Ephemeral) | Core NATS routing is ephemeral and in-memory. Messages are written directly to subscriber TCP socket buffers without accumulating in resident heap memory. |
| **`x_core_peak`** | Throughput | **NO** | Excluded (Ephemeral) | Peak ephemeral routing sizes CPU cycles, not resident process memory. |
| **`c_core`** | Fan-Out | **NO** | Excluded (Zero Buffer) | Core broadcast delivery writes directly to existing subscriber sockets without maintaining cursor state in RAM. |
| **`n_conn_avg`** | Connections | **NO** | Excluded (Peak Sized) | Memory limits must be provisioned for peak socket connection load (`n_conn_peak`) to prevent kernel socket buffer exhaustion. |
| **`n_consumer_avg`** | Resources | **NO** | Excluded (Peak Sized) | Memory limits must be provisioned for peak durable consumer count (`n_consumer_peak`) to prevent Go heap OOM kills during scale-out. |
| **`t`** | Retention | **NO** | Excluded (Disk Storage) | Message retention over days or weeks governs accumulated Persistent Disk storage ($D_{\text{node}}$). Messages on disk do not occupy resident process RAM. |

---

### 1.2 Standard System Constants (Audit of All 7 Constants)

| System Constant | Default Value | Included in RAM? | Role in Memory Sizing | Physical Justification |
|---|---|:---:|---|---|
| **`HEADROOM`** | **30%** (0.30) | **YES** | **Sizing Divisor** | Divides raw process RAM demand by $1 - \text{HEADROOM} = 0.70$. Absorbs Go GC garbage accumulation spikes (`GOGC=100`), OS kernel overhead, and guarantees $N-1$ cluster failover survivability. |
| **`BATCH`** | **500 msgs** | **NO** | Excluded (CPU Divisor) | Amortizes pull fetch RPC requests. Consumer cursor tracking memory depends on pending unacknowledged message state, not batch fetch size. |
| **`S_PROTO`** | **32 bytes** | **NO** | Excluded (Wire / Disk) | Wire framing overhead per message line. Sized under Network Bandwidth and Disk Storage. |
| **`T_RAFT`** | **250 ms** (4 ops/s) | **NO** | Excluded (CPU Rate) | Raft leader heartbeat timer consumes CPU execution cycles, with zero incremental memory footprint. |
| **`T_ROUTE`** | **2 seconds** (0.5 ops/s) | **NO** | Excluded (CPU Rate) | Inter-node TCP route keepalives consume negligible static buffer memory already captured in the baseline floor. |
| **`T_PING`** | **60 seconds** (1 op/min) | **NO** | Excluded (CPU Rate) | Client ping processing consumes CPU, while socket buffer memory is governed by `n_conn_peak`. |
| **`T_HB`** | **5 seconds** (0.2 ops/s) | **NO** | Excluded (Idle-Only) | Consumer idle heartbeats are suppressed during active message flow and require zero dedicated memory allocations. |

---

## 2. Deduction of Mathematical Model

Memory sizing calculates the **Go Process Resident Memory** required for safe, OOM-free cluster operation:

$$\text{RAM}_{\text{provisioned}} = \frac{\text{RAM}_{\text{process, node}}}{1 - \text{HEADROOM}} = \frac{\text{RAM}_{\text{process, node}}}{0.70}$$

---

### Process Resident Memory Formulation ($\text{RAM}_{\text{process, node}}$)

The Go runtime resident footprint per node is deduced from empirical benchmark measurements across isolated resource dimensions:

$$\text{RAM}_{\text{process, node}} = \text{RAM}_{\text{base}} + (\beta_{\text{subj}} \cdot n_{\text{subj}}) + (\beta_{\text{stream}} \cdot n_{\text{stream}}) + (\beta_{\text{consumer}} \cdot n_{\text{consumer\_peak}}) + \left(\beta_{\text{conn}} \cdot \frac{n_{\text{conn\_peak}}}{3}\right)$$

#### 1. Baseline Go Process Floor ($\text{RAM}_{\text{base}}$)
* **Empirical Observation**: Before injecting workload traffic, a 3-node NATS cluster with JetStream enabled consumes ~64 MB to ~96 MB of resident RSS per node. This covers the Go runtime allocator, mcache/mcentral span structures, runtime poller threads, internal monitoring state, and pre-allocated ring buffers.
* **Model Coefficient**: Sizing a conservative baseline floor absorbs process startup overhead:
  $$\text{RAM}_{\text{base}} = 128 \text{ MB}$$

#### 2. Sublist Prefix Trie Memory ($\beta_{\text{subj}}$)
* **Benchmark Evidence (`subject-cardinality`)**: Scaling registered subjects from 100 to 10,000 subjects produced a deterministic, linear memory expansion from 72.2 MB to 81.5 MB (+9.3 MB across 9,900 subjects).
* **Deduction**: Each registered subject creates a prefix trie node in the sublist structure containing token string pointers, atomic subscription slices, and child pointer arrays:
  $$\beta_{\text{subj, cluster}} \approx 940 \text{ bytes per subject cluster-wide}$$
  $$\beta_{\text{subj}} = \frac{940\text{ B}}{3} \approx 313 \text{ bytes per subject per node}$$
* *Example*: 10,000 enterprise subjects consume only $10,000 \times 313\text{ B} \approx 3.1\text{ MB}$ of RAM per node.

#### 3. Multi-Raft Stream Descriptors ($\beta_{\text{stream}}$)
* **Benchmark Evidence (`stream-scaling`)**: Scaling from 1 stream to 50 streams (with 49 idle auxiliary streams) increased cluster peak RAM from 239.6 MB to 274.0 MB (+34.4 MB across 49 streams).
* **Deduction**: Each active stream instantiates an independent Multi-Raft consensus group, in-memory log index pointers, term/vote state, and stream configuration structures:
  $$\beta_{\text{stream, cluster}} \approx 702 \text{ KiB per stream cluster-wide}$$
  $$\beta_{\text{stream}} = \frac{702\text{ KiB}}{3} \approx 234 \text{ KiB per stream per node} \quad (0.23 \text{ MB per stream})$$
* *Example*: 20 independent business streams consume $20 \times 0.23\text{ MB} \approx 4.6\text{ MB}$ of RAM per node.

#### 4. Durable Consumer Progress State ($\beta_{\text{consumer}}$)
* **Benchmark Evidence (`consumer-fanout`)**: Scaling concurrent durable consumers from 1 to 20 against an active stream expanded cluster peak RAM from 253.1 MB to 668.9 MB (+415.8 MB across 19 consumers).
* **Deduction**: Each durable consumer tracks delivery sequence maps, pending unacknowledged message bitsets, redelivery timers, and Raft consumer journal buffers:
  $$\beta_{\text{consumer, cluster}} \approx 21.88 \text{ MB per consumer cluster-wide}$$
  $$\beta_{\text{consumer}} = \frac{21.88\text{ MB}}{3} \approx 7.29 \text{ MB per consumer per node}$$
* *Example*: 10 active durable consumers consume $10 \times 7.29\text{ MB} \approx 72.9\text{ MB}$ of RAM per node.

#### 5. Client TCP Socket Buffers ($\beta_{\text{conn}}$)
* **System Mechanism**: Each client connection terminates a TCP socket. The Linux kernel allocates transmit and receive socket buffers (`SO_SNDBUF`, `SO_RCVBUF`), while NATS maintains an associated client struct with inbound read framing buffers.
* **Model Coefficient**: Sizing 64 KiB per client socket covers kernel buffers and client connection descriptors. With connections distributed across nodes via HAProxy:
  $$\beta_{\text{conn}} = 64 \text{ KiB per connection} = 0.0625 \text{ MB per connection}$$
* *Example*: 300 peak connections (100 per node) consume $100 \times 0.0625\text{ MB} \approx 6.25\text{ MB}$ of RAM per node.

---

### How Linux Page Cache Is Satisfied via Cloud VM Packaging

In enterprise infrastructure deployments, operating system Page Cache is not managed through arbitrary mathematical timers or manual kernel parameter tweaking. Instead, it is satisfied **automatically through standard cloud instance packaging**:

1. **Compute Dictates the Instance Tier**:
   Our CPU capacity model sizes the required compute cores (e.g., **2 vCPU**).
2. **Cloud Instances Bundle RAM in Fixed Ratios (1:2 or 1:4)**:
   Commercial cloud VM catalogs (AWS, GCP, Azure) bundle memory at standard packaging ratios:
   * **Compute-Optimized (1:2 ratio)**: 2 vCPU bundles **4 GB RAM** (e.g. AWS `c6i.large`, GCP `c2-standard-4`).
   * **General-Purpose (1:4 ratio)**: 2 vCPU bundles **8 GB RAM** (e.g. AWS `m6i.large`, GCP `n2-standard-2`).
3. **The Natural Page Cache Buffer**:
   Because the NATS Go process only occupies **~300 MB to 650 MB** of resident RAM, the remaining **3.5 GB to 7.5 GB** of the VM is automatically and passively utilized by the Linux kernel as Page Cache.
   At a typical ingress rate of 2.5 MB/s (2,500 msgs/s at 1 KiB), 7 GB of spare RAM holds **over 45 minutes of message data in memory**. This provides abundant cache capacity, guaranteeing that real-time consumers read directly from RAM with **zero physical disk read IOPS**.
