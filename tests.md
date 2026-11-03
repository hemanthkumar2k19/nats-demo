# NATS Capacity Benchmark - Test Case Inventory & Matrices

A structured catalog of benchmark test cases, parameter scaling matrices, and execution procedures designed to evaluate Core NATS and JetStream cluster performance.

---

## 1. Test Methodology & Baseline Rules

All tests adhere to the **Controlled Variable Methodology**:
- **Fixed Infrastructure**: 3-node cluster, 1.0 CPU / 1 GB RAM per NATS node, 0.5 CPU / 512 MB for HAProxy.
- **Idle Baseline**: Every test captures a 15-second idle baseline before workload execution.
- **Single-Variable Variation**: Each test isolates exactly one dimension (concurrency, payload size, fan-out ratio, or distribution pattern).
- **Automated Evidence**: Every run automatically saves immutable records (`metrics.csv`, `summary.csv`, `node-summary.csv`, and `report.html`) in a timestamped folder under `results/<TIMESTAMP>/`.

---

## 2. Test Execution Workflow

Tests can be executed using any of the following standard workflows:

### Option A: Configuration File (Recommended)
Edit `config/core-nats.conf` with the parameters specified in the test catalog, then run:
```bash
cd nats-capacity-benchmark
./benchmark.sh
```

### Option B: Inline Environment Variable Overrides
Run directly without modifying configuration files on disk:
```bash
cd nats-capacity-benchmark

# Example: Run Publish-Only with 4 publishers at 5,000 msgs/sec for 30s:
SCENARIO=publish PUBLISHER_CONCURRENCY=4 TARGET_MSGS_PER_SEC=5000 TEST_DURATION_SECONDS=30 ./benchmark.sh

# Example: Run Queue Group load-balancing with 4 workers:
SCENARIO=pubsub SUBSCRIPTION_PATTERN=queue SUBSCRIBER_CONCURRENCY=4 ./benchmark.sh

# Example: Run Subscribe Setup overhead benchmark:
SCENARIO=subscribe_setup SUBSCRIBER_CONCURRENCY=50 TEST_DURATION_SECONDS=15 ./benchmark.sh
```

---

## 3. Test Case Catalog

### Suite 0: Infrastructure & Preflight Validation
Validates cluster health, port reachability, and baseline idle stability before load generation.

| Test ID | Test Name | Target / Command | Pass Criteria |
| :--- | :--- | :--- | :--- |
| **TEST-0.1** | Preflight Health Check | `./benchmark.sh --check` | Exit 0; all ports (`:4222`, `:8404`, `:8222-:8224`) and cluster routes active. |
| **TEST-0.2** | Idle Baseline Floor | `./benchmark.sh` (idle window) | Cluster CPU < 1.0%, RAM 10-25 MB/node, 0 MB/s Net & Block IO. |

---

### Suite 1: Baseline Reference Tests
Validates core functional scenarios under standard 1 KiB payload and baseline concurrency.

| Test ID | Scenario | Pattern | Pubs | Subs | Payload | Target Rate | Duration | Objective & Verification |
| :--- | :--- | :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **TEST-1.0** | `pubsub` | `pubsub` | 1 | 1 | 1 KiB | 1,000 | 10s | **Quick Smoke Test**: Verify end-to-end workload and `report.html` generation. |
| **TEST-1.1** | `pubsub` | `pubsub` | 8 | 1 | 1 KiB | 5,000 | 60s | **Core NATS Baseline**: Benchmark standard from `benchmarking.md:L140`. CPU < 10%. |
| **TEST-1.2** | `publish` | `pubsub` | 8 | 0 | 1 KiB | 5,000 | 60s | **Publish-Only Baseline**: Pure ingress routing; subscriber card displays `N/A`. |
| **TEST-1.3** | `subscribe_setup` | - | 0 | 50 | - | - | 15s | **Subscribe Setup Overhead**: Measures connection rate (>50 conns/s) and RAM/conn. |

---

### Suite 2: Dimensional Scaling Matrices

#### Matrix A: Publisher Concurrency Scaling
Measures multi-client TCP efficiency and HAProxy round-robin distribution.
- **Fixed Settings**: `SCENARIO="publish"`, `SUBS=0`, `Payload=1 KiB`, `Rate=10,000 msgs/s`, `Duration=30s`.

