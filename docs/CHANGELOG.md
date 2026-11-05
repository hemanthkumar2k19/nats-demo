# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased] - 2026-11-05

### Added
- **Executive Benchmark Suite Runner (`suite.sh`)**: Added automated multi-test execution harness supporting sequential test matrices with cluster isolation (`nats stream rm -f`), mandatory 10-second quiescence cooldowns between runs, and automatic executive aggregation.
- **Predefined Executive Suites (`suites/*.conf`)**:
  - `jetstream-guarantees.conf`: Compares Core NATS, JetStream Memory R=3, JetStream File R=1, and JetStream File R=3.
  - `core-ladder.conf`: Core NATS throughput saturation ladder (1k to 100k msgs/s).
  - `jetstream-ladder.conf`: JetStream sync saturation ladder (250 to 10,000 msgs/s).
  - `payload-scaling.conf`: Payload dimension scaling from 128 B to 32 KiB.
- **Executive Comparative Aggregator (`scripts/aggregate_suite.py`)**:
  - Computes executive cluster sizing recommendations (sustained capacity, peak rate, hardware bottlenecks, storage delta).
  - Generates zero-dependency inline SVG Throughput vs Latency saturation curve ("hockey stick" curve) with P50/P99 latency knee callout.
  - Compiles the "Cost of Guarantees" comparative matrix table with deep-links to individual test reports.
  - Produces standalone `executive-report.html` and portable `executive-summary.md`.
- **Empirical Host Directory Disk Sizing**: Host filesystem scanner (`os.scandir`) measuring physical JetStream storage growth per node under `deploy/data/nats-{1,2,3}` to ensure accurate storage telemetry on platforms where container cgroup blkio is suppressed.
- **Strict POSIX Monitor Mode (`set -m`)**: Enforced monitor mode in `scripts/workload.sh` to prevent POSIX `SIG_IGN` background signal masking and ensure reliable graceful shutdown of `nats bench`.

### Changed
- Refactored `benchmark.sh` timeout handling to distinguish between nominal duration, grace window, and hard watchdog termination.
- Upgraded report visual hierarchy with dedicated setup details, test details, and results cards.

