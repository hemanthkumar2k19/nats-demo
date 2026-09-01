# NATS Demo - Developer Guide

This guide provides developers with an overview of the NATS Platform Demo repository structure, architecture, and NATS capability mappings.

## 1. Project Purpose

The purpose of this project is to evaluate and demonstrate Core NATS and JetStream capabilities using a simple, clear, and easy-to-understand Go backend and React frontend workspace.

---

## 2. Repository Structure

```text
nats-demo/
+-- AGENTS.md               # Coding rules and agent guidelines
+-- README.md               # High-level overview and run instructions
+-- docs/                   # Specifications and guides
|   +-- DEVELOPER_GUIDE.md  # This developer guide
|   +-- DEPLOYMENT_GUIDE.md # Run and deploy guide
|   +-- CHANGELOG.md        # Log of repository changes
|   +-- *.md                # API and component specs
+-- deploy/                 # Docker Compose and NATS configuration files
+-- backend/                # Go Backend Service Workspace
|   +-- src/
|       +-- cmd/            # Entry points (demo-service, processor-service)
|       +-- internal/       # Common configurations, jobs, and messaging clients
+-- frontend/               # React SPA Dashboard application
```

---

## 3. Service Responsibilities

### Backend Services
1. **Demo Service (`cmd/demo-service`)**:
   - Serves the dashboard HTTP APIs.
   - Accepts jobs, validates payloads, and queries job statuses.
   - Publishes jobs to NATS (`jobs.submitted`).
   - Listens to wildcard lifecycle events (`jobs.>`) to update an in-memory job store.
2. **Processor Service (`cmd/processor-service`)**:
   - Background worker service simulating task execution.
   - Subscribes to `jobs.submitted` to receive jobs.
   - Subscribes to `jobs.validate` to answer validation requests.
   - Publishes lifecycle events (`jobs.received`, `jobs.completed`, `jobs.failed`,
     `jobs.request.received`, `jobs.reply.sent`, etc.) to track processing.
   - Dynamically toggles processing state (ON/OFF) via control NATS subjects.

### Frontend
- **React SPA Dashboard (`frontend`)**:
   - Provides controls to submit jobs in Core NATS or JetStream mode.
   - Provides a dedicated Request/Reply panel to send validation requests and observe responses.
   - Toggles the processor ON/OFF.
   - Inspects job histories, wildcard events, and subject addressing logs.

---

## 4. NATS Capability Mapping

| NATS Capability | Implementation Details |
| :--- | :--- |
| **Addressing (Wildcards)** | Demonstrates exact matching (`jobs.submitted`), single-level wildcard (`jobs.*`), and multi-level wildcard (`jobs.>`) routing. |
| **Transient Pub/Sub (Core NATS)** | Jobs sent via `CORE` delivery mode are not stored and are discarded if the processor is offline. |
| **Durable Streaming (JetStream)** | Jobs sent via `JETSTREAM` delivery mode are persisted in the `JOBS` stream, allowing offline processing. |
| **Consumer Groups / Competing Consumers** | Multiple processor workers (`processor-1`, `processor-2`) pull from the same stream to balance workloads. |
| **Durable vs Ephemeral Consumers** | Supports durable (`job-processor`) and dynamic ephemeral pull consumers configured via Consumer Lab. |
| **Ordering** | Ordered consumer demonstration ensuring message delivery order follows stream sequence. |
| **At-Least-Once & Redelivery** | Failure simulation triggers `msg.Nak()`, causing JetStream to redeliver with incremented delivery counts. |
| **JetStream Deduplication** | Publishes with `Nats-Msg-Id` within the 2-minute deduplication window recognize duplicates (`DEDUPLICATED`). |
| **Request/Reply** | Sync job validation is processed on the subject `jobs.validate` with a 2-second requester timeout. |
| **Replay / Rewind** | Replays historical stream events from the `JOBS` stream based on sequence number or time constraints. |

---

## 5. Important Implementation Concepts

### Durable Consumer Lifecycle
To prevent the client library from deleting the durable consumer on shutdown, the consumer is created explicitly on NATS during initialization (`AddConsumer`), and the subscriber binds to it explicitly using:
```go
js.PullSubscribe(subject, "processor-durable", nats.Bind("JOBS", "processor-durable"))
```

### Stable Dashboard Log Ordering
Events happening within the same second are sorted by logical state sequence (`PUBLISHED` -> `RECEIVED` -> `COMPLETED`) and grouped by `JobID` in the backend before being sent to the UI.

### Request/Reply Lifecycle Flow

```text
UI                 demo-service          NATS          processor-service
|                       |                 |                    |
| POST /jobs/validate   |                 |                    |
|---------------------->|                 |                    |
|                       | RequestMsg      |                    |
|                       |-- jobs.validate -->                  |
|                       |                 | Deliver            |
|                       |                 |-- jobs.validate -->|
|                       |                 |                    | Publish jobs.request.received
|                       |<-- jobs.request.received (via jobs.>) |
|                       |                 |                    | Respond
|                       |                 |<-- Reply ----------|
|                       |                 |                    | Publish jobs.reply.sent
|                       |<-- jobs.reply.sent (via jobs.>)     |
|                       | Reply received  |                    |
| 200 OK                |                 |                    |
|<----------------------|                 |                    |
```

**Correlation ID propagation**:
- The HTTP client sends (or the backend generates) a `X-Correlation-Id` header.
- `demo-service` includes it in the NATS `RequestMsg` headers.
- `processor-service` extracts it and forwards it in the `jobs.request.received` and `jobs.reply.sent` lifecycle events.
- All activity log entries carry the same correlation ID so the full flow can be traced.

**Natural timeout when Processor is OFF**:
- When the processor state is toggled to `OFF`, it calls `unsubscribeValidation()`, which removes the `jobs.validate` subscriber.
- The NATS `RequestMsg` in `demo-service` finds no active responder and returns `nats.ErrNoResponders` (or `nats.ErrTimeout` after 2 seconds).
- `publisher.RequestJobValidation` converts this to `messaging.ErrRequestTimeout`.
- The HTTP handler detects this sentinel error and returns HTTP `504 Gateway Timeout`.
- No artificial timeout generation is used; the behavior is an authentic NATS timeout.

---

## 6. How to Make Common Changes

### Adding a New Subject or Event Type
1. Define the NATS subject constant in `internal/messaging/subjects.go`.
2. Register the status mappings in `ProcessLifecycleEvent()` within `internal/jobs/service.go`.
3. If necessary, assign a logical weight in `getStatusWeight` in `service.go` to preserve correct chronological log sorting.
4. Update the badges or column handlers in the React component `ActivityPanel.tsx`.

### Extending Request/Reply with a New Subject
1. Add the subject constant in `internal/messaging/subjects.go`.
2. Add a `Subscribe<Name>` method in `internal/messaging/consumer.go` following the `SubscribeJobValidate` pattern.
3. Add a `Request<Name>` method in `internal/messaging/publisher.go` following `RequestJobValidation`.
4. Add the corresponding `subscribe<Name>` / `unsubscribe<Name>` lifecycle methods in `processor-service/main.go`.
5. Wire the new API endpoint in `api/http/handler.go` and register it in `api/http/routes.go`.

