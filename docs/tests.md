# NATS Capacity Benchmark: Consolidated Test Inventory & Matrix Specification

## 1. Overview & Execution Workflow

### 1.1 Objective and Scope
This document is the authoritative test inventory for the NATS Capacity Benchmark suite. It consolidates all test specifications across both **Core NATS** (in-memory pub/sub, queue groups, subject taxonomy) and **JetStream** (File/Memory storage, Raft consensus replication, stream density, and durable consumer fan-out).

The benchmark adheres to the **OpenMessaging Benchmark (OMB)** framework, measuring sustainable throughput, tail latency percentiles, and per-node resource consumption under fixed infrastructure limits.

### 1.2 Fixed Infrastructure Environment

All test suites execute against the fixed 3-node NATS cluster topology with HAProxy L4 load balancing and container resource limits defined in `docs/benchmarking.md`.

### 1.3 Execution Modes

#### Mode A: Automated Suite Execution (Recommended)
Execute entire test ladders or comparative matrices unattended with automated cluster isolation, 10-second quiescence settling between runs, and consolidated executive reporting (`executive-report.html` and `suite-report.html`):

```bash
# Execute any of the 7 canonical suites:
./suite.sh core-ladder
./suite.sh payload-scaling
./suite.sh subject-cardinality
./suite.sh jetstream-ladder
./suite.sh jetstream-guarantees
./suite.sh stream-scaling
./suite.sh consumer-fanout
```

#### Mode B: Single Test Execution via Environment Overrides
Execute an isolated test case with custom parameter overrides:

```bash
SCENARIO=js-pub-sync \
THROUGHPUT=2000 \
PUBS=4 \
SUBS=2 \
STORAGE=file \
REPLICAS=3 \
./benchmark.sh
```

---

## 2. The 7 Canonical Executable Suites Inventory

The parameter space is structured into 7 distinct suites. Each suite evaluates a single physical dimension while keeping all other parameters constant.

```
+---------------------------------------------------------------------------------------+
|                               7 CANONICAL BENCHMARK SUITES                            |
+--------------------+---------------------+--------------------------------------------+
| Core NATS Suites   | core-ladder         | Throughput ladder: 1k to 100k msgs/s       |
|                    | payload-scaling     | Message size ladder: 128 B to 32 KiB       |
|                    | subject-cardinality | Taxonomy scaling: 1 to 10k subjs, wildcard |
+--------------------+---------------------+--------------------------------------------+
| JetStream Suites   | jetstream-ladder    | Sync File R=3 ladder: 250 to 10k msgs/s    |
|                    | jetstream-guarantees| Guarantees: Core vs Mem R=3 vs File R=1/R=3|
|                    | stream-scaling      | Multi-Raft density: 1 to 50 streams        |
|                    | consumer-fanout     | ACK write amplification: 1 to 20 consumers |
+--------------------+---------------------+--------------------------------------------+
```

---

### Suite 1: `core-ladder` (Core NATS Saturation Ladder)
* **Configuration File**: `suites/core-ladder.conf`
* **Scenario**: `pub-sub` (Core NATS normal pub/sub broadcast)
* **Fixed Settings**: Payload = 1,024 B, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Identify the maximum in-memory throughput ceiling, saturation knee, and the onset of socket buffer bloat.

| Test ID | Target Rate | Payload | Pubs | Subs | Duration | Explicit Test Objective |
| :--- | ---:| :--- | ---:| ---:| :--- | :--- |
| **`TEST-CL-1K`** | 1,000 /s | 1024 B | 4 | 4 | 30s | Low-throughput baseline; measure microsecond routing latency. |
| **`TEST-CL-10K`** | 10,000 /s | 1024 B | 4 | 4 | 30s | Moderate operational load; evaluate Go runtime socket batching. |
| **`TEST-CL-25K`** | 25,000 /s | 1024 B | 4 | 4 | 30s | Elevated throughput; evaluate multi-client connection scaling. |
| **`TEST-CL-50K`** | 50,000 /s | 1024 B | 4 | 4 | 30s | High load near saturation boundary; detect latency inflection. |
| **`TEST-CL-100K`** | 100,000 /s | 1024 B | 4 | 4 | 30s | Over-offered saturation; observe broker backpressure and memory growth. |

---

### Suite 2: `payload-scaling` (Message Payload Dimension Scaling)
* **Configuration File**: `suites/payload-scaling.conf`
* **Scenario**: `pub-sub` (Core NATS normal pub/sub broadcast)
* **Fixed Settings**: Target Rate = 10,000 msgs/s, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Measure network interface serialization limits, bandwidth scaling (MB/s), and socket buffer memory expansion across message payload sizes.

