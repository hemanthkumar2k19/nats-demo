# NATS Capacity Benchmark - User & Execution Guide

A lightweight, portable benchmarking utility to evaluate NATS cluster throughput, latency distributions, and node resource consumption (CPU, Memory, Network I/O, Disk Block I/O).

---

## 1. Architecture Overview

The benchmarking utility tests a fixed multi-node NATS cluster through an L4 TCP load balancer:

```
[ Benchmark Runner (Host) ]
            |
            v  (Port :4222)
    [ HAProxy L4 LB ]  (:4222 traffic, :8404 stats)
      /     |     \
     /      |      \
    v       v       v
[nats-1] [nats-2] [nats-3]  (3-node cluster, full mesh routing :6222)
```

- **Client Isolation**: The benchmark runner executes on the host machine outside container resource limits.
- **L4 Distribution**: Client connections are balanced across backend NATS nodes via HAProxy round-robin.
- **Resource Constraints**: Enforced at container level (1.0 CPU and 1 GB RAM per NATS node; 0.5 CPU and 512 MB for HAProxy).
- **Telemetry Collection**: Podman statistics collector samples per-container CPU, RAM, Network RX/TX, and Disk Block Read/Write in the background.

---

## 2. Prerequisites

Ensure the following tools are installed on your workstation:
- **Bash** (v3.2+ compatible)
- **NATS CLI** (`nats`)
- **Python 3** (Standard library only; no external pip dependencies)
- **Podman** & **Podman Compose**
- **jq**

---

## 3. Environment Setup & Deployment

The 3-node NATS cluster and HAProxy load balancer are configured in `deploy/compose.yaml`.

```bash
# 1. Start the cluster and load balancer in the background:
cd deploy
podman compose up -d

# 2. Verify containers are healthy and running:
podman ps

# 3. Access HAProxy live health and connection statistics:
open http://127.0.0.1:8404
```

To tear down the environment after testing:
```bash
cd deploy
podman compose down
```

---

## 4. Configuration Guide

Workload and environment configurations are strictly decoupled under `config/`:

### A. Environment Configuration (`config/environment.conf`)

Controls connection endpoints, metrics collectors, and output paths:

```bash
# Endpoint for benchmark traffic (points to HAProxy L4 listener)
NATS_URL="nats://127.0.0.1:4222"

# Infrastructure telemetry backend: "podman" or "none"
METRICS_BACKEND="podman"

# Podman container names to monitor
NATS_CONTAINERS="nats-1 nats-2 nats-3"

# Telemetry sample interval (seconds) and idle baseline duration (seconds)
METRICS_SAMPLE_INTERVAL=5
BASELINE_DURATION_SECONDS=15

# Results directory (relative to nats-capacity-benchmark/ or absolute path)
RESULTS_DIR="results"
```

### B. Core NATS Workload Configuration (`config/core-nats.conf`)

Controls the messaging profile, scenarios, concurrency, and volume:

```bash
# 1. Test Scenario:
#   pubsub          - Concurrent publish and subscribe (default)
#   publish         - Publishers send without active subscribers
#   subscribe_setup - Connection and subscription establishment overhead
SCENARIO="pubsub"

# 2. Subscription Pattern (for pubsub scenario):
#   pubsub          - Normal Pub/Sub broadcast (fan-out = SUBSCRIBER_CONCURRENCY)
#   queue           - Queue group load balancing (fan-out = 1x)
SUBSCRIPTION_PATTERN="pubsub"
QUEUE_NAME="benchmark-core-group"

# 3. Subject Distribution:
#   single          - Single subject (SUBJECT)
#   multi           - Multiple subjects (--multisubject token distribution)
SUBJECT="benchmark.core.test"
SUBJECT_DISTRIBUTION="single"
MULTI_SUBJECT_MAX=10000

# 4. Workload Rate & Concurrency Limits:
TARGET_MSGS_PER_SEC=1000       # Target publication rate ceiling (msgs/sec)
MESSAGE_SIZE=1024              # Message payload size in bytes (e.g. 1024 = 1 KiB)
PUBLISHER_CONCURRENCY=1        # Concurrent publisher client connections
SUBSCRIBER_CONCURRENCY=1       # Concurrent subscriber client connections
TEST_DURATION_SECONDS=60       # Nominal test duration in seconds
```

---

## 5. Execution Guide

