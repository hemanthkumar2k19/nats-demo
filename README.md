# NATS Platform Demo

A production-pattern evaluation playground and interactive demonstration platform for **Core NATS** and **JetStream**, built with **Go** microservices and a modern **React SPA dashboard**.

---

## 1. Overview

This repository demonstrates how to architect, operate, and observe real-world messaging patterns with NATS. It uses a **Job Processing & Distributed Order Fulfillment** domain to showcase both foundational messaging and advanced event-driven orchestration without production complexity.

### Key Highlights
*   **Zero Demo Intrusion in Business Code**: The business microservice (`job-service`) contains zero demo harness or mock code. All live activity tracking, wildcard taps, and test controls are handled through a dedicated gateway (`demo-control-service`).
*   **Dual Engine Evaluation**: Compare transient, in-memory **Core NATS** pub/sub side-by-side with durable, persistent **JetStream** streaming.
*   **Interactive Capability Studio**: Test 7 distinct NATS patterns directly from the browser with single-click demo presets and instant visual feedback.
*   **Observable by Design**: Features a real-time Activity Log with a **Message Classification Switcher** (`Business Messages [A]` vs `Flow/Lifecycle Events [B]`), Modal Job Inspector, and full Grafana LGTM stack (Prometheus, Loki, Tempo) with W3C OpenTelemetry distributed tracing.

---

## 2. NATS Capabilities Demonstrated

| Capability | Pattern | Real-World Scenario | How to Observe in Demo |
| :--- | :--- | :--- | :--- |
| **Core NATS Pub/Sub** | Fire-and-Forget | Fast telemetry, metrics, transient broadcast | Submit in `CORE` mode; message drops if worker is offline. |
| **Subject Wildcards** | Subject Routing | Tiered routing (`jobs.*`, `jobs.>`) | Subject Addressing observer compares single vs multi-level wildcards. |
| **Queue Groups** | Server-Side Load Balancing | High-throughput worker distribution | Core NATS distributes `jobs.queue` across worker pool with zero broker state. |
| **JetStream Streaming** | Persistent Storage | Critical business transactions & audits | Stream `JOBS` stores messages on disk; workers process backlog when restarted. |
| **Competing Consumers** | Stream Pull Consumers | Scalable background processing | Multiple worker routines pull and compete for tasks on stream `JOBS`. |
| **Request / Reply** | Synchronous RPC over NATS | Instant payload validation | `POST /jobs/validate` requests validation over `jobs.validate` with timeout fallback. |
| **Deduplication** | Sliding Window Idempotency | Network retry de-duping | Publishes with duplicate `Nats-Msg-Id` are suppressed by JetStream. |
| **Delayed & Retries** | Backoff & Redelivery | Rate-limited external APIs, transient errors | Test `msg.NakWithDelay` backoff, `AckWait` timeout recovery, and scheduled timers. |
| **Dead Letter Queue (DLQ)**| Poison Message Isolation | Unrecoverable job handling | Repeated failures route to `JOBS_DLQ` after 3 attempts, leaving main stream healthy. |
| **Stream Replay** | Event Sourcing & Audit | Disaster recovery, historical reprocessing | Ephemeral consumer rewinds stream events by sequence range or time window. |
| **Distributed Saga** | Multi-step Orchestration | Multi-service order fulfillment | 2-Op workflow (`Reserve Inventory` -> `Process Payment` -> `Completed`) with automatic compensating rollback (`Release Inventory`) on failure. |
| **Observability (LGTM)** | Full-Stack Telemetry | Production monitoring | NATS Prometheus metrics, Loki server logs, and Tempo distributed traces. |

---

## 3. System Architecture

