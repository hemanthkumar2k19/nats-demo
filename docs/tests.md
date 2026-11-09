# Core NATS Capacity Benchmark - Test Inventory

## 1. Objective and Methodology

The objective is to establish an empirical performance baseline for Core NATS and understand how workload characteristics influence throughput and infrastructure resource consumption.

The methodology adheres to the **OpenMessaging Benchmark (OMB)** framework and **Synadia's enterprise NATS benchmarking standards**, evaluating time-bound workloads across standardized 1 KiB payloads, publisher/subscriber concurrency, broadcast fan-out, queue groups, and dynamic subject routing.

The benchmark focuses on:

- Publisher concurrency.
- Subscriber concurrency.
- Message payload size.
- Subscription delivery patterns.
- Subject cardinality.
- Sustainable throughput and resource utilization.

### Benchmark Environment

| Parameter | Configuration |
|---|---|
| NATS Cluster | 3 nodes |
| CPU per NATS node | 1 vCPU |
| Memory per NATS node | 1 GiB |
| HAProxy | 0.5 vCPU / 512 MiB |
| Load Balancing | Round Robin |
| Client Location | Outside Podman containers |
| Messaging Type | Core NATS |

### Benchmark Rules

1. Keep infrastructure configuration constant throughout the benchmark series.
2. Change one workload dimension at a time.
3. Capture a 60-second idle resource baseline before benchmark execution.
4. Capture per-node CPU, memory, network I/O and block I/O.
5. Record achieved publishing and subscription delivery throughput.
6. Repeat important benchmark configurations three times to assess consistency.
7. Preserve raw telemetry in `metrics.json`, benchmark metrics in `pub.csv`/`sub.csv`, and consolidate other test information in `results.json` and `report.html`.

---

## 2. Test Execution Workflow

### Option A: Configuration File

Configure the test ID and workload parameters in `benchmark.conf`.

```bash
./benchmark.sh
```

### Option B: Environment Variable Overrides

Override individual workload parameters during execution.

```bash
PUBS=4 \
THROUGHPUT=5000 \
./benchmark.sh
```

### Option C: Automated Executive Suite Execution

Execute entire test ladders or comparative matrices unattended with automated cluster isolation, 10-second quiescence settling between runs, and consolidated executive reporting (`executive-report.html` and `suite-report.html`):

```bash
# Run Core NATS saturation ladder suite:
./suite.sh core-ladder

# Run payload dimension scaling suite:
./suite.sh payload-scaling
```

Each execution should produce:

```text
results/
  <TEST_ID>/
    metrics.json
    pub.csv
    sub.csv
    test-config.txt
    results.json
    report.html
```

---

## 3. Test Case Inventory

### Suite 0: Infrastructure and Preflight Validation

Validates cluster health, connectivity and idle resource consumption.

| Test ID | Test Name | Objective | Expected Result |
|---|---|---|---|
| TEST-0.1 | Preflight Health Check | Verify NATS connectivity, cluster routes and HAProxy backend health. | All expected nodes healthy and client connectivity successful. |
| TEST-0.2 | Idle Baseline | Capture idle CPU, memory, network and block I/O. | Establish per-node idle resource baseline. |

### Suite 1: Baseline Reference Tests

Establishes reference measurements for fundamental Core NATS workloads.

| Test ID | Scenario | Publishers | Subscribers | Payload | Objective |
|---|---|---:|---:|---|---|
| TEST-1.0 | Pub/Sub Smoke Test | 1 | 1 | 1 KiB | Verify workload execution, message delivery and report generation. |
| TEST-1.1 | Pub/Sub Baseline | 8 | 1 | 1 KiB | Establish baseline publishing and subscription delivery performance. |
| TEST-1.2 | Publish-only Baseline | 8 | 0 | 1 KiB | Measure publishing throughput without subscribers. |
| TEST-1.3 | Subscription Setup | 0 | 50 | N/A | Evaluate connection and subscription setup overhead. |

### Suite 2: Dimensional Scaling Tests

These tests isolate individual workload dimensions while keeping the remaining parameters constant.

#### Matrix A: Publisher Concurrency Scaling

**Objective:** Understand how publisher concurrency affects publishing throughput, CPU utilization and connection distribution.

Fixed settings:
- Scenario: Publish-only.
- Subscribers: 0.
- Payload: 1 KiB.
- Publish rate: Unthrottled.
- Total payload volume: At least 1 GiB per run.

| Test ID | Publishers | Target Rate | Objective |
|---|---:|---|---|
| TEST-2.A1 | 1 | Unthrottled | Establish single-publisher performance. |
| TEST-2.A2 | 2 | Unthrottled | Evaluate throughput scaling with two publishers. |
| TEST-2.A3 | 4 | Unthrottled | Evaluate multi-publisher scaling. |
| TEST-2.A4 | 8 | Unthrottled | Evaluate higher publisher concurrency and resource consumption. |

#### Matrix B: Message Payload Scaling

**Objective:** Understand the impact of message size on throughput, network bandwidth and resource consumption.

Fixed settings:
- Scenario: Publish-only.
- Publishers: 8.
- Subscribers: 0.
- Publish rate: Unthrottled.
- Total payload volume: At least 1 GiB per run.

