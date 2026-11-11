# NATS Benchmark Telemetry & Sizing Guide: Metrics, Calculations, and Infrastructure Planning

This guide is a comprehensive technical reference for understanding the metrics, mathematical formulas, operational dynamics, and infrastructure capacity sizing standards produced by the NATS Capacity Benchmark suite.

---

## 1. Workload & Throughput Metrics

### 1.1 Target Rate vs. Achieved Rate

* **Target Rate (Offered Load)**:
  The rate at which the benchmark client publishers attempt to inject messages into NATS, configured via the `--throughput <RATE>` flag in `nats bench`.
  * *Example*: `target_rate = 100,000 msgs/sec`. The client timer attempts to dispatch 100,000 messages every second across its concurrent publisher routines.
* **Achieved Rate (Delivered Throughput)**:
  The actual, empirical rate of messages successfully published through the cluster over the duration of the test:
  $$\text{Achieved Rate (msgs/sec)} = \frac{\text{Total Messages Published}}{\text{Elapsed Duration (seconds)}}$$
  * *Example*: In `TEST-CL-100K`, the target was 100,000 msgs/sec, but the achieved rate was `60,219 msgs/sec`.

#### Who or what hinders the Achieved Rate?
When Achieved Rate falls below Target Rate, the publisher is being throttled by one of three physical bottlenecks:

1. **TCP Socket Send Buffer Backpressure (`SO_SNDBUF`)**:
   In Go, the NATS client invokes the standard library `net.Conn.Write()`. When the publisher attempts to send faster than the network interface (NIC) can transmit or faster than the broker can read, the operating system kernel socket send buffer fills completely. Once full, **the client's `Write()` call synchronously blocks**. The client's rate-limiter is paused because the thread is suspended in a kernel I/O wait state. (This bottleneck was observed when 32 KiB payloads hit the 103 MB/s loopback ceiling).
2. **Synchronous Disk Fsync & Raft Consensus (JetStream Storage)**:
   In synchronous JetStream (`js-pub-sync`), the publisher sends a message and **halts until it receives an explicit acknowledgment (`PUB-ACK`) from the server**. The broker cannot issue that acknowledgment until the message has been written to disk via an operating system `fsync()` and confirmed by a Raft quorum of peer replicas (2 out of 3 nodes). If the physical storage subsystem caps at ~7,000 fsync operations per second, the client cannot publish faster than 7,000 msgs/sec.
3. **Broker Event Loop Saturation**:
   If the NATS broker's CPU reaches saturation or lock contention arises, the server's TCP read loop slows down. The broker reduces its TCP receive window (`TCP window update = 0`), which forces the client kernel to stop sending.

#### How it matters for overall sizing:
* **The Sizing Knee**: The point where Achieved Rate diverges from Target Rate marks the **hard capacity ceiling** of that hardware configuration.
* **Over-provisioning Danger**: Setting client publish rates above the achieved ceiling does not yield more throughput; it only increases client thread queueing, blows out tail latency, and triggers buffer bloat.

---

### 1.2 Delivery Efficiency (%)

Delivery Efficiency measures message integrity: of the messages that the publishers successfully pushed into the broker, what percentage reached subscriber applications?

#### Mathematical Formula:
$$\text{Expected Total Deliveries} = \begin{cases} \text{Published Messages} & \text{if Queue Group (1:1 load-balanced)} \\ \text{Published Messages} \times N_{\text{subscribers}} & \text{if Broadcast Fan-Out (1:N)} \end{cases}$$

$$\text{Delivery Efficiency (\%)} = \left(\frac{\text{Actual Received Messages}}{\text{Expected Total Deliveries}}\right) \times 100$$

#### Why is Delivery % often 100% even when Achieved Rate is less than Target Rate?
* **Throttling is not message loss.**
  If a publisher targets 10,000 msgs/s but disk fsync throttles it to 7,000 msgs/s, the publisher simply emits fewer messages (e.g. 210,000 messages instead of 300,000 in 30 seconds).
  However, **all 210,000 messages that were published were successfully stored, routed, and delivered to subscribers without loss**.
  Therefore, Delivery Efficiency is **100%**.
