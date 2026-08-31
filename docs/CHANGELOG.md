# Changelog

All notable changes to this project will be documented in this file.

## 2026-08-31

### Added
- Implemented **NATS Subject Addressing Demo** feature displaying active subscriptions (`exact`, `single-level`, and `multi-level`) and observed message matching routing activity.
- Created `internal/messaging/observer.go` thread-safe store in the backend to record message delivery activity.
- Created NATS observer subscriptions on subjects `jobs.submitted` (`exact`), `jobs.*` (`single-level`), and `jobs.>` (`multi-level`) in `demo-service`.
- Added endpoints `GET /messaging/subscriptions` and `GET /messaging/activity` to serve active subscriptions and observed events.
- Created frontend `AddressingPanel.tsx` visual console widget displaying the NATS subscription patterns and routing activity.
- Added message routing events publication (e.g., `jobs.processing.started`, `jobs.processing.completed`, and `jobs.processing.failed`) in `processor-service`.

### Fixed
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
