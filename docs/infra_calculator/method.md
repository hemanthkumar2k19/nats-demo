# Mathematical & Systems Engineering Principles

This document formalizes the mathematical foundations, systems engineering methodologies, and scientific principles used to derive the NATS capacity sizing framework across Compute (CPU), Memory (RAM), Storage (Persistent Disk), and Network.

---

## 1. OFAT (One-Factor-At-A-Time) Controlled Dimensional Isolation

### Theoretical Foundation
In systems performance engineering, composite load tests where multiple dimensions are scaled simultaneously create confounding variables. If latency inflects or CPU spikes during a test combining 10,000 subjects, 50 streams, 20 consumers, and 16 KiB payloads, it is impossible to determine the physical root cause (e.g. TCP socket buffer saturation, sublist trie cache misses, disk fsync serialization, or Raft heartbeat scheduling).

The **OFAT (One-Factor-At-A-Time)** methodology scales strictly **one independent variable at a time** across logarithmic decadal spans while holding all other parameters fixed. This isolates the marginal resource consumption of each individual operation.

### Applied In:
* **Compute (CPU)**:
  * `core-ladder`: Isolates in-memory routing CPU ($\alpha_{\text{core\_pub}}$, $\alpha_{\text{core\_sub}}$) by varying message rate while holding payload size and stream count fixed.
  * `consumer-fanout`: Isolates the consumer delivery and explicit ACK tax ($\alpha_{\text{ack}}$) by varying consumer count against a fixed 1,000 msgs/s publish stream.
  * `stream-scaling`: Isolates multi-Raft control-plane lease overhead ($\alpha_{\text{stream}}$) by scaling idle auxiliary streams while holding traffic constant.
* **Memory (RAM)**:
  * `subject-cardinality`: Isolates the sublist prefix trie node cost ($\beta_{\text{subj}} = 313\text{ B/node}$) by varying registered subjects from 1 to 10,000 under fixed rate and payload.
  * `stream-scaling`: Isolates multi-Raft stream descriptor state ($\beta_{\text{stream}} = 234\text{ KiB/node}$) across 1 to 50 streams.
  * `consumer-fanout`: Isolates durable consumer progress state ($\beta_{\text{consumer}} = 7.29\text{ MB/node}$) across 1 to 20 consumers.
* **Persistent Disk Storage**:
  * Retention sweeps: Isolates accumulated persistent storage bytes ($D_{\text{node}} = x_{\text{js}} \times S \times t$) from write IOPS.
* **Network Bandwidth**:
  * `payload-scaling`: Isolates TCP framing overhead and line-rate saturation across payload sizes from 128 B to 32 KiB.

---

## 2. The Superposition Principle (Linear Composition of Micro-Workloads)

### Theoretical Foundation
In distributed systems operating below physical resource saturation knees, total resource demand is the linear sum of baseline fixed cluster costs plus the independent marginal service taxes of all active operations:

$$R_{\text{total}} = R_{\text{base}} + \sum_{i=1}^{n} (\text{Rate}_i \times \text{Cost}_i)$$

Because each independent operational cost was isolated via OFAT, the superposition principle permits composing these micro-coefficients into a single unified predictive model for any arbitrary composite workload profile.

### Applied In:
* **Compute (CPU)**:
  * Composes baseline idle overhead ($\text{mCPU}_{\text{base}}$), Core NATS pub/sub, JetStream replicated writes, consumer explicit ACKs, pull batch RPCs, Raft stream leases, and client socket polling:
    $$\text{mCPU}_{\text{peak}} = \text{mCPU}_{\text{base}} + (\alpha_{\text{core\_pub}} \cdot x_{\text{core\_peak}}) + (\alpha_{\text{core\_sub}} \cdot y_{\text{core\_peak}}) + (\alpha_{\text{js\_write}} \cdot x_{\text{js\_peak}}) + (\alpha_{\text{ack}} \cdot y_{\text{js\_peak}}) + (\alpha_{\text{batch}} \cdot R_{\text{batch\_peak}}) + (\alpha_{\text{stream}} \cdot n_{\text{stream}}) + \left(\alpha_{\text{conn}} \cdot \frac{n_{\text{conn\_peak}}}{3}\right)$$
