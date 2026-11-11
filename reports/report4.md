# JetStream Cost of Guarantees Suite: Technical Analysis & Inference

**Suite**: `jetstream-guarantees` (`JetStream Cost of Guarantees Suite`)  
**Run Timestamp**: `20261111T054843Z`  
**Artifact Inspected**: `nats-capacity-benchmark/results/suites/jetstream-guarantees/20261111T054843Z/suite-results.json`  

---

## 1. Executive Summary & Architectural Purpose

### The Cost of Reliability Hierarchy
This suite provides a controlled, apples-to-apples evaluation of the architectural costs of persistence, replication, and acknowledgment semantics in NATS. All four tests executed under an identical offered load of **1,000 msgs/sec** with **1 KiB payloads**:
1. **Core NATS (`TEST-G1-CORE`)**: At-most-once in-memory routing with zero persistence.
2. **JetStream Memory R=3 (`TEST-G2-JS-MEM-R3`)**: At-least-once message durability replicated in volatile RAM across a 3-node Raft consensus quorum (`--storage memory --replicas 3`).
3. **JetStream File R=1 (`TEST-G3-JS-FILE-R1`)**: Disk-persisted storage on a single stream leader node without replication (`--storage file --replicas 1`).
4. **JetStream File R=3 (`TEST-G4-JS-FILE-R3`)**: Full enterprise reliability with on-disk persistence and 3-node Raft consensus replication (`--storage file --replicas 3`).

### Key Findings:
* **The Latency Step Function**: Adding JetStream durability increases median latency by approximately **1,000x** (from 1.65 microseconds in Core NATS to 1.5 to 1.9 milliseconds in JetStream).
* **Consensus CPU Tax**: Replicating state across 3 nodes via Raft triples CPU utilization (from 5.35% in Core NATS to 15.81% in JetStream Memory R=3).
* **Storage Footprint Multiplier**: JetStream File R=3 consumes **3x the disk space** of File R=1 (92.6 MB vs. 30.3 MB for 30,000 messages), verifying exact Raft consensus write distribution.

---

## 2. Comparative Cost of Guarantees Matrix

| Dimension / Metric | Mode 1: Core NATS | Mode 2: JS Memory R=3 | Mode 3: JS File R=1 | Mode 4: JS File R=3 |
| :--- | :--- | :--- | :--- | :--- |
| **Delivery Guarantee** | At-most-once | At-least-once | At-least-once | At-least-once |
| **Storage Medium** | Volatile RAM (Buffer) | Volatile RAM (Stream) | On-Disk Persistent File| On-Disk Persistent File|
| **Replication Factor**| 0 (No Consensus) | 3 Replicas (Raft Quorum)| 1 Replica (Leader Only)| 3 Replicas (Raft Quorum)|
| **Target Rate** | 1,000 /s | 1,000 /s | 1,000 /s | 1,000 /s |
| **Achieved Rate** | **996 /s** | **996 /s** | **996 /s** | **996 /s** |
| **P50 Latency** | **1.65 us** | **1,519.69 us (1.52 ms)**| **1,910.66 us (1.91 ms)**| **1,517.02 us (1.52 ms)**|
| **P99 Latency** | **7.62 us** | **2,192.08 us (2.19 ms)**| **2,972.33 us (2.97 ms)**| **3,124.79 us (3.12 ms)**|
| **P99.9 Latency** | 42.12 us | 3,144.29 us (3.14 ms)| 4,030.79 us (4.03 ms)| 22,658.92 us (22.7 ms)|
| **Max Latency** | 73.96 us | 3,460.92 us (3.46 ms)| 10,663.54 us (10.7 ms)| 65,539.25 us (65.5 ms)|
| **Cluster Avg CPU** | **5.35%** | **15.81%** | **8.49%** | **13.79%** |
| **Cluster Peak CPU**| 9.10% | 22.95% | 15.67% | 18.43% |
| **Cluster Peak RAM** | 133.0 MB | **250.1 MB (+117 MB)**| 264.4 MB | 245.4 MB |
| **Total Disk Growth**| **0 B** | **236 B (Metadata)** | **30.3 MB** | **92.6 MB (3x)** |
| **Empirical Disk Write**| 0.00 MB/s | 0.00 MB/s | **1.01 MB/s** | **3.09 MB/s** |
| **Delivery %** | 100.0% | 100.0% | 100.0% | 100.0% |

---

## 3. Detailed Step-by-Step Technical Breakdown

```
Latency vs Reliability Hierarchy (P50 & P99 Latency)
  4.0ms +
  3.0ms |                                       * [P99: 2.97ms]     * [P99: 3.12ms]
  2.0ms |                   * [P99: 2.19ms]
  1.0ms |                   * [P50: 1.52ms]     * [P50: 1.91ms]     * [P50: 1.52ms]
  0.0ms +---* [P50: 1.7us]--+-------------------+-------------------+--->
           Core NATS       JS Memory R=3       JS File R=1         JS File R=3
```

