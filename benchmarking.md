# NATS Capacity Benchmarking

## 1. Overview

The objective of NATS Capacity Benchmarking is to understand NATS Server resource consumption under different messaging workloads and identify sustainable throughput limits.

The benchmarking exercise focuses on:

- Understanding Core NATS and JetStream performance characteristics.
- Measuring CPU, memory, network and storage consumption.
- Understanding the impact of message size, concurrency, replication and consumer configuration.
- Identifying maximum sustainable throughput under a fixed infrastructure configuration.
- Generating resource consumption data to support infrastructure capacity planning and TCO estimation.

The benchmarking approach follows a controlled, workload-driven methodology inspired by Synadia's NATS benchmarking exercises.

## 2. Benchmarking Test Scenarios

Benchmarking is divided into Core NATS and JetStream. Each messaging type has three primary scenarios.

### 2.1 Core NATS

| Test Scenario | Description | Objective |
|---|---|---|
| Publish | Publishers send messages without active subscribers. | Measure publishing throughput and server resource consumption. |
| Subscribe Setup | Subscribers establish connections and subscriptions without active publishers. | Understand connection and subscription overhead. |
| Publish + Subscribe | Publishers and subscribers operate simultaneously. | Measure message delivery throughput and fan-out impact. |

**Considerations:**

- Core NATS does not persist messages.
- Subscribe Setup is not a message-throughput benchmark; it measures subscription and connection overhead.
- Normal Pub/Sub and Queue Group consumption should be treated as separate workload profiles.
- In normal Pub/Sub, each matching subscriber receives a copy of the message.
- In Queue Groups, messages are distributed among group members.

### 2.2 JetStream

| Test Scenario | Description | Objective |
|---|---|---|
| Publish | Publishers send messages to a JetStream Stream without active consumers. | Measure persistence and replication write performance. |
| Consume | Consumers retrieve previously persisted messages without concurrent publishing. | Measure historical consumption and replay throughput. |
| Publish + Consume | Publishers and consumers operate simultaneously. | Measure sustained throughput under concurrent publishing, persistence, replication and consumption. |

**Considerations:**

- Publish-only tests should run without active consumers.
- Consume-only tests should use a pre-populated Stream.
- Publish + Consume tests should run publishing and consumption concurrently.
- Consumer throughput should be recorded both per consumer and in aggregate.
- The initial consumption profile will use durable pull consumers.

## 3. Benchmark Dimensions

Benchmark dimensions define the workload and NATS configuration used during each test.

### 3.1 Common Workload Dimensions

Applicable to both Core NATS and JetStream.

| Dimension | Description | Initial Values |
|---|---|---|
| Message Size | Payload size of individual messages. | 1 KiB, 5 KiB, 10 KiB, 100 KiB |
| Publisher Concurrency | Number of concurrent publisher clients. | 1, 2, 4, 8 |
| Consumer Concurrency | Number of concurrent subscriber/consumer clients. | 1, 2, 4 |
| Target Throughput | Configured publishing rate. | Unthrottled initially; fixed-rate tests subsequently |
| Test Volume | Total payload volume processed. | At least 1 GiB per run as a reference |
| Test Duration | Measurement duration. | Consistent duration sufficient to reach steady state |
| Subject Distribution | Number of subjects used by the workload. | Single subject initially |

Publisher and consumer concurrency should be varied independently to understand their individual impact.

### 3.2 Core NATS Dimensions

| Dimension | Description |
|---|---|
| Publisher Clients | Number of concurrent publishing clients. |
| Subscriber Clients | Number of concurrent subscribing clients. |
| Subscription Pattern | Normal Pub/Sub or Queue Group. |
| Fan-out | Number of subscribers receiving each published message. |
| Subject Distribution | Single or multiple subjects. |

### 3.3 JetStream Stream Dimensions

Stream configuration determines persistence, replication and storage characteristics.

| Dimension | Description | Initial Baseline |
|---|---|---|
| Storage Type | File or Memory storage. | File |
| Replication Factor | Number of Stream replicas. | 3 |
| Retention Policy | Limits, WorkQueue or Interest. | Limits |
| Stream Count | Number of Streams. | 1 |
| Subjects per Stream | Number of subjects captured by a Stream. | 1 |
| Maximum Stream Bytes | Configured storage limit. | Fixed and documented |

The initial benchmark should use one Stream, one subject, File storage, replication factor 3 and Limits retention.

Additional configurations should be evaluated independently rather than testing every possible combination.

### 3.4 JetStream Consumer Dimensions

