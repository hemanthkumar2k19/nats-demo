# Capacity Planning Notes & Architectural Rationale

This document captures architectural design rationale, empirical observations, and parameter sensitivities for the NATS capacity sizing framework.

---

## 1. Why Subject Count Does Not Impact Routing Latency or CPU

In Core NATS and JetStream routing, subject cardinality ($n_{\text{subj}}$) does not degrade routing performance:

* **Algorithmic Invariance ($O(K)$)**: NATS organizes subjects into an internal prefix trie (the `sublist` data structure) split by the dot separator (`.`). Lookup time depends strictly on the token hierarchy depth ($K$), not the total number of registered subjects ($N$).
* **Empirical Validation**: Benchmarks across 1 to 10,000 subjects and wildcard patterns (`>`) demonstrated that median routing latency remains strictly invariant at $0.32\ \mu\text{s}$ to $0.35\ \mu\text{s}$, with no measurable CPU penalty.
* **Heap Memory Footprint**: Each registered subject allocates a small trie node in the Go runtime heap (~313 bytes per node, or ~940 bytes cluster-wide). Even 10,000 distinct subjects consume only ~3.1 MB of RAM per node.
* **Sizing Implication**: Subject count is a non-dominant parameter for CPU and latency sizing. It acts merely as a sanity check on baseline resident heap memory.

---

## 2. JetStream: Why Throughput x Payload Governs Sizing

Physical resource sizing for JetStream clusters is governed almost exclusively by data velocity ($\text{Throughput} \times \text{Payload Size}$) rather than subject cardinality:

* **Append-Only Block Storage**: JetStream stores messages sequentially in append-only segment files (`.blk`) partitioned by Stream, not by subject. Storage growth depends strictly on aggregate ingested bytes over the retention window ($x_{\text{js}} \times S \times t$).
* **Disk IOPS & Fsync Serialization**: Kernel write buffers and group-commit fsync operations flush batches of arriving messages. Whether 1,000 msgs/s arrive on 1 subject or across 5,000 subjects, the physical disk write and sync workload is identical.
* **Network NIC Bandwidth**: Line-rate wire consumption is strictly driven by ingress bytes and consumer egress replication ($\text{Throughput} \times \text{Payload} \times (1 + \text{Fan-Out})$).
* **Linux OS Page Cache**: Real-time consumer replay reads directly from memory pages maintained in spare host RAM, avoiding physical disk reads without requiring custom memory cache formulas.

### Architectural Exceptions Where Subject Count Matters
Subject cardinality becomes a sizing factor only under two specific configurations:
1. **`MaxMsgsPerSubject` / KV Buckets**: When a stream retains only the latest message per subject (e.g. KV stores or IoT state caches), NATS maintains an in-memory index mapping each subject to its latest sequence. Millions of subjects require dedicated Go heap memory (~100-200 bytes per key).
2. **Unbounded Ephemeral UUID Subjects**: Generating unique random subjects without cleanup causes continuous sublist trie heap expansion.

---

## 3. Resolving Sizing Intuition Paradoxes: 1 vCPU, ~300 MB RAM, ~800 GB Disk

When platform teams first review NATS JetStream sizing outputs, an immediate reaction is often:
*"How can a broker handling thousands of messages per second require only 1 vCPU and ~300 MB of RAM, while asking for over 800 GB of disk? Did we miss memory variables, or is disk wildly overestimated?"*

This section provides the physical mechanics and architectural rationale behind these numbers, structured as compact points for technical discussions and architecture review boards.

### 3.1 The Apparent Paradox: Why It Looks Counter-Intuitive

The skepticism stems from two common industry biases:
1. **JVM / Heavy-Broker Bias**: Engineers accustomed to Kafka, RabbitMQ, or ActiveMQ expect brokers to allocate 8 GB to 32 GB of heap memory per node for message buffers, object pools, and queue caches.
2. **Dimension Confusion (Velocity vs. Accumulation)**: Comparing CPU and Memory (which scale with instantaneous throughput per second) against Disk (which accumulates data continuously across days) makes storage appear disproportionately large.

---

### 3.2 Memory Reality: Why Go Process RAM is ~300 MB (No Missed Variables)

* **NATS Does NOT Buffer Messages in User-Space Heap**:
  In JetStream File Store (`StorageType: File`), incoming messages are read from kernel socket buffers directly into temporary reusable slice buffers (`sync.Pool`), written sequentially to append-only segment files (`.blk`), committed via `fsync()`, and released immediately. Message payloads never accumulate in the Go process heap.
