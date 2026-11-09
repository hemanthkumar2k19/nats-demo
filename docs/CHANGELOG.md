# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic Versioning.

## [Unreleased] - 2026-11-09

### Fixed
- **Suite Executive Report Generation**: Resolved bug preventing `suite.sh` from producing or rendering `executive-report.html` and `suite-report.html` on external systems:
  - Exported caller `NATS_URL` into child test subshells so remote endpoints are not clobbered by defaults in `benchmark.conf`.
  - Preserved `TIMESTAMP_SUBFOLDER` across `benchmark.sh` configuration sourcing, preventing unexpected nested test output directories.
  - Replaced literal sentinel string matching with multi-line regex substitution in `aggregate_suite.py` and `aggregate.py`, eliminating sensitivity to whitespace or CRLF line endings across different environments.
  - Hardened `compute_recommendations()` and rate parsing in `aggregate_suite.py` with `safe_float` and `safe_int` coercions, preventing unhandled exceptions on unthrottled (zero rate), missing, or malformed metrics.
  - Implemented recursive child result discovery (`child.rglob("results.json")`) and synthetic fallback run records from `test-config.txt` so `executive-report.html`, `suite-report.html`, and `suite-results.json` are always generated even during partial or failed runs.
  - Updated `static/report.html` suite rendering: dynamically sets `#status-badge` (`COMPLETED`, `PARTIAL`, `FAILED`, or `NO DATA`) and `#sub-header`, adds responsive SVG `viewBox="0 0 800 340"`, and activates the comparison tab prior to SVG curve calculation.

### Added
- **Developer Ergonomics and Inline Architectural Documentation**: Added comprehensive argument contract headers (e.g. 17-parameter positional table in `workload.sh`, invocation signature in `metrics.sh`, subshell isolation documentation in `suite.sh`), inline rationale for the environment-save/restore pattern in `benchmark.sh`, explanation of the Podman cumulative `cpu_time` delta calculation, JetStream consumer setup rationale, and delivery efficiency calculation formulas for broadcast fan-out vs queue groups.
- **Template Injection Safeguards**: Added explicit warning comments and programmatic verification in `aggregate.py` and `aggregate_suite.py` to check for the `<script id="benchmark-data">` sentinel tag before attempting HTML report generation, preventing silent failures.
- **Structured Dashboard Code Sections**: Added numbered ASCII section dividers in `static/report.html` organizing the client-side logic into discrete modules (State & Utilities, Single-Run KPI/Latency/Node Rendering, Suite Comparison Table, Child Run Drill-Down, SVG S-Curve, Configurator, and File Drop/DOM Initialization).

### Changed
- **README.md architecture and execution modernization**: Updated root `README.md` to align with the unified configuration and reporting architecture:
  - Replaced the outdated 4-file decoupled configuration guide (`config/*.conf`) with the canonical `benchmark.conf` parameters and environment variable override semantics.
  - Modernized the Execution Guide to include single-configuration invocations, runtime environment overrides, and the interactive Web UI runner (`./benchmark.sh --serve`).
  - Corrected Section 6 (Supported Workload Scenarios) to match current engine capabilities: Core NATS (`pubsub`, `publish`, `subscribe` with `QUEUE` fan-out controls) and JetStream (`js-pub-sync`, `js-pub-async`, `js-consume`).
  - Aligned Section 7 (Results & Output Artifacts) with the canonical artifact inventory (`results.json`, `report.html`, `metrics.json`, `pub.csv`, `sub.csv`, `test-config.txt`, `suite-results.json`, `executive-report.html`), eliminating deprecated file references.
  - Updated cross-reference links in Section 9 to point to `docs/tests.md`, `docs/tests-jetstream.md`, and `docs/benchmarking.md`.
- **Documentation synchronized with unified architecture**: Updated `docs/benchmarking.md`, `docs/tests.md`, and `docs/tests-jetstream.md` to remove legacy artifact references (`summary.csv`, `node-summary.csv`, `metrics.csv`, `workload.log`, `subscriber.log`, `executive-summary.md`, and `config/*.conf`). Updated workflow examples and artifact inventories to reference canonical files (`benchmark.conf`, `results.json`, `metrics.json`, `pub.csv`, `sub.csv`, `suite-results.json`, `report.html`, and `executive-report.html`).
- **Suite `.conf` files updated to canonical variable names**: All four suite configuration files (`core-ladder.conf`, `jetstream-guarantees.conf`, `jetstream-ladder.conf`, `payload-scaling.conf`) updated from legacy override names (`TARGET_MSGS_PER_SEC`, `PUBLISHER_CONCURRENCY`, `SUBSCRIBER_CONCURRENCY`, `TEST_DURATION_SECONDS`, `MESSAGE_SIZE`, `STORAGE_TYPE`, `REPLICATION_FACTOR`, `SUBSCRIPTION_PATTERN`) to current canonical names (`THROUGHPUT`, `PUBS`, `SUBS`, `DURATION`, `SIZE`, `STORAGE`, `REPLICAS`, `QUEUE`). Stale config file references (`config/core-nats.conf`, `config/jetstream.conf`) replaced with `benchmark.conf` in all suite entries.
- **JetStream suite entries now specify `SCENARIO` explicitly**: `jetstream-guarantees.conf` and `jetstream-ladder.conf` entries now include `SCENARIO=js-pub-sync` and `STORAGE`/`REPLICAS` overrides, making each test case self-describing without relying on a deleted config file to carry those settings.

