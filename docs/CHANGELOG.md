# Changelog

All notable changes to this project will be documented in this file.

## 2026-09-08

### Added (Multiple Worker Failure Scenarios from docs/feature.md)
- **Scenario 7 — One Worker Crashes (`worker.go`, `http_server.go`, `CoreFlowProcessor.tsx`)**:
  - Implemented isolated worker crash semantics in `crash_before_ack`: when 1 worker crashes in a multi-worker pool, surviving workers continue pulling from `job-processor` uninterrupted.
  - Set status to `DEGRADED: X/Y active` with `crashed_worker` identification.
  - Upon AckWait (5s) expiration, surviving healthy peer picks up the redelivered message and ACKs.
  - Supervisor revives the crashed worker after 5.5s, restoring pool capacity.
- **Scenario 8 — One Worker Is Slow (`worker.go`, `CoreFlowProcessor.tsx`)**:
  - In `exceed_ack_wait`, slow worker delays 7s (>5s AckWait) while surviving peer workers continue pulling and executing concurrent jobs without delay.
  - At T = 5s, server marks slow message for redelivery; healthy peer pulls attempt #2 and ACKs at T = 6s.
  - Slow worker wakes up at T = 7s and emits late ACK notice.
- **Scenarios 9 & 10 — Worker Pool Scales Down & Scales Up (`worker.go`, `CoreFlowProcessor.tsx`)**:
  - Verified incremental worker scaling (`ScaleWorkers`): scaling down stops excess workers from the tail while remaining workers process backlog with zero loss; scaling up spawns new goroutines that immediately join the competing pull.

### Documentation
- **Core Flow & Failure Scenarios Comprehensive Guide (`docs/demo.md`)**:
  - Fully refreshed `docs/demo.md` to reflect the reorganized Stage 3 layout (JetStream consumer 2x2 grid on top, followed by `processor-service:8082` non-editable status inputs).
  - Clarified backend endpoints on `:8082` (`PUT /processor/state`, `GET /processor/status`, `PUT /processor/workers`, `PUT /processor/scenario`, `POST /processor/worker/restart`) and documented deprecation of `GET /processor/events`.
  - Documented real-time terminal load distribution summaries (`Pool: [processor-1=X, processor-2=Y]`).
  - Detailed Failure Lab failure states (`RUNNING`, `DEGRADED: X/Y active`, `CRASHED`) and supervisor/manual revival mechanisms.
  - Expanded the 10-step live demonstration runbook and added NATS CLI inspection commands.

### Added (Worker Observability & Incremental Scaling)
- **Restored Aggregated Worker Load Distribution Logging (`backend/services/cmd/processor-service/`)**:
  - Re-introduced `consumerDistribution` map and `consumerDistMu` mutex on `App` in `main.go`.
  - Added `getLoadDistributionSummary()` and `recordWorkerPull()` in `worker.go` to display cumulative per-worker load and pool distribution (`Worker load: N (Pool: [processor-1=X, processor-2=Y])`).
  - Integrated real-time distribution summaries into `[PULLED]`, `[REDELIVERED]`, `[ACK SENT]`, and `[LATE ACK SENT]` console logs.
  - Handled counter reset on `consumer.reset` in `control.go`.
  - Added helper `getLoadDistributionSummary()` to format active pool load distribution (`processor-1=X, processor-2=Y, ...`).
  - Added real-time load distribution logging on message `[PULLED]`: `Worker load: N (Pool: [processor-1=X, ...])`.
  - Added real-time load distribution logging on message `[REDELIVERED]`, `[ACK SENT]`, and `[LATE ACK SENT]`.
- **Incremental Scaling in Control Responders (`backend/services/cmd/processor-service/control.go`)**:
  - Updated `messaging.SubjectConsumerConfigSet` handler to call `a.ScaleWorkers(req.Workers)` instead of restarting the entire worker pool, ensuring zero-disruption worker scaling from all interfaces.

