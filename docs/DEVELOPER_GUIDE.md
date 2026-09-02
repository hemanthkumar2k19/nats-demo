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
|   +-- DEVELOPER_GUIDE.md          # This developer guide
|   +-- DEPLOYMENT_GUIDE.md         # Run and deploy guide
|   +-- FUNCTIONAL_TESTING_GUIDE.md # Hands-on testing & evaluation scenarios
|   +-- CHANGELOG.md                # Log of repository changes
|   +-- *.md                # API and component specs
+-- deploy/                 # Docker Compose and NATS configuration files
+-- backend/                # Go Backend Service Workspace
|   +-- src/
|       +-- cmd/            # Entry points (demo-control-service, job-service, processor-service)
|       +-- internal/       # Common configurations, jobs, and messaging clients
+-- frontend/               # React SPA Dashboard application
```

---

## 3. Service Responsibilities

### Backend Services
1. **Job Service (`cmd/job-service`)**:
   - Pure business microservice listening on `:8081`.
   - Accepts job submissions (`POST /jobs`), validation requests (`POST /jobs/validate`), and job status queries (`GET /jobs`).
   - Publishes jobs to NATS (`jobs.submitted`).
   - Injects standard W3C OpenTelemetry trace context into message headers.
   - Contains zero demo harness code and zero in-memory activity ring buffers.
2. **Demo Control Service (`cmd/demo-control-service`)**:
   - Dedicated UI gateway and observability harness listening on `:8080`.
   - Passively taps NATS lifecycle events (`jobs.>`) to maintain the live activity stream.
   - Powers the wildcard subject addressing comparison (`Observer`).
   - Manages ephemeral JetStream replay consumers (`POST /jobs/replay`).
   - Relays processor state and consumer lab configuration changes over NATS.
3. **Processor Service (`cmd/processor-service`)**:
   - Background worker service simulating task execution.
   - Subscribes to `jobs.submitted` to receive jobs.
   - Subscribes to `jobs.validate` to answer validation requests.
   - Publishes lifecycle events (`jobs.received`, `jobs.completed`, `jobs.failed`,
     `jobs.request.received`, `jobs.reply.sent`, etc.) to track processing.
   - Dynamically toggles processing state (ON/OFF) via control NATS subjects.

### Frontend
- **React SPA Dashboard (`frontend`)**:
   - Displays interactive **Current Demo Setup** pairing runtime topology on the left with embedded **Consumer Lab** controls and live metrics on the right.
   - Accurately visualizes a two-tier architecture: Tier 1 with **React UI** and **Demo Control Service** connected to Tier 2 with **Job Service**, **NATS Server**, and **Processor Service**, clearly distinguishing deployed runtime components from internal NATS/JetStream resources (`JOBS Stream`, `job-processor Consumer`) and worker pool routines (`processor-1`, `processor-2`).
   - Provides contextual **NATS Information** popovers via `(i)` indicators across all sections explaining core NATS concepts, usage, and trivia.
   - Provides controls to publish jobs in Core NATS or JetStream mode (`JobPanel` - Pub Sub).
   - Provides a dedicated Message Deduplication panel (`DeduplicationPanel`) displaying stream deduplication parameters (`2m` window, `Nats-Msg-Id`), NATS docs knowledge, and live duplicate testing.
   - Provides a dedicated Request/Reply panel to send validation requests and observe responses.
   - Configures JetStream consumers (Durable vs Ephemeral, 1 or 2 Workers, Normal vs Ordered) directly within Demo Setup.
   - Toggles the processor ON/OFF directly from the global status bar.
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
| **Replay / Rewind** | Replays historical stream events from the `JOBS` stream based on sequence number or time constraints via an ephemeral consumer without modifying stored stream entries. |
| **Metrics Observability** | OpenTelemetry OTLP metrics exported to Grafana OTEL-LGTM stack alongside NATS Prometheus Exporter infrastructure metrics. |
| **Distributed Tracing** | End-to-end W3C trace context propagation (`traceparent`) via NATS headers with OpenTelemetry spans visualized in Tempo. |
| **Dead Letter Queue (DLQ)** | Demonstrates application-level DLQ routing on JetStream: messages failing repeatedly are NAKed until reaching `max_delivery_attempts` (default: 3), then routed to stream `JOBS_DLQ` (`jobs.dlq`), emitting `DLQ_PUBLISHED` and inspected by consumer `dlq-inspector`. |

---

## 5. Important Implementation Concepts

### Dead Letter Queue (DLQ) Pattern Flow
- NATS does not require DLQ to be a special server-side component; DLQ is an application-level architectural pattern built using JetStream primitives (persistence, NAK, redelivery counters, stream routing).
- When a JetStream job fails processing, the worker issues `msg.Nak()`. JetStream increments `NumDelivered` and redelivers the message.
- Once `NumDelivered` reaches `max_delivery_attempts` (default: 3), the worker isolates the failed message into stream `JOBS_DLQ` on subject `jobs.dlq`, publishes `jobs.dlq.published` (status `DLQ_PUBLISHED`), and issues `msg.Ack()` on the original `JOBS` stream message to halt redeliveries.
- Consumer `dlq-inspector` monitors `JOBS_DLQ`, allowing administrators to inspect failed messages without altering the primary business stream.

### Durable Consumer Lifecycle
To prevent the client library from deleting the durable consumer on shutdown, the consumer is created explicitly on NATS during initialization (`AddConsumer`), and the subscriber binds to it explicitly using:
```go
js.PullSubscribe(subject, "processor-durable", nats.Bind("JOBS", "processor-durable"))
```

### Stable Dashboard Log Ordering
Events happening within the same second are sorted by logical state sequence (`PUBLISHED` -> `RECEIVED` -> `COMPLETED`) and grouped by `JobID` in the backend before being sent to the UI.

### JetStream Replay Flow
- The dashboard triggers `POST /jobs/replay` with sequence range (`start_sequence`, `end_sequence`) or time window (`start_time`, `end_time`), and replay mode (`instant` or `original`).
- `job-service` creates an ephemeral push consumer on the `JOBS` stream configured with `DeliverByStartSequencePolicy` or `DeliverByStartTimePolicy`, and `ReplayInstantPolicy` or `ReplayOriginalPolicy`.
- Delivered historical messages are published to `jobs.replayed` with status `REPLAYED` so the Activity Log records them.
- Original stream messages remain immutable and untouched; the ephemeral consumer is cleanly torn down after completion.

### Request/Reply Lifecycle Flow

```text
UI                 job-service           NATS          processor-service
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
- `job-service` includes it in the NATS `RequestMsg` headers.
- `processor-service` extracts it and forwards it in the `jobs.request.received` and `jobs.reply.sent` lifecycle events.
- All activity log entries carry the same correlation ID so the full flow can be traced.