Navigate to the benchmark workspace:
```bash
cd nats-capacity-benchmark
```

### 1. Run Preflight Health Check
Validates local dependencies, port reachability (`:4222`, `:8404`, monitoring ports `:8222-:8224`), and cluster routes:
```bash
./benchmark.sh --check
```

### 2. Run Default Benchmark
Executes the workload defined in `config/core-nats.conf` against `config/environment.conf`:
```bash
./benchmark.sh
```

### 3. Run with Custom Configuration Files
Execute specific workloads without modifying base configuration files:
```bash
./benchmark.sh path/to/custom-workload.conf path/to/custom-env.conf
```

### 4. Re-aggregate Past Test Runs
Regenerate summary CSVs and HTML reports from an existing test directory without re-running the test:
```bash
python3 scripts/aggregate.py results/<TEST_ID>
```

---

## 6. Supported Core NATS Workload Scenarios

1. **Publish + Subscribe (`SCENARIO="pubsub"`)**:
   - Publishers and subscribers run simultaneously.
   - **Broadcast Mode (`SUBSCRIPTION_PATTERN="pubsub"`)**: Each subscriber receives a copy of all published messages ($Fan\text{-}out = \text{Subscriber Concurrency}$).
   - **Queue Group Mode (`SUBSCRIPTION_PATTERN="queue"`)**: Subscribers join a shared queue group, load-balancing messages across workers ($Fan\text{-}out = 1\times$).
2. **Publish-Only (`SCENARIO="publish"`)**:
   - Publishers push messages without active subscribers to measure raw publishing throughput, connection handling, and server ingress routing without delivery costs.
3. **Subscribe Setup (`SCENARIO="subscribe_setup"`)**:
   - Measures connection and subscription establishment overhead without message traffic. Reports connection rate (`conns/sec`) and server memory overhead per subscription.

---

## 7. Results & Output Artifacts

Every benchmark run creates an immutable, timestamped directory under `results/<TIMESTAMP>/`:

| File | Description | Primary Purpose |
| :--- | :--- | :--- |
| **`report.html`** | Standalone, self-contained offline HTML report with dark mode, responsive KPI cards, latency percentiles, node infrastructure table, cluster totals, and print/PDF export styling. | Visual review, architecture reports, executive dashboards |
| **`summary.csv`** | Benchmark execution summary with test parameters, achieved publication/delivery rates, fan-out multiplier, cluster average/peak CPU and RAM, and exit codes. | Automated data collection, CI/CD, spreadsheet analysis |
| **`node-summary.csv`** | Per-container resource consumption breakdown: average and peak CPU %, RAM usage, Network RX/TX (MB/s), and Block Read/Write (MB/s). | Node balance analysis, identifying cluster bottlenecks |
| **`metrics.csv`** | Raw time-series telemetry sampled every `METRICS_SAMPLE_INTERVAL` seconds across all monitored cluster nodes. | Time-series charting, deep-dive resource analysis |
| **`workload.log`** | Execution console output from publisher clients and NATS CLI. | Latency percentiles (min, avg, P50, P90, P99, P99.9, max) |
| **`subscriber.log`** | Execution console output from subscriber clients. | Delivery verification, bandwidth confirmation, connection setup stats |
| **`test-config.txt`** | Complete snapshot of runtime parameters and configurations used during the test. | Test auditability, repeatability |

---

## 8. Operational Notes & Troubleshooting

- **Rootless Podman**: When running rootless Podman on macOS, ensure the Podman machine has sufficient memory allocated (`podman machine info`).
- **Core NATS Block I/O**: Block read and write throughput for Core NATS will correctly read `0.000 MB/s` because Core NATS operates purely in-memory.
- **HAProxy Round-Robin**: If publisher or subscriber concurrency is less than the cluster node count (e.g. 1 pub and 1 sub across 3 nodes), only the connected nodes will observe workload ingress/egress. Idle nodes will show ~1 KB/s of background cluster heartbeat traffic.

---

## 9. Next Steps & Cross-References

- To plan and select test matrices across concurrency, payload sizes, and fan-out multipliers, see the **[Test Case Inventory (`tests.md`)](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/%20capacity-bench/tests.md)**.
- For the architectural design, evaluation methodology, and capacity planning standards, see the **[Benchmarking Design Document (`benchmarking.md`)](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/%20capacity-bench/benchmarking.md)**.