| Test ID | Payload Size | Objective |
|---|---:|---|
| TEST-2.B1 | 1 KiB | Establish small-message baseline. |
| TEST-2.B2 | 5 KiB | Evaluate medium-message performance. |
| TEST-2.B3 | 10 KiB | Evaluate larger-message performance. |

100 KiB can be included as an optional extended workload.

#### Matrix C: Normal Pub/Sub Fan-out

**Objective:** Understand the additional delivery and network overhead when one publication is delivered to multiple subscribers.

Fixed settings:
- Scenario: Normal Pub/Sub.
- Publishers: 1.
- Payload: 1 KiB.
- Aggregate publish rate: 2,000 messages/sec.

| Test ID | Subscribers | Expected Aggregate Delivery | Objective |
|---|---:|---:|---|
| TEST-2.C1 | 1 | 2,000 msg/sec | Establish 1:1 delivery baseline. |
| TEST-2.C2 | 2 | 4,000 msg/sec | Evaluate 2x delivery fan-out. |
| TEST-2.C3 | 4 | 8,000 msg/sec | Evaluate 4x delivery fan-out. |
| TEST-2.C4 | 8 | 16,000 msg/sec | Evaluate higher fan-out overhead. |

#### Matrix D: Queue Group Distribution

**Objective:** Understand how Core NATS distributes messages across queue group subscribers.

Fixed settings:
- Scenario: Queue Group.
- Publishers: 1.
- Payload: 1 KiB.
- Aggregate publish rate: 2,000 messages/sec.
- Queue group name: `benchmark-core-group`.

| Test ID | Queue Workers | Expected Aggregate Delivery | Objective |
|---|---:|---:|---|
| TEST-2.D1 | 2 | 2,000 msg/sec | Evaluate two-worker distribution. |
| TEST-2.D2 | 4 | 2,000 msg/sec | Evaluate four-worker distribution. |
| TEST-2.D3 | 8 | 2,000 msg/sec | Evaluate eight-worker distribution. |

Individual worker distribution is observed, not required to be exactly equal.

#### Matrix E: Subject Cardinality (Optional)

**Objective:** Understand the impact of subject cardinality on routing and resource consumption.

Fixed settings:
- Scenario: Pub/Sub.
- Publishers: 2.
- Subscribers: 2.
- Payload: 1 KiB.
- Aggregate publish rate: 2,000 messages/sec.

| Test ID | Subject Cardinality | Objective |
|---|---:|---|
| TEST-2.E1 | 1 | Establish single-subject baseline. |
| TEST-2.E2 | 1,000 | Evaluate multiple-subject routing. |
| TEST-2.E3 | 10,000 | Evaluate higher subject cardinality. |

---

## 4. Capacity Boundary Identification

### Suite 3: Target-Rate Load Ladder

**Objective:** Identify sustainable publishing throughput and understand resource utilization as offered load increases.

Fixed settings:
- Scenario: Pub/Sub.
- Publishers: 4.
- Subscribers: 2.
- Payload: 1 KiB.
- Test duration: 60 seconds per rate.
- Target rate: Aggregate publishing rate.

| Test ID | Target Publish Rate | Duration | Objective |
|---|---:|---:|---|
| TEST-3.1 | 5,000 msg/sec | 60 sec | Establish low-load behaviour. |
| TEST-3.2 | 15,000 msg/sec | 60 sec | Evaluate moderate-load behaviour. |
| TEST-3.3 | 30,000 msg/sec | 60 sec | Evaluate increased resource consumption. |
| TEST-3.4 | 50,000 msg/sec | 60 sec | Evaluate high-load behaviour. |
| TEST-3.5 | Unthrottled | Until workload completes | Identify maximum achievable throughput. |

Evaluate:
- Achieved publishing throughput.
- Subscription delivery throughput.
- CPU and memory utilization per node.
- Network and block I/O.
- Message delivery correctness.
- Latency, where available.

The maximum achievable throughput is not automatically considered sustainable capacity. Identify the highest operating rate that satisfies the applicable performance and resource requirements.

---

## 5. Result Tracking

Every benchmark execution should capture:

| Category | Measurements |
|---|---|
| Configuration | Test ID, scenario, payload, publishers, subscribers, target rate |
| Throughput | Achieved publish rate and delivery rate |
| CPU | Average and peak per node |
| Memory | Average and peak per node |
| Network | RX/TX throughput per node |
| Block I/O | Read/write throughput per node |
| Latency | Available latency percentiles |
| Correctness | Expected versus received messages |
| Execution | Actual duration and exit status |

Results should be compared across repeated runs to identify consistency and performance variation.

---

## 6. Benchmark Deliverables

The Core NATS benchmark should establish:

1. Baseline publishing and Pub/Sub performance.
2. Impact of publisher concurrency.
3. Impact of message payload size.
4. Impact of subscriber fan-out.
5. Queue group worker distribution behaviour.
6. Impact of subject cardinality, where relevant.
7. Maximum achievable and sustainable throughput.
8. Resource consumption per node at different workload levels.

These measurements form the Core NATS performance baseline for subsequent capacity planning and infrastructure TCO estimation.