* **Memory (RAM)**:
  * Composes resident Go heap baseline, prefix trie state, stream descriptors, consumer progress tracking, and client socket buffers:
    $$\text{RAM}_{\text{heap, node}} = \text{RAM}_{\text{base}} + (\beta_{\text{subj}} \cdot n_{\text{subj}}) + (\beta_{\text{stream}} \cdot n_{\text{stream}}) + (\beta_{\text{consumer}} \cdot n_{\text{consumer\_peak}}) + \left(\beta_{\text{conn}} \cdot \frac{n_{\text{conn\_peak}}}{3}\right)$$
* **Network Bandwidth**:
  * Composes client ingress traffic, inter-node Raft consensus replication traffic to 2 peer nodes, and outbound consumer delivery egress:
    $$\text{Wire}_{\text{cluster}} = (x_{\text{js\_peak}} \cdot S_{\text{peak}} \cdot 3) + (y_{\text{js\_peak}} \cdot S_{\text{peak}}) + (x_{\text{core\_peak}} \cdot S_{\text{peak}}) + (y_{\text{core\_peak}} \cdot S_{\text{peak}})$$

---

## 3. Conservative Upper-Bound Sizing (The Secant Envelope)

### Theoretical Foundation
In high-performance networking and operating system runtimes, service cost curves are almost never purely linear. They exhibit **sub-linear (concave)** behavior ($f''(x) < 0$) due to kernel socket batching, NATS ring buffer amortizations, and system call optimizations:
* **Unbatched Regime (Low/Moderate Rates)**: Messages arrive sparsely. Every single message triggers an independent network hardware interrupt, OS kernel context switch, `epoll_wait` wakeup, and `read()`/`write()` syscall. The marginal CPU cost per message is at its **highest**.
* **Batched Regime (High Rates)**: Packets queue in socket buffers. NATS reads and flushes tens of messages in a single syscall, sharply reducing the marginal cost per message ($\Delta \text{CPU} / \Delta \text{Rate}$).

If a capacity model fits a linear slope to the flatter high-rate points, it will severely **under-provision** real-world production workloads running at lower or bursty throughput. 

To prevent under-sizing, we apply the **Secant Envelope Principle**: we anchor our linear slopes on the steeper, un-amortized measurements observed across lower/moderate throughput. This guarantees an upper-bound envelope:
$$\text{Model}(x) \ge \text{Actual}(x) \quad \forall x$$

```
CPU Demand
 ^
 |             / (Our Model: Conservative Upper Secant)
 |            /          . Actual System (Amortized / Batched)
 |           /      . '
 |          / . ' 
 |         /. '
 |       .' /
 |     .'  / (Under-provisioned High-Rate Tangent)
 |   .'   /
 +--------------------------------------------> Throughput
```

### Applied In:
* **Compute (CPU)**:
  * **Core NATS Routing**: Anchoring on the steeper slope between 1,000 and 10,000 msgs/s ($0.0011\text{ mCPU/op}$) rather than the flat slope between 25,000 and 50,000 msgs/s ($0.0002\text{ mCPU/op}$).
  * **JetStream Writes**: Anchoring write consensus at $\alpha_{\text{js\_write}} = 0.0320\text{ mCPU/msg}$ across the unbatched 250 to 5,000 msgs/s range.
  * **Raft Streams**: Provisioning $\alpha_{\text{stream}} = 0.66\text{ mCPU}$ (a 4x conservative margin over measured idle state) to safely absorb heartbeat bursts and periodic leader election timer checks.
* **Storage IOPS**:
  * Sizing disk write IOPS based on unbatched fsync sync rates rather than relying on synthetic continuous group-commit batching.

---

## 4. Differential Slope Extraction (Elimination of Containerization Overhead)