* **When does Delivery % drop below 100%?**
  Delivery drops below 100% only when messages are dropped in transit:
  1. *Core NATS Slow Consumers*: Under severe saturation (such as 100k broadcast), if subscribers cannot read from the socket fast enough, the server's pending buffer limits are exceeded and Core NATS drops messages to protect the broker.
  2. *Network packet drops*: Dropped frames on lossy or saturated physical links.
  3. *JetStream retention discards*: If stream limits (`max_bytes`, `max_msgs`) are breached and older messages are discarded before a slow consumer reads them.

#### How it matters for overall sizing:
* A cluster running at 100% delivery efficiency with sub-microsecond latency is operating inside its safe envelope.
* If Delivery % drops below 100% on Core NATS, the cluster is in an active buffer-overflow state. You must either scale the broker, partition topics across more nodes, or migrate subscribers to Queue Groups.

---

### 1.3 Bandwidth (MB/s) and Payload Dimension

* **Bandwidth (MB/s)**:
  The rate of raw payload data transferred through the network interfaces by publishers in megabytes per second:
  $$\text{Bandwidth (MB/s)} = \frac{\text{Throughput (msgs/sec)} \times \text{Payload Size (bytes)}}{1024 \times 1024}$$
* **Payload Size**:
  The discrete size in bytes of an individual message body (`SIZE`, e.g. `1024B`, `8KB`, `32KB`).

#### Why Payload is displayed under Bandwidth:
Bandwidth is the direct mathematical product of Achieved Throughput and Payload Size:
* Publishing **30,000 msgs/sec** with a **1 KiB (1,024 B)** payload generates **29.30 MB/s** (234.4 Mbps).
* Publishing the same **30,000 msgs/sec** with an **8 KiB (8,192 B)** payload generates **234.38 MB/s** (1.87 Gbps).
* Publishing only **3,000 msgs/sec** with a **32 KiB (32,768 B)** payload generates **93.75 MB/s** (750.0 Mbps).

Displaying Payload directly beneath Bandwidth provides immediate context for network capacity utilization: it reveals whether low message throughput is caused by large payloads exhausting network interface limits, or if high message throughput is operating well within network capacity.

#### How it matters for overall sizing:
* **Pinpointing the Physical Constraint**: CPU-bound workloads are characterized by high message throughput with low bandwidth (e.g. 100,000 msgs/s of 128 B = 12.2 MB/s). Network-bound workloads are characterized by low message throughput with high bandwidth (e.g. 3,300 msgs/s of 32 KiB = 103 MB/s).
* **Network Interface Card (NIC) Sizing**: When sizing bare-metal or cloud instances (e.g. AWS c6i.xlarge with 12.5 Gbps vs standard 1 Gbps), bandwidth calculations dictate whether standard 1 GbE interfaces are sufficient or if 10 GbE/25 GbE interfaces are required. A 3-node cluster generating 100 MB/s ingress and 400 MB/s broadcast egress will immediately saturate a 1 Gbps NIC.

---

## 2. Latency Distribution Metrics

The benchmark measures end-to-end operation latency from the client's perspective across standard percentiles:
* **P50 (Median)**: The latency experienced by 50% of messages. Represents normal, uncontended processing time.
* **P90**: 90th percentile processing time under minor queueing.
* **P99 (Tail Latency)**: 99th percentile processing time. Reflects OS thread scheduling delays, minor network contention, or GC pauses.
* **P99.9**: Extreme tail latency experienced by 1 in 1,000 messages.
* **Max Latency**: The single worst observed latency event during the entire run.

### 2.1 The "Low-Throughput Latency Penalty"
In Core NATS, P50 latency at 1,000 msgs/s (~2.2 us) is often *higher* than at 25,000 msgs/s (~0.24 us).
* **Reason**: At low throughput (e.g. 1 message every 4 ms per client), the OS TCP stack and the Go runtime do not batch packets. The socket buffer is emptied after each message, forcing repeated thread context switches between user-space and kernel-space.
* At higher throughput (10k - 50k msgs/s), Go goroutines and kernel socket buffers remain continuously primed. Multiple messages are coalesced into single TCP frames and processed in a hot CPU cache, lowering per-message latency by up to 10x.