| Test ID | Pubs | Target Rate | Key Observation & Validation |
| :--- | :---: | :---: | :--- |
| **TEST-2.A1** | 1 | 10,000 msgs/s | Single-client socket throughput; measure client latency distribution. |
| **TEST-2.A2** | 2 | 10,000 msgs/s | Two-node HAProxy distribution (e.g. `nats-1` and `nats-2` active). |
| **TEST-2.A3** | 4 | 10,000 msgs/s | Symmetrical distribution across all 3 backend cluster nodes. |
| **TEST-2.A4** | 8 | 10,000 msgs/s | High client concurrency; observe server CPU headroom and connection pooling. |

#### Matrix B: Payload Size & Wire Bandwidth
Measures network throughput scaling and socket memory buffering.
- **Fixed Settings**: `SCENARIO="pubsub"`, `PUBS=2`, `SUBS=2`, `Rate=2,000 msgs/s`, `Duration=30s`.

| Test ID | Payload | Wire Throughput | Key Observation & Validation |
| :--- | :---: | :---: | :--- |
| **TEST-2.B1** | 1 KiB (1,024 B) | ~2.0 MB/s | Baseline payload size; verify low socket buffer utilization. |
| **TEST-2.B2** | 5 KiB (5,120 B) | ~10.0 MB/s | Linear network ingress/egress scaling (~10 MB/s in `report.html`). |
| **TEST-2.B3** | 10 KiB (10,240 B) | ~20.0 MB/s | High bandwidth wire transfer; monitor network interface counters. |
| **TEST-2.B4** | 100 KiB (102,400 B) | ~200.0 MB/s | Heavy payload stress; observe cluster route socket buffer growth in RAM. |

#### Matrix C: Fan-Out Multipliers (Normal Pub/Sub Broadcast)
Measures inter-cluster route replication overhead as 1 publication is copied to $N$ subscribers.
- **Fixed Settings**: `SCENARIO="pubsub"`, `PATTERN="pubsub"`, `PUBS=1`, `Payload=1 KiB`, `Rate=2,000 msgs/s`, `Duration=30s`.

| Test ID | Subs | Fan-Out | Pub Rate | Delivery Rate | Cluster Egress | Key Observation & Validation |
| :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **TEST-2.C1** | 1 | 1x | 2,000 | 2,000 msgs/s | ~2.2 MB/s | 1:1 publication to delivery parity. |
| **TEST-2.C2** | 2 | 2x | 2,000 | 4,000 msgs/s | ~4.4 MB/s | 2x multiplier; verify delivered msgs in `report.html`. |
| **TEST-2.C3** | 4 | 4x | 2,000 | 8,000 msgs/s | ~8.8 MB/s | 4x multiplier; check route forward traffic across all nodes. |
| **TEST-2.C4** | 8 | 8x | 2,000 | 16,000 msgs/s | ~17.6 MB/s | High broadcast fan-out; observe server CPU replication cost. |

#### Matrix D: Queue Groups (Load-Balancing Pattern)
Measures worker distribution where messages are shared across subscribers (Fan-out = 1x).
- **Fixed Settings**: `SCENARIO="pubsub"`, `PATTERN="queue"`, `QUEUE_NAME="benchmark-core-group"`, `PUBS=1`, `Payload=1 KiB`, `Rate=2,000 msgs/s`, `Duration=30s`.

| Test ID | Workers (`SUBS`) | Fan-Out | Pub Rate | Aggregate Delivery | Per-Worker Share | Key Observation & Validation |
| :--- | :---: | :---: | :---: | :---: | :---: | :--- |
| **TEST-2.D1** | 2 | 1x | 2,000 | 2,000 msgs/s | ~1,000 msgs/s | Verified load sharing between 2 workers. |
| **TEST-2.D2** | 4 | 1x | 2,000 | 2,000 msgs/s | ~500 msgs/s | 4-worker load balancing; verify `Fan-out: 1x` card. |
| **TEST-2.D3** | 8 | 1x | 2,000 | 2,000 msgs/s | ~250 msgs/s | 8-worker distribution; verify even message allocation. |