| Test ID | Payload Size | Target Rate | Pubs | Subs | Expected Bandwidth | Explicit Test Objective |
| :--- | ---:| ---:| ---:| ---:| :--- | :--- |
| **`TEST-PS-128B`** | 128 B | 10,000 /s | 4 | 4 | ~1.2 MB/s | Small telemetry/IoT payload; evaluate high-frequency message routing. |
| **`TEST-PS-1KB`** | 1,024 B | 10,000 /s | 4 | 4 | ~9.8 MB/s | Enterprise standard event payload baseline. |
| **`TEST-PS-8KB`** | 8,192 B | 10,000 /s | 4 | 4 | ~78.1 MB/s | Medium document payload; evaluate network buffer utilization. |
| **`TEST-PS-32KB`** | 32,768 B | 10,000 /s | 4 | 4 | ~103 MB/s (Capped)| Large payload; test 1 Gbps physical network interface saturation wall. |

---

### Suite 3: `subject-cardinality` (Prefix Trie Routing & Taxonomy Scaling)
* **Configuration File**: `suites/subject-cardinality.conf`
* **Scenario**: `pub-sub` (Core NATS dynamic subject pub/sub)
* **Fixed Settings**: Target Rate = 10,000 msgs/s, Payload = 1,024 B, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Evaluate the NATS sublist prefix trie lookup complexity, measuring whether high subject cardinality degrades routing latency or expands Go heap memory.

| Test ID | Subject Hierarchy | Cardinality | Randomize | Pub Subject | Sub Subject | Explicit Test Objective |
| :--- | :--- | ---:| :--- | :--- | :--- | :--- |
| **`TEST-SC-1`** | `benchmark.core.test` | 1 | False | Exact match | Exact match | Single-subject baseline for trie cache comparison. |
| **`TEST-SC-100`** | `benchmark.core.test.N`| 100 | True | Random (0-99) | Random (0-99) | Evaluate 100 dynamic subject branches. |
| **`TEST-SC-1K`** | `benchmark.core.test.N`| 1,000 | True | Random (0-999) | Random (0-999) | Measure sublist trie memory growth across 1k subjects. |
| **`TEST-SC-10K`** | `benchmark.core.test.N`| 10,000 | True | Random (0-9999)| Random (0-9999)| Large taxonomy stress; prove $O(K)$ lookup invariance. |
| **`TEST-SC-WILDCARD`**| `benchmark.core.test.N`| 1,000 | True | Random (0-999) | `...test.>` | Test multi-token wildcard (`>`) routing efficiency parity. |

---

### Suite 4: `jetstream-ladder` (Synchronous File R=3 Saturation Ladder)
* **Configuration File**: `suites/jetstream-ladder.conf`
* **Scenario**: `js-pub-sync` (Synchronous JetStream publish waiting for quorum ACK)
* **Fixed Settings**: Storage = File, Replicas = 3, Max Bytes = 5 GB, Payload = 1,024 B, Publishers = 8, Duration = 30s
* **Primary Objective**: Measure synchronous disk `fsync()` write batching, Raft quorum latency, and identify the rotational/SSD physical storage write barrier.

| Test ID | Target Rate | Storage | Replicas | Publishers | Duration | Explicit Test Objective |
| :--- | ---:| :--- | ---:| ---:| :--- | :--- |
| **`TEST-JL-250`** | 250 /s | File | 3 | 8 | 30s | Isolated single-message fsyncs; high per-message latency baseline. |
| **`TEST-JL-500`** | 500 /s | File | 3 | 8 | 30s | Low-rate disk sync evaluation. |
| **`TEST-JL-1K`** | 1,000 /s | File | 3 | 8 | 30s | Moderate rate; onset of Raft quorum write batching. |
| **`TEST-JL-2.5K`** | 2,500 /s | File | 3 | 8 | 30s | High write concurrency; inverted latency curve acceleration. |
| **`TEST-JL-5K`** | 5,000 /s | File | 3 | 8 | 30s | Performance sweet spot; optimal amortized disk sync commit. |
| **`TEST-JL-10K`** | 10,000 /s | File | 3 | 8 | 30s | Hard storage saturation ceiling; identify maximum disk write limit. |

---

### Suite 5: `jetstream-guarantees` (Cost of Durability & Guarantees)
* **Configuration File**: `suites/jetstream-guarantees.conf`
* **Scenario**: Stepped across Core, Memory JetStream, File R=1, and File R=3
* **Fixed Settings**: Target Rate = 10,000 msgs/s, Payload = 1,024 B, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Isolate the exact latency, CPU, and disk tax imposed by each level of messaging guarantee.

| Test ID | Engine Tier | Storage | Replicas | Max Bytes | Explicit Test Objective |
| :--- | :--- | :--- | ---:| :--- | :--- |
| **`TEST-G1-CORE-INMEM`** | Core NATS | Memory (none)| 0 | N/A | In-memory baseline: zero disk I/O, sub-microsecond latency. |
| **`TEST-G2-JS-MEM-R3`** | JetStream | Memory | 3 | 256 MB | Replicated in-memory streaming: Raft consensus without disk sync. |
| **`TEST-G3-JS-FILE-R1`** | JetStream | File | 1 | 5 GB | Single-node disk persistence: disk sync without Raft network quorum. |
| **`TEST-G4-JS-FILE-R3`** | JetStream | File | 3 | 5 GB | Enterprise standard: disk persistence + 3-node Raft consensus. |

