# JetStream Capacity Benchmark – Test Inventory

## 1. Objective and Methodology

The objective is to establish an empirical performance baseline for JetStream (persisted, replicated messaging) and understand how storage engines, replication factors, consumer configurations, and workload characteristics influence throughput and infrastructure resource consumption.

The methodology adheres to the **OpenMessaging Benchmark (OMB)** streaming framework and **Synadia's enterprise NATS vs Kafka performance equivalence standards** (McKnight Consulting Group evaluation), utilizing time-bound workloads across standardized 1 KiB payloads, multi-node Raft consensus quorums ($R=3$), persistent File storage, and durable pull consumers with explicit acknowledgments.

JetStream introduces key architectural dimensions beyond Core NATS:
- **Raft Consensus Replication**: Synchronous replication quorum across 3 nodes ($R=3$).
- **Disk Persistence**: Write-ahead commit log and index persistence to `/data/jetstream`.
- **Consumer Delivery Contracts**: Durable pull consumers, explicit receipt acknowledgments (`Explicit` ACK), and historical message replay.
- **Resource Footprint**: Memory-mapped index/data buffers, journal commit logs, and disk I/O read/write saturation.

### Benchmark Environment

| Parameter | Configuration |
|---|---|
| NATS Cluster | 3 nodes (`nats-1`, `nats-2`, `nats-3`) |
| CPU per NATS node | 1 vCPU (enforced via container limits) |
| Memory per NATS node | 1 GiB (enforced via container limits) |
| Storage Directory | `/data/jetstream` on volume mount |
| Default Storage Engine | `File` (disk storage) |
| Default Replication | `3` ($R=3$ Raft consensus cluster) |
| HAProxy | 0.5 vCPU / 512 MiB (Round Robin on port 4222) |
| Client Location | Host system outside Podman containers |
| Messaging Type | JetStream |

### Benchmark Rules

1. Keep infrastructure configuration constant throughout the benchmark series.
2. Change one workload dimension at a time.
3. Capture an idle resource baseline before each benchmark execution.
4. Capture per-node CPU, memory, network I/O, and disk block I/O.
5. In JetStream benchmarks, disk block I/O (Read/Write MB/s) is a primary bottleneck indicator.
6. Manage stream and consumer lifecycles automatically: create, purge between runs, and cleanup.
7. Preserve raw metrics in `metrics.csv` and consolidate test analysis in `report.html`.

---

## 2. Test Execution Workflow

### Option A: Configuration File

Configure the test ID and workload parameters in the JetStream configuration file.

```bash
./benchmark.sh config/jetstream.conf
```

### Option B: Environment Variable Overrides

Override individual workload parameters during execution.

```bash
TARGET_MSGS_PER_SEC=2000 \
PUBLISHER_CONCURRENCY=4 \
SUBSCRIBER_CONCURRENCY=2 \
./benchmark.sh config/jetstream.conf
```

### Option C: Automated Executive Suite Execution

Execute entire test matrices or throughput saturation ladders unattended with automated cluster isolation, 10-second quiescence settling between runs, and consolidated executive reporting (`executive-report.html` and `executive-summary.md`):

```bash
# Run JetStream Cost of Guarantees suite:
./suite.sh jetstream-guarantees

# Run JetStream throughput saturation ladder suite:
./suite.sh jetstream-ladder
```

Each execution produces:

```text
results/
└── <TEST_ID>/
    ├── metrics.csv
    ├── node-summary.csv
    ├── summary.csv
    ├── workload.log
    ├── subscriber.log
    ├── test-config.txt
    └── report.html
```

---

## 3. Test Case Inventory

### Suite 0: Infrastructure and Preflight Validation

Validates JetStream cluster health, storage directory access, and idle resource consumption.

| Test ID | Test Name | Objective | Expected Result |
|---|---|---|---|
| TEST-JS-0.1 | JetStream Cluster Quorum | Verify 3-node JetStream enablement, account quotas, and Raft consensus leader election. | Raft consensus healthy, leader elected, storage paths writable. |
| TEST-JS-0.2 | Pre-Workload Idle Baseline | Capture 15s-60s background resource usage without active streams. | Per-node idle CPU (~0.2-0.3%), RAM (~6-10MB), zero disk write rate. |

---

### Suite 1: Baseline Reference Tests (File Storage, $R=3$)

Establishes reference measurements for fundamental JetStream workloads under enterprise durability settings.