#### Matrix E: Subject Cardinality
Measures subject trie resolution overhead and sublist memory footprint.
- **Fixed Settings**: `SCENARIO="pubsub"`, `PUBS=2`, `SUBS=2`, `Payload=1 KiB`, `Rate=2,000 msgs/s`, `Duration=30s`.

| Test ID | Distribution | Subject Pattern | Cardinality | Key Observation & Validation |
| :--- | :--- | :--- | :---: | :--- |
| **TEST-2.E1** | `single` | `benchmark.core.test` | 1 | Baseline single-subject lookup. |
| **TEST-2.E2** | `multi` | `benchmark.core.test.*` | 1,000 | Tokenized subject routing (`--multisubject`); verify route trie cost. |
| **TEST-2.E3** | `multi` | `benchmark.core.test.*` | 10,000 | High cardinality routing; verify server sublist memory stability. |

---

### Suite 3: Capacity & Boundary Identification (Load Ladder)
Progressively increases throughput to identify maximum sustainable throughput before latency or CPU saturation.
- **Fixed Settings**: `SCENARIO="pubsub"`, `PUBS=4`, `SUBS=2`, `Payload=1 KiB`, `Duration=60s`.

| Test ID | Target Rate | Total Messages | Evaluation Focus & Headroom Thresholds |
| :--- | :---: | :---: | :--- |
| **TEST-3.1** | 5,000 msgs/s | 300,000 | Low load step; verify CPU < 15% and P99 latency < 50 us. |
| **TEST-3.2** | 15,000 msgs/s | 900,000 | Moderate load step; verify CPU 25-35% and linear network scaling. |
| **TEST-3.3** | 30,000 msgs/s | 1,800,000 | Substantial load; observe CPU distribution across cluster nodes. |
| **TEST-3.4** | 50,000 msgs/s | 3,000,000 | Near-capacity step; check if P99 latency remains within acceptable SLA. |
| **TEST-3.5** | Unthrottled (`0`) | Max Achievable | Maximum client throughput limit; identify saturation boundary. |

---

### Suite 4: JetStream Planned Test Suite (Roadmap)
Baseline configurations from `benchmarking.md:L150-L185` (R=3, File Storage, Limits Retention, Durable Pull, Explicit ACK).

| Test ID | Test Name | Scenario | Pubs | Subs | Payload | Evaluation Focus |
| :--- | :--- | :--- | :---: | :---: | :---: | :--- |
| **TEST-4.1** | JS Publish Baseline | Publish to Stream | 8 | 0 | 1, 5, 10 KiB | Disk persistence write rate and Raft consensus replication latency. |
| **TEST-4.2** | JS Consume Baseline | Consume Prepopulated | 0 | 1, 2, 4 | 1, 5, 10 KiB | Historical stream replay throughput and batch ACK processing. |
| **TEST-4.3** | JS Pub + Consume | Concurrent Pub & Sub | 8 | 1, 2, 4 | 1, 5, 10 KiB | Sustained concurrent persistence, replication, and consumer delivery. |

---

## 4. Test Results Tracking Sheet

| Test ID | Date / Time | Target Rate | Pub Rate | Sub Rate | Fan-Out | Peak CPU | Peak RAM | Net RX/TX (MB/s) | P99 Latency | Status |
| :--- | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| TEST-0.1 | `<TIMESTAMP>` | N/A | N/A | N/A | N/A | N/A | N/A | N/A | N/A | PASS |
| TEST-1.0 | `<TIMESTAMP>` | 1,000 | 999 msgs/s | 1,000 msgs/s | 1x | 1.07% | 39.7 MiB | 2.18 / 2.21 | 15.83 us | PASS |
| TEST-1.1 | | 5,000 | | | 1x | | | | | |
| TEST-1.2 | | 5,000 | | N/A | 0x | | | | | |
| TEST-1.3 | | N/A | N/A | N/A | 0x | | | | | |
| TEST-2.A1 | | 10,000 | | N/A | 0x | | | | | |
| TEST-2.B2 | | 2,000 | | | 1x | | | | | |
| TEST-2.C3 | | 2,000 | | | 4x | | | | | |
| TEST-2.D2 | | 2,000 | | | 1x | | | | | |
| TEST-3.1 | | 5,000 | | | 1x | | | | | |