### Theoretical Foundation
When benchmarking NATS inside containerized environments (such as Podman or Kubernetes), telemetry gathered via cgroup accounting (`cpu.stat`, `memory.current`) records aggregate container consumption:

$$\text{Metric}_{\text{measured}} = C_{\text{overhead}} + (\alpha \cdot \text{Rate})$$

Where $C_{\text{overhead}}$ represents the static virtualization, hypervisor scheduling, and background daemon tax. 

To isolate the pure marginal cost of NATS software execution from hosting overhead, we evaluate the **differential rate of change** between consecutive benchmark steps:

$$\frac{\Delta \text{Metric}}{\Delta \text{Rate}} = \frac{(C_{\text{overhead}} + \alpha \cdot \text{Rate}_2) - (C_{\text{overhead}} + \alpha \cdot \text{Rate}_1)}{\text{Rate}_2 - \text{Rate}_1} = \alpha$$

The constant containerization tax ($C_{\text{overhead}}$) mathematically cancels out. This isolates the pure software-intrinsic coefficient ($\alpha$), ensuring portability across local containers, cloud VMs, and bare-metal nodes. Static background costs are then cleanly consolidated into dedicated baseline floors.

### Applied In:
* **Compute (CPU)**:
  * Differentiating CPU between load steps isolates the pure marginal cost of NATS frame parsing, Raft log serialization, and ACK state mutation.
  * Static background container taxes, route keepalives, and monitoring scrapers are consolidated into the flat $\text{mCPU}_{\text{base}} = 50\text{ mCPU}$ floor.
* **Memory (RAM)**:
  * Differentiating memory between steady-state steps ($\Delta \text{RAM} / \Delta N$) isolates true incremental struct allocations, eliminating initial process startup allocations (warmup allocations).
  * Startup runtime allocations, mcache/mcentral spans, and runtime poller threads are consolidated into the flat $\text{RAM}_{\text{base}} = 128\text{ MB}$ floor.

---

## 5. Two-Tier Memory Architecture (Application RSS vs. Kernel Page Cache)

### Theoretical Foundation
High-performance distributed message brokers do not manage large user-space data caches in application heap memory. Managing multi-gigabyte data caches in application runtimes (such as the Go heap) triggers severe garbage collection pauses (`STW`), pointer-scanning overhead, and memory fragmentation.

Instead, NATS relies on a **Two-Tier Memory Architecture**:
1. **Tier 1: Application Process Heap (RSS)**: Remains flat, bounded, and compact (~250 MB to 650 MB), holding only active execution metadata, routing tries, and cursor tracking.
2. **Tier 2: Linux OS Page Cache**: Leverages the kernel Virtual File System (VFS) to dynamically cache sequential append-only file blocks (`.blk`) in unused node RAM.

Because standard cloud VM compute profiles bundle RAM in fixed ratios (1:2 or 1:4 with vCPU), sizing the node compute core automatically supplies several gigabytes of spare host RAM. The Linux kernel uses this spare RAM as Page Cache, establishing the **Zero-Disk Read Replay Principle**: live streaming consumers read directly from memory and generate **zero physical disk read IOPS**.

### Applied In:
* **Memory (RAM)**:
  * Sizes the deterministic Go Process Resident Memory ($\text{RAM}_{\text{process, node}}$) covering baseline runtime structures, routing tries, stream state, consumer cursors, and socket buffers with 30% Headroom for GC:
    $$\text{RAM}_{\text{provisioned}} = \frac{\text{RAM}_{\text{process, node}}}{0.70}$$
  * Relies on cloud instance bundling to passively provide generous kernel Page Cache (holding 45+ minutes of recent ingress data in spare RAM) without synthetic timer parameters.
* **Persistent Disk Storage**:
  * Guarantees that disk storage volumes are sized purely for write ingestion and long-term retention ($t$), eliminating write/read IOPS contention on the disk bus.

---

## 6. Mechanical Sympathy with OS & Storage Subsystems