**Natural timeout when Processor is OFF**:
- When the processor state is toggled to `OFF`, it calls `unsubscribeValidation()`, which removes the `jobs.validate` subscriber.
- The NATS `RequestMsg` in `job-service` finds no active responder and returns `nats.ErrNoResponders` (or `nats.ErrTimeout` after 2 seconds).
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

---

## 7. OpenTelemetry Distributed Tracing & Telemetry

### Trace Propagation across NATS
Distributed traces use the standard W3C `traceparent` HTTP header format (`00-<trace-id>-<span-id>-01`), carried natively inside `nats.Msg.Header` (`map[string][]string`).

```text
React UI              job-service                 NATS Server              processor-service
   |                       |                           |                           |
   | POST /jobs            |                           |                           |
   |---------------------->| Span: POST /jobs (Server) |                           |
   |                       | Span: NATS Publish (Prod) |                           |
   |                       | Inject traceparent        |                           |
   |                       |-- Publish msg + header -->|                           |
   |                       |                           |-- Deliver msg + header -->|
   |                       |                           |                           | Extract traceparent
   |                       |                           |                           | Span: Consumer Receive (Cons)
   |                       |                           |                           | Span: Process Job (Internal)
   |                       |                           |                           | Ack / Nak
   | 202 Accepted          |                           |                           |
   |<----------------------|                           |                           |
```

### Trace Span Hierarchy
1. **Asynchronous Job Processing**:
   - `POST /jobs` (`SpanKindServer`): Root span created by HTTP handler in `job-service`.
     - `NATS Publish jobs.submitted` (`SpanKindProducer`): Child span created by `Publisher.PublishJobSubmitted`.
       - `Consumer Receive` (`SpanKindConsumer`): Span created upon message delivery in `processor-service`.
         - `Process Job` (`SpanKindInternal`): Child span representing job execution. Records errors and failure status if simulation fails.
2. **Synchronous Validation RPC (Request/Reply)**:
   - `POST /jobs/validate` (`SpanKindServer`): Root span in `job-service`.
     - `NATS Request jobs.validate` (`SpanKindClient`): Child span in `job-service` covering the 2-second timeout window.
       - `Process Validation Request` (`SpanKindServer`): Span in `processor-service` validating parameters.
         - `NATS Reply` (`SpanKindProducer`): Response span dispatched back to requester inbox.

### Central Telemetry Package (`internal/telemetry`)
- `Init(ctx, serviceName, otelEndpoint, insecure)`: Initializes both `TracerProvider` and `MeterProvider` targeting the OTLP gRPC collector.
- `InjectTraceContext(ctx, header)`: Injects active span context into `nats.Header`.
- `ExtractTraceContext(ctx, header)`: Extracts span context from `nats.Header` to become parent context for consumer spans.
- `StartSpan(ctx, name, opts...)`: Convenience wrapper around OpenTelemetry tracer.


