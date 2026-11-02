# System Baseline

## Objective

Establish the baseline resource consumption of the NATS system without benchmark-generated workload. These measurements serve as the reference point for identifying incremental resource consumption during subsequent workload tests.

## Baseline Measurements

| Category | Metrics | Purpose |
| --- | --- | --- |
| CPU | CPU utilization per NATS node | Understand idle CPU consumption |
| Memory | Process/container memory per node | Establish memory footprint |
| Network | Network ingress and egress per node | Understand background network activity |
| Disk | Disk read/write throughput and IOPS | Establish background storage activity |
| Storage | Persistent storage consumption | Understand existing storage usage |
| Connections | Active client and route connections | Understand baseline connection count |
| JetStream | Stream and consumer count, stored messages and bytes | Establish the existing persistence footprint |

## Measurement Approach

1. Ensure no benchmark workload is running.
2. Allow the system to settle after deployment or restart.
3. Collect metrics over a fixed observation window (for example, 15-30 minutes).
4. Record average, minimum, and peak values for each resource metric.
5. Repeat measurements if needed to establish a stable baseline.

## Baseline Output Profile

The result should be a baseline profile for the NATS deployment:

| Metric | Measurement unit |
|---|---|
| CPU consumption | CPU cores or core-seconds |
| Memory consumption | MB (RSS) |
| Network ingress | MB/s |
| Network egress | MB/s |
| Disk read throughput | MB/s |
| Disk write throughput | MB/s |
| Active connections | Count |
| JetStream storage | GB |

> [!NOTE]
> For JetStream, also capture the existing stream storage footprint and replication configuration alongside node metrics.

## Baseline Utilization in Benchmarking

During workload benchmarking, compare the measured resource consumption against the baseline:

$$Incremental\ Resource\ Usage = Benchmark\ Resource\ Usage - Baseline\ Resource\ Usage$$

This helps distinguish the resources consumed by the benchmark workload from the resources already consumed by the running system.

## Measurement Scope Separation

Capture both the NATS process/container resource usage and the underlying host or node resource usage where available. They represent distinct measurement scopes and should not be combined:

* **Process / Container Scope**: Direct NATS daemon resource consumption (cgroup CPU/memory, network sockets, JetStream storage directory).
* **Host / Node Scope**: Aggregate machine resource utilization (OS overhead, kernel buffers and page cache, physical NIC, and storage volume saturation).

## Next Steps

Finalize this baseline scope first, then define the exact metrics and telemetry collection sources:
* NATS monitoring endpoints (`/varz`, `/connz`, `/jsz`, `/routez`)
* OpenShift Container Platform (OCP) / Prometheus metrics
* Host-level storage and network telemetry
