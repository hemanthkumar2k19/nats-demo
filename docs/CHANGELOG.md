# Changelog

All notable changes to this project will be documented in this file.

## 2026-09-03

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