| Test ID | Scenario | Publishers | Consumers | Payload | Target Rate | Objective |
|---|---|---:|---:|---|---|---|
| TEST-JS-1.0 | JetStream Smoke Test | 1 | 1 | 1 KiB | 500 msg/s | Verify stream creation, sync publishing, pull consumption, and HTML report generation. |
| TEST-JS-1.1 | Publish-only Baseline | 4 | 0 | 1 KiB | Unthrottled | Measure raw Raft write rate, consensus commit latency, and server disk write MB/s. |
| TEST-JS-1.2 | Consume-only Replay | 0 | 2 | 1 KiB | Unthrottled | Replay from a pre-populated 60,000-message stream; measure disk read MB/s and batch ACK throughput. |
| TEST-JS-1.3 | Concurrent Pub + Consume | 4 | 2 | 1 KiB | 2,000 msg/s | Measure concurrent disk read/write contention, delivery latencies, and server resource consumption. |

---

### Suite 2: Dimensional Scaling Tests

These tests isolate individual JetStream dimensions while keeping remaining parameters constant.

#### Matrix A: Publisher Concurrency Scaling (Publish-Only, $R=3$, File)

**Objective:** Understand how publisher concurrency affects write throughput, Raft commit batching, and disk write rates.

Fixed settings:
- Scenario: Publish-only.
- Consumers: 0.
- Payload: 1 KiB.
- Storage: File, Replicas: 3.
- Rate: Unthrottled.
- Publish mode: Synchronous (`pub sync`).

| Test ID | Publishers | Target Rate | Objective |
|---|---:|---|---|
| TEST-JS-2.A1 | 1 | Unthrottled | Establish single-publisher synchronous write performance. |
| TEST-JS-2.A2 | 2 | Unthrottled | Evaluate scaling with two concurrent publishers. |
| TEST-JS-2.A3 | 4 | Unthrottled | Evaluate multi-publisher scaling and server write batching. |
| TEST-JS-2.A4 | 8 | Unthrottled | Evaluate higher publisher concurrency and server disk contention. |

#### Matrix B: Message Payload Scaling (Publish-Only, 4 Pubs, $R=3$, File)

**Objective:** Understand the impact of message size on disk write bandwidth (MB/s) and server I/O saturation.

Fixed settings:
- Scenario: Publish-only.
- Publishers: 4.
- Consumers: 0.
- Storage: File, Replicas: 3.
- Rate: Unthrottled.

| Test ID | Payload Size | Objective |
|---|---:|---|
| TEST-JS-2.B1 | 1 KiB | Small message baseline (high IOPS, lower bandwidth). |
| TEST-JS-2.B2 | 5 KiB | Medium message scaling. |
| TEST-JS-2.B3 | 10 KiB | Large message throughput evaluation. |
| TEST-JS-2.B4 | 100 KiB | Extended message payload (disk write bandwidth saturation). |

#### Matrix C: Consumer Concurrency & Batch Size Scaling (Consume-Only Replay)

**Objective:** Understand pull consumer batch fetch efficiency, disk read rates, and acknowledgment processing.

Fixed settings:
- Scenario: Consume-only.
- Pre-populated messages: 60,000 messages.
- Payload: 1 KiB.
- ACK policy: `Explicit`.
- Delivery mode: `Pull` (callback).

| Test ID | Consumers | Batch Size | Objective |
|---|---:|---:|---|
| TEST-JS-2.C1 | 1 | 500 | Establish single-consumer pull replay baseline. |
| TEST-JS-2.C2 | 2 | 500 | Evaluate dual-consumer parallel replay. |
| TEST-JS-2.C3 | 4 | 500 | Evaluate multi-consumer replay distribution. |
| TEST-JS-2.C4 | 2 | 100 | Evaluate smaller pull batch size overhead. |
| TEST-JS-2.C5 | 2 | 1000 | Evaluate larger pull batch size efficiency. |

#### Matrix D: Replication Factor Comparison ($R=1$ vs $R=3$)

**Objective:** Quantify the overhead of Raft consensus quorum: write amplification across 3 nodes and inter-node cluster network traffic (port 6222).

Fixed settings:
- Scenario: Publish-only.
- Publishers: 4.
- Payload: 1 KiB.
- Storage: File.
- Rate: Unthrottled.

