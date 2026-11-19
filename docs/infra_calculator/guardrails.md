# Operational Boundaries & Physical Sizing Guardrails

This document catalogs the non-linear physical saturation knees, hardware boundaries, and architectural limitations observed across empirical testing. 

Linear superposition modeling ($R_{\text{total}} = R_{\text{base}} + \sum \alpha_i \cdot \text{Rate}_i$) remains strictly valid **only below these physical saturation thresholds**. When workloads breach these boundaries, systems transition into non-linear queueing behavior, tail-latency explosions, and operational failure.

---

## Summary of Physical Guardrails

| Guardrail | Resource Layer | Physical Threshold | Saturation Symptom | Primary Mitigation |
|---|---|---|---|---|
| **1. ACK Amplification Wall** | Consensus & CPU | $\le 10,000$ ACKs/s per stream | Tail latency jumps from 1 ms to 39 ms; 50% write drop | Shard consumers across streams; batch ACK policies |
| **2. Disk Fsync Ceiling** | Storage & Disk IOPS | $\le 7,000$ msgs/s per stream | Group-commit reverses; max latency spikes to 18.9 ms | Horizontally partition stream ($n_{\text{stream}} \ge \lceil x / 7,000 \rceil$) |
| **3. Stream Leader Skew** | CPU Distribution | 25% to 30% leader CPU skew | Uneven node utilization under single-stream workloads | Absorbed by 30% Headroom; L4 multi-stream balancing |
| **4. Cold Read Disk Thrashing**| Memory & Disk I/O | Lag bytes exceed spare host RAM | Eviction from kernel Page Cache forces physical disk reads | Provision higher memory tier VM; catch-up partitions |
| **5. Socket Buffer Expansion** | Memory (RAM) | Payload size $> 8\text{ KiB}$ | TCP auto-tuning inflates buffer memory 5x to 10x | Scale connection buffer RAM ($\beta_{\text{conn}} \approx 4 \times \text{payload}$) |
| **6. In-Memory Stream Trap** | Memory (RAM) | `StorageType: Memory` | 100% of retention volume consumes Go heap RAM | Use `StorageType: File` for streaming retention |
| **7. Ephemeral Subject Leaks** | Memory (Go Heap) | Unbounded UUID subject strings | Sublist prefix trie memory leaks monotonically | Enforce naming governance (UUIDs in headers/payload) |
| **8. Physical Network NIC Wall** | Network Line Rate | $> 70\text{ MB/s}$ on 1 Gbps NICs | TCP socket buffers block; P99 latency jumps 7,300x | Upgrade to 10 Gbps NIC; verify line-rate wire bandwidth |
| **9. Multi-Raft Stream Density**| Control Plane & CPU | $> 1,000$ concurrent streams | Goroutine scheduler and Raft lease heartbeat load | Provision 1 vCPU and 1 GB RAM per 500 idle streams |

---

## 1. Compute & Consensus Guardrails

### Guardrail 1: The ACK Amplification Wall ($\le 10,000$ ACKs/sec per Stream)
* **Physical Mechanism**: In JetStream, consumer message deliveries and explicit acknowledgments (`+ACK`) mutate consumer state machines committed through Raft. On a single stream, state transitions serialize through an internal Go channel and Raft lock.
* **Empirical Observation (`consumer-fanout`)**:
  * Scaling from 1 to 10 consumers produced linear CPU scaling ($0.0330\text{ mCPU per ACK/s}$) with flat median latency ($1.0\text{ ms}$).
  * Scaling to 20 consumers ($20,000\text{ ACKs/s}$) saturated the single-stream Raft commit channel. Ingress write throughput dropped by 50% (996 down to 500 msgs/s), median latency rose 7x (1.0 ms to 7.7 ms), and P99.9 latency exploded to 39 ms.
* **Architectural Rule**: Linear consumer ACK sizing applies strictly up to **10,000 ACKs/sec per stream** (or $\le 10$ active consumers per stream). Workloads exceeding 10,000 ACKs/sec must:
  1. Horizontally shard consumers across partitioned streams.
  2. Implement client-side batch acknowledgment policies (`AckWait` / batching).