Consumer configuration determines message delivery, acknowledgement and processing behaviour.

| Dimension | Description | Initial Baseline |
|---|---|---|
| Consumer Type | Durable or Ephemeral. | Durable |
| Delivery Mode | Pull or Push. | Pull |
| ACK Policy | Explicit, None or All. | Explicit |
| Consumer Count | Number of independent consumers. | 1 |
| Worker Concurrency | Number of workers processing messages per consumer. | 1 |
| Fetch Batch Size | Number of messages fetched per request. | Fixed and documented |
| Max ACK Pending | Maximum number of unacknowledged messages. | Fixed and documented |
| ACK Wait | Acknowledgement timeout. | Fixed and documented |

**Important distinctions:**

- Consumer count and worker concurrency are independent dimensions.
- Multiple workers can share a consumer, whereas multiple consumers maintain independent consumption state.
- Pull and Push delivery should be compared as separate workload profiles.
- ACK policies should be evaluated independently.
- Durable versus Ephemeral consumer comparison can be introduced as a subsequent experiment.

## 4. Initial Benchmark Configuration

The following configuration establishes a consistent starting point for the initial benchmark series.

### 4.1 Infrastructure Baseline

| Parameter | Configuration |
|---|---|
| NATS Nodes | 3 |
| Deployment | Fixed three-node cluster |
| Infrastructure Resources | Fixed CPU, Memory and Storage limits |
| Client Location | Separate from NATS server nodes |
| Client Connectivity | Exposed NATS endpoint / HAProxy |
| Monitoring | Available container or platform metrics |

Infrastructure configuration must remain unchanged throughout a benchmark series.

### 4.2 Core NATS Baseline

| Parameter | Configuration |
|---|---|
| Subject | Single benchmark subject |
| Message Size | 1 KiB |
| Publisher Concurrency | 8 |
| Subscriber Concurrency | 1 |
| Subscription Pattern | Normal Pub/Sub |

### 4.3 JetStream Baseline

**Stream Configuration**

| Parameter | Configuration |
|---|---|
| Stream Count | 1 |
| Subjects | 1 |
| Storage | File |
| Replication | 3 |
| Retention | Limits |

**Consumer Configuration**

| Parameter | Configuration |
|---|---|
| Consumer Type | Durable |
| Delivery Mode | Pull |
| ACK Policy | Explicit |
| Consumer Count | 1 |
| Worker Concurrency | 1 |

All other consumer parameters should be fixed and recorded for repeatability.

## 5. Initial Benchmark Test Matrix

The initial benchmark series should focus on establishing baseline performance and understanding the impact of message size and concurrency.

| Messaging Type | Scenario | Publishers | Consumers | Message Sizes |
|---|---|---:|---:|---|
| Core NATS | Publish | 1, 2, 4, 8 | 0 | 1, 5, 10 KiB |
| Core NATS | Publish + Subscribe | 8 | 1, 2, 4 | 1, 5, 10 KiB |
| JetStream | Publish | 8 | 0 | 1, 5, 10 KiB |
| JetStream | Consume | 0 | 1, 2, 4 | 1, 5, 10 KiB |
| JetStream | Publish + Consume | 8 | 1, 2, 4 | 1, 5, 10 KiB |

For JetStream Consume tests, prepopulate the Stream before starting the measurement.

The initial test matrix is not intended to exhaust every possible configuration. Additional dimensions should be introduced after establishing the baseline.

## 6. Results to Capture

Results are divided into workload performance, infrastructure resource consumption and JetStream-specific measurements.

### 6.1 Workload Performance

| Metric | Description |
|---|---|
| Publish Throughput | Successfully published messages per second. |
| Publish Bandwidth | Published data volume per second (MiB/sec). |
| Consume Throughput | Consumed messages per second. |
| Consume Bandwidth | Consumed data volume per second (MiB/sec). |
| Aggregate Consume Throughput | Total consumption throughput across consumers. |
| Per-Consumer Throughput | Individual consumer throughput. |
| Achieved Throughput | Actual throughput compared with configured target. |
| Total Messages | Successfully published and consumed messages. |
| Test Duration | Actual elapsed measurement duration. |
| Errors / Timeouts | Failed, timed-out or incomplete operations. |

For Core NATS fan-out workloads, distinguish published messages from aggregate subscriber deliveries.

### 6.2 Infrastructure Resource Consumption

Infrastructure metrics should be captured individually for every NATS node.