### Theoretical Foundation
"Mechanical sympathy" (a term coined by Martin Thompson) means designing software and capacity models that align with the physical realities of the underlying hardware and operating system kernel:
* **Sequential Append-Only Disk I/O**: Hard disks and SSDs achieve orders of magnitude higher throughput on sequential append operations than random access. JetStream formats messages into append-only segment blocks (`.blk`), matching Linux sequential readahead algorithms.
* **Fsync Group Commit Amortization**: Under increasing concurrent load, physical disk sync latency drops because NATS combines multiple in-flight transactions into shared disk sync batches.
* **Shared Memory Pages**: When 10 consumers read the same stream, the Linux VFS serves all 10 consumers from the same physical 4 KiB memory page in RAM, rather than duplicating memory.

### Applied In:
* **Compute (CPU)**:
  * Reflects socket auto-tuning and kernel epoll batching at high throughput.
* **Memory (RAM)**:
  * Consumer fan-out ($c_{\text{js}}$) is deliberately excluded from Page Cache volume sizing because the kernel shares file cache pages across consumers.
* **Persistent Disk Storage**:
  * Sizing acknowledges the single-stream disk fsync ceiling ($\le 7,000\text{ msgs/s}$) governed by physical drive sync latency.

---

## 7. The USE Method (Utilization, Saturation, and Errors)

### Theoretical Foundation
Formulated by Brendan Gregg (performance engineering lead at Netflix and Sun Microsystems), the **USE Method** directs the systematic analysis of every physical and logical resource across three vectors:
1. **Utilization**: The percentage of time that a resource was busy servicing work.
2. **Saturation**: The degree to which work has queued up waiting for an unavailable resource.
3. **Errors**: The count of rejected, dropped, or failed operations.

Linear capacity modeling is valid strictly within the **non-saturated regime**. When a resource saturates, systems transition into non-linear queueing behavior where latency spikes asymptotically.

### Applied In:
* **All Sizing Guardrails**:
  * **ACK Amplification Wall**: Identified saturation of internal Raft commit channels beyond $10,000\text{ ACKs/s}$ per stream, where latency spiked from 1.0 ms to 39 ms and throughput dropped 50%.
  * **Disk Fsync Ceiling**: Identified saturation of single-stream disk write queues beyond $7,234\text{ msgs/s}$, where goroutines spun on disk locks and CPU spiked to 35.2%.
  * **Network NIC Wall**: Identified saturation of physical network adapters at payloads $>8\text{ KiB}$ (reaching 103.6 MB/s on 1 Gbps links), causing TCP socket buffers to fill and block `write()` syscalls.

---

## 8. Queueing Theory & Target Utilization (M/M/1 Headroom Sizing)

### Theoretical Foundation
In queueing theory (Kingman's formula / M/M/1 queueing models), mean wait time in a service queue ($W_q$) grows asymptotically as utilization ($\rho$) approaches 100%:

$$W_q \propto \frac{\rho}{1 - \rho}$$

When sustained server utilization exceeds 70% ($\rho > 0.70$), the queue wait time inflects sharply upward (the "knee of the curve"), triggering latency explosions, buffer overruns, and client timeout cascades.

Reserving a **30% Headroom buffer (targeting $\le 70\%$ peak sustained utilization)** divides raw aggregate demand by $0.70$:
$$\text{Provisioned} = \frac{\text{Raw Demand}}{0.70}$$

In a 3-node cluster, this 30% headroom provides **mathematical $N-1$ failover survivability**: if 1 node fails, the surviving 2 nodes absorb $1.5\times$ the workload:
$$70\% \times 1.5 = 105\%$$
The surviving nodes absorb the failover traffic without service collapse while the failed node recovers.

### Applied In:
* **Compute (CPU)**:
  * Scales raw peak mCPU by $1 / 0.70$ to guarantee sub-millisecond tail latency and absorb stream leader CPU skew (25-30%).
* **Memory (RAM)**:
  * Scales raw memory demand by $1 / 0.70$ to safely absorb Go runtime garbage accumulation spikes (`GOGC=100`) without triggering Linux OOM killer termination.