### Guardrail 2: Single-Stream Disk Fsync Ceiling ($\le 7,000$ msgs/sec per Stream)
* **Physical Mechanism**: Persistent JetStream streams ($R=3$, File storage) commit messages to append-only block files (`.blk`) using synchronous or group `fsync()` system calls. A single stream write log is constrained by the physical IOPS and flush serialization latency of the underlying persistent storage controller.
* **Empirical Observation (`jetstream-ladder`)**:
  * Replicated persistent write throughput saturated at **7,234 msgs/s** on a single stream.
  * When offered load reached 10,000 msgs/s, the disk write queue saturated, reversing group-commit amortization: median latency rose, maximum latency spiked to 18.9 ms, and cluster CPU jumped to 35.2% due to goroutines spinning on disk commit locks.
* **Architectural Rule**: Single-stream persistent write throughput is physically capped at **~7,000 msgs/s**. If peak ingress exceeds 7,000 msgs/s, the workload must be horizontally partitioned across multiple streams:
  $$n_{\text{stream}} \ge \left\lceil \frac{x_{\text{js\_peak}}}{7,000} \right\rceil$$
  Adding more CPU or RAM will not overcome a physical single-stream disk sync ceiling.

### Guardrail 3: Stream Leader vs Follower Node CPU Asymmetry
* **Physical Mechanism**: In a 3-node JetStream cluster ($R=3$), the stream Raft leader node performs significantly more computational work than follower nodes:
  * The leader terminates client TCP sockets, parses incoming payloads, creates Raft log proposals, dispatches `AppendEntries` RPCs to two followers, collates quorums, delivers messages to consumers, and generates `PUB-ACK` replies.
  * Follower nodes only receive pre-serialized log entries, write them to disk, and send acknowledgement RPCs.
* **Empirical Observation**: The leader node consistently consumes **25% to 30% higher CPU** than follower nodes (e.g. 41.2% leader vs 32.9% follower under heavy load).
* **Architectural Rule**:
  * In multi-stream deployments, HAProxy Layer 4 load balancing and round-robin stream leader assignment naturally distribute leader load across all 3 nodes.
  * In single-stream deployments, the **30% Headroom buffer (targeting $\le 70\%$ peak utilization)** safely absorbs single-node leader skew without starving any individual node.

---

## 2. Memory & Caching Guardrails

### Guardrail 4: The Cold Read Disk Thrashing Knee (Lag Exceeding Spare Host RAM)
* **Physical Mechanism**: NATS relies on the Linux OS Page Cache maintained in spare host RAM to serve real-time consumer reads directly from memory (Zero-Disk Read Replay). When consumer processing stalls or falls behind such that accumulated backlog exceeds available spare host RAM ($\text{Backlog Bytes} > \text{Host RAM} - \text{RAM}_{\text{process}}$), the requested message blocks will have been evicted from RAM by newer incoming writes.
* **Behavioral Impact**: The kernel must fetch cold message blocks from physical disk storage. Real-time publishers (writing to disk) and catch-up consumers (reading from disk) will compete for physical storage IOPS and bus bandwidth, causing write latency to spike and tail latency to degrade.
* **Architectural Rule**: Workloads with expected long consumer catch-up periods or batch replay requirements must:
  1. Provision a higher memory-tier VM (e.g., General Purpose 1:4 vCPU:RAM ratio rather than Compute-Optimized 1:2) to provide a larger kernel cache cushion.
  2. Direct batch catch-up consumers to dedicated read replicas or off-peak processing windows.

### Guardrail 5: Large Payload Socket Memory Amplification ($>8\text{ KiB}$)
* **Physical Mechanism**: For standard message payloads ($\le 4\text{ KiB}$), Linux TCP socket buffers remain compact (~64 KiB per socket). When message payloads exceed 8 KiB (e.g. 32 KiB payloads), the Linux kernel TCP auto-tuning mechanism dynamically expands receive and transmit buffers (`SO_RCVBUF`, `SO_SNDBUF`) up to several megabytes per connection.
* **Behavioral Impact**: Under high concurrent connections ($n_{\text{conn}} > 1,000$), large payloads can inflate socket memory consumption by 5x to 10x, risking container OOM termination.
* **Architectural Rule**: For workloads where peak payload exceeds 8 KiB, socket buffer memory must be scaled proportionally:
  $$\beta_{\text{conn}} \approx 4 \times \text{payload}_{\text{peak}}$$