### 2.2 The "Inverted Latency Curve" in JetStream File Storage
In synchronous JetStream File publishing (`js-pub-sync`), latency displays a pronounced inverted curve:
* At **250 msgs/sec**: P50 latency is **4.32 ms**.
* At **1,000 msgs/sec**: P50 latency drops to **2.85 ms**.
* At **5,000 msgs/sec**: P50 latency drops to **0.93 ms (sub-millisecond)**.
* **Reason**: JetStream automatically batches concurrent publish writes into shared disk `fsync()` blocks and coalesced Raft quorum rounds. At 250 msgs/s, each message arrives alone and must pay the full rotational/SSD sync penalty (3-4 ms). At 5,000 msgs/s, dozens of concurrent messages share a single disk flush, reducing the amortized per-message commit time.

#### How it matters for overall sizing:
* **SLA Boundaries**: Enterprise SLAs are typically defined at the P99 or P99.9 level. Sizing cannot be based on P50 alone.
* **Early Warning Indicator**: When a cluster begins to saturate, P50 latency often remains deceptively fast (nanoseconds for local socket writes), while **P99 and P99.9 latency explodes into the tens or hundreds of milliseconds**. A rising P99 is the first mathematical signal that TCP transmit buffers are backing up.

---

### 2.3 Latency Distribution Quantiles, Cumulative Volume, and Standard Deviation

The benchmark provides a detailed breakdown of client operation latency percentiles and statistical dispersion:

#### 1. Percentiles as Quantile Thresholds:
Each percentile row represents a boundary value $L_p$ such that $p\%$ of observed message latencies were $\le L_p$:
* **Min**: Fastest recorded message ($1^{\text{st}}$ sample).
* **Median (P50)**: Median processing time (50% of messages).
* **P90**: 90% of messages completed within this time.
* **P99 (Tail Latency)**: 99% of messages completed within this time (1 in 100 outlier).
* **P99.9 (Extreme Tail)**: 99.9% of messages completed within this time (1 in 1,000 outlier).
* **Max**: Absolute worst-case latency event recorded during the test (100% of messages).

#### 2. Cumulative Deliveries:
Cumulative Deliveries represent the exact count of messages that fell within or below that percentile boundary:
$$\text{Cumulative Deliveries}(P_k) = \text{Round}\left(\text{Total Published Messages} \times \frac{k}{100}\right)$$

* *Example with 300,000 Published Messages*:
  * **Min Latency**: 1 message ($< 0.01\%$)
  * **Median (P50)**: 150,000 messages (50.0%)
  * **P90 Latency**: 270,000 messages (90.0%)
  * **P99 Latency**: 297,000 messages (99.0%)
  * **P99.9 Latency**: 299,700 messages (99.9%)
  * **Max Latency**: 300,000 messages (100.0%)

#### 3. Standard Deviation ($\sigma$):
Standard deviation measures the dispersion or spread of message latencies around the arithmetic mean ($\mu$):
$$\sigma = \sqrt{\frac{1}{N} \sum_{i=1}^{N} (x_i - \mu)^2}$$

* **Why Standard Deviation is a Global Metric, Not a Per-Percentile Column**:
  Standard deviation is a single statistical property of the entire distribution dataset. A single percentile threshold (like P50 or P99) does not have its own separate standard deviation.
  * In naive reporting tables, placing "Standard Deviation" as a table column results in empty dashes (`-`) across all percentile rows except one.
  * The correct architectural reporting standard is to present **Total Samples**, **Mean Latency**, and **Standard Deviation** in a summary header bar, reserving the table exclusively for quantile thresholds and cumulative volume slices.

#### How it matters for overall sizing:
* **Jitter vs. Steady State**: A large standard deviation relative to mean latency (e.g. Mean = 0.86 us, Std Dev = 52.95 us) indicates high jitter caused by intermittent OS thread preemption, GC cycles, or lock contention.
* **SLA Verification**: Production capacity sizing must verify that the cumulative message count exceeding the SLA threshold (e.g. total messages above P99) matches acceptable business error budgets.

---

## 3. Cluster Resource Telemetry: CPU & Memory

### 3.1 CPU Utilization: Average vs. Peak

* **Calculation Method**:
  Container CPU is derived from cumulative CPU time (`cpu_time` reported by the container runtime):
  $$\Delta t = \text{Current Timestamp} - \text{Previous Timestamp}$$
  $$\Delta \text{cpu\_time} = \text{Cumulative CPU Time}_{\text{now}} - \text{Cumulative CPU Time}_{\text{prev}}$$
  $$\text{Interval CPU \%} = \left(\frac{\Delta \text{cpu\_time}}{\Delta t}\right) \times 100$$