```text
+-----------------------------------------------------------------------------------+
|                                 REACT SPA DASHBOARD                               |
|                             (http://localhost:5173)                               |
+-----------------------------------------------------------------------------------+
        |                                                   |
   Domain Calls                                       Demo Inspection
  (POST /jobs, validate)                             (Status, Activities,
        |                                            Replay, Consumer Lab, Saga)
        v                                                   v
+-------------------------------+             +-------------------------------------+
|      Job Service (:8081)      |             |     Demo Control Service (:8080)    |
|  - Pure Business REST API     |             |  - UI Gateway & Activity Ring Buffer|
|  - Domain Job Validation      |             |  - Passive Wildcard Tap (jobs.>,    |
|  - W3C OTEL Trace Context     |             |    saga.>)                          |
|  - Publishes to NATS          |             |  - Ephemeral JetStream Replay Engine|
|  - Zero Demo Harness Code     |             |  - Saga Orchestrator Coordinator    |
+-------------------------------+             +-------------------------------------+
        |                                                   |
     Publishes                                        Passive Taps &
     jobs.submitted                                   Control Subjects
     jobs.validate                                          |
        |                                                   |
        +-------------------------+-------------------------+
                                  |
                                  v
+-----------------------------------------------------------------------------------+
|                                   NATS BROKER                                     |
|  - Core NATS Engine (Transient Pub/Sub, Queue Groups, Request/Reply, Wildcards)   |
|  - JetStream Engine (JOBS Stream, JOBS_DLQ Stream, Durable & Ephemeral Consumers) |
+-----------------------------------------------------------------------------------+
                                  |
                              Delivers
                             jobs.submitted
                             jobs.queue
                             saga.*
                                  |
                                  v
+-----------------------------------------------------------------------------------+
|                             Processor Service Workers                             |
|  - Background Worker Pool (competing consumer routines)                           |
|  - Executes task logic, handles ACKs, NAKs, and NAK with delay                    |
|  - Handles Request/Reply validation on jobs.validate                              |
|  - Responds to Saga step commands and executes compensating release actions       |
|  - Toggled dynamically (ON/OFF) via control subjects for failure simulation       |
+-----------------------------------------------------------------------------------+
```

---

## 4. Quickstart: Up and Running in 3 Minutes

### Prerequisites
*   **Docker & Docker Compose** (or Podman)
*   **Go**: 1.22+
*   **Node.js**: 18+ & npm

---

### Step 1: Start NATS & Observability Stack

Launch the NATS message broker and pre-configured Grafana LGTM stack:

```bash
docker compose -f deploy/docker-compose.yaml up -d
```

Verify service availability:
*   **NATS Broker**: `nats://localhost:4222`
*   **NATS Prometheus Exporter**: http://localhost:7777/metrics
*   **Grafana Dashboard (LGTM Stack)**: http://localhost:3000 (Credentials: `admin` / `admin`)

---

### Step 2: Start Backend Microservices

Copy the environment configuration template:
```bash
cp backend/src/.env.example backend/src/.env
```

Open three terminal windows to launch the backend services:

**Terminal 1 -- Demo Control Service (UI Gateway & Observability Hub)**:
```bash
cd backend/src
go run cmd/demo-control-service/main.go
```
*Listening on port `:8080`.*

**Terminal 2 -- Job Service (Pure Business REST API)**:
```bash
cd backend/src
go run cmd/job-service/main.go
```
*Listening on port `:8081`.*

**Terminal 3 -- Processor Service (Worker Daemon Pool)**:
```bash
cd backend/src
go run cmd/processor-service/main.go
```
*Connected to NATS, listening on `jobs.submitted`, `jobs.validate`, and `saga.*`.*

---

### Step 3: Start the React Frontend Dashboard

In a new terminal window:
```bash
cd frontend
npm install
npm run dev
```

Open **http://localhost:5173** in your web browser.

---

## 5. Endpoints & Ports Reference

| Component | Port | Description | URL / Protocol |
| :--- | :--- | :--- | :--- |
| **React Dashboard** | `5173` | Interactive demo UI | http://localhost:5173 |
| **Demo Control Service** | `8080` | Observability gateway & activity stream | http://localhost:8080 |
| **Job Service** | `8081` | Pure business REST API | http://localhost:8081 |
| **NATS Server** | `4222` | NATS client protocol | `nats://localhost:4222` |
| **NATS Monitoring** | `8222` | Internal NATS server statistics | http://localhost:8222 |
| **NATS Exporter** | `7777` | Prometheus metrics surface | http://localhost:7777/metrics |
| **Grafana LGTM Stack** | `3000` | Dashboards, logs, and traces | http://localhost:3000 |
| **Tempo (OTEL Tracing)**| `4317`/`4318` | OpenTelemetry gRPC / HTTP ingest | http://localhost:3200 |

---

## 6. Interactive Guided Tour

Once the dashboard is open at **http://localhost:5173**, explore these key demonstrations:

### Tour 1: Core NATS vs JetStream Persistence
1. Navigate to the **Pub/Sub & Stream** tab.
2. In the top Topology panel, toggle the Processor state to **OFF**.
3. Submit a job in `CORE` mode -> Notice the message drops because Core NATS is fire-and-forget.
4. Submit a job in `JETSTREAM` mode -> Notice the message enters the `JOBS` stream and stays `STORED`.
5. Toggle Processor back to **ON** -> Watch the worker pull and process the queued JetStream job immediately.

### Tour 2: Request / Reply Synchronous RPC & Timeouts
1. Switch to the **Request / Reply** tab.
2. Click **Validate Job** -> Observe an instantaneous response (`HTTP 200`, validation result).
3. Toggle the Processor state to **OFF** and click **Validate Job** again.
4. Observe a natural NATS Request timeout (`HTTP 504`) after 2 seconds with zero hanging goroutines.

### Tour 3: Server-Side Queue Groups
1. Switch to the **Queue Groups** tab.
2. Click **Publish to Queue Group** multiple times (or click **Dispatch 5x Burst**).
3. Watch Core NATS balance deliveries evenly across competing worker threads (`Worker-1` and `Worker-2`) with zero JetStream consumer configuration.

### Tour 4: Distributed Saga & Compensating Rollback
1. Switch to the **Saga** tab in Capability Studio.
2. Click **Normal Success Flow** -> Watch `Op 1 (Reserve Inventory)` complete, followed by `Op 2 (Process Payment)`, reaching `COMPLETED`.
3. Click **Payment Declined (Triggers Rollback)** -> Watch Op 1 succeed, then Op 2 fail with a declined card. Observe the orchestrator immediately trigger compensating action: `saga.op1.compensate` releasing the reserved inventory.

### Tour 5: Activity Log Classification Switcher
1. In the bottom **Live Activity Log**, observe the 3-way toggle switcher:
   *   **All Messages**: Displays both business domain payloads and worker lifecycle telemetry.
   *   **Business Messages (A)**: Isolates actual domain commands, submissions, and Saga order transactions (`jobs.submitted`, `jobs.queue`, `saga.*`).
   *   **Flow / Lifecycle Events (B)**: Isolates internal worker telemetry (`RECEIVED`, `ACKED`, `PROCESSING`, `STORED`, `DLQ_PUBLISHED`).
2. Click on any row to open the **Modal Job Inspector** and view headers, W3C trace context, and raw payload data.

---

## 7. CLI Testing with `curl`

Prefer working from the terminal? All capabilities can be triggered via standard HTTP calls:

```bash
# 1. Submit a job to Job Service (:8081)
curl -X POST http://localhost:8081/jobs \
  -H "Content-Type: application/json" \
  -d '{"job_id": "job-cli-1", "type": "report-generation", "payload": {"file": "annual.pdf"}, "delivery_mode": "JETSTREAM"}'

# 2. Execute a synchronous Request/Reply validation (:8081)
curl -X POST http://localhost:8081/jobs/validate \
  -H "Content-Type: application/json" \
  -d '{"job_id": "val-cli-1", "type": "report-generation", "payload": {"file": "annual.pdf"}}'

# 3. Trigger a distributed Saga order workflow (:8080)
curl -X POST http://localhost:8080/sagas/start \
  -H "Content-Type: application/json" \
  -d '{"job_id": "order-cli-1", "item_id": "ITEM-1", "amount": 199.99, "auto_advance": true}'

# 4. Read the live activity stream (:8080)
curl http://localhost:8080/activities

# 5. Check cluster status (:8080)
curl http://localhost:8080/status
```

---

## 8. Documentation Roadmap

For in-depth architectural specifications and implementation guides, refer to:

*   [Developer Guide](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/docs/DEVELOPER_GUIDE.md): Architecture patterns, code walkthrough, service boundaries, and coding conventions.
*   [Deployment Guide](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/docs/DEPLOYMENT_GUIDE.md): Docker Compose topology, environment variables, production deployment considerations.
*   [Functional Testing Guide](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/docs/FUNCTIONAL_TESTING_GUIDE.md): Step-by-step verification checklists for every test scenario.
*   [Changelog](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/docs/CHANGELOG.md): Historical record of features, enhancements, and fixes.
*   [Coding Rules & Agent Guidelines](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/nats-demo/AGENTS.md): Repository principles, simplicity guidelines, and constraints.