### Removed
- **Dead fallback paths in `benchmark.sh`**: Removed unreachable fallback lookup for `config/benchmark.conf` and `config/core-nats.conf` (the `config/` directory was deleted in the prior refactoring session).
- **`summary.csv` backwards-compatibility write in `aggregate.py`**: Removed the `summary.csv` output that was marked "backwards compatibility". The canonical output is `results.json`. Nothing in the current pipeline reads `summary.csv`.

## [Unreleased] - 2026-11-05

### Added
- **Unified Configuration Model (`benchmark.conf`)**: Consolidated disparate configuration files into a single configuration matching `nats bench` CLI flags (`--server`, `--clients`, `--throughput`, `--duration`, `--size`, `--stream`, `--consumer`, `--batch`, `--acks`).
- **Constant Static HTML Dashboard (`static/report.html`)**: Introduced a permanent, self-contained, zero-dependency HTML dashboard with single-run KPI cards, latency tables, dynamic SVG throughput-vs-latency curves, and a visual configurator with "Copy Command" and "Download benchmark.conf" capabilities.
- **Dual Execution Modes (CLI + Web UI)**: Added optional runner server (`scripts/server.py` invoked via `./benchmark.sh --serve`) allowing users to configure and trigger benchmark runs directly from their browser, while terminal execution via `./benchmark.sh` remains standard.
- **Direct CSV Ingestion**: Integrated native `nats bench --csv` flags across all pub, sub, and JetStream operations, eliminating regex log scraping.

### Removed
- **Runtime Python HTML & SVG Generation**: Removed 2,000+ lines of Python string concatenation in `scripts/aggregate.py` and `scripts/aggregate_suite.py`.
- **Runtime Markdown Summaries**: Permanently removed `executive-summary.md` generation.
- **Redundant Scripts**: Removed obsolete `scripts/results.sh` and split config files in `config/`.

### Changed
- **Massive Codebase Reduction**: Reduced overall benchmark harness lines of code from 3,643 to 1,467 lines (~60% net reduction).
- **Streamlined Aggregators**: Slashed `scripts/aggregate.py` from 1,635 to 208 lines and `scripts/aggregate_suite.py` from 578 to 66 lines.
- **Local Environment Hardening & Resource Enforcements (`deploy/`)**:
  - Configured `max_mem_store: 512MB` and `max_file_store: 10GB` in `nats-{1,2,3}.conf` to prevent unbounded memory or host disk exhaustion under saturation workloads.
  - Added connection buffers and write deadlines (`write_deadline: "5s"`, `max_pending: 32MB`, `max_payload: 8MB`) to eliminate slow-consumer buffer accumulation in container RAM.
  - Hardened `deploy/compose.yaml` with explicit swap isolation (`memswap_limit`), `restart: unless-stopped`, and `stop_grace_period: 15s`.
  - Configured `maxconn 4096` and `timeout queue 10s` in `deploy/haproxy.cfg` to protect Podman's user-mode network proxy (`gvproxy`) from ephemeral socket starvation.
  - Documented Podman Machine sizing guidelines (>= 5 vCPUs, >= 6 GiB RAM) in `README.md`.
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
- **Suite Matrix Calibration & Single-Variable Parity**:
  - Calibrated `suites/jetstream-guarantees.conf` with `SUBSCRIPTION_PATTERN=queue` on Core NATS baseline for exact 1x load-balanced delivery matching JetStream pull consumer semantics.
  - Standardized `suites/jetstream-ladder.conf` with constant 8 publishers and 2 consumers across all 6 steps (250 to 10k msg/s) to isolate offered load as the sole independent variable.
  - Standardized `suites/payload-scaling.conf` at fixed 5,000 msg/s with 1x queue delivery, isolating message size (128 B to 32 KiB) without network broadcast multiplication.
  - Added post-suite stream cleanup in `suite.sh`.
- **Publisher Watchdog & Subscriber Drain Refinement**:
  - Expanded publisher watchdog in `scripts/workload.sh` to `DURATION + PUBLISHER_GRACE_SECONDS` to guarantee healthy workloads write their full stats tables without premature interruption.
  - Eliminated premature `kill -INT` signal broadcast on subscribers, allowing clean drain completion.
- Refactored `benchmark.sh` timeout handling to distinguish between nominal duration, grace window, and hard watchdog termination.
- Upgraded report visual hierarchy with dedicated setup details, test details, and results cards.