* **Average CPU (%)**: The arithmetic mean of CPU utilization across the active test window.
* **Peak CPU (%)**: The maximum single sample observed across the active test window.

#### How it matters for overall sizing:
* **Average CPU** sizes baseline operational costs and steady-state infrastructure sizing.
* **Peak CPU** establishes the cluster's safety margin:
  $$\text{CPU Headroom (\%)} = 100.0 - \text{Cluster Peak CPU \%}$$
* In enterprise production environments (e.g. OpenShift, Kubernetes), clusters should be sized so that peak workload CPU does not exceed **60% to 70%**, leaving at least 30% to 40% headroom for sudden traffic spikes or node failover re-balancing.

---

### 3.2 CPU Efficiency (msgs/sec per Core)

CPU Efficiency quantifies the computational economy of the messaging engine:
$$\text{CPU Efficiency} = \frac{\text{Achieved Throughput (msgs/sec)}}{\frac{\text{Average Cluster CPU \%}}{100}}$$

#### What the number means:
It represents: **"How many messages per second does the cluster route for every 1.0 full CPU core (100% CPU) consumed?"**

* *At 1k msgs/s*: CPU Efficiency is **~15,500 msgs/core**.
* *At 50k msgs/s*: CPU Efficiency climbs to **~157,600 msgs/core** (a 10x improvement in computational efficiency).

#### How it matters for overall sizing:
* CPU Efficiency enables precise capacity forecasting:
  $$\text{Estimated Cores Needed} = \frac{\text{Target Production Throughput}}{\text{CPU Efficiency at Expected Load}} \times \text{Safety Factor (e.g. 1.5)}$$
* If an enterprise plans to process 300,000 msgs/sec of Core NATS traffic with 1 KiB payloads, and the benchmark demonstrates an efficiency of 150,000 msgs/core at high load:
  $$\text{Estimated Cores} = \left(\frac{300,000}{150,000}\right) \times 1.5 = 3.0\text{ CPU Cores across the cluster}$$

---

### 3.3 Memory Footprint: Peak RAM vs. Average RAM

The benchmark reports four memory metrics: Baseline RAM, Average RAM, Peak RAM, and Memory Growth (Delta).

#### Why the Executive Matrix shows Peak RAM instead of Average RAM:
1. **Linux Cgroup & Go Runtime Mechanics**: Memory in Go applications is "sticky." Once the Go runtime requests memory pages from the OS (via `mmap`) to expand socket buffers or stream caches, it does not instantly return those pages back to the operating system when the burst subsides.
2. **Kubernetes OOM Killer Protection**: In containerized environments, **Peak RAM is the only metric that dictates container limits**. If you size Pod memory limits (`resources.limits.memory`) based on "Average RAM", the Linux Out-Of-Memory (OOM) killer will immediately terminate your NATS broker the first time a traffic spike causes buffer bloat.

#### Memory Growth as a Diagnostic for Server Buffer Bloat:
* In healthy Core NATS operation, memory remains near-flat:
  * 1k msgs/s: Peak RAM ~43 MB (+12 MB delta).
  * 50k msgs/s: Peak RAM ~46 MB (+3.8 MB delta).
* Under saturation (100k msgs/s), cluster RAM ballooned to **423.9 MB (+381.6 MB delta)**!
* **Architectural Inference**: Memory growth on a Core NATS broker is a direct physical measure of **unconsumed messages trapped in server socket buffers**. When subscribers lag behind publishers, the broker buffers data in RAM.

#### How it matters for overall sizing:
* **Pod Memory Request**: Set to `Baseline RAM + Normal Workload Buffer` (e.g. 256 MB per node for Core NATS; 1 GB to 2 GB for JetStream).
* **Pod Memory Limit**: Set to at least **2x to 3x baseline** to absorb temporary subscriber lag without being killed by the OS.
* **Alert Rule**: Trigger a high-priority operational alert if Core NATS container RAM grows by more than 100 MB above baseline, indicating that slow consumers are falling behind.

---

## 4. Network & Storage Telemetry: I/O Dynamics

### 4.1 Network RX vs. TX (Ingress vs. Egress)

