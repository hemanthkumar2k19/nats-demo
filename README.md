# NATS Capacity Benchmark

A lightweight utility to benchmark NATS throughput and measure resource consumption across a NATS cluster.

## Prerequisites

Ensure the following tools are installed:

- Bash
- NATS CLI (`nats`)
- Python 3
- Podman (for local container metrics)
- jq

## Configuration

Update the configuration files under `config/`.

**1. `environment.conf`**

Configure the target environment:

```bash
NATS_URL="nats://127.0.0.1:4222"
METRICS_BACKEND="podman"
NATS_CONTAINERS="nats-1 nats-2 nats-3"
METRICS_SAMPLE_INTERVAL=5
BASELINE_DURATION_SECONDS=15
RESULTS_DIR="results"
```

**2. `core-nats.conf`**

Configure the Core NATS workload:

```bash
SUBJECT="benchmark.core.test"
TARGET_MSGS_PER_SEC=1000
MESSAGE_SIZE=1024
PUBLISHER_CONCURRENCY=1
SUBSCRIBER_CONCURRENCY=1
TEST_DURATION_SECONDS=60
```

## Running the Benchmark

Navigate to the project directory:

```bash
cd nats-capacity-benchmark
```

Execute the benchmark using the orchestrator script:

```bash
# Run infrastructure validation only
./benchmark.sh --check

# Run benchmark normally
./benchmark.sh

# Run benchmark with custom configs
./benchmark.sh config/core-nats.conf config/environment.conf
```

## Results

Each execution creates a timestamped directory under `results/`.

| File | Description |
|---|---|
| `test-config.txt` | Configuration used for the benchmark |
| `workload.log` | Publisher workload output |
| `subscriber.log` | Subscriber workload output |
| `metrics.csv` | Time-series resource metrics |
| `node-summary.csv` | Per-node average and peak resource summary |
| `summary.csv` | Benchmark execution summary and cluster-wide aggregates |

## Notes

- The benchmark client runs on the host machine, outside the NATS containers.
- `NATS_URL` should point to the endpoint being benchmarked, such as HAProxy.
- `NATS_CONTAINERS` must match the actual Podman container names when using Podman metrics.
- Network and block I/O metrics depend on the availability of statistics from the underlying environment.
- Block I/O measurements represent bytes transferred, not physical disk IOPS.
- Client-side CPU and memory consumption are not included in NATS container metrics.