### Guardrail 6: In-Memory Stream Trap (`StorageType: Memory` vs `File`)
* **Physical Mechanism**: This capacity sizing framework models **File-based JetStream streams** (`StorageType: File`), where messages are persisted to disk and only recent writes occupy the Page Cache.
* **Behavioral Impact**: If an in-memory stream (`StorageType: Memory`) is configured, 100% of all retained messages must reside permanently in Go runtime heap RAM.
* **Architectural Rule**: In-memory streams require dedicating the full storage volume ($x_{\text{js}} \times S \times t$) directly to host RAM. In-memory streams must be reserved strictly for transient cache or ephemeral buffering use cases.

### Guardrail 7: Ephemeral Subject Memory Leak (Unbounded UUIDs)
* **Physical Mechanism**: Every distinct subject registered in NATS allocates a prefix trie node in the Go heap sublist (~313 bytes per node per subject).
* **Behavioral Impact**: If an application publishes to unique, un-bounded dynamic subjects (e.g. `events.order.<uuid>`), the sublist trie expands monotonically over time, permanently leaking Go heap memory and triggering garbage collection thrashing.
* **Architectural Rule**: Enterprise naming governance must prohibit dynamic IDs in subject hierarchies. Unique IDs (UUIDs, transaction IDs) must reside in message headers (`Nats-Msg-Id`) or payloads, keeping registered subjects fixed and bounded.

---

## 3. Network & Control Plane Guardrails

### Guardrail 8: Physical Network Interface Wall ($\le 70\text{ MB/s}$ on 1 Gbps NICs)
* **Physical Mechanism**: In distributed clusters, network wire consumption is amplified by client ingress, inter-node Raft consensus replication ($R=3$ writes replicate to 2 peers), and consumer delivery egress.
* **Empirical Observation (`payload-scaling`)**:
  * At 32 KiB payloads and 5,000 msgs/s, network wire throughput reached 103.6 MB/s (~869 Mbps), hitting the physical line-rate ceiling of a standard 1 Gbps network interface.
  * Ingress throughput was throttled down to 3,312 msgs/s, and P99 tail latency exploded by over 7,300x (from $3.4\ \mu\text{s}$ to $25.1\text{ ms}$) as TCP send socket buffers filled and blocked `write()` system calls.
* **Architectural Rule**: Sizing models assume CPU is independent of payload size up to 8 KiB. For workloads with large payloads ($>8\text{ KiB}$) or high ingress rates, aggregate wire bandwidth must be verified against physical host network adapter limits:
  $$\text{Wire}_{\text{cluster}} = (x_{\text{js\_peak}} \cdot S_{\text{peak}} \cdot 3) + (y_{\text{js\_peak}} \cdot S_{\text{peak}}) + (x_{\text{core\_peak}} \cdot S_{\text{peak}}) + (y_{\text{core\_peak}} \cdot S_{\text{peak}})$$
  Workloads exceeding 70 MB/s sustained traffic require upgrading to 10 Gbps network interfaces.

### Guardrail 9: Multi-Raft Stream Density Scaling Limits ($> 1,000$ Streams)
* **Physical Mechanism**: In NATS JetStream, each stream operates as an independent Raft consensus group maintaining its own leader lease timer (`T_RAFT = 250\text{ ms}`, 4 ops/s per stream).
* **Empirical Observation (`stream-scaling`)**:
  * Scaling from 1 stream to 50 streams added only +8.1 mCPU and +34.4 MB RAM cluster-wide, demonstrating excellent control-plane multiplexing.
  * However, running thousands of independent Raft streams per node creates goroutine scheduling contention and multiplexing overhead on inter-node route connections.
* **Architectural Rule**: Clusters hosting more than 1,000 active streams should provision at least 1 dedicated vCPU and 1 GB of RAM per 500 idle/low-activity streams, or partition streams across multiple independent NATS clusters.