* **RX (Receive / Ingress)**: Network bandwidth received *into* the container or network interface.
* **TX (Transmit / Egress)**: Network bandwidth sent *out of* the container or network interface.

#### The Broadcast Fan-Out Amplification Formula:
In a publish/subscribe system without queue groups, every subscriber receives a distinct copy of every message:
$$\text{Cluster Egress Bandwidth (TX)} \approx \text{Cluster Ingress Bandwidth (RX)} \times N_{\text{subscribers}}$$

* *At 50k msgs/s with 4 subscribers (1 KiB payload)*:
  * Ingress (RX) = **48.8 MB/s**.
  * Egress (TX) = **195.2 MB/s** across the cluster.
* *At 100k target with 8 subscribers*:
  * Ingress (RX) = **100 MB/s**.
  * Egress (TX) would require **800 MB/s (6.4 Gbps)**!

#### How it matters for overall sizing:
* **Network Sizing Wall**: Network egress (TX), not CPU or memory, is the primary physical bottleneck for high-concurrency Core NATS broadcast clusters.
* **Architectural Pattern**: Whenever subscriber counts grow beyond 2-4 clients at rates exceeding 25,000 msgs/sec, applications must migrate from broadcast pub/sub to **Queue Groups (`QUEUE=worker_group`)**. Queue groups load-balance deliveries across workers ($1\times\text{ fan-out}$), capping egress bandwidth at $1\times\text{ ingress}$.

---

### 4.2 Storage Block I/O and Disk Growth

#### Definition and Measurement Methodology:
* **Disk Growth (Bytes)**:
  The net change in on-disk storage footprint of the persistent data directory for each NATS node (`data/nats-1`, `data/nats-2`, `data/nats-3`) during the benchmark execution window:
  $$\text{Disk Growth (bytes)} = \text{DirSize}_{\text{post-test}} - \text{DirSize}_{\text{pre-test}}$$
  Pre-test and post-test directory sizes are captured via filesystem inspection (`du -sk`) across the container volume mounts.
* **Block Write Rate vs. Disk Growth**:
  * **Block Write Rate (`disk_write_mb_sec`)**: The continuous I/O throughput rate at which the operating system flushes blocks to the storage subsystem (including write-ahead logs, Raft commit logs, and directory metadata).
  * **Disk Growth (`disk_growth_bytes`)**: The net retained file volume that remains stored on disk after the workload concludes.
* **The Host Bind-Mount Cgroup Limiting Factor**:
  In containerized deployments (such as Podman or Docker Compose), NATS node storage directories are mounted from host directories (e.g. `./data/nats-1:/data`). The Linux kernel cgroup `blkio` (or `io` in cgroup v2) controller monitors block operations on virtual block devices (such as `/dev/vda`). Writes directed to host bind-mounted filesystems traverse the host Virtual File System (VFS) layer and bypass container cgroup accounting. Consequently, `podman stats` reports `BLOCK I/O: 0B / 0B` continuously.
* **Empirical Write Rate Formulation**:
  When container runtime block counters report `0.00 MB/s` due to host volume mounts, the benchmark derives the true physical disk write rate from physical directory growth:
  $$\text{Empirical Write Rate (MB/s)} = \frac{\text{Disk Growth (bytes)}}{1024 \times 1024 \times \text{Duration (seconds)}}$$
* **Auto-Scaled Unit Progression for Capacity Sizing**:
  Reporting disk growth in locked units (such as hardcoded `KiB`) obscures the magnitude of data persistence (e.g. displaying 928 MB as `950649.3 KiB`). The benchmark standard dynamically scales units:
  * **B** ($< 1\text{ KiB}$): Empty baseline or Core NATS in-memory runs.
  * **KiB** ($< 1\text{ MB}$): Stream headers, cluster metadata, and raft state initialization.
  * **MB** ($< 1\text{ GB}$): Standard operational benchmark runs and saturation steps.
  * **GB** ($\ge 1\text{ GB}$): Sustained multi-minute stress tests and high-throughput production runs.

