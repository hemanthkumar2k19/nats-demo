# NATS Demo - Developer Guide

This guide provides developers with an overview of the NATS Platform Demo repository structure, architecture, and NATS capability mappings.

## 1. Project Purpose

The purpose of this project is to evaluate and demonstrate Core NATS and JetStream capabilities using a simple, clear, and easy-to-understand Go backend and React frontend workspace.

---

## 2. Repository Structure

```text
nats-demo/
├── AGENTS.md               # Coding rules and agent guidelines
├── README.md               # High-level overview and run instructions
├── docs/                   # Specifications and guides
│   ├── DEVELOPER_GUIDE.md  # This developer guide
│   ├── DEPLOYMENT_GUIDE.md # Run and deploy guide
│   ├── CHANGELOG.md        # Log of repository changes
│   └── *.md                # API and component specs
├── deploy/                 # Docker Compose and NATS configuration files
├── backend/                # Go Backend Service Workspace
│   └── src/
│       ├── cmd/            # Entry points (demo-service, processor-service)
│       └── internal/       # Common configurations, jobs, and messaging clients
└── frontend/               # React SPA Dashboard application
```

---

## 3. Service Responsibilities

### Backend Services
1. **Demo Service (`cmd/demo-service`)**: 
   - Serves the dashboard HTTP APIs.
   - Accepts jobs, validates payloads, and queries job statuses.
   - Publishes jobs to NATS (`jobs.submitted`).
   - Listens to wildcard lifecycle events (`jobs.*`) to update an in-memory job store.
2. **Processor Service (`cmd/processor-service`)**:
   - Background worker service simulating task execution.
   - Subscribes to `jobs.submitted` to receive jobs.
   - Publishes lifecycle events (`jobs.received`, `jobs.completed`, `jobs.failed`, etc.) to track processing.
   - Dynamically toggles processing state (ON/OFF) via control NATS subjects.

### Frontend
- **React SPA Dashboard (`frontend`)**:
   - Provides controls to submit jobs in Core NATS or JetStream mode.
   - Toggles the processor ON/OFF.
   - Inspects job histories, wildcard events, and subject addressing logs.

---

## 4. NATS Capability Mapping

| NATS Capability | Implementation Details |
| :--- | :--- |
| **Addressing (Wildcards)** | Demonstrates exact matching (`jobs.submitted`), single-level wildcard (`jobs.*`), and multi-level wildcard (`jobs.>`) routing. |
| **Transient Pub/Sub (Core NATS)** | Jobs sent via `CORE` delivery mode are not stored and are discarded if the processor is offline. |
| **Durable Streaming (JetStream)** | Jobs sent via `JETSTREAM` delivery mode are persisted in the `JOBS` stream, allowing offline processing. |
| **Competing Consumers** | Multiple processor instances can share the durable consumer `processor-durable` to load balance jobs. |
| **Request/Reply** | Sync job validation is processed on the subject `jobs.validate` with a requester timeout. |
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

---

## 6. How to Make Common Changes

### Adding a New Subject or Event Type
1. Define the NATS subject constant in `internal/messaging/subjects.go`.
2. Register the status mappings in `ProcessLifecycleEvent()` within `internal/jobs/service.go`.
3. If necessary, assign a logical weight in `getStatusWeight` in `service.go` to preserve correct chronological log sorting.
4. Update the badges or column handlers in the React component `ActivityPanel.tsx`.