| Metric | Aggregation |
|---|---|
| CPU Utilization | Average and Peak (%) |
| CPU Consumption | Average and Peak (cores) |
| Memory Utilization | Average and Peak (%) |
| Memory Consumption | Average and Peak (bytes) |
| Network Ingress | Average and Peak (MiB/sec) |
| Network Egress | Average and Peak (MiB/sec) |
| Disk Read Throughput | Average and Peak (MiB/sec) |
| Disk Write Throughput | Average and Peak (MiB/sec) |
| Disk IOPS | Average and Peak, where available |
| Disk Utilization | Used and Available Storage, where available |

Cluster-level CPU and memory consumption should also be calculated.

Per-node results must be retained to identify uneven resource distribution and individual node bottlenecks.

### 6.3 JetStream-Specific Measurements

| Metric | Purpose |
|---|---|
| Stream Message Count | Verify persisted message count. |
| Stream Storage Consumed | Understand persistence footprint. |
| Consumer Pending Messages | Identify consumer backlog. |
| Consumer ACK Pending | Identify unacknowledged messages. |
| Redelivered Messages | Identify repeated deliveries. |
| Consumer Lag | Understand processing delay. |

These measurements should be captured before and after each test, and during execution where practical.

## 7. Benchmark Execution Methodology

The benchmark follows a Fixed Infrastructure + Incremental Load approach.

### Step 1: Establish Infrastructure Baseline

- Deploy the NATS cluster.
- Fix CPU, memory and storage limits.
- Verify cluster health and replication configuration.
- Verify client connectivity.
- Capture idle resource consumption.

### Step 2: Execute Workload

- Select a benchmark scenario.
- Configure message size and client concurrency.
- Execute the workload.
- Capture workload and infrastructure metrics.
- Record the actual test duration and processed message volume.

### Step 3: Incremental Load Testing

- Increase target throughput progressively.
- Keep other workload dimensions unchanged.
- Capture resource consumption at each throughput level.
- Identify the maximum sustainable throughput.
- Perform finer-grained tests around the identified capacity boundary.

### Step 4: Configuration Comparison

After establishing the baseline, vary one dimension at a time.

Examples:

- Message size.
- Publisher concurrency.
- Consumer count.
- Worker concurrency.
- Stream storage type.
- Replication factor.
- Retention policy.
- Consumer delivery mode.
- ACK policy.

### Step 5: Result Aggregation

For each test:

- Preserve raw time-series metrics.
- Generate average and peak resource consumption.
- Calculate network and disk throughput from counter deltas.
- Generate per-node summaries.
- Generate cluster-level summaries.
- Record workload configuration and achieved throughput.

### Step 6: Capacity Identification

Identify maximum sustainable throughput based on:

- Successful message processing.
- Acceptable error and timeout rates.
- CPU and memory headroom.
- Network and disk resource consumption.
- Stable consumer backlog and processing behaviour.

The objective is to identify sustainable operating capacity, not the point at which the cluster fails.

## 8. Result Organization

Each benchmark execution should produce the following artifacts.

| Artifact | Description |
|---|---|
| `config.json` | Complete workload and NATS configuration. |
| `metrics.csv` | Raw infrastructure time-series metrics. |
| `summary.csv` | Aggregated results for each benchmark test. |
| `node-summary.csv` | Aggregated resource metrics for each NATS node. |
| `workload.log` | Benchmark execution output and errors. |

Each result should be identifiable by its messaging type, scenario and configuration.

## 9. Benchmarking Phases

| Phase | Scope | Objective |
|---|---|---|
| Phase 1 | Core NATS and JetStream baseline scenarios | Establish baseline performance and resource consumption. |
| Phase 2 | Message size and publisher/consumer concurrency | Understand workload scaling characteristics. |
| Phase 3 | JetStream Stream configuration | Understand storage, replication and retention impact. |
| Phase 4 | JetStream Consumer configuration | Understand ACK, delivery mode, consumer and worker concurrency impact. |
| Phase 5 | Capacity analysis | Identify sustainable throughput and infrastructure sizing requirements. |

## 10. Expected Outcome

The benchmarking exercise should provide the following:

1. Maximum sustainable message throughput for Core NATS and JetStream.
2. Resource consumption at different workload levels.
3. Per-node and cluster-level CPU, memory, network and storage requirements.
4. Impact of message size and client concurrency.
5. Impact of JetStream replication, storage and consumer configuration.
6. Resource requirements for expected production workloads.
7. Data to support NATS infrastructure capacity planning and TCO estimation.

## References
- https://www.synadia.com/downloads/nats-kafka-tco-report.pdf
