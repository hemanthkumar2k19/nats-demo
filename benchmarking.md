# NATS Capacity Benchmarking

## Objective

Understand NATS Server CPU, memory, network and storage resource consumption under different messaging workloads to support infrastructure capacity planning and TCO estimation.

## Benchmarking

### 1. Workload Inputs

| Input | Description |
|---|---|
| Messaging Type | Core NATS / JetStream |
| Message Throughput | Target messages published per second |
| Message Size | Payload size in bytes |
| Publisher Concurrency | Number of concurrent publishers |
| Consumer Concurrency | Number of concurrent consumers |

For JetStream, keep the following configuration fixed initially:

- Replication factor: 3 (subject to verification).
- Retention policy: Limits.
- Stream count: 1.

### 2. Benchmark Client – NATS Bench

Use `nats bench`, available through the deployed NATS Box, to generate controlled publishing and consumption workloads.

The benchmark client will be used to:

- Generate messages at a specified throughput.
- Vary message size.
- Vary publisher concurrency.
- Vary consumer concurrency.
- Capture achieved publishing and consumption throughput.

The same benchmark commands can also be executed from a local machine, provided it has connectivity to the NATS endpoint and the required credentials.

### 3. Capacity Metrics (Outputs)

Focus on the following infrastructure metrics:

| Metric | What to Capture | Purpose |
|---|---|---|
| CPU Utilization | Average and peak CPU consumption per NATS node (cores and %) | Understand compute requirements |
| Memory Utilization | Average and peak memory consumption per NATS node (MB/GB) | Understand memory requirements |
| Network Ingress/Egress | Incoming and outgoing network throughput per NATS node (MB/s) | Understand network capacity requirements |
| Disk I/O | Disk read/write throughput and IOPS per NATS node | Understand storage performance requirements |
| Disk Utilization | Persistent storage consumed per NATS node (GB) | Understand storage capacity requirements |

### 4. Benchmark Results

For every workload execution, capture the following:

| Information | Description |
|---|---|
| Test Configuration | Workload input parameters |
| CPU Utilization | Average and peak per node |
| Memory Utilization | Average and peak per node |
| Network Ingress | Average and peak per node |
| Network Egress | Average and peak per node |
| Disk Read/Write | Average and peak throughput per node |
| Disk Utilization | Storage consumed per node |

Also record achieved publish and consume throughput alongside the results to validate whether the requested workload was generated and consumed successfully.

## Measurement Approach

Use the monitoring facilities already available in each environment rather than introducing additional monitoring infrastructure.

| Metric | Potential Measurement Source |
|---|---|
| CPU | OCP Pod Metrics / container metrics |
| Memory | OCP Pod Metrics / container metrics |
| Network | Node/container network metrics, if available |
| Disk I/O | Node/container storage metrics, if available |
| Disk Utilization | Persistent Volume metrics |

NATS native monitoring can supplement infrastructure measurements with server-level CPU, memory and network counters. However, it does not expose all disk I/O metrics.

## Execution Strategy

1. Start with Core NATS and establish baseline resource consumption.
2. Vary one workload input at a time while keeping other inputs constant.
3. Record per-node resource consumption for every workload execution.
4. Repeat the exercise with JetStream, including persistence, replication and storage-related resource consumption.
5. Use the observed resource consumption against workload throughput to estimate infrastructure capacity requirements and support TCO calculations.
