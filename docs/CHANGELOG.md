# Changelog

All notable changes to this project will be documented in this file.

## 2026-09-01

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
  - Right Column (Live Observability): Activity Log (`ActivityPanel`), Job Details Inspector (`JobInspectorPanel` positioned between Activity and Addressing when opened), Subject Routing & Addressing (`AddressingPanel`).
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