#### Behavior Across NATS Messaging Engines:
* **Core NATS**: Strictly in-memory pub/sub routing. Messages are never written to disk. Confirmed at `disk_growth_bytes: 0 B` and `disk_write_mb_sec: 0.0 MB/s` across all nodes.
* **JetStream Memory Storage (`--storage memory`)**: Stream messages and consumer state are held exclusively in volatile RAM. Storage disk growth is `0 B`.
* **JetStream File Storage R=1 (Single Replica)**:
  * Only the single node designated as the JetStream stream leader writes message blocks and indices to disk (~32.0 MB per 30,000 messages of 1 KiB).
  * Peer cluster nodes record **`0 B` disk growth**, as they do not participate in storage for that stream.
* **JetStream File Storage R=3 (Three Replicas with Raft Consensus)**:
  * Messages and Raft state logs are synchronously replicated across all 3 cluster nodes.
  * Every node in the quorum writes identical byte volumes (**~32.7 MB per node, totaling 98.0 MB across the cluster** for 30,000 messages of 1 KiB).

#### Disk Sizing Standard for JetStream:
$$\text{Daily Disk Growth (GB/day)} = \left(\frac{\text{Throughput (msgs/sec)} \times \text{Payload Size (bytes)} \times \text{Replicas}}{10^9}\right) \times 86,400 \times 1.05$$

* At **1,000 msgs/sec** of 1 KiB messages with R=3:
  $$\text{Storage} = \left(\frac{1000 \times 1024 \times 3}{10^9}\right) \times 86,400 \times 1.05 \approx \mathbf{278\text{ GB / day}}$$
* At **5,000 msgs/sec** with R=3:
  $$\text{Storage} \approx \mathbf{1.39\text{ TB / day}}$$

#### How it matters for overall sizing:
1. **Retention Limits are Mandatory**: JetStream streams must define retention limits (`max_bytes`, `max_age`) to prevent unbounded disk growth.
2. **IOPS & Fsync Requirements**: Synchronous JetStream clusters must be deployed on **local NVMe SSDs or high-IOPS provisioned storage (e.g. AWS EBS gp3/io2 with dedicated IOPS)**. Standard network-attached storage with high fsync latency (> 5 ms) will throttle cluster throughput below 2,000 msgs/sec.

---

## 5. Enterprise Capacity Planning & Sizing Matrix

Synthesizing all benchmark dimensions yields the following operational sizing boundaries for a 3-node NATS cluster:

| Messaging Engine | Safe Operating Limit | Sizing Knee Boundary | Hard Saturation Ceiling | Primary Limiting Bottleneck |
| :--- | :--- | :--- | :--- | :--- |
| **Core NATS (1 KiB)** | **35,000 msgs/sec** | 50,000 msgs/sec | 60,219 msgs/sec | Network Egress & TCP socket buffer bloat |
| **Core NATS (32 KiB)**| **2,500 msgs/sec** | 3,000 msgs/sec | 3,302 msgs/sec | Physical Network Interface Bandwidth (1 Gbps) |
| **JetStream Memory R=3**| **25,000 msgs/sec** | 35,000 msgs/sec | 40,000 msgs/sec | Cluster RAM quota (`max_mem_store`) & Raft CPU |
| **JetStream File R=3** | **5,000 msgs/sec** | 6,000 msgs/sec | 7,003 msgs/sec | Storage `fsync()` write latency & Raft quorum |

---

## 6. Sizing Rules of Thumb for Platform Engineers

1. **The 70% Network Bandwidth Rule**:
   Do not plan steady-state traffic above 70% of network interface capacity (e.g. cap traffic at 70 MB/s on a 1 Gbps link). Exceeding 80% causes TCP send buffer queueing that inflates P99 tail latency from microseconds to tens of milliseconds.
2. **The 8 KiB Payload Standard (Claim-Check Pattern)**:
   Keep messaging payloads $\le 8\text{ KiB}$ to ensure sub-millisecond P99 latency. Payloads $> 32\text{ KiB}$ should be stored in an object store (e.g. NATS Object Store or S3), passing only an 8-byte reference over the messaging topic.
3. **The JetStream Durability Tax**:
   Budget for an **8x to 10x throughput reduction** and a **1,000x latency expansion** (microseconds to milliseconds) when migrating a workload from in-memory Core NATS to durable JetStream File R=3 persistence.
4. **Queue Groups for High Fan-Out**:
   Always specify `QUEUE=<name>` for consumers when message rates exceed 25,000 msgs/sec to prevent $N\times$ network egress multiplication from exhausting broker network interfaces.