* **Audit of Go Process Resident Memory (RSS)**:
  The ~203 MB raw demand (provisioned to ~291 MB with 30% Headroom) holds strictly metadata and active routing state:
  - **Go Runtime Baseline Floor (`RAM_base`)**: 128 MB (allocator, thread stacks, runtime poller, monitoring).
  - **Durable Consumers (`n_consumer_peak = 10`)**: 72.9 MB (~7.29 MB per consumer per node for sequence maps, unacknowledged delivery bitsets, and redelivery timers).
  - **Multi-Raft Streams (`n_stream = 5`)**: 1.1 MB (~234 KiB per stream per node for consensus state machines and log index pointers).
  - **Sublist Prefix Trie (`n_subj = 1,000`)**: 0.3 MB (~313 B per subject per node for token tree nodes).
  - **Client TCP Sockets (`n_conn_peak = 50`)**: 1.0 MB (~64 KiB per connection for kernel/app socket descriptors distributed across 3 nodes).
* **Process RSS vs. Host Machine RAM**:
  A node **cannot** run on a 512 MB host. If host RAM is restricted to match the process RSS, the Linux kernel has zero memory for **Page Cache**, forcing all consumer reads to fetch from physical disk and thrashing I/O.
* **How Cloud VM Packaging Solves This**:
  Compute sizing dictates the VM instance tier. Standard cloud VM catalogs bundle RAM at 1:2 or 1:4 ratios with vCPU:
  - Sizing for 1 or 2 vCPU provisions a VM with **2 GB or 4 GB of host RAM**.
  - The NATS Go process consumes **~291 MB**.
  - The remaining **~1.7 GB to ~3.7 GB is utilized by the Linux kernel as Page Cache**.
  - At 2,000 msgs/s avg (2.24 MB/s ingress), 3.7 GB of spare RAM holds over 27 minutes of active messages in memory, ensuring live consumers achieve **zero physical disk read IOPS**.

---

### 3.3 Storage Reality: Why Disk is ~800 GB (The 518 Million Message Accumulation)

* **Velocity vs. Time Accumulator**:
  - Compute (CPU) and Memory (RAM) handle the workload of **one single second** (e.g. 6,000 msgs/s peak). Once dispatched and ACKed, CPU drops to baseline and temporary buffers are recycled.
  - Storage (Disk) is a **continuous mathematical integral over time**:
    $$\text{Volume} = \int_{0}^{T} \text{Throughput}(t) \cdot \text{RecordSize} \cdot dt$$
* **The Raw Storage Math is Unforgiving**:
  - Ingress rate: `x_js_avg = 2,000 msgs/s`.
  - Effective record size: `1,024 B (payload) + 64 B (header) + 32 B (protocol) = 1,120 bytes`.
  - Retention window: `72 hours = 259,200 seconds` (3 continuous days).
  - **Total Message Count**:
    $$2,000\text{ msgs/s} \times 259,200\text{ s} = \mathbf{518,400,000\text{ messages (518.4 Million)}}$$
  - **Raw Stored Bytes**:
    $$518,400,000 \times 1,120\text{ B} = 580,608,000,000\text{ bytes} = \mathbf{580.6\text{ GB}} \ (540.7\text{ GiB})$$
* **Why 30% Headroom (829.4 GB Provisioned) is Mandatory**:
  Provisioning exactly 580 GB would cause operational failure on production storage:
  1. **Filesystem Extent Fragmentation**: Linux filesystems (`ext4`, `xfs`) suffer non-linear allocation latency spikes once disk utilization exceeds 80% to 85%, directly stalling synchronous `fsync()` operations.
  2. **OS Root Reservation**: The Linux kernel automatically reserves 5% of filesystem blocks strictly for root (`tune2fs`), reducing usable user capacity.
  3. **Segment Rollover and Asynchronous Cleanup**: JetStream writes into 8 MB or 16 MB `.blk` files. Expired segments are deleted in asynchronous background sweeps; during high ingestion, new blocks are allocated before old blocks are unlinked.
* **The Business Lever**:
  Disk storage is high solely because of the 72-hour retention requirement. Sizing scales linearly:
  - **72 hours (3 days)**: 580.6 GB raw -> **829.4 GB provisioned / node** (2.5 TB cluster).
  - **24 hours (1 day)**: 193.5 GB raw -> **276.5 GB provisioned / node** (830 GB cluster).
  - **12 hours (Half day)**: 96.8 GB raw -> **138.2 GB provisioned / node** (415 GB cluster).

---

### 3.4 Compute Reality: Why 1 vCPU is Sized and 2 vCPU is Deployed