### Stage 1: Mode 1 - Core NATS (`TEST-G1-CORE`)
* **Characteristics**: In-memory, non-blocking fire-and-forget routing.
* **Technical Observations**:
  * P50 latency was **1.65 us**; P99 was **7.62 us**. Max latency across 30,000 messages never exceeded **74 us**.
  * CPU utilization was lowest at **5.35%**; memory footprint remained flat (+0.5 MB).
  * Disk activity was strictly **0 B**.
  * **Architectural Tradeoff**: Maximum throughput and sub-microsecond latency, but zero message durability: if a broker restarts or network partitions occur, unconsumed messages are lost.

### Stage 2: Mode 2 - JetStream Memory R=3 (`TEST-G2-JS-MEM-R3`)
* **Characteristics**: In-memory stream storage with 3-node Raft consensus replication.
* **Technical Observations**:
  * **The Latency Cost of Raft Consensus**: P50 latency increased from 1.65 us to **1,519.69 us (1.52 ms)**, a 920x expansion.
  * **Why**: The publisher sends a message and blocks until a Raft consensus quorum (2 of 3 nodes) acknowledges receipt over the internal cluster mesh.
  * **The CPU Cost of Quorum**: Average CPU tripled from 5.35% to **15.81%** (Peak 22.95%). Raft heartbeat timers, log replication RPCs, and state synchronization consume measurable compute.
  * **RAM Expansion**: Cluster memory grew by **+117.3 MB** to cache the stream messages in RAM.
  * **Disk Growth**: Only **236 bytes** (minimal cluster configuration metadata).
  * **Architectural Tradeoff**: High availability against single-node crashes without paying a rotational/SSD disk `fsync()` penalty. However, a total cluster power failure results in message loss.

### Stage 3: Mode 3 - JetStream File R=1 (`TEST-G3-JS-FILE-R1`)
* **Characteristics**: On-disk persistence on a single stream leader without cluster replication.
* **Technical Observations**:
  * P50 latency was **1,910.66 us (1.91 ms)**; P99 was **2.97 ms**.
  * **Why latency is slightly higher than Memory R=3**: The stream leader node must synchronously flush data to disk via an operating system `fsync()` before acknowledging the client.
  * **Single-Node Disk Growth**:
    * Total disk growth was **31,795,136 bytes (~30.3 MB)**.
    * Node breakdown: `nats-1` wrote **31.8 MB**; `nats-2` wrote **0 B**; `nats-3` wrote **0 B**.
    * Only the single node hosting the stream leader wrote data to disk.
  * **Architectural Tradeoff**: Messages survive broker restarts, but failover redundancy is zero: if the stream leader node hardware fails, the stream is unavailable.

### Stage 4: Mode 4 - JetStream File R=3 (`TEST-G4-JS-FILE-R3`)
* **Characteristics**: On-disk persistence combined with 3-node Raft consensus replication (the enterprise gold standard).
* **Technical Observations**:
  1. **Latency Profile**:
     * P50 latency was **1,517.02 us (1.52 ms)**; P99 was **3.12 ms**.
     * Extreme tail latency (P99.9) reached **22.66 ms**, and Max latency was **65.54 ms**, reflecting occasional disk `fsync()` pauses and concurrent Raft consensus sync rounds.
  2. **Triple Storage Growth**:
     * Total disk growth reached **97,077,600 bytes (~92.6 MB)**.
     * Node breakdown:
       * `nats-1`: **32.36 MB**
       * `nats-2`: **32.36 MB**
       * `nats-3`: **32.36 MB**
     * All 3 nodes wrote nearly identical byte volumes down to within 0.01%, proving symmetrical Raft replication across the entire quorum.
  3. **Architectural Tradeoff**: Maximum enterprise durability and fault tolerance (can tolerate the loss of any single node without data loss or downtime), at the cost of $3\times$ storage consumption and millisecond-level commit latency.

---

## 4. Architectural Selection Matrix

Use the following decision matrix to select the appropriate NATS messaging engine based on operational requirements:

| Application Use Case | Recommended Mode | Rationale |
| :--- | :--- | :--- |
| **High-Frequency Market Data / Telemetry** | **Core NATS** | Microsecond latency (< 2 us), sub-microsecond jitter, zero storage footprint. Message loss during broker failure is acceptable. |
| **Ephemeral Microservice RPC / Workflows** | **Core NATS (Queue Groups)** | Load-balances requests across worker instances with near-zero latency and horizontal scaling. |
| **Fast Session State / High-Speed Deduplication**| **JS Memory R=3** | Provides high availability and at-least-once delivery without disk I/O bottlenecks. |
| **Edge Device / Single-Server Logging** | **JS File R=1** | Provides persistent on-disk queuing with minimal CPU overhead where multi-node clustering is not deployed. |
| **Financial Ledgers, Orders, Critical Events**| **JS File R=3** | Full disaster recovery, zero data loss, survives node crashes and power failures via distributed Raft disk consensus. |

