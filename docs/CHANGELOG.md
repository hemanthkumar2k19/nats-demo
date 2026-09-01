# Changelog

All notable changes to this project will be documented in this file.

## 2026-09-01

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
  produced `{valid: false, message: ""}` — a zero-value struct with no error, causing
  `ValidateJob` to record `REPLY_RECEIVED` and return HTTP 200 instead of 504.
  Fixed by changing the stream `Subjects` from `["jobs.>"]` to `["jobs.submitted"]` —
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