* **Low CPU Overhead**: NATS is written in idiomatic Go with non-blocking epoll network loops, zero-copy socket reads, and lockless message fan-out. Processing 6,000 persistent msgs/s and 12,000 consumer deliveries/s requires only ~650 mCPU of raw compute.
* **Exact Provisioned**: $650 / 0.70 = 928.6\text{ mCPU} \approx 0.93\text{ vCPU}$.
* **Production Recommendation**: Standard cloud VM packaging does not offer sub-vCPU slices for bare VMs. Platform teams provision **2 vCPU** (e.g. AWS `c6i.large`), providing 50%+ CPU headroom to absorb kernel softirqs, TCP packet reassembly, and burst traffic.

---

### 3.5 Quick Architecture Review Cheat Sheet

| Question / Skepticism | Physical System Fact | Architecture Justification |
|---|---|---|
| **"Why is RAM so low (~300 MB)? Did we miss variables?"** | Message payloads are not cached in Go heap. | JetStream persists directly to disk via `.blk` files. Go RSS only tracks metadata (cursors, Raft terms, tries). |
| **"Can we deploy on a 512 MB VM then?"** | No. Host RAM must provide OS Page Cache. | Sizing 1-2 vCPU bundles 2-4 GB host RAM; the spare 1.7-3.7 GB serves as Page Cache for zero-disk consumer reads. |
| **"Why is Disk so large (>800 GB) for moderate TPS?"** | Time accumulation: 72 hours is 518 million messages. | 2,000 msgs/s x 1.12 KB x 259,200 s = 580.6 GB raw. 830 GB guarantees utilization stays <= 70% to prevent ext4/xfs fragmentation. |
| **"Can we reduce disk cost?"** | Yes, by adjusting the retention window ($t$). | Reducing retention from 72h to 24h cuts disk demand by 66% (from 830 GB down to 276 GB per node). |
| **"Is 1 vCPU realistic for 6,000 msgs/s?"** | Yes. NATS Go architecture is lightweight. | Raw compute is ~650 mCPU. Recommended rounded deployment is 2 vCPU for cloud VM packaging and burst headroom. |

---

### 3.6 Topology-Driven Sizing: Bridging Kubernetes Pods to NATS Mechanics

In production deployments, sizing parameters should not be guessed in isolation. They are mathematically bound to Kubernetes application topology:

1. **Client TCP Sockets (`n_conn_peak`) Derivation**:
   - Standard NATS client SDKs (Go, Java, Node.js, Python, C#) maintain **1 multiplexed TCP connection per pod/process**. Inside that connection, threads/goroutines publish and subscribe concurrently.
   - Total client sockets across the cluster is:
     $$n_{\text{conn\_peak}} = (n_{\text{pub\_services}} \cdot \text{pub\_replicas} + n_{\text{sub\_services}} \cdot \text{sub\_replicas}) \cdot \text{conn\_per\_pod} + n_{\text{conn\_admin}}$$
   - *Example*: 5 publishers (3 replicas = 15 pods) + 10 consumers (3 replicas = 30 pods) = 45 total pods. With 1 socket per pod + 5 admin/monitoring sockets, $n_{\text{conn\_peak}} = 50\text{ connections}$.

2. **Durable Consumers (`n_consumer_peak`) Derivation**:
   - Each downstream subscribing microservice requiring its own independent progress tracking creates a **Durable Consumer Group**.
   - With $n_{\text{stream}}$ active streams and an average of $c_{\text{js}}$ subscribing services per stream:
     $$n_{\text{consumer\_peak}} = n_{\text{stream}} \cdot c_{\text{js}}$$
   - *Example*: 5 business streams with an average fan-out of 2 subscribing services per stream yields $5 \times 2 = \mathbf{10\text{ durable consumers}}$.

3. **Traffic Peak Multipliers ($k_{\text{peak}}$)**:
   - Rather than guessing arbitrary raw peak numbers, peak velocity is modeled via business multipliers applied to steady-state average ingress:
     $$x_{\text{peak}} = x_{\text{avg}} \cdot k_{\text{peak}}$$
   - Realistic industry presets:
     - **2.0x (Banking / SaaS)**: Routine morning/lunch peak or salary day credit.
     - **3.0x (E-Commerce)**: Evening prime-time rush (8 PM - 10 PM) and weekend promotions.
     - **5.0x (Retail Peak)**: Black Friday, Cyber Monday, or festival season promotions.
     - **10.0x (Flash Sale)**: 10-15 minute ticket drops or product launch bursts.

4. **Taxonomy Ordering (Core NATS -> JetStream)**:
   - In architecture reviews, presenting **Core NATS (In-Memory Ephemeral)** before **JetStream (Persistent Multi-Raft Quorum)** reflects the natural progression of message-routing taxonomy: from lightweight broadcast pub/sub to persistent, consensus-backed event streaming.


