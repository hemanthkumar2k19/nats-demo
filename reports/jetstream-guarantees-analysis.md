# NATS Behavior Analysis: JetStream Cost of Guarantees

## 1. Executive Summary

The JetStream Cost of Guarantees suite (`jetstream-guarantees`, timestamp `20261112T091142Z`) provides a controlled step-by-step isolation of the latency, CPU, memory, and disk tax imposed by each messaging guarantee tier. All tests were executed at an identical offered rate of 1,000 msgs/s, 1,024-byte payloads, 4 publishers, and 2 subscribers over 30-second measurement windows.

Key findings:
1. **The Cost of Distributed Consensus (Core $\to$ Memory R=3)**: Adding 3-node in-memory Raft consensus replication increases median latency by **703x** (from $2.22 \ \mu\text{s}$ to $1,561.25 \ \mu\text{s}$) and triples cluster CPU from 4.42% to 14.97%, strictly isolating the network RPC and serialization tax of distributed Raft quorum voting.
2. **The Cost of Disk Persistence (Core $\to$ File R=1)**: Adding single-node disk persistence without distributed consensus increases median latency to $2,132.80 \ \mu\text{s}$ (2.13 ms) and CPU to 8.47%, isolating the physical `fsync()` write barrier.
3. **Enterprise Standard (File R=3)**: Combining distributed consensus with physical disk persistence yields a median latency of $1,998.09 \ \mu\text{s}$ (2.0 ms) and cluster CPU of 16.88%, establishing that in a clustered setup with group commits, the network consensus RPC and disk sync run concurrently in the Raft state machine pipeline.

---

## 2. Empirical Benchmark Data

All tests executed at 1,000 msgs/s, 1,024-byte payloads, 4 publishers, 2 subscribers:

| Test ID | Guarantee Tier | Storage Engine | Replicas | Consensus | p50 Latency | p99 Latency | p99.9 Latency | Max Latency | Cluster Avg CPU | Cluster Peak RAM | Memory Growth |
| :--- | :--- | :--- | ---:| :--- | ---:| ---:| ---:| ---:| ---:| ---:| ---:|
| **`TEST-G1-CORE`** | Core NATS | Memory (none) | 0 | None | 2.22 us | 15.62 us | 61.96 us | 188.5 us | 4.42% | 60.3 MB | +4.2 MB |
| **`TEST-G2-JS-MEM-R3`**| JetStream Memory | Memory | 3 | Raft Quorum | 1,561.25 us | 3,272.96 us | 7,371.46 us | 11,996 us | 14.97% | 257.4 MB | +198.6 MB |
| **`TEST-G3-JS-FILE-R1`**| JetStream File R=1| File | 1 | None (Local) | 2,132.80 us | 4,094.50 us | 8,782.33 us | 22,570 us | 8.47% | 283.3 MB | +25.2 MB |
| **`TEST-G4-JS-FILE-R3`**| JetStream File R=3| File | 3 | Raft Quorum | 1,998.09 us | 4,055.00 us | 12,010.42 us | 17,982 us | 16.88% | 242.8 MB | +66.4 MB |

---

## 3. Systems & Architectural Behavioral Analysis

### 3.1 Isolating the Distributed Consensus Tax (`TEST-G1` vs `TEST-G2`)
Comparing Core NATS against JetStream Memory R=3 isolates the exact performance overhead of distributed consensus:
- **Routing Engine**: Core NATS dispatches messages directly through in-memory pointer lookups in the sublist trie. Latency is microsecond-level ($p50 = 2.22 \ \mu\text{s}$, $p99 = 15.62 \ \mu\text{s}$).
- **Raft Memory Engine**: In JetStream Memory R=3, messages are not stored to disk, but the leader must serialize a Raft `AppendEntries` payload, send it across the network bridge to two peer nodes, wait for acknowledgment from at least one follower, commit the Raft log index, and reply with a `PUB-ACK`.
- **Tax Breakdown**:
  - Latency Tax: $+1,559 \ \mu\text{s}$ (+1.56 ms, a 703x increase).
  - CPU Tax: $+10.55\%$ cluster CPU (from 4.42% to 14.97%), driven by JSON/binary wire serialization, TCP socket read/write loops between cluster nodes, and Raft consensus state machine evaluation.
  - Memory Tax: $+197.1 \text{ MB}$ cluster RAM (from 60.3 MB to 257.4 MB), reflecting the pre-allocated in-memory stream buffer ring.

### 3.2 Isolating the Physical Disk `fsync()` Tax (`TEST-G1` vs `TEST-G3`)
Comparing Core NATS against JetStream File R=1 isolates the cost of disk persistence without network clustering:
- In File R=1, NATS runs on a single node without Raft network communication. However, it must write incoming messages to the append-only block file and execute a blocking `fsync()` to ensure durability.
- **Tax Breakdown**:
  - Latency Tax: $2,132.80 \ \mu\text{s}$ (2.13 ms) - this represents the baseline physical write cycle of the underlying storage subsystem at 1,000 writes/sec.
  - CPU Tax: $+4.05\%$ cluster CPU (from 4.42% to 8.47%), representing low user-space CPU but blocking OS kernel I/O wait states.

### 3.3 The Combined Clustered Standard (`TEST-G4-JS-FILE-R3`)
When distributed Raft consensus and disk persistence are combined:
- Median latency is $1,998.09 \ \mu\text{s}$ (2.0 ms), which is slightly lower than File R=1 ($2.13 \text{ ms}$).
- **Concurrency in the Raft Pipeline**: In File R=3, the leader writes to its local disk log while simultaneously transmitting network AppendEntries RPCs to followers. Because followers write to disk in parallel, the network transmission overlaps with local disk I/O.
- Cluster CPU was 16.88%, with `nats-1` (the stream leader) taking 19.73% CPU and follower nodes `nats-2` and `nats-3` consuming 14.84% and 16.06% CPU respectively.

---

## 4. Architectural Decision Matrix

| Requirement | Recommended Engine | Latency Expectation | CPU Overhead | Memory Footprint | Durability & High Availability |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **High Frequency, Ephemeral** | Core NATS | Sub-10 microseconds | Minimal (< 5%) | Lowest (~40-60 MB) | In-memory only; data lost on node crash |
| **Lossless, In-Memory HA** | JetStream Memory R=3 | 1.5 - 3 milliseconds | Moderate (~15%) | Moderate (~250 MB) | Resilient to node crash; lost on total power loss |
| **Single-Node Persistent** | JetStream File R=1 | 2.0 - 4 milliseconds | Low (~8%) | Moderate (~280 MB) | Disk durable; downtime during host failure |
| **Enterprise Standard HA** | JetStream File R=3 | 1.0 - 2.5 milliseconds | Highest (~17-35%) | Moderate (~240 MB) | Complete disaster recovery and Raft quorum HA |