| Test ID | Replication Factor | Storage | Objective |
|---|---:|---|---|
| TEST-JS-2.D1 | 1 | File | Single-node write performance without Raft consensus overhead. |
| TEST-JS-2.D2 | 3 | File | Standard 3-node Raft consensus quorum with 3x write amplification. |

#### Matrix E: Storage Engine Comparison (File vs Memory)

**Objective:** Isolate disk I/O latency from the JetStream streaming engine to identify the pure in-memory streaming ceiling.

Fixed settings:
- Scenario: Concurrent Pub + Consume.
- Publishers: 4, Consumers: 2.
- Payload: 1 KiB.
- Replicas: 3.
- Rate: 2,000 msg/s.

| Test ID | Storage Engine | Replicas | Objective |
|---|---|---:|---|
| TEST-JS-2.E1 | File | 3 | Standard persistent disk storage. |
| TEST-JS-2.E2 | Memory | 3 | In-memory JetStream storage (zero disk I/O, RAM-backed Raft). |

#### Matrix F: Publish Acknowledgment Mode (Sync vs Async)

**Objective:** Evaluate the throughput and latency impact of pipelined asynchronous publishing versus synchronous per-message waiting.

Fixed settings:
- Scenario: Publish-only.
- Publishers: 4.
- Payload: 1 KiB.
- Storage: File, Replicas: 3.
- Rate: Unthrottled.

| Test ID | Publish Mode | Objective |
|---|---|---|
| TEST-JS-2.F1 | `sync` | Synchronous JetStream publish (client waits for server consensus ACK). |
| TEST-JS-2.F2 | `async` | Pipelined asynchronous publish with in-flight message windows. |

---

### Suite 3: Target-Rate Load Ladder (Capacity Boundary)

**Objective:** Identify the maximum sustainable JetStream throughput under fixed infrastructure limits (1 vCPU, 1 GB RAM, R=3 File storage).

Fixed settings:
- Scenario: Concurrent Pub + Consume.
- Publishers: 4, Consumers: 2.
- Payload: 1 KiB.
- Storage: File, Replicas: 3.
- Duration: 60 seconds per rate step.

| Test ID | Target Publish Rate | Duration | Objective |
|---|---:|---:|---|
| TEST-JS-3.1 | 1,000 msg/s | 60 sec | Low-load baseline verification. |
| TEST-JS-3.2 | 2,500 msg/s | 60 sec | Moderate workload evaluation. |
| TEST-JS-3.3 | 5,000 msg/s | 60 sec | Elevated load and disk write evaluation. |
| TEST-JS-3.4 | 10,000 msg/s | 60 sec | High load near saturation boundary. |
| TEST-JS-3.5 | Unthrottled | 60 sec | Maximum achievable throughput boundary under time-bound saturation. |

Capacity criteria:
- **Delivery Rate**: 100% of published messages successfully delivered and acknowledged.
- **Server CPU**: Peak CPU across all nodes $<80\%$.
- **Publish Latency**: P99 synchronous publish latency $<15\text{ ms}$.
- **Storage**: Stable disk write throughput without unbounded message accumulation.

---

## 4. Result Tracking

Every benchmark execution captures:

| Category | Measurements |
|---|---|
| Configuration | Test ID, messaging type (`JetStream`), storage (`file`/`memory`), replicas (`1`/`3`), stream name, payload size, publishers, consumers, publish mode (`sync`/`async`) |
| Throughput | Achieved publish rate (msgs/s) and consumer delivery rate (msgs/s) |
| Latencies | Min, Avg, P50, P90, P99, P99.9, Max synchronous publish latency |
| Disk Block I/O | Per-node Disk Read MB/s and Disk Write MB/s (avg and peak), plus empirical physical storage growth (bytes/MB) |
| Server CPU | Per-node and cluster-wide average and peak CPU (% and vCPU) |
| Server Memory | Per-node and cluster-wide average and peak RAM |
| Network I/O | Per-node Network RX/TX throughput (client ingress + Raft replication) |
| Execution | Nominal duration, actual duration, and exit status |

---

## 5. Benchmark Deliverables

The JetStream capacity benchmark establishes:
1. Baseline synchronous publishing and pull consumer delivery performance.
2. Quantified cost of Raft consensus ($R=1$ vs $R=3$ write amplification).
3. Disk I/O saturation points and throughput limits for File storage.
4. Memory-backed vs disk-backed JetStream performance profiles.
5. Maximum sustainable throughput boundary for enterprise capacity planning.

