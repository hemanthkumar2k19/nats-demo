# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/) and adheres to [AGENTS.md](AGENTS.md).

## 2026-09-08

### Added
- Delivery Sequence, First Sequence, & Cursor State educational model with number line diagram and formula Q&A in Stage 2 (NATS View & CLI) info popover.
- Interactive "Recreate JOBS Stream (Seq 1)" action in NATS Demo View and direct `job-service` API (`POST /stream/reset` on `:8081`) to reset stream sequence to 1 and consumer cursor to 0 on demand.
- Multi-worker resiliency scenarios in Failure Lab: worker crash failover (Scenario 7), slow worker timeout racing (Scenario 8), and dynamic worker pool scaling (Scenarios 9 & 10).
- Lightweight execution toggle `MODE=demo` (and `PROCESSOR_MODE`) to run strictly JetStream pull workers without transient subscriber overhead.

### Changed
- Reorganized Stage 3 UI to prioritize the JetStream `job-processor` durable consumer view above worker details.
- Suppressed periodic `/health` polling logs in `job-service` router.

### Fixed
- Decoupled NATS broker health indicator from microservice health in header, preventing false `NATS DISCONNECTED` alerts.
- Added dual-channel status checks (HTTP `:8082` fallback) in frontend and `demo-control-service`.
- Registered control responders in `processor-service` under `demo` mode so `status.processor` responds to NATS pings.
- Filtered JetStream broker duplicate publications from transient subscriber console logs.

### Removed
- Fully pruned legacy Consumer Lab feature from UI, API gateway, and backend services.
- Removed legacy Saga Orchestration module, worker responders, and HTTP routes.

### Documentation
- Converted internal mechanism diagrams to Mermaid syntax and consolidated demonstrated capabilities into a single NATS-centric table in `docs/demo.md`.

## 2026-09-07

### Added
- Single-worker failure scenarios in Failure Lab: worker crash before ACK, processing exceeds AckWait (7s), explicit NAK retry, and terminal poison pill (TERM).
- Direct lightweight HTTP diagnostic server on `processor-service` (:8082) for direct state, status, and failure scenario controls.
- Generic NATS message publisher supporting custom subjects, transport modes, and arbitrary wire headers.
- Reference guides for NATS terminology (`docs/NATS.md`) and consumer failure testing (`docs/consumer_testing.md`).

### Changed
- Modularized backend into two standalone Go modules: `nats-demo/services` (`job-service`, `processor-service`) and `nats-demo/control` (`demo-control-service`).
- Modularized `processor-service` into clean, single-responsibility files (`main.go`, `worker.go`, `validation.go`, `queue_group.go`, `control.go`).
- Refocused NATS Demo View around an end-to-end 3-stage lifecycle: 1. Submit Job -> 2. NATS Broker & CLI -> 3. Process Job.
- Added multi-account authentication support (`app_user` for tenant APP, `sys_admin` for system monitoring).

### Removed
- Removed legacy `CURRENT DEMO SETUP` panel and obsolete `nats-ui` container to keep focus on native CLI and direct developer experience.

## 2026-09-04

### Added
- Dead Letter Queue (`JOBS_DLQ`) stream and management panel with on-demand provisioning, payload inspection, purging, and message reprocessing.
- Configurable JetStream Deliver Policies (`all`, `new`, `last`, `last_per_subject`) and Ack Policies (`explicit`, `none`, `all`).

### Changed
- Migrated JetStream client calls across all services to the modern `jetstream.New(nc)` API (`CreateOrUpdateStream`, `CreateOrUpdateConsumer`, and `Consumer.Fetch`).
- Separated Activity Log into NATS Business Messages (Category A) and Platform / Worker Telemetry (Category B), with inline message journey tracking.

## 2026-09-03

### Added
- Interactive 2-operation event-driven Saga pattern (`saga.start` -> `Op 1: Reserve` -> `Op 2: Payment` -> `Completed` / `Compensate: Release`) demonstrating distributed rollbacks over NATS.
- Segmented category switcher in Activity Log with live counters distinguishing business messages from internal lifecycle events.

### Documentation
- Overhauled root `README.md` with capability evaluation matrix, ASCII architecture diagram, and quickstart tour.

## 2026-09-02

### Added
- Core NATS Queue Groups demonstration (`jobs.queue`, group `job-workers`) illustrating stateless 1-of-N load balancing alongside JetStream stateful consumers.
- Dynamic worker pool scaling (1 to 5 competing workers) with real-time per-worker distribution tracking.

### Changed
- Standardized distributed tracing on W3C Trace Context (`traceparent` header over NATS) and Grafana Tempo, deprecating legacy correlation IDs.
- Renamed primary publisher service from `demo-service` to `job-service`.
- Enhanced educational popovers (`natsInfo.ts`) structured around Role, Concepts, Demo Usage, and Architecture Trivia.

## 2026-09-01

### Added
- Distributed tracing with OpenTelemetry and Grafana Tempo across HTTP and NATS boundaries.
- Prometheus metrics observability via `nats-exporter` and Grafana dashboard covering health, throughput, and JetStream storage.
- Multi-token subject wildcard addressing demonstration comparing single-level (`jobs.*`) and multi-level (`jobs.>`) routing.
- Real-time search and filter toolbar in Activity Log supporting full-text search, event filters, and worker filters.
- JetStream message deduplication using `Nats-Msg-Id` within a 2-minute deduplication window.

### Documentation
- Created end-to-end `docs/FUNCTIONAL_TESTING_GUIDE.md` covering 11 hands-on evaluation scenarios.

## 2026-08-31

### Added
- Initial project architecture: Go microservices (`job-service`, `processor-service`) and React SPA dashboard.
- Core NATS transient Pub/Sub and JetStream durable streaming comparison on `jobs.submitted`.
- Synchronous Request/Reply demonstration on `jobs.validate` with natural timeout propagation.
- Subject addressing observation for exact, single-level (`*`), and multi-level (`>`) subscriptions.
- JetStream stream replay capability and Job Details Inspector with timeline tracking.
