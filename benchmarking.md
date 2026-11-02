# NATS Capacity Benchmarking

## Objective

For proposing NATS TCO, we need to understand the resource consumption of the current NATS deployment by conducting capacity benchmarking in the Dev environment.

The objective is to understand how workload characteristics influence infrastructure resource consumption and use the observations as inputs for capacity planning.

## Deployment

- 3 NATS Cluster Deployments
  - Cluster communication through routes on port `6222`.
  - JetStream enabled.
    - `max_memory_store`: 512 MB
    - `max_file_store`: 5 GB
  - Client port `4222` exposed.
  - Monitoring port `8222` exposed.
- Kubernetes Service
- Kubernetes NodePort
- NATS Box
  - Connects through NodePort: `{OCP_WORKER_HOST_IP:30222}`.
  - Includes NATS CLI and benchmarking utilities.
  - Reference: [NATS Box](https://github.com/nats-io/nats-box).

## Benchmarking

### 1. Workload Inputs

Define the workload characteristics that will be varied during benchmarking.

| Input                 | Description                          |
| --------------------- | ------------------------------------ |
| Messaging Type        | Core NATS / JetStream                |
| Message Throughput    | Target messages published per second |
| Message Size          | Payload size in bytes                |
| Publisher Concurrency | Number of concurrent publishers      |
| Consumer Concurrency  | Number of concurrent consumers       |

For JetStream, keep the following configuration fixed initially:

- Replication factor: 3 (subject to verification).
- Retention policy: Limits.
- Stream count: 1.

### 2. Benchmark Client – NATS Bench

Use `nats bench` available through the deployed NATS Box to generate controlled publishing and consumption workloads.

The benchmark client will be used to:

- Generate messages at a specified throughput.
- Vary message size.
- Vary publisher concurrency.
- Vary consumer concurrency.
- Capture achieved throughput and latency.

The same benchmark commands can also be executed from a local machine, provided it has connectivity to the NATS endpoint and the required credentials.

### 3. Capacity Metrics

Collect the following metrics while executing benchmark workloads.

| Metric                           | Description                           |
| -------------------------------- | ------------------------------------- |
| CPU Utilization                  | CPU consumption per NATS node         |
| Memory Utilization               | Memory consumption per NATS node      |
| Network Ingress/Egress           | Network traffic per NATS node         |
| Disk I/O                         | Disk read/write throughput and IOPS   |
| Disk Utilization                 | Persistent storage consumption        |
| Publish Throughput               | Actual messages published per second  |
| Consume Throughput               | Actual messages consumed per second   |
| Publish Latency                  | Publishing latency                    |
| Consumer Acknowledgement Latency | Acknowledgement latency for JetStream |


## Implementation Plan

### Deployment Configuration Constraints