---

### Suite 6: `stream-scaling` (Stream Density & Multi-Raft Scaling)
* **Configuration File**: `suites/stream-scaling.conf`
* **Scenario**: `js-pub-sync` (Active primary stream under load + auxiliary background streams)
* **Fixed Settings**: Target Rate = 1,000 msgs/s, Payload = 1,024 B, Storage = File, Replicas = 3, Max Bytes = 100 MB, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Measure the multi-Raft control-plane tax, heartbeat CPU overhead, and baseline memory growth across 1 to 50 concurrent Raft streams.

| Test ID | Active Streams | Auxiliary Streams | Total Cluster Raft Groups | Cluster Storage Reserved | Explicit Test Objective |
| :--- | ---:| ---:| ---:| ---:| :--- |
| **`TEST-SS-1`** | 1 | 0 | 1 | 300 MB | Single-stream baseline performance. |
| **`TEST-SS-5`** | 1 | 4 | 5 | 1.5 GB | Measure onset of multi-Raft heartbeat loop timers. |
| **`TEST-SS-20`** | 1 | 19 | 20 | 6.0 GB | Evaluate multi-Raft state machine scaling and memory expansion. |
| **`TEST-SS-50`** | 1 | 49 | 50 | 15.0 GB (5 GB/node)| High stream density; evaluate Raft heartbeat aggregation efficiency. |

---

### Suite 7: `consumer-fanout` (Consumer Fan-Out & ACK Amplification)
* **Configuration File**: `suites/consumer-fanout.conf`
* **Scenario**: `js-pub-sync` (Active primary stream with concurrent durable consumers)
* **Fixed Settings**: Target Rate = 1,000 msgs/s, Payload = 1,024 B, Storage = File, Replicas = 3, Max Bytes = 1 GB, Publishers = 4, Subscribers = 4, Duration = 30s
* **Primary Objective**: Measure Persistent Disk write multiplication, network egress amplification, and ACK consensus commit overhead across 1 to 20 durable pull consumers.

| Test ID | Durable Consumers | Total Deliveries | Total ACKs Processed | Explicit Test Objective |
| :--- | ---:| ---:| ---:| :--- |
| **`TEST-CF-1`** | 1 | 60,000 | 60,000 | Single-consumer baseline; measure base consumer ACK write overhead. |
| **`TEST-CF-3`** | 3 | 180,000 | 180,000 | 3x ACK amplification; evaluate concurrent consumer write batching. |
| **`TEST-CF-10`** | 10 | 600,000 | 600,000 | 10x ACK amplification; detect onset of ACK queueing and latency rise. |
| **`TEST-CF-20`** | 20 | 1,200,000 | 1,200,000 | 20x ACK amplification; identify hard disk IOPS / ACK consensus knee. |

---

## 3. Preflight & Baseline Validation Tests

Before executing workload suites, the benchmark harness runs two foundational checks:

| Test ID | Name | Objective | Acceptance Criteria |
| :--- | :--- | :--- | :--- |
| **`TEST-PRE-0.1`** | Cluster & Storage Preflight | Validate NATS route connectivity, Raft quorum, and storage admission quotas via `/jsz`. | All 3 nodes healthy, routes active, declared storage $\le$ cluster max limits. |
| **`TEST-IDLE-0.2`** | Background Idle Baseline | Sample 15s of idle cluster metrics before applying workload. | Node CPU $<0.5\%$, RAM baseline recorded, zero disk write activity. |

---

## 4. Result Telemetry & Metrics Schema Mapping

Every test execution generates structured metrics captured in `results.json` and rolled up in `suite-results.json`:

| Schema Field | Source | Description |
| :--- | :--- | :--- |
| `client.throughput_msgs_sec` | `pub.csv` | Empirical publishing throughput achieved across the test window. |
| `client.latency_us.p50` | `pub.csv` | Median end-to-end publish latency in microseconds. |
| `client.latency_us.p99` | `pub.csv` | 99th percentile tail latency. |
| `cluster.avg_cpu_pct` | `metrics.json` | Arithmetic mean of CPU utilization across all 3 nodes. |
| `cluster.total_peak_mem_mb` | `metrics.json` | Sum of peak RAM consumed across all 3 nodes. |
| `nodes[].disk_growth_bytes` | Directory scan | Net physical bytes written to `/data/jetstream` per node. |
| `metadata` | `test-config.txt` | Complete snapshot of 20+ runtime parameters for audit repeatability. |