### Changed
- **Stage 3 UI Reorganization & Non-Editable Configuration Fields (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Moved the `job-processor` JetStream Consumer card to the top above `processor-service`.
  - Renamed the consumer badge from `NATS Object` to `JetStream Object`.
  - Converted consumer settings (Attached Stream, Filter Subject, Ack Policy, AckWait Threshold) into styled non-editable form input fields (`readOnly`) with clear uppercase labels.
  - Removed redundant `BOUND WORKER` and `BROKER DURABILITY` fields from the consumer card for a clean 2x2 grid.
  - Converted `processor-service:8082` parameters (Role, Consuming From, Goroutine State) into the same non-editable input field format.
  - Removed the `ACTIVE WORKER` status badge from `processor-service:8082`.
- **Stage 3 UI Failure Lab Cleanup (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Removed heading `Goroutine Failure Scenarios` next to the `FAILURE LAB` badge.
  - Converted the collapsible section indicator from text `[+]`/`[-]` to an animated SVG chevron dropdown arrow.
  - Changed the default state of `isFailureLabOpen` to `false` (collapsed by default).

### Fixed
- **Empty Critical Section in Processor Service (`backend/services/cmd/processor-service/control.go`, `main.go`)**:
  - Eliminated empty critical section (`a.consumerMu.Lock()` / `a.consumerMu.Unlock()`) in the `consumer.reset` responder.
  - Removed unused `consumerMu sync.Mutex` field from struct `App`.
  - Safely reset `consumerDistribution` under `a.consumerDistMu` before sending response, and included zeroed distribution in `ConsumerStatusResponse`.
- **JSX Unescaped Token Fix (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Escaped unescaped literal `>` character as `&gt;` inside JSX text in scenario description to resolve TSX/JSX parser syntax error.
- **Undefined Variable Fix in Processor Service (`backend/services/cmd/processor-service/main.go`)**:
  - Fixed Go compiler error `undefined: attemptsMu` by correctly qualifying receiver fields `a.attempts` and `&a.attemptsMu` in the `subscribeControlResponders` call.
- **Batch Publish Job Interface Compliance (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Fixed TypeScript compiler error where `status` and `max_retries` were incorrectly included on the `Job` object literal in `handleBatchPublish`.
  - Aligned batch publish payload with the canonical `Job` interface and single publish flow (`source`, `content_type`, `headers`, `payload`).

### Removed
- **Grafana Header Link Removal (`frontend/src/components/Header.tsx`)**:
  - Removed the `[Grafana (:3000)->]` external dashboard link from the top navigation bar header.
- **Worker Load Distribution UI & API Removal (`frontend/`, `backend/services/cmd/processor-service/`)**:
  - Removed the `Active Load Distribution (Competing Pull)` section and per-worker badges from Stage 3 UI (`CoreFlowProcessor.tsx`).
  - Removed `worker_distribution` from `ProcessorDirectStatus` in `demoApi.ts`.
  - Removed `worker_distribution` from `GET /processor/status` and `PUT /processor/workers` JSON responses in `http_server.go`.
  - Removed `consumerDistribution` tracking and logging from `worker.go`, `main.go`, and `control.go`.
- **Saga Orchestration Removal from Studio View (`frontend/src/`)**:
  - Removed "Saga Orchestration" tab from `CapabilityStudio.tsx` (`StudioTab` type, nav bar button, panel render).
  - Deleted unused standalone `frontend/src/components/SagaPanel.tsx`.
  - Removed obsolete Saga API functions (`startSaga`, `advanceSagaStep`, `getSagaStatus`, `injectSagaFailure`, `cancelSaga`, `listSagas`) and models from `frontend/src/api/demoApi.ts`.
  - Removed `saga-orchestration` knowledge card from `frontend/src/content/natsInfo.ts`.
- **Saga Worker Responders from Processor Service (`backend/services/cmd/processor-service/main.go`)**:
  - Removed auxiliary `sagaWorkers *saga.WorkerResponders` and import from `processor-service`.
- **Complete Deletion of Saga Package & Job Service Endpoints (`backend/services/`)**:
  - Deleted obsolete `backend/services/internal/saga/` package (`model.go`, `orchestrator.go`, `worker.go`).
  - Deleted `backend/services/api/http/saga_handler.go`.
  - Removed Saga Orchestrator initialization, routes, and `WithSagaOrchestrator` fallback from `job-service` (`cmd/job-service/main.go`, `api/http/job_handler.go`, `api/http/routes.go`).

### Changed
- **Processor Service Code Cleanup & Stale Code Removal (`backend/services/cmd/processor-service/`)**:
  - Removed redundant parameter threading of struct fields `attempts` and `attemptsMu` across `buildCoreJobHandler`, `subscribeControlResponders`, `startWorkers`, `jsPullLoop`, and `handleJetStreamMsg`, using direct receiver field access.
  - Eliminated dead fallback code in `handleJetStreamMsg` where `attemptCount <= 0` was impossible under JetStream message metadata.
  - Consolidated duplicate failure lifecycle events in `buildCoreJobHandler` to publish solely to canonical `jobs.failed`.
  - Cleaned up stale hardcoded line numbers in the simulated panic trace log.
- **Incremental Worker Pool Scaling (`backend/services/cmd/processor-service/worker.go`)**:
  - Replaced the stop-and-recreate worker pool strategy with true incremental scaling in `ScaleWorkers`.
  - On scale UP (e.g. 2 -> 3): Keeps existing workers running uninterrupted without context cancellation, spawning only the delta (`processor-3`).
  - On scale DOWN (e.g. 4 -> 1): Cancels only excess worker goroutines from the tail (stops `processor-4`, `processor-3`, `processor-2`), keeping `processor-1` running uninterrupted.

### Added (Multiple Processors & Competing Consumer Scaling)
- **Dynamic Worker Pool Scaling & Live Logs (`backend/services/cmd/processor-service/`)**:
  - Implemented dynamic worker pool resizing via `PUT /processor/workers` without restarting the daemon or modifying the NATS durable consumer.
  - Enhanced `startWorkers` and added `ScaleWorkers` with explicit, prominent ASCII logs detailing pool transitions: `[WORKERS] Scaling worker pool request: N -> M worker(s)`, stopping existing goroutines, and initializing new worker goroutines.
  - Added worker distribution tracking to `GET /processor/status` (`worker_distribution: {"processor-1": X, "processor-2": Y}`).
- **Interactive Worker Scaling & Batch Publishing UI (`frontend/src/`)**:
  - In Stage 1 (`CoreFlowPublisher.tsx`), added a `Batch (6 Jobs)` action to instantly dispatch 6 concurrent jobs to the `JOBS` stream.
  - In Stage 3 (`CoreFlowProcessor.tsx`), replaced static worker count with a dynamic stepper `[ 1W ] [ 2W ] [ 3W ] [ 5W ]` in Card 1.
  - Added live Load Distribution badges in Card 1 displaying real-time job allocation per worker (`processor-1: 2`, `processor-2: 2`, `processor-3: 2`) along with throughput speedup metrics (~3x speedup with 3 workers).
  - Added `updateProcessorWorkers` in `demoApi.ts`.
- **Multi Worker Testing Reference (`docs/consumer_testing.md`)**:
  - Populated Section 2 (Multi Worker Testing) with single-table test matrix covering Competing Consumers Load Balancing, Throughput Scaling (Linear Speedup), and Dynamic Worker Pool Resizing (Zero Downtime).
  - Updated `docs/DEVELOPER_GUIDE.md` with Competing Consumers & Horizontal Scaling architecture.

## 2026-09-07

### Added (Durable Consumer Failure Scenarios & Rich Goroutine Diagnostics)
- **Consumer Failure Testing Reference (`docs/consumer_testing.md`)**:
  - Created standardized testing guide featuring a single table format detailing `Failure`, `Description`, `How NATS works in this`, and `How to do demo`.
  - Added Single Worker Testing covering Scenario 1 (Worker Crash Before ACK), Scenario 2 (Processing Exceeds AckWait / Late ACK), Scenario 3 (Worker Unavailable / Backlog Accumulation), Scenario 4 (Worker NAKs Message), Scenario 5 (Worker Terminates Message), and Scenario 6 (Consumer State Retention Across Restart).
  - Added Section 2 for Multi Worker Testing as an extensible template for upcoming test cases.
- **Collapsible Vertical Failure Lab (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`, `frontend/src/api/demoApi.ts`)**:
  - Converted the Failure Lab in Stage 3 to a collapsible container with interactive header toggle and ASCII indicator (`[-]` / `[+]`).
  - Arranged failure scenarios into a vertical stack of dedicated cards:
    - 1. Worker Crash Before ACK (`crash_before_ack`)
    - 2. Exceed AckWait Threshold (`exceed_ack_wait`)
    - 4. Worker NAKs Message (`nak_message` with explicit `msg.Nak()` and immediate redelivery)
    - 5. Worker Terminates Message (`term_message` with `msg.Term()`, advancing ACK floor without redelivery)
    - 6. Durable Consumer State Retention Across Restart (interactive restart demonstration)
  - Extended `FailureScenario` TypeScript type and API client.
  - Retained clean reset button and copyable `nats consumer info` CLI helper.
- **Additional Failure Scenarios Backend Engine (`backend/services/cmd/processor-service/worker.go`, `http_server.go`)**:
  - Implemented Scenario 4 (Explicit NAK): Sends `msg.Nak()` on attempt #1, emits `[NAK SENT]` log and event; JetStream immediately queues for attempt #2 where it completes with `msg.Ack()`.
  - Implemented Scenario 5 (Terminal ACK): Sends `msg.Term()` for poison messages, emits `[TERM SENT]` log; broker marks permanently terminated and advances ACK floor without redelivery.
  - Implemented Scenario 6 (Durable State Inspection): Startup logs existing consumer state (`[DURABLE STATE] Ack Floor: Stream Seq N | Outstanding ACKs: N | Stream Backlog: N`).
  - Updated `PUT /processor/scenario` to accept `nak_message` and `term_message`.
- **True Goroutine Termination & Supervisor Respawn (`backend/services/cmd/processor-service/worker.go`, `main.go`, `http_server.go`)**:
  - Implemented Scenario 1 (Worker Goroutine Crash Before ACK): Authentic panic simulation with Go stack trace printed to stdout, actual goroutine exit from `jsPullLoop`, 0 active goroutines during the 5s AckWait window, and 5.5s supervisor timer launching a genuine new worker goroutine that pulls redelivery #2 and completes with `msg.Ack()`.
  - Implemented Scenario 2 (Processing Exceeds AckWait): 7s simulated execution taking longer than 5s server AckWait, verifying NATS broker redelivery while execution is ongoing and delayed `msg.Ack()` completion.
  - Implemented Scenario 3 (Worker Unavailable): Toggling worker pull loop pauses goroutines, allowing messages to accumulate in `JOBS` stream and drain immediately upon resuming.
  - Added standardized startup logs: `[CONFIG] Consumer: job-processor...`, `[WORKERS] Initializing...`, and `[processor-1] Worker goroutine READY...`.
  - Added HTTP endpoints `PUT /processor/scenario` and `POST /processor/restart` on port :8082 for direct worker control.
- **Stage 3 Failure Lab & Broker Inspection UI (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`, `frontend/src/api/demoApi.ts`)**:
  - Added Card 3 (Failure Lab) with 1-click arming for "1. Crash Before ACK", "2. Exceed AckWait (7s)", and "Reset", along with copyable NATS CLI check command (`nats consumer info JOBS job-processor`).
  - Added Goroutine State row in Card 1 with manual "Revive Goroutine Now" action.
  - Added server-side Ack Policy (`Explicit`) and AckWait (`5s`) configuration badges to Card 2.
  - Decluttered Stage 3 by removing the redundant activity log panel to focus presenter attention on live terminal output.

### Changed (NATS Demo View 3-Stage Lifecycle Panel Cleanups)
- **Stage 1 Wire Envelope Cleanup (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Removed the redundant `OUTGOING NATS WIRE ENVELOPE` box from Stage 1 to declutter the message publishing interface.
- **Stage 2 CLI Snippet Simplification (`frontend/src/components/CoreFlow/CoreFlowNatsCli.tsx`)**:
  - Removed verbose descriptions from under each CLI command, keeping clean, readable Heading + Monospace Command + Copy Button blocks.
  - Removed the bottom `Tip: Exporting NATS_CONTEXT=local-app...` footer div.
- **Stage 3 Architectural Plane Separation & Tactile Toggle (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Decoupled the Application Microservice Plane (`processor-service` on port `:8082`) from the NATS Broker Consumer Plane (`job-processor` server-side durable pull consumer) into two distinct, dedicated cards.
  - Added explicit binding relationships between the two cards (`Consuming From: job-processor (Pull)`, `Bound Worker: processor-service (Pull Loop)`, and `CreateOrUpdateConsumer("job-processor")` connector).
  - Clarified the local HTTP diagnostic server on port `:8082` (`HTTP Control Port: :8082 (Direct)`).
  - Upgraded the worker pull loop toggle from a coarse button/banner into a modern, tactile segmented switch control (`[ ON | OFF ]`) with live status indicators and subtle pulsing glow.

### Changed (Backend Modularization into 2 Go Modules & CURRENT DEMO SETUP Removal)
- **Frontend Current Demo Setup Removal (`frontend/src/App.tsx`, `index.css`, `frontend/src/components/DemoSetup/`)**:
  - Completely removed the legacy `CURRENT DEMO SETUP` panel, `DemoTopology.tsx`, and `DemoSummary.tsx` from the frontend, decluttering the `CapabilityStudio` view.
  - Extracted `InfoPopover.tsx` to `frontend/src/components/InfoPopover.tsx` to maintain full interactive contextual information across remaining cards.
  - Cleaned up obsolete `.demo-setup-*` CSS rules from `index.css`.
- **Backend Architecture Separation (`backend/services/`, `backend/control/`)**:
  - Split backend workspace into two completely decoupled, self-contained Go modules without requiring Go workspaces:
    1. **Module `nats-demo/services` (`backend/services/`)**: Consolidates `job-service` (port `:8081`) and `processor-service` (port `:8082`), acting purely as business logic holders and direct NATS Go client communicators. Retains Stage 3 NATS Demo inspector endpoints (`/processor/events`, `/processor/status`, `/processor/state`).
    2. **Module `nats-demo/control` (`backend/control/`)**: Houses `demo-control-service` (port `:8080`), cleanly isolating Capability Studio test triggers, DLQ management, activity ring-buffers, and $SYS advisory monitoring with its own self-contained domain models and client wrappers.
  - Removed legacy monolithic `backend/src` directory.
- **Documentation & Run Guides (`README.md`, `docs/DEPLOYMENT_GUIDE.md`, `docs/DEVELOPER_GUIDE.md`, `docs/FUNCTIONAL_TESTING_GUIDE.md`)**:
  - Updated startup commands to reflect new module paths (`cd backend/control`, `cd backend/services`).
  - Formatted NATS CLI context setup commands in `README.md` as single-line copyable commands.
  - Added `processor-service` (port `:8082`) to the Endpoints & Ports Reference table in `README.md`.
  - Updated Capability Studio tab listings in `DEVELOPER_GUIDE.md` and testing verification steps in `FUNCTIONAL_TESTING_GUIDE.md` to remove legacy `CURRENT DEMO SETUP` panel references.
  - Standardized backend startup commands across all services to use package directory syntax (`go run ./cmd/<service>`) instead of single-file arguments (`go run cmd/<service>/main.go`), ensuring multi-file packages like `processor-service` compile all sibling package files.
  - Verified 100% synchronization across backend Go modules (`nats-demo/services`, `nats-demo/control`), frontend React API clients (`demoApi.ts`), and technical documentation.

### Fixed (Processor Worker Undefined nakReason Fix)
- **Undefined `nakReason` Variable in JetStream Processor Worker (`backend/src/cmd/processor-service/worker.go`)**:
  - Declared and formatted `nakReason` before passing it to `recordWorkerEvent` during `NakWithDelay` handling, resolving the Go compilation error `undefined: nakReason`.

### Added (Job Service & Publisher Wire Reception Logging)
- **Message Ingestion & NATS Wire Header Logging (`backend/src/api/http/job_handler.go`, `backend/src/internal/messaging/publisher.go`)**:
  - Added explicit ASCII log lines on `POST /jobs` reception in `job_handler.go` displaying received `job_id`, `msg_id`, `subject`, `delivery_mode`, and `source`.
  - Added explicit ASCII log lines in `publisher.go` displaying the exact subject, `Nats-Msg-Id`, and transport mode being published to NATS.

### Changed (Pure JetStream Stage 1 & Direct Processor HTTP API)
- **Pure JetStream in Stage 1 (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Removed Core NATS transient delivery mode switcher from Stage 1 in the NATS Demo scope.
  - Form is now exclusively dedicated to NATS JetStream (`JOBS` stream) with `delivery_mode: "JETSTREAM"`.
  - Removed Core NATS queue group presets (`jobs.queue`) to keep attention focused on stream ingestion.
- **Direct Processor HTTP API & Zero-NATS Telemetry Decoupling (`backend/src/cmd/processor-service/`, `frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Added dedicated lightweight HTTP API server to `processor-service` listening on port `:8082` (`http_server.go`).
  - Added endpoints `GET /processor/events`, `DELETE /processor/events`, `GET /processor/status`, and `PUT /processor/state`.
  - Replaced NATS publishing of internal lifecycle events (`jobs.delivered`, `jobs.completed`, `jobs.acked`) with in-memory thread-safe event tracking, keeping the NATS broker 100% pure and free of fake telemetry messages.
  - Connected Stage 3 Processor View directly to `processor-service` (`:8082`), bypassing `demo-control-service` for the primary NATS demo flow.

### Changed (Stage 1 Generic Message Publisher & Extensible NATS Headers)
- **Generic NATS Message Publisher (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`, `CoreFlowView.tsx`)**:
  - Transformed Stage 1 from a job-specific form ("Submit Job") into an extensible, generic "Publish Message" screen.
  - Added user-configurable controls for Target Subject (with presets `jobs.submitted`, `jobs.queue`), Delivery Mode (`JETSTREAM` vs `CORE`), Message Identity (`Nats-Msg-Id`, `X-Source`, `Content-Type`), and payload JSON editor.
  - Added an extensible Custom Headers builder allowing users to add and manage arbitrary key-value pairs on the NATS message envelope.
  - Added live Outgoing NATS Wire Envelope inspector reflecting all message parameters before transmission.
- **Backend Model & Dynamic Publisher Support (`backend/src/internal/jobs/model.go`, `service.go`, `backend/src/internal/messaging/publisher.go`)**:
  - Extended `Job` struct and `demoApi.ts` with optional envelope fields: `Subject`, `MsgID`, `Source`, `ContentType`, and `Headers` (`map[string]string`).
  - Updated `PublishJobSubmitted` to dynamically route messages to the specified subject, apply customized headers, set `Nats-Msg-Id`, and forward custom key-value headers to the NATS broker.
  - Maintained 100% backward compatibility for existing job processing pipelines and durable consumers.

### Changed (Clean Demo Startup Isolation, On-Demand Capability Prerequisites, & Backend Modularization)
- **Primary Demo Broker Isolation (`backend/src/internal/natsclient/client.go`)**:
  - Confined initial application bootstrap to ONLY create the primary demo objects: `JOBS` Stream and `job-processor` durable consumer.
  - Eliminated automatic startup creation of auxiliary streams (`JOBS_DLQ`) and secondary consumers (`processor-durable`), ensuring CLI commands (`nats stream ls`, `nats consumer ls JOBS`) remain completely clean and uncluttered during the primary NATS Demo presentation.
- **Capability Studio Prerequisites & Cleanup (`frontend/src/components/DLQPanel.tsx`, `frontend/src/api/demoApi.ts`, `backend/src/api/http/control_handler.go`, `backend/src/api/http/routes.go`)**:
  - Introduced explicit **`[Setup Prerequisites]`** and **`[Cleanup]`** action buttons inside the DLQ Capability Studio panel.
  - Added REST endpoints `POST /dlq/setup` (calls `EnsureDLQStream()`) and `POST /dlq/cleanup` (calls `DeleteDLQStream()`).
  - Added visual readiness badge (`READY (JOBS_DLQ ACTIVE)` vs `NOT PROVISIONED`) and graceful fallback when secondary streams are not yet provisioned.
- **Processor-Service Backend Modularization (`backend/src/cmd/processor-service/`)**:
  - Deconstructed monolithic 1543-line `main.go` into clean, decoupled, single-responsibility files (all under `package main`):
    1. `main.go`: Application lifecycle, configuration loading, component instantiation, and graceful shutdown signal management (~215 lines).
    2. `worker.go`: Core message processing logic, JetStream pull consumer loops (`Fetch(1)`), Core NATS message handling, telemetry tracing, failure simulation, and ACK/NAK/TERM/DLQ routing (~440 lines).
    3. `validation.go`: NATS Request/Reply pattern handler on `jobs.validate` (~130 lines).
    4. `queue_group.go`: Core NATS Queue Group competing consumer implementation on `jobs.queue` / `job-workers` (~100 lines).
    5. `control.go`: Runtime demo instrumentation responders (`status.processor`, `consumer.config.set`, `consumer.reset`, `processor.state.set`, `queue.config.set`, `queue.status`, `queue.reset`) (~240 lines).
  - Preserved 100% backward compatibility, symbol availability, and runtime API contracts while making backend code immediately readable and presentable.
- **Documentation & Run Command Updates (`README.md`, `docs/DEPLOYMENT_GUIDE.md`, `docs/FUNCTIONAL_TESTING_GUIDE.md`, `docs/DEVELOPER_GUIDE.md`)**:
  - Updated service execution commands to package mode (`go run ./cmd/processor-service`) so that all modular files (`main.go`, `worker.go`, `validation.go`, `queue_group.go`, `control.go`) compile together seamlessly.

### Changed (NATS View & CLI: Setup Phase Context & Context-Free Commands)
- **Terminal Setup Phase Snippets (`frontend/src/components/CoreFlow/CoreFlowNatsCli.tsx`)**:
  - Added dedicated setup phase commands to configure terminal context: `export NATS_CONTEXT=local-app` (app developer scope) and `export NATS_CONTEXT=sys-admin` (sys admin scope).
- **Stream & Consumer Operational Commands (`frontend/src/components/CoreFlow/CoreFlowNatsCli.tsx`)**:
  - Replaced legacy commands having inline `--context` flags with clean, handy operational commands:
    1. **View Streams**: `nats stream ls` and `nats stream info JOBS`.
    2. **View Messages in Streams - Last 5**: `nats stream view JOBS 5`.
    3. **View Consumers**: `nats consumer ls JOBS`.
    4. **View Consumer Message Processing Data**: `nats consumer info JOBS job-processor`.
    5. **View Consumer Processing Report**: `nats consumer report JOBS`.
- **UI Simplification (`frontend/src/components/CoreFlow/CoreFlowNatsCli.tsx`)**:
  - Removed the in-card live telemetry banner (Stream msgs, Consumer status, In-Flight pending) and duplicate instruction sentence to eliminate visual confusion and keep the panel focused exclusively on the CLI commands.

### Changed (Centered NATS Demo View on Submit Job & Processing)
- **Submit Job Focus (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Centered Stage 1 on **Submit Job** as the single domain use case, removing domain presets (`invoice-processing`, `order-settlement`, `customer-onboarding`).
  - Set default job payload to standard image-processing workload (`{"file": "image-101.jpg"}`) with standard type `image-processing`.
  - Updated panel title to `Submit Job`, publish button to `Submit Job to NATS JetStream ->`, and activity log to `Job Submission Log:`.
- **Pipeline Tracker Alignment (`frontend/src/components/CoreFlow/CoreFlowView.tsx`)**:
  - Aligned the 3-step pipeline banner and subtitle to: `1. Submit Job -> 2. NATS Broker (Stream/Subject) -> 3. Process Job (Worker)`.

### Added (NATS Terminology & Lifecycle Reference: docs/NATS.md)
- **NATS Reference Guide (`docs/NATS.md`)**:
  - Created a single reference table covering NATS primitives across: `Word / Term`, `Description`, `Type & Location (Where It Lives: App side / JetStream / Core NATS)`, `Component (Core NATS / JetStream)`, and `Lifecycle (Create / Change Config / Delete)`.
  - Clarified physical boundaries distinguishing what lives on the application side (Worker processes, client subscriptions), what lives on the broker in Core NATS (subjects, dynamic queue groups), and what lives in JetStream storage (streams, durable/ephemeral consumer cursors, KV, object stores).

### Changed (4-Phase Processor Activity Log & ACK Detection Fix)
- **Processor Activity Log Formatting (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Implemented the 4 distinct processor lifecycle events in the Processor Activity Log:
    1. `[PULLED]`: Message fetched from NATS JetStream stream.
    2. `[PROCESSING]`: Business logic execution started.
    3. `[COMPLETED]`: Domain workload completed successfully.
    4. `[ACK SENT]`: Explicit acknowledgment (`msg.Ack()`) dispatched back to broker.
  - Formatted each event as a clean, single-line entry with fixed-width tags matching the Publisher Activity Log layout.
  - Removed the redundant metrics summary strip (`Total Processed`, `ACK`, `NAK`) to keep the panel uncluttered.

### Changed (NATS Demo View Simplification, JetStream-Only Publisher, and Processor Service Inspection)
- **Top-Level Navigation &amp; Default View (`frontend/src/components/Header.tsx`, `frontend/src/App.tsx`)**:
  - Reordered view switcher: `NATS Demo` is now positioned on the left and set as the default view; `Capability Studio` is on the right.
- **Uncluttered NATS Demo Presentation (`frontend/src/App.tsx`)**:
  - Confined `Current Demo Setup` and `Observability Setup (LGTM Architecture)` panels to only render within the `Capability Studio` view.
- **JetStream-Only Publisher Simplification (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Removed Core NATS and Failure Simulation options from the Publisher view, locking the transport to NATS JetStream (`JOBS` stream).
  - Added a compact `Publisher Activity Log` at the bottom of the panel recording recent outgoing published messages.
- **Processor Service Microservice &amp; Consumer Details (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`, `frontend/src/components/CoreFlow/CoreFlowView.tsx`)**:
  - Displayed microservice identity (`processor-service`), NATS consumer type (`DURABLE`), consumer name (`job-processor`), and attached stream (`JOBS`).
  - Added an animated processing status sign (`PROCESSING ENGINE: LISTENING` vs `PAUSED`) and active status toggle.
  - Displayed processed message logs at the bottom with quick inspection links.
- **Immediate Worker Pause Enforcement (`backend/src/cmd/processor-service/main.go`)**:
  - Explicitly cancelled worker loop contexts and called `unsubscribeJetStream()` on pause to prevent any lingering message processing while paused.

### Changed (Core Flow: Publisher UI Design Upgrade & Processor Business View)
- **Publisher -&gt; Processor Terminology &amp; Flow (`frontend/src/components/CoreFlow/CoreFlowView.tsx`)**:
  - Renamed Stage 3 from "Consumer View" to "Processor View" to eliminate terminology collision with NATS server-side JetStream consumers (which are inspected in Stage 2).
  - Updated lifecycle pipeline header to: `1. Publisher -&gt; 2. NATS Broker (Stream/Subject) -&gt; 3. Processor (Worker Execution)`.
- **Processor View Business Logic POV (`frontend/src/components/CoreFlow/CoreFlowProcessor.tsx`)**:
  - Replaced the generic activity event log with a business execution feed showing processed message records, domain actions performed, emitted completion events (`jobs.completed`), worker identity, and delivery counts.
  - Added worker status metrics strip tracking processed jobs, completed (ACK), and retried (NAK).
  - Maintained `CoreFlowConsumer.tsx` as a backward-compatible wrapper component.
- **Publisher View Design Overhaul (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Replaced plain button presets with interactive preset cards with status indicators, badges, and intent descriptions.
  - Replaced browser radio buttons with a sleek segmented transport switcher (`JetStream (Durable)` vs `Core NATS (Transient)`).
  - Upgraded envelope inspector to a dark terminal card format and added JSON formatting/reset controls to the payload editor.
  - Enhanced publish action button with glow effects and active sending feedback.

### Fixed (Frontend Activity Type Mismatch)
- **Activity Property Access in Job Selection (`frontend/src/App.tsx`)**:
  - Replaced `latest.type` with `latest.job_type` in `handleSelectJob` fallback logic to match the `Activity` interface definition and backend schema.

### Added (Platform Core Flow & Platform Internal Inspection View)
- **Top-Level View Switcher (`frontend/src/components/Header.tsx`, `frontend/src/App.tsx`)**:
  - Added navigation pills in the main application header allowing presenters to toggle between the comprehensive **Capability Studio** and the focused **Platform Core Flow (Stage 1-2-3)**.
- **Core Flow Container (`frontend/src/components/CoreFlow/CoreFlowView.tsx`)**:
  - Created a 3-column architecture tracing end-to-end message lifecycle: Publisher View -> NATS Broker/CLI View -> Consumer View.
- **Publisher View (`frontend/src/components/CoreFlow/CoreFlowPublisher.tsx`)**:
  - Outgoing message envelope inspector displaying target subject (`jobs.submitted`), delivery mode (`JETSTREAM` vs `CORE`), deduplication header (`Nats-Msg-Id`), and JSON payload.
  - One-click presets for JetStream Persisted, Core NATS Transient, and Simulated Worker Failure (NAK).
- **NATS Broker & CLI View (`frontend/src/components/CoreFlow/CoreFlowNatsCli.tsx`)**:
  - Live JetStream stream (`JOBS`) and consumer metrics (message count, stream sequence cursor, pending count, ack wait).
  - Curated, copyable `nats` CLI terminal commands pre-configured with `--context local-app` and `--context sys-admin` (`stream info`, `consumer info`, `stream view`, `sub "jobs.>"`, `server report`).
- **Consumer View (`frontend/src/components/CoreFlow/CoreFlowConsumer.tsx`)**:
  - Live worker processing timeline displaying worker identity, delivery attempts, and explicit acknowledgment status (`ACK`, `NAK_WITH_DELAY`, `TIMEOUT`).
  - Worker pause/resume toggle to demonstrate messages buffering in the NATS stream before worker consumption.
- **Documentation (`docs/DEVELOPER_GUIDE.md`, `docs/CHANGELOG.md`)**:
  - Documented the Core Flow architecture and view switching.

### Added (Top-to-Down Collapsible Dashboard Panels)
- **Collapsible Panel System (`frontend/src/index.css`)**:
  - Added `.panel-collapsible`, `.panel-header-interactive`, `.collapse-toggle-btn`, `.collapse-chevron`, and `.panel-collapsible-body` CSS classes.
  - Implemented top-to-down transitions with max-height and opacity easing.
- **Current Demo Setup Panel (`frontend/src/components/DemoSetup/DemoSetupPanel.tsx`)**:
  - Made the panel collapsible top-to-down with default `collapsed` state (`isExpanded: false`).
  - Added interactive header with `Expand / Collapse` button and rotating chevron indicator.
- **Unified Activity Log & Subject Addressing Collapsible Container (`frontend/src/components/ObservabilityPanelContainer.tsx`, `frontend/src/components/ActivityPanel.tsx`)**:
  - Implemented a unified top-to-down collapsible container on `ObservabilityPanelContainer` with default `collapsed` state (`isExpanded: false`).
  - Added a single `Expand / Collapse` button in the segmented switcher bar controlling both **Live Activity Log** and **Subject Addressing & Wildcards** views.
  - Preserved live counts for both tabs (`{activities.length}` events and `{subscriptions.length}` subs) in the switcher bar when collapsed.
  - Automatically expands the selected view if a user clicks either tab while collapsed.
- **Observability Setup Panel (`frontend/src/components/ObservabilityPanel.tsx`)**:
  - Made the LGTM architecture panel collapsible top-to-down with default `collapsed` state (`isExpanded: false`).
  - Added `Expand / Collapse` toggle button in the header action bar alongside telemetry links.
- **Documentation (`docs/DEVELOPER_GUIDE.md`, `docs/CHANGELOG.md`)**:
  - Updated developer guide with details on collapsible panel states and behavior.

### Fixed (NATS Server Startup Config Fix)
- **NATS Configuration (`deploy/nats/nats.conf`)**:
  - Removed unsupported `log_format: "json"` directive which caused NATS server startup failure (`unknown field "log_format"`).

### Added (NATS Multi-Account Authentication & Environment Variable Configuration)

- **Backend Configuration & Client (`internal/config/config.go`, `internal/natsclient/client.go`)**:
  - Added `NATS_USER`, `NATS_PASSWORD`, `NATS_SYS_USER`, and `NATS_SYS_PASSWORD` to runtime configuration with sensible defaults matching `deploy/nats/nats.conf`.
  - Added `ConnectWithAuth` helper in `natsclient` to inject `nats.UserInfo(user, password)` into client connection options.
- **Backend Microservices (`cmd/job-service/main.go`, `cmd/processor-service/main.go`, `cmd/demo-control-service/main.go`)**:
  - Updated `job-service` and `processor-service` to connect using `app_user` credentials for the `APP` tenant account.
  - Updated `demo-control-service` to connect its business client with `app_user` and its operational `AdvisoryListener` with `sys_admin` credentials to monitor `$SYS.ACCOUNT.*` events in the system account.
- **Documentation & CLI (`README.md`, `docs/DEPLOYMENT_GUIDE.md`)**:
  - Documented environment variables in `.env.example` and deployment guides.
  - Added NATS CLI context configuration examples for `sys-admin` and `local-app` contexts.

### Removed (NATS UI Web Management Console Removal)

- **Docker Compose Orchestration (`deploy/docker-compose.yaml`, `deploy/nats/nats.conf`)**:
  - Removed `nats-ui` service container (`ghcr.io/gastbob40/nats-ui:latest`) and host port mapping `3001:3000`.
  - Removed WebSocket port `9222:9222` mapping from `nats` service and removed `websocket` block from `nats.conf`.
- **Frontend Components (`frontend/src/components/Header.tsx`, `frontend/src/components/ObservabilityPanel.tsx`, `frontend/src/components/DemoSetup/DemoTopology.tsx`)**:
  - Removed `NATS UI (:3001)` quick-launch button from top application Header.
  - Removed `NATS UI (:3001)` quick-launch button from Observability Panel header action bar.
  - Reverted NATS Server port tag in Demo Topology from `Port 4222 / 8222 / 9222 (WS)` to `Port 4222 / 8222`.
- **Documentation (`README.md`, `docs/DEVELOPER_GUIDE.md`, `docs/DEPLOYMENT_GUIDE.md`)**:
  - Removed references, verification steps, and capability table rows for NATS UI and port 3001.



## 2026-09-04


### Added (NATS UI Web Management Console Integration)
- **Docker Compose Orchestration (`deploy/docker-compose.yaml`)**:
  - Integrated `nats-ui` container using image `ghcr.io/gastbob40/nats-ui:latest`.
  - Configured host port mapping `3001:3000` to prevent port collisions with Grafana (`:3000`).
  - Attached to network `nats-net` with dependency on the core `nats` broker.
- **Frontend Dashboard Integration (`frontend/src/components/ObservabilityPanel.tsx`, `frontend/src/components/Header.tsx`, `frontend/src/components/DemoSetup/DemoTopology.tsx`)**:
  - Added dedicated quick-launch link button `NATS UI (:3001) ->` in the Observability Panel header action bar.
  - Added direct navigation link for `NATS UI (:3001)` in the top application header bar.
  - Updated NATS Server port tag in Demo Topology to indicate active WebSocket port `Port 4222 / 8222 / 9222 (WS)`.
- **Documentation (`docs/DEPLOYMENT_GUIDE.md`, `docs/DEVELOPER_GUIDE.md`, `README.md`)**:
  - Documented `nats-ui` service configuration, image, and port binding (`:3001`).
  - Added NATS UI availability and verification steps in the deployment and quickstart guides.

### Added (Comprehensive JetStream Deliver and Ack Policies in Consumer Lab)
- **Backend Domain and Controls (`internal/jobs/model.go`, `cmd/processor-service/main.go`, `api/http/control_handler.go`)**:
  - Extended `ConsumerConfig` and `ConsumerStatusResponse` with configurable `deliver_policy` (`all`, `new`, `last`, `last_per_subject`) and `ack_policy` (`explicit`, `none`, `all`).
  - Added dynamic mapping to `jetstream.DeliverPolicy` and `jetstream.AckPolicy` in `subscribeJetStream()`.
  - Added clean recreation handling for durable consumers when policies change, satisfying NATS JetStream immutability rules.
  - Guarded worker `msg.Ack()` and DLQ routing calls so `msg.Ack()` is bypassed when `AckPolicy == "none"`.
- **Frontend API and Consumer Lab UI (`frontend/src/api/demoApi.ts`, `frontend/src/components/ConsumerLabPanel.tsx`)**:
  - Added interactive toggle groups for all 4 Deliver Policies and all 3 Ack Policies with descriptive explanations.
  - Added live NATS Consumer status badge bar displaying active consumer name, durability type, deliver policy, and ack policy.

### Fixed (JobDetailResponse Payload Type and Model Alignment)
- **Frontend API (`frontend/src/api/demoApi.ts`)**:
  - Added optional `payload?: Record<string, any>` to `JobDetailResponse` interface, resolving the TypeScript compiler error on `payload: saga.payload` in `getJobDetail`.
- **Backend Job Model and Store (`internal/jobs/model.go`, `internal/jobs/store.go`)**:
  - Added `DeliveryMode`, `Worker`, and `Payload` fields to `JobDetailResponse` struct in `model.go`, aligning with `JobHandler` response mapping.
  - Updated in-memory `JobStore.AddJob` to preserve `DeliveryMode` and `Payload` from submitted jobs for inspection.

### Fixed (Job Details Inspector for Saga Workflow Orders)
- **Backend Job Handler Saga Resolution (`api/http/job_handler.go`, `cmd/job-service/main.go`)**:
  - Connected `sagaOrchestrator` to `JobHandler` via `.WithSagaOrchestrator(a.sagaOrchestrator)`.
  - Updated `GET /jobs/:job_id` to fall back to `sagaOrchestrator.GetSaga(jobID)` when a requested ID is a Saga order, adapting the `SagaInstance` (steps, timeline, payload) into a standard `JobDetailResponse`.
- **Frontend API & Inspector Fallback (`frontend/src/api/demoApi.ts`, `App.tsx`, `JobInspectorPanel.tsx`)**:
  - Updated `getJobDetail` to fall back to `/sagas/jobs/${jobId}`.
  - Added seamless fallback in `handleSelectJob` to synthesize timeline details from known in-memory activity events if backend is temporarily restarted.
  - Added badge mapping for Saga states (`SAGA_STARTED`, `OP1_COMPLETED`, `OP2_COMPLETED`, `COMPENSATING`, `COMPENSATED`, `SAGA_COMPLETED`, `SAGA_FAILED`).

### Fixed (Demo Setup Core NATS Queue Group Subscriber Dynamic Status)
- **Demo Topology View (`frontend/src/components/DemoSetup/DemoTopology.tsx`)**:
  - Replaced hardcoded `pill-green` on Core NATS Queue Group subscriber cards with dynamic state tracking (`pill-green` when active, `pill-amber` when paused/idle, `pill-red` when offline).
  - Updated card container class to apply `worker-paused` when processing is paused, matching JetStream pull workers.
  - Made the compartment header badge dynamically indicate `OFFLINE` when the Processor Service is disconnected.

### Fixed (Dead Letter Queue Double-Push and Double-Reprocessing)
- **JOBS_DLQ Stream Subject Isolation (`internal/natsclient/client.go`)**:
  - Restricted `JOBS_DLQ` stream subjects from `[]string{"jobs.dlq", "jobs.dlq.>"}` to `[]string{"jobs.dlq"}`.
  - Eliminated unintentional capture of `jobs.dlq.published` lifecycle telemetry events into the DLQ stream, ensuring exactly 1 message is stored per dead-lettered job.
- **Activity Tracker Event Deduplication (`internal/activity/tracker.go`)**:
  - Explicitly ignored raw `jobs.dlq` stream messages in `ProcessLifecycleEvent` (matching `jobs.validate` behavior) while preserving `jobs.dlq.published` as the single `DLQ_PUBLISHED` event.
- **DLQ Reprocessing & Inspection Hardening (`api/http/control_handler.go`)**:
  - Filtered telemetry and non-payload entries from `GetDLQMessages`.
  - Added in-memory deduplication (`seenJobs`) to `ReprocessDLQ` to ensure each distinct job is re-injected into `jobs.submitted` exactly once even if duplicate or corrupt records pre-existed in the DLQ store.
  - Removed duplicate direct call to `activityTracker.AddEvent` during reprocessing, relying on NATS publication to `SubjectJobReprocessed` (`jobs.reprocessed`) to record the event once via wildcard listener.

### Changed (Activity Log UX: NATS Business Messages vs. Platform Telemetry Separation)
- **Backend Activity Tracker (`internal/activity/tracker.go`)**:
  - Enhanced `Activity` model with `Category` (`"BUSINESS"` vs `"LIFECYCLE"`) and human-readable `Action`.
  - Added `DetermineCategory` and `DetermineAction` helper functions to classify events automatically upon ingestion.
  - Provided descriptive operational action labels (`"JetStream Stream Ingestion"`, `"Worker Pull / Delivery"`, `"Message Acknowledged (msg.Ack())"`, `"Retry Delay Requested (msg.NakWithDelay())"`, `"AckWait Missing ACK Timeout"`, `"Poison Pill Routed to DLQ"`).
- **Frontend Activity Log (`ActivityPanel.tsx`, `demoApi.ts`, `index.css`)**:
  - Added top explanatory **Legend Banner** clarifying that `NATS Message` represents domain payloads on topics, while `Platform Telemetry` represents internal worker and broker execution signals.
  - Enhanced top-bar switcher with descriptive subtitles (`All Stream`, `NATS Business Messages (A)`, `Platform & Worker Telemetry (B)`).
  - **Section A (NATS Business Messages)**:
    - Displays exactly one row per published payload on business topics (`jobs.submitted`, `jobs.queue`, `saga.*`).
    - Added interactive **Inline Message Journey**: clicking `[+] Journey` expands a vertical timeline directly underneath the message showing each execution milestone from stream ingestion to worker ACK.
  - **Section B (Platform & Worker Telemetry)**:
    - Replaced confusing `Subject` column with primary `Platform Action`, displaying target stream/topic as secondary context to eliminate the misconception that internal signals are business channels.

### Fixed
- **JetStream Publish Activity Capture (`cmd/demo-control-service/main.go`)**:
  - Resolved issue where JetStream publishes to `jobs.submitted` (such as 1st publish and 2nd duplicate publish in Message Deduplication) were discarded by the activity log subscriber.
  - Replaced overly broad `if msg.Reply != ""` check with explicit `if msg.Subject == "jobs.validate"`, ensuring all JetStream messages containing internal `PubAck` reply inboxes are properly captured as Section A business messages.
  - Added `category: 'BUSINESS'` and `action: 'Published by Client'` to frontend optimistic activity dispatch in `App.tsx`.

### Changed (Modern NATS JetStream API Migration)
- **Centralized JetStream Client Initialization (`internal/natsclient/client.go`)**:
  - Replaced legacy `nc.JetStream()` context initialization with modern `jetstream.New(nc)`.
  - Added `JS jetstream.JetStream` directly to `Client` struct for centralized, thread-safe access.
  - Replaced manual `StreamInfo` / `AddStream` / `UpdateStream` branching with modern, idempotent `c.JS.CreateOrUpdateStream(...)` for `JOBS` and `JOBS_DLQ`.
  - Configured durable consumers (`job-processor`, `processor-durable`, `dlq-inspector`) via idempotent `stream.CreateOrUpdateConsumer(...)`.
- **JetStream Message Publishing (`internal/messaging/publisher.go`)**:
  - Migrated `PublishJobSubmitted` from `js.PublishMsg(msg)` to modern `p.client.JS.PublishMsg(pubCtx, msg)`, passing active OpenTelemetry trace context.
  - Kept sequence numbers (`ack.Sequence`) and deduplication recognition (`ack.Duplicate`) fully compatible.
- **Worker Pull Loop & Consumer Engine (`cmd/processor-service/main.go`)**:
  - Replaced legacy `*nats.Subscription` (`js.PullSubscribe`) with first-class `jetstream.Consumer` (`a.jsConsumer`).
  - Updated worker pull loop from `jsSub.Fetch(1)` to `a.jsConsumer.Fetch(1, jetstream.FetchMaxWait(500*time.Millisecond))`, reading messages via `batch.Messages()`.
  - Updated `handleJetStreamMsg` signature to accept `jetstream.Msg`, retrieving headers via `msg.Headers()`, payload via `msg.Data()`, and delivery count via `msg.Metadata()`.
  - Retained native `msg.Ack()`, `msg.Nak()`, and `msg.NakWithDelay(...)` handling.
  - Updated DLQ failure routing to publish directly via `a.natsClient.JS.PublishMsg(...)`.
  - Migrated dynamic consumer configuration and distribution reset responders to query `stream.Consumer(...)` and `cons.Info(...)`.
- **Observability Tap & UI Control Gateway (`api/http/control_handler.go`)**:
  - Replaced legacy `js.StreamInfo` and `js.ConsumerInfo` in `GetStatus` and `GetConsumerStatus` with modern `stream.Info(ctx)` and `cons.Info(ctx)`.
  - Updated `ReplayStream` to instantiate ephemeral replay consumers via `stream.CreateConsumer(ctx, consumerCfg)` and teardown cleanly via `stream.DeleteConsumer(...)`.
  - Updated `GetDLQStatus`, `GetDLQMessages`, and `ReprocessDLQ` to use `stream.GetMsg(ctx, seq)` on `JOBS_DLQ`.
  - Updated `PurgeDLQ` to invoke `stream.Purge(ctx)`.

### Documentation
- Updated `docs/DEVELOPER_GUIDE.md` under `Durable Consumer Lifecycle` to showcase `stream.CreateOrUpdateConsumer` and `consumer.Fetch(...)`.

## 2026-09-03

### Fixed
- Fixed Go compilation error in `internal/saga` (`undefined: StepAllocate`, `undefined: SagaCommandPayload`, etc.) by restoring worker command step constants (`StepAllocate`, `StepPrepare`, `StepExecute`, `StepRelease`) and structs (`SagaCommandPayload`, `SagaCommandResponse`) in `backend/src/internal/saga/model.go`.
- Fixed TypeScript compiler error `Type 'boolean | null' is not assignable to type 'boolean | undefined'` in `frontend/src/components/SagaPanel.tsx` by wrapping `isOp1Active`, `isOp2Active`, and `isTerminal` expressions with `Boolean(...)`.
- Fixed TypeScript compiler error `Cannot find namespace 'NodeJS'` in `frontend/src/components/SagaPanel.tsx` by changing `NodeJS.Timeout` to `ReturnType<typeof setInterval>`.

### Documentation
- **Improvised README.md for First-Time Evaluators**:
  - Overhauled root `README.md` with an evaluation matrix of all 12 NATS capabilities, ASCII architectural diagram, 3-minute quickstart guide, ports/endpoints reference table, interactive UI guided tour (Tours 1 through 5), and CLI testing curl commands.
- **Synchronized Developer Guide (`docs/DEVELOPER_GUIDE.md`)**:
  - Added the 7th Capability Studio tab (`Saga Orchestration`) and the Saga distributed transaction mapping row.
  - Added documentation for the Activity Log Message Classification Switcher (`Business Messages [A]` vs `Flow / Lifecycle Events [B]`).

### Added (Activity Log Message Classification Switcher)
- **Top-Bar Message Category Switcher (`ActivityPanel.tsx`, `index.css`)**:
  - Added a 3-way segmented toggle bar (`All Messages`, `Business Messages (A)`, `Flow / Lifecycle Events (B)`) to the top toolbar of the Activity Log.
  - Implemented automatic classification engine separating domain business payload messages (`jobs.submitted`, `jobs.queue`, `saga.start`, `jobs.validate`, etc.) from demo lifecycle and telemetry events (`jobs.received`, `jobs.processing`, `jobs.acked`, `jobs.completed`, `jobs.stored`, `jobs.dlq`, etc.).
  - Ensured all distributed Saga orchestration transitions (`OP1_COMPLETED`, `OP2_COMPLETED`, `SAGA_COMPLETED`, `OP1_COMPENSATED`, `SAGA_FAILED`) are accurately categorized under Business Messages (A) as core transaction domain events.
  - Added live pill counters for each category dynamically displaying count totals.
  - Added visual category badges (`[BUSINESS]` in blue, `[LIFECYCLE]` in purple) directly beneath the Event pill in each table row for clarity.

### Changed (Saga Panel UI/UX Redesign)
- **Unified Design Tokens & Aesthetic Refactoring (`SagaPanel.tsx`, `index.css`)**:
  - Redesigned the Saga Orchestration panel to match the visual language, typography, and spacing of other studio panels (`ConsumerLabPanel`, `QueueGroupPanel`, `DelayedRetryPanel`).
  - Added structured metadata summary chips (`queue-meta-row`, `queue-meta-chip`) detailing the 2-Op Saga Pattern, Forward Path (`saga.op1.reserve -> saga.op2.payment`), Rollback Compensation (`saga.op1.compensate -> release`), and Core NATS event transport.
  - Replaced unstyled radio controls and text inputs with standard `.form-label`, `.form-input`, and segmented toggle buttons (`btn-worker-toggle`).
  - Added Quick Scenario demo presets (`Normal Success Flow`, `Payment Declined (Triggers Rollback)`, `Inventory Out of Stock`) for single-click demonstrations.
  - Eliminated vertical height distortion in the visual pipeline by replacing the nested compensation branch with a balanced 3-card horizontal forward pipeline and a dedicated, full-width Compensating Rollback Track.
  - Upgraded step action buttons with clean inline SVG icons (check / cross / undo) and subtle hover effects instead of bracketed text.
  - Refactored the event transmission stream into an aligned ledger table with monospace timestamps, NATS subject badges, and latency metrics.

### Added (Interactive Event-Driven 2-Op Saga Pattern)
- **Interactive 2-Operation Workflow (`saga.start` -> `Op 1: Reserve` -> `Op 2: Payment` -> `Completed` / `Compensate: Release`)**:
  - Implemented event-driven Saga flow purely over NATS: `saga.start` -> `saga.op1.reserve` -> `saga.op1.completed` -> `saga.op2.payment` -> `saga.op2.completed` -> `saga.completed`.
  - Added rollback compensation for Op 1 on Op 2 failure: `saga.op2.failed` -> `saga.op1.compensate` -> `saga.op1.compensated` -> `saga.failed`.
  - Added `POST /sagas/jobs/:job_id/step` to allow the UI to manually advance or fail each stage interactively.
  - Rebuilt `SagaPanel.tsx` with interactive buttons directly on the Op 1 and Op 2 cards (`[Complete Op 1]`, `[Fail Op 1]`, `[Complete Op 2]`, `[Fail Op 2]`), allowing step-by-step presentation of both happy and unhappy rollback paths.
  - Added mode toggle between `Interactive Buttons` (manual presenter step-through) and `Auto Run`.

### Added (Saga Orchestration Pattern - docs/feature.md)
- **Saga Orchestrator Engine (`backend/src/internal/saga/`)**:
  - Implemented multi-step Saga Orchestrator in `internal/saga/orchestrator.go` with forward step progression (`STARTED` -> `ALLOCATING` -> `PREPARING` -> `EXECUTING` -> `COMPLETED`) and compensating rollback (`COMPENSATING` -> `FAILED` / `COMPENSATION_FAILED`).
  - Added data models in `internal/saga/model.go` for Saga instances, step logs, failure injection configurations, and event payloads.
  - Implemented worker command responders in `internal/saga/worker.go` for command subjects `saga.job.allocate`, `saga.job.prepare`, `saga.job.execute`, and `saga.job.release`.
- **Saga HTTP REST APIs (`api/http/saga_handler.go`, `routes.go`)**:
  - `POST /sagas/jobs`: Initiates new Saga workflow.
  - `GET /sagas/jobs/:job_id`: Queries active or completed Saga status and step execution timeline.
  - `POST /sagas/jobs/:job_id/fail`: Injects controlled failure at a designated step or compensation stage.
  - `POST /sagas/jobs/:job_id/cancel`: Requests cancellation of an active Saga and initiates compensation for completed steps.
  - `GET /sagas/jobs`: Lists recent Saga transactions.
- **NATS Subjects for Saga Pattern (`internal/messaging/subjects.go`)**:
  - Commands: `saga.job.allocate`, `saga.job.prepare`, `saga.job.execute`, `saga.job.release`.
  - Events: `saga.job.started`, `saga.job.step.completed`, `saga.job.failed`, `saga.job.compensation.started`, `saga.job.compensation.completed`, `saga.job.completed`.
  - Guaranteed zero collision or interference with existing `jobs.*` flows.
- **React UI Component & Studio Integration (`SagaPanel.tsx`, `CapabilityStudio.tsx`, `demoApi.ts`, `index.css`)**:
  - Added dedicated **Saga Orchestration** tab in NATS Capability Studio.
  - Built interactive test launcher for 5 demo scenarios (Normal Success, Fail at Execute, Fail at Prepare, Fail at Allocate, and Compensation Failure).
  - Built visual state machine pipeline depicting forward steps and rollback compensation nodes with live polling.
  - Added step execution timeline table and recent Saga inspect cards.
  - Added explanatory info modal card in `natsInfo.ts`.
  - Integrated `saga.job.*` events into the global `Activity Tracker` (`demo-control-service`) so that Saga transitions (`SAGA_STARTED`, `SAGA_STEP`, `SAGA_COMPENSATING`, `SAGA_COMPLETED`, `SAGA_FAILED`) also stream into the main Activity Log.

### Added (Complete NATS Observability Setup: Metrics, Logs, Events, Tracing - docs/fix.md)
- **Complete NATS Prometheus Metrics Surface (`docker-compose.yaml`)**:
  - Expanded `nats-exporter` flags to include all required NATS monitoring categories without filtering: `-varz`, `-connz`, `-connz_detailed`, `-subz`, `-routez`, `-gatewayz`, `-leafz`, `-accountz`, `-accstatz`, `-healthz`, `-jsz=all`.
- **NATS Server Logging to Loki (`nats.conf`, `fluent-bit.conf`, `docker-compose.yaml`)**:
  - Configured NATS Server to write structured logs with timestamps to `/data/nats.log` in shared `nats-data` volume.
  - Added lightweight `nats-log-collector` (Fluent Bit) container tailing `/data/nats.log` and pushing to Loki (`http://otel-lgtm:3100/loki/api/v1/push`) with labels `service="nats"`, `server="nats"`, `cluster="nats-demo"`.
  - Exposed Loki port `3100:3100` on `otel-lgtm` in `docker-compose.yaml`.
- **NATS Operational Events & JetStream Advisories Pipeline (`advisory_listener.go`, `main.go`)**:
  - Built `AdvisoryListener` background component subscribing to `$SYS.ACCOUNT.*.CONNECT`, `$SYS.ACCOUNT.*.DISCONNECT`, and `$JS.EVENT.ADVISORY.>`.
  - Normalized raw NATS advisories into structured JSON event records (timestamp, subject, event_type, stream, consumer, server, account, payload) and pushed to Loki (`service="nats-events"`).
- **Grafana LGTM Dashboard & UI Updates (`nats-demo-dashboard.json`, `ObservabilityPanel.tsx`, `index.css`)**:
  - Re-implemented the **Observability Setup Panel** into a 3-column **T-Shape Architecture Layout with Inner LGTM Boxes**:
    - **Source Columns**: Left column displays App Services (`Job Service :8081` & `Processor Service`); Right column displays NATS Infrastructure (`NATS Server :4222/:8222` & Exporter/Daemons).
    - **Coloured Conduit Bridges**: In between the sources and central stack, embedded directional colored tracks for MELT signals:
      - Left bridge: Green `M | METRICS` and Indigo `T | TRACES` with OTLP gRPC protocol badges.
      - Right bridge: Green `M | SCRAPE`, Amber `L | LOGS`, and Magenta `E | EVENTS` with capture details.
    - **Central LGTM Box with Inner Boxes**: Main `GRAFANA OTEL-LGTM` container containing the `OpenTelemetry Collector Gateway (:4317/:4318)` on top and an interactive 2x2 grid of all 4 inner engine boxes (Loki `:3100`, Grafana `:3000`, Tempo `:3200`, Prometheus `:9090`).
    - Cut down all verbose paragraph text and long descriptions while preserving high-density component cards and tags.
  - Added dedicated Loki log panels to Grafana dashboard for NATS Server Centralized Logs and NATS Operational Advisories.
  - Added `otel-collector` (`localhost:8889`) scrape target to `deploy/lgtm/prometheus.yaml` so Prometheus ingests application-level OTLP metrics (`jobs_submitted_total`, `jobs_processed_total`, etc.) emitted by Go services.
  - Added explicit Grafana datasource provisioning (`deploy/lgtm/grafana/provisioning/datasources/datasources.yaml`) for Prometheus, Tempo, and Loki with persistent UIDs (`prometheus`, `tempo`, `loki`).
  - Updated React `ObservabilityPanel` header tabs with dedicated actions: `NATS Logs`, `NATS Events`, `NATS Metrics`, `Application Traces`, and `Open Grafana`.
  - Removed the bottom URL endpoint cards layer to streamline the panel layout.
- **Tracing Scope Boundary Clarification (`DEVELOPER_GUIDE.md`, `DEPLOYMENT_GUIDE.md`, `fix.md`)**:
  - Explicitly documented that NATS Server is diagnostic-only and not a native OTLP span producer.
  - Clarified that end-to-end distributed tracing is application-level via W3C `traceparent` context injection across NATS messages.
- **Reason**:
  - Implement all backlog items in `docs/fix.md` to establish complete local LGTM observability.
- **Affected Area**:
  - Infrastructure (`deploy/docker-compose.yaml`, `deploy/nats/nats.conf`, `deploy/lgtm/fluent-bit/fluent-bit.conf`, `deploy/lgtm/grafana/dashboards/nats-demo-dashboard.json`).
  - Backend (`internal/events/advisory_listener.go`, `cmd/demo-control-service/main.go`).
  - Frontend (`ObservabilityPanel.tsx`).
  - Documentation (`docs/fix.md`, `docs/DEVELOPER_GUIDE.md`, `docs/DEPLOYMENT_GUIDE.md`, `docs/CHANGELOG.md`).

### Fixed
- **Assignment Mismatch in Scheduled Job Dispatch (`service.go`)**:
  - Fixed assignment mismatch in background goroutine of `ScheduleJob` by unpacking both return values `(*JobStatusResponse, error)` with `_, _ = s.SubmitJob(context.Background(), j)`.
  - Resolved compiler error `assignment mismatch: 1 variable but s.SubmitJob returns 2 values`.
- **Affected Area**:
  - `backend/src/internal/jobs/service.go`

### Added (Activity Log Clear Button)
- **Backend (`tracker.go`, `control_handler.go`, `routes.go`)**:
  - Added `ClearActivities()` method to `activity.Tracker` that resets the in-memory activity buffer.
  - Added `ClearActivities` handler on `ControlHandler` returning `{"status": "cleared"}`.
  - Registered `DELETE /activities` route on demo-control-service.
- **Frontend (`demoApi.ts`, `ActivityPanel.tsx`, `ObservabilityPanelContainer.tsx`, `App.tsx`)**:
  - Added `clearActivity()` API function calling `DELETE /activities`.
  - Added "Clear Log" button in the Activity Log panel header (visible when events exist).
  - Threaded `onClearActivity` callback through `ObservabilityPanelContainer` to `App.tsx`.
- **Reason**:
  - Allow presenters to reset the activity log between demo scenarios for a clean starting point.
- **Affected Area**:
  - Backend (`internal/activity/tracker.go`, `api/http/control_handler.go`, `api/http/routes.go`), Frontend (`demoApi.ts`, `ActivityPanel.tsx`, `ObservabilityPanelContainer.tsx`, `App.tsx`).

### Changed (Move Queue Group & Consumer Lab into Capability Studio)
- **Frontend Layout Refactor (`CapabilityStudio.tsx`, `DemoSetupPanel.tsx`, `App.tsx`)**:
  - Moved "Core NATS Queue Group" and "JetStream Consumer Lab" from the `DemoSetupPanel` side-by-side switcher into the NATS Capability Studio as dedicated tabs ("Queue Groups" and "Consumer Lab").
  - Simplified `DemoSetupPanel` to display only the architecture topology visualizer.
  - Threaded `onConfigChanged`, `onActivityUpdated`, and `isProcessing` props through `CapabilityStudio` to `ConsumerLabPanel`.
- **Reason**:
  - Consolidate all interactive NATS capability demos under the unified Capability Studio for a cleaner presentation workflow.
- **CSS Full-Width Topology Expansion (`index.css`)**:
  - Changed `.demo-topology-container` from `width: fit-content` to `width: 100%` so the topology fills the full panel width.
  - Removed `max-width: 210px` cap from `.deployed-card` and added `flex: 1` so service cards expand into available space.
  - Reduced `.nats-server-col-wide` `min-width` from `520px` to `420px` for better responsive behavior.
  - Cleaned up unused `.demo-consumer-lab-wrapper` and `.consumer-lab-embedded` CSS rules from the old two-column layout.
- **Affected Area**:
  - Frontend (`CapabilityStudio.tsx`, `DemoSetupPanel.tsx`, `App.tsx`, `index.css`).

### Changed (High-Visibility Inter-Tier Topology Connectors & Grid Alignment)
- **Grid Alignment & Connector Symmetry (`DemoTopology.tsx`, `index.css`)**:
  - Aligned Tier 1, Inter-tier Bridge, and Tier 2 into a clean 3-column topology grid:
    - Column 1 (width: 220px): `React UI` -> `Vertical Connector (HTTP REST Ingress :8081)` -> `Job Service`.
    - Column 2 (width: 120px): Horizontal connector `UI Gateway (:8080)` in Tier 1 -> spacer in Bridge -> horizontal connector `Publish / RPC (:4222)` in Tier 2.
    - Column 3: `Demo Control Service` (width: 240px) -> `Vertical Connector (NATS TCP Client :4222)` -> `NATS Server` (expanding via `flex: 1`).
  - Upgraded vertical inter-tier connectors (`UI -> Job Service` and `Demo Service -> NATS`) to prominent, high-visibility connector cards matching horizontal connector styling:
    - Active green state (`#10B981`) with vertical line and arrow (`v`).
    - Prominent uppercase monospace labels (`HTTP REST Ingress (:8081)` and `NATS TCP Client (:4222)`).
    - Clear endpoint listings (`POST /jobs, /schedule, /validate` and `jobs.> Tap | Replay | Control RPC`).
- **Reason**:
  - Fix misaligned and faint vertical inter-tier connections so `UI -> Job Service` and `Demo Service -> NATS` are prominently visible and structurally aligned with the rest of the topology.
- **Affected Area**:
  - Frontend (`DemoTopology.tsx`, `index.css`), Documentation (`CHANGELOG.md`).

### Added (Rich Architecture Connector Boxes for Job Service -> NATS and NATS -> Processor)
- **Job Service -> NATS Ingress Box (`DemoTopology.tsx`, `index.css`)**:
  - Upgraded horizontal connector between Job Service and NATS Server into a prominent, data-rich card box (`NATS TCP INGRESS (:4222)`).
  - Included color-coded tag badges and exact subjects: `PUB: jobs.submitted | jobs.queue`, `RPC: jobs.validate (Request)`, `HDR: W3C traceparent | Msg-Id`.
  - Added matching rich UI Gateway box in Tier 1 (`UI GATEWAY (:8080)`) with endpoints `REST: GET /activities | /status`, `CTRL: PUT /consumer | /queue-group`, `DATA: Activity Polling & DLQ Reprocess`.
- **NATS -> Processor Service Delivery Bridge Grid (`DemoTopology.tsx`, `index.css`)**:
  - Replaced generic text bridge between NATS Server and Processor Service with a dual-card rich architecture grid:
    1. **Downstream Message Delivery (NATS -> Processor)**:
       - Displays live delivery status badge (`DELIVERY ACTIVE` vs `DELIVERY PAUSED`).
       - Details `PULL`: `Stream JOBS -> Consumer 'job-processor' (Batch: 5)`.
       - Details `QUEUE`: `Core NATS 1-of-N -> Group 'job-workers' on jobs.queue`.
       - Details `RPC`: `Sync Request Dispatch -> Responder jobs.validate`.
       - Details `POLICY`: `AckWait: 5s | NakWithDelay Backoff | Ordering`.
    2. **Upstream Protocol Acks & Lifecycle Feedback (Processor -> NATS)**:
       - Displays `BIDIRECTIONAL FEEDBACK` badge in cyan.
       - Details `ACKS`: `Explicit msg.Ack() | msg.NakWithDelay(d) | msg.Term()`.
       - Details `EVENTS`: `jobs.received | jobs.processing | jobs.completed | jobs.failed`.
       - Details `POISON`: `Max Deliveries (3) Routing -> Stream JOBS_DLQ (jobs.dlq)`.
       - Details `METRICS`: `Delivery Counts | Stream Sequence | Worker Attribution`.
- **Reason**:
  - Provide comprehensive architectural and operational context directly on all connectors in the topology visualizer.
- **Affected Area**:
  - Frontend (`DemoTopology.tsx`, `index.css`), Documentation (`CHANGELOG.md`).

### Added (NATS Delayed & Retry Delivery Demo)
- **NAK with Delay (`processor-service/main.go`, `subjects.go`, `tracker.go`)**:
  - Implemented explicit negative acknowledgement with retry backoff using `msg.NakWithDelay(d)`.
  - Added subject `jobs.nak.delayed` with `NAK_WITH_DELAY` lifecycle event status.
  - Demonstrated explicit worker-driven retry delay preventing immediate worker thrashing.
- **AckWait Missing ACK Recovery (`processor-service/main.go`, `natsclient/client.go`, `subjects.go`)**:
  - Configured JetStream durable consumer `job-processor` with `AckWait: 5 * time.Second`.
  - Added `simulate_no_ack` failure payload flag and `jobs.ack.timeout` lifecycle event (`ACK_TIMEOUT_SIMULATED`).
  - Demonstrated broker-level redelivery across competing workers when a worker hangs or crashes without acknowledging.
- **Application-Level Scheduled Delivery (`job-service/main.go`, `service.go`, `routes.go`, `demoApi.ts`)**:
  - Implemented `POST /jobs/schedule` with application timer holding message before publishing to NATS.
  - Added `jobs.scheduled` lifecycle event (`SCHEDULED`) emitted immediately on schedule request.
  - Demonstrated that timestamp scheduling is handled by application schedulers rather than native JetStream timers.
- **Delayed & Retry Delivery Studio Tab (`DelayedRetryPanel.tsx`, `CapabilityStudio.tsx`, `natsInfo.ts`)**:
  - Added dedicated **"Delayed & Retry"** tab in Capability Studio featuring 3 interactive demo cards with real-time feedback and educational callouts.
  - Added `delayed-retry-delivery` info modal content with concepts, demo scenarios, and trivia.

### Added (DLQ Message Reprocessing, Purging & Activity Log Recovery)
- **DLQ Reprocessing & Purge API (`control_handler.go`, `routes.go`, `demoApi.ts`)**:
  - Implemented `POST /dlq/reprocess` supporting batch or single job reprocessing from `JOBS_DLQ` back into the active `JOBS` stream on `jobs.submitted` with failure simulation flags cleared.
  - Implemented `POST /dlq/purge` to purge all poison messages from `JOBS_DLQ` storage.
  - Added subjects `jobs.reprocessed` and `jobs.dlq.reprocessed` with `REPROCESSED` event status mapping and green badge styling.
- **DLQ Operator Action Controls (`DLQPanel.tsx`, `CapabilityStudio.tsx`, `index.css`)**:
  - Added primary **"Reprocess DLQ Messages"** and **"Purge"** action buttons to the DLQ header.
  - Added per-row **"Reprocess"** button on each message card in `JOBS_DLQ` list.
  - Connected `onActivityUpdated` lifecycle triggers to live-stream recovery events (`REPROCESSED` -> `DELIVERED` -> `COMPLETED` -> `ACKED`) into the Activity Log.

### Changed (UI Chips, 3-Per-Row Processor Grid & Consumer Policy Display)
- **Delivery Type Badges & Chips Cleanup (`QueueGroupPanel.tsx`, `ConsumerLabPanel.tsx`)**:
  - Removed `MULTI-GROUP` from Core NATS Queue Group panel and replaced with `DELIVERY TYPE: Push (Server-Dispatched)`.
  - Removed `STATUS` from JetStream Consumer Lab and replaced with `DELIVERY TYPE: Pull (Client-Fetched)`.
  - Added dedicated **Internal NATS Consumer Policy** inspector showing real-time `DeliverPolicy` (`DeliverAll` / `DeliverNew`), `AckPolicy: Explicit`, and `FilterSubject: jobs.submitted`.
- **Processor Service Responsive Grid (`DemoTopology.tsx`)**:
  - Constrained Tier 3 worker blocks to a maximum of 3 per row (`repeat(min(count, 3), 1fr)`) so 4th and 5th workers wrap to a second line without stretching the card width or encroaching on neighboring panels.
- **Backend Ephemeral Consumer Policy (`processor-service/main.go`)**:
  - Explicitly configured `DeliverPolicy: nats.DeliverNewPolicy`, `AckPolicy: nats.AckExplicitPolicy`, `FilterSubject: messaging.SubjectJobSubmitted` when creating ephemeral consumers.

### Fixed (Activity Log Live Streaming & Test Bursts)
- **Continuous Activity Polling & Fast Action Triggers (`App.tsx`, `DemoSetupPanel.tsx`, `ConsumerLabPanel.tsx`, `QueueGroupPanel.tsx`)**:
  - Integrated `refreshActivity(true)` directly into the primary 2.5s polling loop in `App.tsx` so all incoming asynchronous NATS events are automatically streamed into the Activity Log.
  - Added `onActivityUpdated` callback across `ConsumerLabPanel` and `QueueGroupPanel` to trigger immediate and staggered activity log refreshes upon dispatching test message bursts.
- **Worker Lifecycle Completion Event Publishing (`processor-service/main.go`)**:
  - Added simulated processing delay and `jobs.queue.completed` lifecycle event emission for Core NATS queue group message handling.
  - Added `jobs.completed` lifecycle event emission alongside `jobs.acked` in JetStream pull worker processing.

### Changed (Demo Setup Topology Redesign & Dual-Engine NATS Server)
- **Three-Tier Runtime Architecture Layout (`DemoTopology.tsx`, `index.css`)**:
  - Reorganized the architecture visualization into 3 clean vertical tiers:
    1. **Tier 1 (Client & Gateway Tier)**: React UI (:5173) and Demo Control Service (:8080) with explicit HTTP REST and SSE/polling metadata.
    2. **Tier 2 (Business Ingress & NATS Server Tier)**: Job Service (:8081) on the left publishing into an expanded, wide NATS Server (:4222) layout.
    3. **Tier 3 (Worker Daemon & Processing Tier)**: Processor Service positioned directly below NATS Server with bidirectional delivery (`Pull/Queue/RPC`) and lifecycle feedback (`Ack/Nak`, `jobs.received`, `jobs.completed`, `jobs.failed`, `jobs.dlq.published`) flows.
- **Widened Dual-Engine NATS Server Block**:
  - Expanded NATS Server into a square/wide card with side-by-side internal compartments:
    - **Core NATS Engine**: In-Memory Transient Pub/Sub (`jobs.*`, `jobs.>`), Queue Groups (`job-workers` on `jobs.queue` with 1-of-N distribution), and Request/Reply RPC (`jobs.validate`).
    - **JetStream Persistence Engine**: Stream `JOBS` (Persistent Log for `jobs.submitted`) -> Consumer `job-processor` (Pull Mode with durability and ordering properties), plus Stream `JOBS_DLQ` (Poison Store for `jobs.dlq`) -> Consumer `dlq-inspector` (Durable Cursor).
    - Removed transient message counter badges (`Stored Messages`, `Pending`, `Ack Pending`, `Redelivered`) from the architectural diagram to eliminate clutter, keeping them dedicated to the interactive **Consumer Lab Panel**, **Queue Group Panel**, and **DLQ Panel**.
- **Wide Processor Service Worker Grid**:
  - Displayed side-by-side compartments for JetStream Competing Pull Workers (1-5 workers binding to `job-processor`) and Core NATS Queue Group Subscribers (1-5 workers in group `job-workers`).
- **Enhanced Inter-Service Communication Details**:
  - Added explicit protocol labels, port annotations (:5173, :8080, :8081, :4222), subject patterns, W3C trace context badges, and interaction descriptions across all connectors.

### Changed (Platform Status Panel Simplification)
- **High-Level Platform Status Bar (`StatusPanel.tsx`)**:
  - Simplified the top-level Platform Status panel to display only high-level connectivity and service availability.
  - Removed granular stream and consumer badges (`Stream: JOBS`, `Pending`, `Workers`, `Consumer`) from the top-level header bar.
  - Consolidated status indicators into a clean single-row layout displaying: NATS Server, JetStream availability, Demo Control Service (:8080), Job Service (:8081), Processor Service, and the Processing ON/OFF toggle.
  - Stream backlogs and worker distributions remain available in their dedicated contextual panels (`ConsumerLabPanel`, `DemoTopology`, and `DLQPanel`).
- **Educational Metadata (`natsInfo.ts`)**:
  - Updated `platform-status` information popover to focus on overall service health and JetStream availability.

## 2026-09-02

### Fixed
- **ControlHandler Package Import**:
  - Added missing `"io"` import to `control_handler.go`, resolving compiler error `undefined: io` on `io.ReadAll(c.Request.Body)` in `PublishStreamJobs`.
- **Affected Area**:
  - `backend/src/api/http/control_handler.go`
- **JobHandler Interface Definition**:
  - Added missing `SubmitStreamJobs` method signature to `JobServiceDomain` interface in `job_handler.go`, resolving compiler error when calling `h.jobService.SubmitStreamJobs`.
- **Affected Area**:
  - `backend/src/api/http/job_handler.go`

### Added (Core NATS Queue Groups Demonstration)
- **Core NATS Queue Groups Implementation**:
  - Implemented Core NATS queue group load-balancing demonstration on subject `jobs.queue` with queue group `job-workers`.
  - Contrasted transient Core NATS in-memory work distribution (1-of-N delivery, no persistence, no JetStream consumer state) with JetStream Competing Consumers.
- **Backend Services**:
  - **`processor-service`**:
    - Added Core NATS Queue Group subscriptions via `Conn.QueueSubscribe(jobs.queue, job-workers)`.
    - Dynamic worker reconfiguration (1 to 5 workers) via NATS control subject `queuegroup.config.set` and status reporting on `queuegroup.status`.
    - Added `queuegroup.reset` responder resetting worker distribution counters to zero on demand.
    - Maintained per-worker message distribution counters (`processor-1` through `processor-5`).
    - Emitted `jobs.queue.received` lifecycle events identifying the specific receiving worker.
  - **`job-service`**:
    - Added `POST /jobs/queue` endpoint and `PublishJobQueue` publisher method for publishing single or batch test messages to `jobs.queue` with delivery mode `CORE`.
  - **`demo-control-service`**:
    - Added `GET /queue-group`, `PUT /queue-group`, `POST /queue-group/reset`, and `POST /jobs/queue` proxy endpoints.
    - Updated activity tracker `ProcessLifecycleEvent` to capture `jobs.queue` submissions and worker receipt events.
- **Frontend Dashboard**:
  - Enhanced `QueueGroupPanel.tsx`:
    - Added dynamic worker selector buttons for 1 to 5 active workers (`processor-1` to `processor-5`).
    - Added "Reset Counters" button to reset worker distribution counters back to zero.
    - Added comprehensive delivery semantics badges (Load-Balanced 1 of N, At-Most-Once Best-Effort, Stateless / No ACK-NAK, Multi-Group Fanout).
    - Added dynamic distribution progress bars for all active workers with distinct color coding.
    - Expanded educational popover (`natsInfo.ts`) thoroughly defining all terminology shown on the panel.
  - **Current Demo Setup Integration (`DemoSetupPanel.tsx` & `DemoTopology.tsx`)**:
    - Placed `Core NATS Queue Group` on the left of the lab switcher and `JetStream Consumer Lab` on the right, defaulting to Core NATS Queue Group.
    - Dynamically rendered 1 to 5 active worker subscriber cards inside Processor Service in `DemoTopology`.

### Changed (JetStream Consumer Lab Visual & Functional Parity)
- **Visual Design & Architecture Symmetry (`ConsumerLabPanel.tsx`)**:
  - Refactored `ConsumerLabPanel` to mirror the rich aesthetic and layout of `QueueGroupPanel`:
    - Meta chips displaying Stream (`JOBS`), Consumer name, Delivery Mode (`Pull`), Guarantees (`At-Least-Once`), State (`Stateful Cursors`), and Status.
    - Segmented toggle buttons for Consumer Durability (`Durable` vs `Ephemeral`), Message Ordering (`Normal` vs `Ordered`), and Active Pull Workers (`1` to `5` Competing Workers).
    - Added "Messages to Publish to Stream" selector (`5`, `10`, `20`, or custom count).
    - Added "Worker Distribution" section with real-time counters, per-worker progress bars (`processor-1` through `processor-5`), stream metrics (Pending, Ack Pending, Redelivered), and "Reset Counters" button.
    - Added "Send Test Messages to Stream" primary action button publishing batch JetStream jobs directly to the `JOBS` stream via new atomic `POST /jobs/stream` endpoint.
- **Backend Tracking & Responders**:
  - Added support for 1 to 5 competing pull consumer workers in `processor-service` and `demo-control-service`.
  - Added atomic batch stream publisher `SubmitStreamJobs` on `POST /jobs/stream` in `job-service` and proxy in `demo-control-service`.
  - Added per-worker JetStream message distribution counters (`a.consumerDistribution`) in `processor-service` initialized for `processor-1` through `processor-5`.
  - Added `SubjectConsumerReset` (`consumer.reset`) responder to reset distribution counters on demand.
  - Added `POST /consumer/reset` endpoint to `demo-control-service`.
  - Exposed `distribution` map in `GET /consumer` responses.
- **Documentation**:
  - Updated `api-spec.md` with `POST /jobs/stream`, `POST /consumer/reset`, and 1-5 worker support for `PUT /consumer`.
- **Reason**:
  - Enable scaling JetStream competing pull consumers up to 5 workers and provide high-speed batch stream publishing per user request.
- **Affected Area**:
  - Backend (`processor-service`, `job-service`, `demo-control-service`), Frontend (`ConsumerLabPanel`, `DemoTopology`, `demoApi`), Documentation.
- **Documentation**:
  - Updated `DEVELOPER_GUIDE.md` with capability matrix mapping and architecture comparison.
  - Updated `api-spec.md` with HTTP endpoints and NATS subject contracts.
- **Reason**:
  - Fulfill specification in `docs/feature.md` to demonstrate Core NATS Queue Groups alongside JetStream Competing Consumers.
- **Affected Area**:
  - Backend, Frontend, Documentation.

### Changed (UI Information Content Aligned with NATS Capability Mapping)
- **NATS Capability Popovers (`natsInfo.ts`)**:
  - Aligned all `(i)` educational popover entries across the platform with the **NATS Capability Mapping -- NATS Native vs Incumbents** model per `docs/feature.md`.
  - Structured every entry around the 4-question mental model:
    - **`Role`**: Concise definition of the component/capability.
    - **`Concepts`**: Accurate technical concepts that teach (e.g., Stream persistence vs Consumer durability, Push vs Pull delivery, Competing Consumers, Message Deduplication, Replay policies, Request/Reply inboxes).
    - **`Demo Usage`**: Contextualized explanation of how this specific demo exercises the capability.
    - **`Trivia`**: Why the platform cares / architectural advantages of native NATS primitives.
  - Distinctly separated **NATS Capability / Resource Components** from **Demo-Specific Components** (which explain demo mechanisms rather than generic documentation).
  - Clarified technical distinctions (Durable vs Ephemeral, Push vs Pull, Stream vs Consumer, Deduplication vs business idempotent processing).
  - Maintained complete UI stability: exactly one `(i)` icon per component with zero visual layout changes.
- **Reason**:
  - Ensure the demonstration UI acts as an authoritative, technically precise learning tool for developers and architects evaluating NATS.
- **Affected Area**:
  - Frontend (`natsInfo.ts`), Documentation (`CHANGELOG.md`).

### Changed (Deprecate Legacy Correlation ID in Favor of W3C Trace Context)
- **Standardized on W3C Distributed Tracing**:
  - Deprecated and removed legacy `correlation_id` / `Corr ID` throughout the backend, frontend, and documentation.
  - The platform now relies entirely on the industry-standard **W3C Trace Context** (`traceparent` header propagated over NATS) and OpenTelemetry `Trace ID` linked to Grafana Tempo, paired with `job_id` for business identity and `Nats-Msg-Id` for JetStream deduplication.
- **Backend Cleanups**:
  - Removed `CorrelationID` from domain structs (`Job`, `JobStatusResponse`, `JobDetailResponse`), store methods, publisher methods (`PublishJobSubmitted`, `RequestJobValidation`, `PublishJobLifecycle`), HTTP handlers (`SubmitJob`, `ValidateJob`), activity tracker (`Activity`, `ProcessLifecycleEvent`), and DLQ message model.
- **Frontend Cleanups**:
  - Removed `correlation_id` from API contracts in `demoApi.ts`.
  - Removed the `Corr ID` column and search query matching in `ActivityPanel.tsx`, reclaiming horizontal table width for Subject, Event, and Worker.
  - Removed Correlation ID from `JobInspectorPanel.tsx`, focusing the inspector on Job ID and the OpenTelemetry Trace ID with its Tempo link.
- **Documentation**:
  - Updated `api-spec.md`, `frontend.md`, `DEVELOPER_GUIDE.md`, and `FUNCTIONAL_TESTING_GUIDE.md`.
- **Reason**:
  - Eliminate redundant tracing abstractions and declutter the Activity Log table.
- **Affected Area**:
  - Backend, Frontend, Documentation.

### Changed (Modal Job Inspector, Event Capping & Observability Switcher)
- **Modal Job Inspector (`JobInspectorPanel.tsx`, `App.tsx`, `index.css`)**:
  - Transformed `JobInspectorPanel` into a focused modal pop-up overlay dialog with dark blurred backdrop, keyboard Escape dismiss, and `[X]` close button.
  - Clicking any row in the Activity Log immediately displays the inspector directly on top of the screen without scrolling down or moving the view.
- **Event Capping (`ActivityPanel.tsx`, `AddressingPanel.tsx`)**:
  - Implemented an event display limit selector (`Cap: [ 15 | 30 | 50 | All ]`, default: 15) and a fixed max-height scrollable container (`420px`) on `ActivityPanel`, preventing vertical page runaway.
  - Capped wildcard match events in `AddressingPanel` to the 10 most recent deliveries with clean scroll overflow.
- **Top Observability View Switcher (`ObservabilityPanelContainer.tsx`, `App.tsx`, `index.css`)**:
  - Created `ObservabilityPanelContainer` in the right column featuring a top segmented switcher: `[ Live Activity Log ]` and `[ Subject Addressing & Wildcards ]`.
  - Restores significant vertical space and eliminates the lower dock, balancing left and right column heights.
- **Reason**:
  - Enhance presenter usability by making job inspection instant via modal overlay, preventing page expansion with event capping, and providing convenient top-level switching to wildcard routing.
- **Affected Area**:
  - Frontend (`JobInspectorPanel.tsx`, `ActivityPanel.tsx`, `AddressingPanel.tsx`, `ObservabilityPanelContainer.tsx`, `App.tsx`, `index.css`), Documentation (`CHANGELOG.md`, `DEVELOPER_GUIDE.md`).

### Changed (Dashboard Layout: NATS Capability Studio & Observability Dock)
- **Frontend Capability Studio (`CapabilityStudio.tsx`, `App.tsx`, `index.css`)**:
  - Replaced the tall vertical stack of 5 action panels on the left with a unified `CapabilityStudio` featuring segmented navigation tabs:
    1. `Pub/Sub & Stream`: Combines standard job submission with an instant toggle for JetStream deduplication testing.
    2. `Request / Reply`: Synchronous RPC validation testing and timeout simulation.
    3. `Dead Letter Queue`: Poison message failure routing and DLQ message inspection.
    4. `Stream Replay`: Historical time-window and sequence rewind controls.
- **Frontend Observability Dock (`ObservabilityDock.tsx`, `App.tsx`, `index.css`)**:
  - Created a coordinated dock directly below `ActivityPanel` housing:
    1. `Subject Addressing & Routing`: Always accessible for real-time wildcard matching demonstrations (`*` and `>`).
    2. `Job Inspector`: Automatically surfaces whenever a message row in the Activity Log is clicked, with full headers, payload, and trace IDs.
- **Layout & Column Balancing (`index.css`)**:
  - Updated grid column proportions to `minmax(380px, 460px) minmax(0, 1fr)` ensuring comfortable control padding and eliminating vertical scrolling.
- **Reason**:
  - Provide an uncluttered, sequential presentation workflow for live demonstrations while keeping PLATFORM STATUS, CURRENT DEMO SETUP, ACTIVITY LOG, and OBSERVABILITY SETUP in their established positions.
- **Affected Area**:
  - Frontend (`CapabilityStudio.tsx`, `ObservabilityDock.tsx`, `App.tsx`, `index.css`), Documentation (`CHANGELOG.md`, `DEVELOPER_GUIDE.md`).

### Changed (Refactor Dead Letter Queue UI for Left Column Proportions)
- **Frontend DLQ Polish (`DLQPanel.tsx`, `index.css`)**:
  - Replaced unstyled `.info-btn` with standard `.node-info-btn` class, fixing the white button glitch and matching the cyan educational `(i)` badge pattern across all dashboard panels.
  - Replaced the wide 5-column HTML table that was overflowing the 360px left column with a responsive, scrollable failed-message card list.
  - Formatted DLQ Stream and Consumer metrics into a clean 2-column grid (`JOBS_DLQ` card + `dlq-inspector` card), eliminating horizontal stretching and wrapping artifacts.
- **Reason**:
  - Resolve visual defects where the `(i)` button appeared unstyled white and the component looked forced into the left column width.
- **Affected Area**:
  - Frontend (`DLQPanel.tsx`, `index.css`), Documentation (`CHANGELOG.md`).

### Added (JetStream Dead Letter Queue Feature)
- **Backend DLQ Stream & Consumer Infrastructure (`client.go`, `subjects.go`)**:
  - Added `SubjectJobDLQ` (`jobs.dlq`) and `SubjectJobDLQPublished` (`jobs.dlq.published`).
  - Added `EnsureDLQStream` in `internal/natsclient/client.go` to guarantee stream `JOBS_DLQ` (subjects `jobs.dlq`, `jobs.dlq.>`) and durable consumer `dlq-inspector` exist.
- **Processor Max Delivery Routing (`processor-service/main.go`)**:
  - Added `max_delivery_attempts` evaluation during simulated failure processing (default: 3).
  - When failure attempts reach max deliveries, routes the failed message to `JOBS_DLQ` on subject `jobs.dlq`, emits `DLQ_PUBLISHED` lifecycle event, and explicitly ACKs the original message in `JOBS` stream to cease redelivery.
- **Demo Control Endpoints (`control_handler.go`, `routes.go`, `tracker.go`)**:
  - Added `GET /dlq/status` returning `JOBS_DLQ` message counts, byte storage, and `dlq-inspector` pending counts.
  - Added `GET /dlq/messages` returning parsed DLQ messages with job IDs, original subjects, delivery attempts, failure reasons, and timestamps.
  - Mapped `jobs.dlq` and `jobs.dlq.published` to `DLQ_PUBLISHED` in `activity.Tracker` with status weight 4.
- **Frontend Dead Letter Queue Dashboard (`DLQPanel.tsx`, `DemoTopology.tsx`, `App.tsx`, `demoApi.ts`, `natsInfo.ts`, `index.css`)**:
  - Implemented compact `DLQPanel` with `Max Delivery Attempts` control, `Send Failing Job` action, live DLQ status badges, and DLQ messages table.
  - Extended DemoTopology visualizer inside NATS Server to display `STREAM: JOBS_DLQ` and `CONSUMER: dlq-inspector` alongside the primary pipeline.
  - Added educational popover entry for `dead-letter-queue`.
  - Added `.badge-dlq` style for Activity Log tracking.
- **Reason**:
  - Fulfill requirements in `docs/feature.md` to demonstrate the application-level Dead Letter Queue pattern on NATS JetStream.
- **Affected Area**:
  - Backend (`subjects.go`, `client.go`, `tracker.go`, `processor-service`, `control_handler.go`, `routes.go`), Frontend (`DLQPanel.tsx`, `DemoTopology.tsx`, `App.tsx`, `ActivityPanel.tsx`, `demoApi.ts`, `natsInfo.ts`, `index.css`), Documentation (`CHANGELOG.md`, `DEVELOPER_GUIDE.md`, `api-spec.md`).

### Changed (Restructure Current Demo Setup Topology in React UI)
- **Frontend Topology Restructuring (`frontend/src/components/DemoSetup/DemoTopology.tsx`)**:
  - Restructured the runtime visualizer into a two-tier layout: Tier 1 displays `React UI (:5173)` pointing (`->`) to `Demo Control Service (:8080)` via horizontal connector, with inter-tier vertical bridge connectors linking down to Tier 2 (`Job Service (:8081) -> NATS Server (:4222) -> Processor Service`).
  - Removed the bottom dashed box titled `OBSERVABILITY & UI CONTROL HARNESS (DECOUPLED FROM BUSINESS LOGIC)` and removed redundant observability phrasing from the top section, keeping observability dedicated to the bottom metrics and distributed tracing panel.
- **Frontend Styling & Educational Content (`index.css`, `natsInfo.ts`)**:
  - Added CSS classes `.topology-tier-header`, `.topology-tier-tag`, `.topology-tier-control`, `.topology-tier-row`, and `.topology-vertical-bridge` for clean two-tier layout alignment.
  - Added educational popover entry for `react-ui` with role, concepts, demo usage, and trivia.
  - Aligned `demo-control-service` role description to "Dedicated UI gateway and demo controller".
- **Reason**:
  - Provide a clear top-to-bottom developer entry point in the architecture visualizer while eliminating confusing and redundant observability labeling in the top section.
- **Affected Area**:
  - Frontend (`DemoTopology.tsx`, `index.css`, `natsInfo.ts`), Documentation (`CHANGELOG.md`, `DEVELOPER_GUIDE.md`).

### Changed (Cross-Stack Synchronization: Code, UI, Docker & Documentation)
- **Documentation & Docker Compose Alignment (`README.md`, `DEPLOYMENT_GUIDE.md`)**:
  - Clarified port `3000` is Grafana (OTEL-LGTM stack with Tempo distributed traces) rather than NATS UI, matching `deploy/docker-compose.yaml` and UI deep links.
  - Aligned verification check to validate Grafana at `http://localhost:3000` (`admin`/`admin`).
- **Testing & Run Guide (`FUNCTIONAL_TESTING_GUIDE.md`)**:
  - Added Terminal 1 instructions to launch `demo-control-service` on `:8080`, required for the React UI to connect and operate.
  - Corrected execution working directory to `cd backend/src` to align with the Go module root.
- **API Specification Alignment (`docs/api-spec.md`)**:
  - Renumbered duplicate Section 2 to Section 3 (`NATS Subjects & Payload Contracts`), Flowchart to Section 4, and Phased Approach to Section 5.
  - Renumbered Demo Control endpoints from `1.6`-`1.12` to `2.2`-`2.8`.
  - Added missing NATS contract rows: `jobs.replayed`, `jobs.received`, `jobs.request.sent`, `jobs.request.timeout`, and `processor.state.set`.
- **Configuration Template (`backend/src/.env.example`)**:
  - Documented `JOB_SERVICE_PORT=8081`, `JOB_SERVICE_URL=http://localhost:8081`, and OpenTelemetry OTLP endpoint variables.
- **Developer Guide (`docs/DEVELOPER_GUIDE.md`)**:
  - Included `demo-control-service` in the repository structure directory tree.

### Fixed (Undefined telemetry.RecordJobSubmission Reference)
- **Backend (`internal/telemetry/telemetry.go`, `api/http/job_handler.go`)**:
  - Added `RecordJobSubmission` in `internal/telemetry/telemetry.go` with `delivery_mode`, `status`, and `duration` attributes matching the invocation in `api/http/job_handler.go`.
  - Preserved `RecordJobSubmitted` as an alias to `RecordJobSubmission` for backward compatibility.
- **Reason**:
  - Resolve compiler error `undefined: telemetry.RecordJobSubmission` when building the job service.
- **Affected Area**:
  - Backend (`internal/telemetry/telemetry.go`, `api/http/job_handler.go`).

### Fixed (Undefined SubjectProcessorState Reference)
- **Backend (`api/http/control_handler.go`, `internal/messaging/subjects.go`)**:
  - Fixed compilation error in `PutProcessorState` where `messaging.SubjectProcessorState` was referenced instead of `messaging.SubjectProcessorStateSet`.
  - Added `SubjectProcessorState` as a constant alias to `SubjectProcessorStateSet` in `internal/messaging/subjects.go` for consistency and backward compatibility.

### Added (Layer 2 Architectural Decoupling: Demo Control Service vs Pure Business Services)
- **New Service (`cmd/demo-control-service`)**:
  - Created a dedicated UI gateway and observability harness running on port `:8080`.
  - Passively taps NATS events (`jobs.>`) via `activity.Tracker` to feed the live activity stream.
  - Houses the subject addressing demo `Observer` and observer subscriptions.
  - Houses the ephemeral JetStream replay consumer engine (`POST /jobs/replay`).
  - Houses status aggregation and remote processor/consumer control APIs (`/status`, `/processor/state`, `/consumer`).
- **Pure Business Refactoring (`cmd/job-service`)**:
  - Stripped all demo harness code, observer subscriptions, and in-memory activity ring buffers.
  - Dedicated to pure domain logic on port `:8081` (`POST /jobs`, `POST /jobs/validate`, `GET /jobs`, `GET /health`).
  - Implements clean domain `JobStore` storing only business job records and status history.
- **Frontend Architecture & Topology Visualization (`demoApi.ts`, `DemoTopology.tsx`, `StatusPanel.tsx`)**:
  - Routed business operations (`submitJob`, `validateJob`, `getJobDetail`) to `http://localhost:8081`.
  - Routed demo inspection and control (`getServiceStatus`, `getActivity`, `replayJobs`, `updateProcessorState`, `getConsumerStatus`) to `http://localhost:8080`.
  - Updated `DemoTopology.tsx` to visualize the 4-tier deployed runtime architecture, featuring an isolated "Observability & UI Control Harness" tier connected to NATS and the React UI.
  - Updated `StatusPanel.tsx` and `natsInfo.ts` to include `Demo Control (:8080)` and `Job Service (:8081)`.

### Reason
- Separate demo-specific instrumentation, in-memory taps, and dashboard endpoints from production-grade business services, providing developers with a clean reference implementation of pure domain services on NATS.

### Affected Area
- Backend (`cmd/demo-control-service`, `cmd/job-service`, `internal/activity`, `internal/jobs`, `api/http`), Frontend (`demoApi.ts`, `DemoTopology.tsx`, `StatusPanel.tsx`, `App.tsx`, `natsInfo.ts`), Documentation (`README.md`, `DEVELOPER_GUIDE.md`, `DEPLOYMENT_GUIDE.md`, `api-spec.md`, `CHANGELOG.md`).

### Fixed (JetStream Stream Metrics Stripping in getServiceStatus)
- **Frontend (`api/demoApi.ts`)**: Fixed `getServiceStatus()` discarding `messages`, `bytes`, `first_seq`, and `last_seq` returned by backend `GET /status`, restoring live stored message counts and sequence ranges in the UI.

### Added (Stored Message Count Display in JetStream Replay)
- **Backend Stream Metric Reporting (`api/http/handler.go`)**:
  - Extended `GET /status` JetStream metadata to query `js.StreamInfo("JOBS")` directly, exposing total stored messages (`messages`), storage size (`bytes`), first sequence (`first_seq`), and last sequence (`last_seq`) alongside consumer pending count.
- **Frontend JetStream Replay Integration (`ReplayPanel.tsx`, `App.tsx`, `demoApi.ts`)**:
  - Displayed live stored message badge (`X stored msgs`) and sequence boundary range (`Seq #first - #last` or `Stream Empty`) in the `Stream: JOBS` row.
  - Added an on-demand `Refresh` button in the panel header to allow immediate status re-query without waiting for background polling.
  - Added dynamic empty-stream guidance banner if no historical messages are currently stored in `JOBS`.
  - Added boundary indicators (`First: #X`, `Last: #Y`) directly above Sequence mode inputs.
  - Displayed live stored message count badge on the `STREAM: JOBS` card in `DemoTopology.tsx`.
- **Documentation**:
  - Updated `docs/api-spec.md` and `docs/CHANGELOG.md` with the new schema and feature details.

### Fixed (Unused consumerName Variable in ReplayMessages)
- **Backend (`api/http/handler.go`)**: Assigned `Name: consumerName` in `consumerCfg := &nats.ConsumerConfig{...}` inside `ReplayMessages`, resolving the Go compiler error `declared and not used: consumerName` while ensuring the ephemeral replay consumer is explicitly named.

### Added (Improve JetStream Replay Controls & Real Backend Replay Engine)
- **JetStream Replay Controls Panel (`ReplayPanel.tsx`)**:
  - Added read-only `Stream: JOBS` indicator identifying the target historical stream.
  - Added `Replay From` selector supporting `Sequence` mode and `Time` mode.
  - Renamed fields to standard JetStream replay terminology: `Start Sequence` and `End Sequence` with integer range validation (`Start >= 1`, `End >= Start`).
  - Added `Start Time` and `End Time` inputs in Time mode with chronological ordering validation (`Start Time < End Time`).
  - Added `Replay Mode` selector supporting `Instant` (as fast as possible) and `Original Timing` (reproducing message emission intervals).
  - Preserved existing `(i)` educational popover trigger and single-entry-point hierarchy.
- **Backend JetStream Ephemeral Replay Engine (`api/http/handler.go`)**:
  - Replaced stubbed mock in `POST /jobs/replay` with actual NATS JetStream ephemeral push consumer creation.
  - Configured `DeliverByStartSequencePolicy` and `DeliverByStartTimePolicy` based on selection.
  - Configured native `ReplayInstantPolicy` and `ReplayOriginalPolicy` dynamically on the consumer.
  - Added background consumer worker delivering historical messages to temporary inboxes and emitting `REPLAYED` events to subject `jobs.replayed`.
  - Automatically deleted the ephemeral replay consumer upon stream boundary completion or timeout.
- **Observability & Log Integration**:
  - Added `SubjectJobReplayed` (`jobs.replayed`) in `subjects.go`.
  - Added `REPLAYED` event handling in `service.go` and rendered `.badge-replayed` in `ActivityPanel.tsx` and `index.css`.

### Reason
- Fulfill requirements in `docs/feature.md` to accurately represent JetStream replay semantics, terminology, and controls, while implementing real backend replay capabilities directly over NATS JetStream.

### Affected Area
- Frontend (`ReplayPanel.tsx`, `demoApi.ts`, `ActivityPanel.tsx`, `index.css`), Backend (`handler.go`, `subjects.go`, `service.go`), Documentation (`DEVELOPER_GUIDE.md`, `frontend.md`, `backend.md`, `api-spec.md`, `CHANGELOG.md`).

### Added (Standalone Message Deduplication Component)
- **Extracted Component (`DeduplicationPanel.tsx`)**: Promoted Message Deduplication from an embedded subsection in `JobPanel.tsx` to a full standalone panel under `DEMO ACTIONS`.
- **Standardized Panel Design**: Matched the exact visual and DOM hierarchy of existing panels (`JobPanel`, `RequestReplyPanel`, `ReplayPanel`), using standard `.panel`, `<h2 className="panel-title">` with SVG shield-check icon, and standard `node-info-btn` for `(i)`.
- **Non-Editable Config Parameters**: Displayed the current active stream parameters directly in the panel: `Stream Config: Duplicates = 2m 0s (120s window)`, `Message ID Header: Nats-Msg-Id`, and `Target Stream: JOBS (Subject: jobs.submitted)`.
- **Educational Popover Encapsulation**: Distilled official NATS/JetStream deduplication architecture into a compact 4-part preview in `natsInfo.ts` covering stream-level ownership, default vs configurable window limits, opt-in `Nats-Msg-Id` semantics, cluster replication vs multi-region boundaries, and server-side memory lookup overhead.
- **Interactive Duplicate Testing**: Streamlined form containing `Message ID (Nats-Msg-Id)` input with `Publish (1st)` and `Publish Duplicate` action buttons and standard inline status feedback.

### Reason
- Ensure Message Deduplication possesses identical visual design, interaction patterns, and educational encapsulation as all other demo action panels.

### Affected Area
- Frontend (`DeduplicationPanel.tsx`, `JobPanel.tsx`, `App.tsx`, `natsInfo.ts`, `index.css`), documentation (`DEVELOPER_GUIDE.md`, `frontend.md`, `CHANGELOG.md`).

### Changed (Rework Frontend Component Educational Information Content)
- **Standardized Educational Hierarchy**: Restructured all component information modals in `natsInfo.ts` into a cohesive 4-part hierarchy: Role (*What is this?*), Concepts (*What technical concepts does it represent?*), Demo Usage (*How is this demonstrated here?*), and Trivia (*What useful NATS fact or terminology should I remember?*).
- **Correct Conceptual Boundaries**:
  - Reinforced that **Streams** persist messages while **Consumers** maintain delivery state and progress cursors.
  - Clarified that **Durable vs. Ephemeral** defines the consumer lifecycle across client sessions, not message persistence.
  - Distinctly separated **Core NATS** (transient pub/sub & req/reply) from **JetStream** (persistence, streams, consumers, acknowledgements, replay).
  - Clarified that `Nats-Msg-Id` serves JetStream server-side duplicate message detection rather than acting merely as an application identifier.
- **Single Entry Point Preserved**: Maintained exactly one `(i)` trigger button per component and panel with no added fields or UI overhead.

### Reason
- Ensure every `(i)` educational popover across the demo has a clear, accurate, and consistent pedagogical purpose, teaching genuine NATS concepts and architectural models rather than generic UI descriptions.

### Affected Area
- Frontend educational content (`frontend/src/content/natsInfo.ts`), documentation (`docs/CHANGELOG.md`).

### Changed (Service Renaming: demo-service to job-service)
- **Backend Service Renaming**: Renamed the primary API and publisher service from `demo-service` to `job-service`. Created entry point `backend/src/cmd/job-service/main.go`, updated OpenTelemetry service registration to `"job-service"`, and updated log output prefixes.
- **Header Attribution & Event Metadata**: Updated `X-Source` header and in-memory event tracking to identify `"job-service"` as the source for `PUBLISHED`, `STORED`, `DEDUPLICATED`, `REQUEST_SENT`, and `REPLY_RECEIVED` transitions.
- **Frontend Alignment**: Updated `StatusPanel.tsx`, `DemoTopology.tsx`, `ObservabilityPanel.tsx`, and `App.tsx` to reference `Job Service (8080)` and query service status for `job-service`.
- **Educational Content**: Added `job-service` entry in `natsInfo.ts` with backward-compatible alias `demo-service`.
- **Documentation**: Updated `README.md`, `DEVELOPER_GUIDE.md`, `DEPLOYMENT_GUIDE.md`, `FUNCTIONAL_TESTING_GUIDE.md`, `api-spec.md`, `backend.md`, and `frontend.md` to reference `job-service` and `Job Service`.

### Reason
- Establish domain-specific, meaningful naming (`job-service`) aligned with its actual role as the job management, API gateway, and publisher component rather than generic demo nomenclature.

### Affected Area
- Backend (`cmd/job-service/main.go`, `api/http/handler.go`, `internal/messaging/publisher.go`, `internal/jobs/service.go`), Frontend (`StatusPanel.tsx`, `DemoTopology.tsx`, `ObservabilityPanel.tsx`, `App.tsx`, `natsInfo.ts`), Documentation (`README.md`, `docs/`).

### Changed (Correct CURRENT DEMO SETUP Architecture Representation)
- **Deployed Runtime Boundaries vs. Logical Resources**: Redesigned the `DemoTopology` visualizer in `DemoSetupPanel.tsx` to clearly differentiate the 3 deployed runtime components (`Demo Service`, `NATS Server`, `Processor Service`) from internal logical NATS/JetStream resources and worker routines.
- **NATS Server Containment Model**: Nested `Core NATS` (Pub/Sub & Req/Reply) and `JetStream` (Persistence & Streaming) capabilities inside the `NATS Server` boundary card. Contained `JOBS Stream` and `job-processor` Consumer within a dedicated JetStream managed resources sub-container, preventing them from being misinterpreted as separate deployable services.
- **Processor Service Worker Pool**: Grouped application workers (`processor-1`, and `processor-2` when configured) inside the `Processor Service` card, showing single-worker execution and dynamic `COMPETING WORKERS` branching pulling from the shared JetStream consumer.
- **Delivery Flow & State Indication**: Connected message delivery directly from the internal Consumer to Processor Service workers. Toggling processing OFF severs the delivery connection with a `[ PAUSED ]` indicator while preserving the Consumer's active state and message buffering inside NATS Server.
- **Contextual Educational Info**: Updated `(i)` popover explanations in `natsInfo.ts` for `nats-server`, `jobs-stream`, `consumer`, and `processor-service` to reinforce the mental model that Streams and Consumers are logical JetStream resources managed inside NATS, not services or containers.
- **Panel Title Renaming**: Renamed the `Submit Job` panel to `Pub Sub` in `JobPanel.tsx` and `natsInfo.ts` for clearer logical parity with other messaging patterns (`Request / Reply`, `JetStream Replay`).

### Reason
- The previous visualization rendered `Demo Service -> NATS Server -> JOBS Stream -> Consumer -> Processor` as peer deployment boxes, misleadingly suggesting that Streams and Consumers are standalone deployable services.

### Affected Area
- Frontend topology visualizer (`DemoTopology.tsx`), stylesheet (`index.css`), educational content (`natsInfo.ts`), developer documentation (`DEVELOPER_GUIDE.md`).

## 2026-09-01

### Added (NATS Multi-Token Subject Addressing Demonstration)
- **Multi-Token Event Publishing**: Activated the 3-token subject `jobs.processing.started` (`a.b.c`) when tasks begin execution in both Core NATS and JetStream worker routines in `processor-service/main.go`.
- **Wildcard Distinction in UI**: Enhanced the **Subject Routing Activity** table in `AddressingPanel.tsx` with dynamic token count badges (`2 tokens` vs `3 tokens`) and highlighted `Yes (> only)` indicators to visibly demonstrate that Single-Level wildcards (`jobs.*`) reject 3-token subjects while Multi-Level wildcards (`jobs.>`) accept them.
- **Testing Guide Update**: Updated Scenario 10 in `FUNCTIONAL_TESTING_GUIDE.md` with step-by-step observation instructions for multi-token wildcard behavior.

### Added (Feature: End-to-End Distributed Tracing with OpenTelemetry and Tempo)
- **Trace Context Propagation over NATS**: Implemented W3C `traceparent` context propagation across NATS message headers (`nats.Msg.Header`) using `telemetry.InjectTraceContext` on publish and `telemetry.ExtractTraceContext` on consumption.
- **Trace Span Hierarchy**:
  - `POST /jobs` (HTTP Server span) -> `NATS Publish jobs.submitted` (Producer span) -> `Consumer Receive` (Consumer span) -> `Process Job` (Internal span) -> ACK / Redelivery events.
  - `POST /jobs/validate` (HTTP Server span) -> `NATS Request jobs.validate` (Client span) -> `Process Validation Request` (Server span) -> `NATS Reply` (Producer span).
- **Tempo Integration**: Connected OpenTelemetry OTLP trace exporter to Grafana OTEL-LGTM Tempo instance via port 4317.
- **Trace ID in Models & Store**: Propagated `trace_id` through `Job`, `JobStatusResponse`, `JobValidationResponse`, `JobDetailResponse`, and `Activity`.
- **UI Trace Exploration**: Added `Trace ID` display with direct `[ View in Tempo -> ]` deep-link in `JobInspectorPanel`, opening the trace waterfall directly in Grafana Tempo Explore (`http://localhost:3000/explore`).
- **Observability Panel Updates**: Expanded Observability architecture diagram and endpoints to showcase both Distributed Tracing (Tempo) and Prometheus Metrics.

### Added (Feature: Metrics Observability)
- **Central Telemetry Package**: Created `backend/src/internal/telemetry` implementing OpenTelemetry metrics instrumentation using standard OTLP gRPC export with a 2-second periodic push interval.
- **Application Metrics**: Instrumented `demo-service` and `processor-service` for job submissions, processing counts, failure simulations, ACK tracking, redelivery occurrences, and latency distributions (p50/p95).
- **Observability Infrastructure**: Added `docker.io/grafana/otel-lgtm` and `natsio/prometheus-nats-exporter` to `deploy/docker-compose.yaml`. Shifted `nats-ui` to port 3001 and mapped Grafana to port 3000.
- **Grafana Dashboard**: Provisioned `NATS Platform Demo - Metrics` dashboard structured into 5 dedicated sections covering the Top 20 NATS metrics: NATS HEALTH (Health, Connections, Subscriptions, CPU, Memory, Uptime), MESSAGING (Messages In/Out rates, Throughput Bytes In/Out), JETSTREAM (Streams, Messages, Bytes, Ingress Rate, Storage), CONSUMERS (Consumers, Pending, Ack Pending, Redeliveries), and JETSTREAM ACTIVITY (Delivery & Ingress rates).
- **Dedicated Observability Setup UI**: Added standalone `OBSERVABILITY SETUP` panel at the bottom of the React dashboard with an architectural pipeline diagram, contextual `(i)` educational popover, and direct `[ Open Grafana -> ]` link.
- **Docker Compose Container Networking**: Added dedicated bridge network `nats-net` connecting `nats`, `nats-exporter`, and `otel-lgtm`. Configured `nats-exporter` target to `http://nats:8222` and Prometheus scrape target to `nats-exporter:7777` for deterministic container DNS resolution.

### Documentation
- **Functional Testing Guide**: Created [docs/FUNCTIONAL_TESTING_GUIDE.md](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/docs/FUNCTIONAL_TESTING_GUIDE.md) providing an end-to-end evaluation guide with 11 hands-on testing scenarios covering Core NATS transient messaging, JetStream persistent delivery, offline message accumulation, message deduplication (`Nats-Msg-Id`), competing consumers, durable vs ephemeral lifecycles, ordered sequence consumption, stream replays, synchronous Request/Reply with timeouts, hierarchical subject wildcard addressing, and Activity Log search/filtering.

### Added (Activity Log ID Columns & Real-time Search and Filter Bar)
- **Activity Log IDs**: Added NATS Message ID (`msg_id` / `Nats-Msg-Id`), Job Type / Category (`job_type`), and Correlation ID (`correlation_id`) across backend store and HTTP activity stream.
- **Hover Legend Tooltips**: Truncated long Correlation IDs and NATS Message IDs in table cells into compact badges with a floating hover legend popover revealing the complete full-length identifier on mouse hover.
- **Layout Bounds & Widescreen Optimization**: Enforced `min-width: 0` on dashboard grid tracks and columns to prevent table overflow from pushing sections off-screen. Expanded overall application `max-width` to `1600px` for comfortable widescreen viewing without disrupting element positioning.
- **Activity Table Columns**: Enhanced Activity Log table with dedicated columns: `Corr ID`, `NATS Msg ID`, and `Type` badge.
- **Search & Filter Toolbar**: Added interactive search and filter bar supporting live full-text search across all identifiers, dynamic Event dropdown filter, Delivery Mode filter (`CORE` vs `JETSTREAM`), Worker filter (`processor-1` vs `processor-2` vs `demo-service`), live result counter, and quick Clear button.

### Changed (Demo Setup Alignment & Universal Info Buttons)
- **Consumer Lab Horizontal Expansion**: Sized Demo Topology container cleanly to fit its nodes (ending right after `processor-2`) and allocated the remaining available space (`minmax(0, 1fr)`) to Consumer Lab, eliminating the empty gap and making Consumer Lab controls and metrics spacious.
- **Horizontal Form Layout**: Grouped Consumer Type and Ordering into a 2-column horizontal row in Consumer Lab, along with Workers and Delivery Semantics, to optimize horizontal width utilization and streamline panel height.
- **Consumer Lab Relocation**: Relocated Consumer Lab into the right side of the `CURRENT DEMO SETUP` container, pairing the interactive consumer controller and live metrics directly alongside the runtime topology visualization. Cleaned up `DEMO ACTIONS` column to focus strictly on demo operations.
- **Direct Stream-to-Consumer Pipeline**: Reorganized Demo Topology into an aligned 3-column architecture where `JOBS Stream`, `Consumer`, and `Processor` reside in a unified vertical column with continuous connectors, eliminating any horizontal offset or visual gap.
- **Visual Competing Workers Branching**: When Workers count is set to 2 in Consumer Lab, Demo Topology dynamically branches the single shared Consumer into two distinct competing worker cards (`processor-1` and `processor-2`) side-by-side with a `COMPETING` branch indicator.
- **Immediate Consumer Lab Synchronization**: Added `onConfigChanged` callback to `ConsumerLabPanel` so that changes to Consumer Type (Durable/Ephemeral), Worker count (1 or 2), and Ordering immediately update the `DemoTopology` and `DemoSummary` cards without waiting for polling intervals.
- **Universal Contextual Info Buttons `(i)`**: Added `(i)` information buttons across all dashboard panels (`Platform Status`, `Submit Job`, `Consumer Lab`, `Request / Reply`, `JetStream Replay`, `Activity Log`, `Job Details`, `Subject Addressing`) opening contextual explanations of relevant NATS concepts and trivia.

### Added (Demo Setup and NATS Information)
- **Current Demo Setup Topology**: Added interactive visual topology diagram beneath Platform Status displaying connected nodes: Demo Service, NATS Server, JOBS Stream, Consumer, and Processor.
- **Dynamic Topology State**: Connector between Consumer and Processor dynamically indicates active vs paused state when Processing is toggled ON/OFF. Component badges reflect live runtime status.
- **Contextual NATS Information**: Each component provides an `(i)` button opening a modal with Role, Core NATS Concepts, Demo Usage, and Architecture Trivia.
- **Current Demo Summary**: Added compact summary card displaying active delivery mode, consumer type, worker pool count, ordering, and processor state.

### Added (NATS Consumer Capabilities Demo)
- **Consumer Lab**: Added interactive Consumer Lab panel in DEMO ACTIONS to configure Consumer Type (Durable vs Ephemeral), Worker pool size (1 or 2 competing workers), and Ordering (Normal vs Ordered).
- **Competing Consumers**: Enabled dynamic worker pool (`processor-1`, `processor-2`) sharing the pull consumer, with worker identifiers clearly visible in the Activity Log.
- **Durable vs Ephemeral Consumers**: Supported durable consumer `job-processor` and dynamic ephemeral consumer lifecycle via NATS JetStream.
- **Ordering**: Implemented ordered consumer demonstration linking stream sequence to delivery sequence.
- **At-Least-Once & Redelivery**: Added NATS JetStream redelivery detection (`meta.NumDelivered > 1`), publishing `REDELIVERED` event with delivery count and explicit ACK tracking.
- **JetStream Message Deduplication**: Handled `ack.Duplicate` on JetStream publish with `Nats-Msg-Id` within 2-minute deduplication window, displaying `DEDUPLICATED` event badges.
- **HTTP Endpoints**: Added `GET /consumer` (queries configuration and live pending/ack_pending/redelivered metrics) and `PUT /consumer` (reconfigures processor consumer over NATS control subject `consumer.config.set`).

### Fixed (Core NATS Worker Distribution)
- Fixed Core NATS jobs only displaying `processor-1`: In `processor-service/main.go`, `jobHandler` was hardcoded to `workerName` (`processor-1`). Added atomic round-robin dispatch across active workers (`a.consumerConfig.Workers`) so that jobs submitted in either `CORE` or `JETSTREAM` mode are distributed across `processor-1` and `processor-2`.

### Fixed (Processor State Toggle)
- Fixed processor toggle button becoming disabled when Processing state was OFF: `demoApi.ts` sets service status to `'stopped'` when `processing: false`, which caused `isProcessorActive` (`status === 'active'`) to evaluate to `false` and disable the button. Replaced condition with `isProcessorOnline` (`status !== 'disconnected' && status !== 'unknown'`), ensuring the toggle button remains enabled while the processor process is running.

### Changed (UI Layout Reorganization)
- Promoted Platform Status to a full-width horizontal status bar directly beneath the header/alerts, displaying NATS Server, Demo Service, Processor Service, Processing toggle (ON/OFF), JetStream availability, Stream name, Pending count, Workers count, and Consumer status.
- Reorganized dashboard into two distinct columns:
  - Left Column (Demo Actions): Submit Job (`JobPanel`), Request / Reply (`RequestReplyPanel`), JetStream Replay (`ReplayPanel`).
  - Right Column (Activity): Activity Log (`ActivityPanel`), Job Details Inspector (`JobInspectorPanel` positioned between Activity and Addressing when opened), Subject Routing & Addressing (`AddressingPanel`).
- Updated Header status indicator to explicitly display `[ NATS CONNECTED ]` / `[ NATS DISCONNECTED ]`.
- Cleaned up non-ASCII symbols in UI components to adhere strictly to the repository ASCII-only rule.

### Reason
- Optimize visual hierarchy and observability by separating demo actions from live monitoring streams, with a prominent global status bar for quick health assessment.

### Affected Area
- Frontend dashboard components (`Header.tsx`, `StatusPanel.tsx`, `App.tsx`), stylesheet (`index.css`), and frontend documentation (`frontend.md`).

## 2026-08-31

### Fixed (Request/Reply Bugs)
- **JetStream intercepting jobs.validate requests (root cause of false REPLY_RECEIVED)**:
  The JOBS JetStream stream was configured with `Subjects: ["jobs.>"]`, which captured
  ALL subjects under `jobs.`, including `jobs.validate`. When demo-service sent a
  `RequestMsg` to `jobs.validate`, JetStream intercepted the message, stored it, and
  immediately published a PubAck (e.g. `{"stream":"JOBS","seq":5}`) back to the request's
  `msg.Reply` inbox. The `RequestMsg` received that PubAck as the "reply" before the
  2-second timeout could fire. Unmarshalling a PubAck into `JobValidationResponse`
  produced `{valid: false, message: ""}` -- a zero-value struct with no error, causing
  `ValidateJob` to record `REPLY_RECEIVED` and return HTTP 200 instead of 504.
  Fixed by changing the stream `Subjects` from `["jobs.>"]` to `["jobs.submitted"]` --
  the only subject that should be durably persisted.
- **Request/Reply panel Job ID not auto-incrementing**: Added auto-increment in the
  `finally` block of `handleSend` in `RequestReplyPanel.tsx`.
- **`processingEnabled` guard added to validation handler**: When the processor is toggled
  OFF, the `jobs.validate` handler now returns `ErrProcessorDisabled` which causes
  `consumer.SubscribeJobValidate` to skip `msg.Respond()`, letting the NATS 2-second
  timeout fire naturally. Added `ErrProcessorDisabled` sentinel to `jobs/model.go`.
- **`msg.Reply != ""` guard in demo-service lifecycle and observer subscriptions**: Prevents
  request messages (which carry a non-empty Reply field) from being mistaken for lifecycle
  events. Belt-and-suspenders defence alongside the JetStream fix.

### Added (Request/Reply Demo)
- Implemented **NATS Request/Reply Demo** feature on subject `jobs.validate`.
- Added dynamic validation subscriber lifecycle management in `processor-service`:
  `subscribeValidation` / `unsubscribeValidation` helpers toggle the `jobs.validate`
  subscription alongside the processor state so that NATS requests time out naturally
  when the processor is OFF, without any artificial timeout generation.
- Processor now publishes `jobs.request.received` and `jobs.reply.sent` lifecycle events
  during each validation request so demo-service can record the full timeline.
- Updated `publisher.RequestJobValidation` to accept and propagate `X-Correlation-Id`
  on the outgoing NATS request; `nats.ErrTimeout` and `nats.ErrNoResponders` are now
  surfaced as `messaging.ErrRequestTimeout` so the HTTP handler can return HTTP 504.
- Updated `ValidateJob` HTTP handler to extract `X-Correlation-Id` from the request header
  and return `504 Gateway Timeout` with a descriptive JSON body on timeout.
- Updated `jobs.Service.ValidateJob` to record `REQUEST_SENT`, `REPLY_RECEIVED`, and
  `REQUEST_TIMEOUT` activity events in the in-memory store.
- Added `jobs.request.received` and `jobs.reply.sent` subject-to-status mappings and
  status weights in `internal/jobs/service.go`.
- Widened the lifecycle subscription in `demo-service` from `jobs.*` to `jobs.>` so
  multi-segment subjects such as `jobs.request.received` are captured.
- Added two new subject constants: `SubjectJobRequestReceived` and `SubjectJobReplySent`
  in `internal/messaging/subjects.go`.
- Updated `messaging.ValidationHandler` signature to include `correlationID string` so
  the processor validation handler can propagate it in lifecycle events.
- Created frontend `RequestReplyPanel.tsx` with Job ID / Type / Payload form, `Send
  Request` action, SUCCESS/INVALID/TIMEOUT status badge, JSON result viewer, and an
  interaction timeline filtered from the live activity log.
- Extended `ActivityPanel.tsx` badge class mapping to cover the five new event types:
  `REQUEST_SENT`, `REQUEST_RECEIVED`, `REPLY_SENT`, `REPLY_RECEIVED`, `REQUEST_TIMEOUT`.
- Wired `RequestReplyPanel` into `App.tsx` left column between `JobPanel` and `ReplayPanel`.
- Updated `demoApi.ts`: `validateJob` now sends `X-Correlation-Id` and returns a typed
  `ValidationResult` with `timedOut: true` on HTTP 504 instead of throwing.

### Changed
- Updated `DEVELOPER_GUIDE.md` with Request/Reply lifecycle diagram, correlation ID
  propagation details, natural timeout explanation, and extension guide.

### Reason
- Demonstrate native NATS Request/Reply messaging with a timeout scenario that is
  controlled through the existing Processor ON/OFF toggle.


### Added (Fire-and-Forget & Durable Streaming)
- Implemented **Fire-and-Forget and Durable Streaming Demo** comparing transient Core NATS Pub/Sub and durable JetStream streaming.
- Added `delivery_mode` selection support (values: `CORE` / `JETSTREAM`) in `POST /jobs` API, Go structures, and the React `JobPanel` UI.
- Implemented automated JetStream stream initialization (`JOBS` stream, subject wildcard `jobs.>`) on `demo-service` startup.
- Configured pull durable JetStream consumer (`processor-durable`) in the `processor-service` for durable streaming.
- Implemented processor state control API (`PUT /processor/state`) to toggle processing ON/OFF dynamically.
- Extended the `GET /status` API to report JetStream stream status metadata (pending message count) and processor status.
- Added event badge classes and table columns (`Mode`, `Seq`) to the frontend `ActivityPanel` to make transient/durable behavior differences observable.
- Updated specification guides (`backend.md`, `frontend.md`) to document the new control APIs and subjects.

### Added
- Implemented **NATS Subject Addressing Demo** feature displaying active subscriptions (`exact`, `single-level`, and `multi-level`) and observed message matching routing activity.
- Created `internal/messaging/observer.go` thread-safe store in the backend to record message delivery activity.
- Created NATS observer subscriptions on subjects `jobs.submitted` (`exact`), `jobs.*` (`single-level`), and `jobs.>` (`multi-level`) in `demo-service`.
- Added endpoints `GET /messaging/subscriptions` and `GET /messaging/activity` to serve active subscriptions and observed events.
- Created frontend `AddressingPanel.tsx` visual console widget displaying the NATS subscription patterns and routing activity.
- Added message routing events publication (e.g., `jobs.processing.started`, `jobs.processing.completed`, and `jobs.processing.failed`) in `processor-service`.

### Fixed
- Fixed JetStream durable consumer lifecycle by explicitly creating the durable consumer `processor-durable` on startup and binding to it using `nats.Bind`, preventing the NATS Go client from automatically deleting the consumer on unsubscribe/shutdown.
- Fixed out-of-order dashboard activity logs by arranging correct `NO CONSUMER` publishing sequence and introducing stable logical status-weighted sorting in the activities endpoint.
- Fixed JetStream Pull subscription failure (`nats: option Durable set more than once`) in the processor service by removing the redundant `nats.Durable` option from `js.PullSubscribe`.
- Resolved TypeScript compiler type mismatch in `StatusIndicator` by adding `'running'` and `'stopped'` status values to `StatusIndicatorProps` and updating CSS styling.
- Added Job ID parsing to NATS observer subscriptions and exposed it as the first column in the frontend Addressing panel routing activity table.
- Resolved duplication of rows for synchronous validation `jobs.validate` messages by using `RequestMsg` with headers and establishing a deterministic payload-based fallback message ID calculation.
- Implemented the **Job Details Inspector** panel displaying status state, correlation ID, delivery count, and a chronological history timeline.
- Integrated the existing `JsonViewer` into the Job Details Inspector to display the raw details payload.
- Implemented the **JetStream Replay** panel form fields for triggering NATS replay sequences.
- Added clickable Job ID link action button to cells in the `Activity Log` table.
- Wired API client and state hooks in `App.tsx` to handle job inspector queries and replay triggers.
- Implemented `POST /jobs/validate` API handler using NATS Request/Reply on subject `jobs.validate`.
- Implemented `GET /jobs` and `GET /jobs/{job_id}` endpoints for tracking job states and detailed histories.
- Implemented `GET /activities` endpoint to expose flat logs of recent NATS wildcard lifecycle events.
- Created `internal/jobs/store.go` containing a thread-safe in-memory store for tracking jobs and activities.
- Added NATS lifecycle events wildcard subscription on `jobs.*` in `demo-service` to capture processing status updates.
- Added attempt tracking and failure/success lifecycle events publishing in the `processor-service`.
- Wired frontend `validateJob` and `getActivity` methods in `demoApi.ts` to fetch real backend APIs.
- Updated `handleJobValidate` in `App.tsx` to handle and display sync validation results.

### Added (Previous)
- Created the core Go project workspace files (`go.mod`, `internal/config/config.go`).
- Added NATS messaging foundation (`internal/messaging/subjects.go`, `internal/messaging/publisher.go`, `internal/messaging/consumer.go`).
- Added job domain models and service logic (`internal/jobs/model.go`, `internal/jobs/service.go`).
- Added HTTP routing and POST handler for Submit Job API (`api/http/handler.go`, `api/http/routes.go`).
- Added command entry points for `demo-service` and `processor-service`.
- Created repository `README.md` at root containing a demo setup and testing user guide.
- Created the React SPA frontend application (`frontend/package.json`, `frontend/vite.config.ts`, `frontend/tsconfig.json`, `frontend/index.html`).
- Added frontend dashboard components (`frontend/src/components/Header.tsx`, `frontend/src/components/StatusPanel.tsx`, `frontend/src/components/JobPanel.tsx`, `frontend/src/components/ActivityPanel.tsx`, `frontend/src/components/JsonViewer.tsx`, `frontend/src/components/StatusIndicator.tsx`).
- Added frontend API client (`frontend/src/api/demoApi.ts`) targeting the Go backend with custom denials for unimplemented endpoints.
- Configured CSS with a premium dark developer console layout (`frontend/src/index.css`).
- Added `GET /status` API endpoint in `demo-service` to serve NATS and service connectivity status.
- Added NATS responder on `status.processor` in `processor-service` for NATS Request/Reply discovery checks.
- Implemented periodic status polling (every 5s) and UI health indicator updates in the React frontend.
- Configured Vite scripts in `frontend/package.json` with loose Rolldown validation to suppress JSX configuration warnings in Vite 8.2.2.

### Changed
- Migrated REST API from standard library `http.ServeMux` to Gin framework in HTTP handlers and routing.
- Separated `Init`, `Run`, and `Stop` lifecycle blocks for both `demo-service` and `processor-service` entry points.
- Refactored `config.go` to load and validate variables from the environment and a `.env` file using `caarlos0/env/v10` and `godotenv`.
- Introduced a `natsclient.Client` wrapper for NATS connection operations.
- Added a custom CORS middleware in Go `demo-service` (`src/api/http/routes.go`) to allow access from Vite frontend.

### Fixed
- Corrected import paths from `nats-platform-demo` to `nats-demo` in `src/api/http/handler.go`, `src/internal/messaging/publisher.go`, and `src/internal/messaging/consumer.go`.

### Reason
- Establish the baseline project architecture, implement Core Pub/Sub messaging (Phase 1) for job submissions, fix compilation errors due to module import path discrepancies, and refactor the code for clean architecture, environment configuration management, and framework integration as requested.

### Affected Area
- Backend codebase framework, routing API, application lifecycle, configuration management, and messaging client architecture.
