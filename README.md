# NATS Platform Demo

This repository is a lightweight demonstration and evaluation workspace for NATS capabilities. It implements a Job Processing workflow where business logic is cleanly decoupled from demonstration and UI observability.

## Overview

The demo demonstrates NATS messaging capabilities (subject addressing, fire-and-forget pub/sub, durable JetStream streaming, competing consumers, request/reply, and stream replay) using a Job Processing domain.

The workspace consists of:
*   **Demo Control Service**: An HTTP API (`:8080`) that acts as the UI and observability gateway. It passively taps NATS events (`jobs.>`), exposes live activity logs, powers wildcard comparison tables, triggers JetStream replay, and controls processor/consumer settings.
*   **Job Service**: A pure business microservice (`:8081`) that accepts job submissions and domain validation requests over HTTP and publishes them to NATS. It contains no demo harness code.
*   **Processor Service**: A pure background worker daemon that consumes jobs from NATS / JetStream, handles validation requests, and publishes lifecycle events.
*   **Frontend Dashboard**: A React SPA developer dashboard (`:5173`) to submit jobs, trigger Request/Reply validation, toggle processor state, view replay, and inspect live NATS activity.
*   **NATS Server & Observability Stack**: Run via Docker Compose or Podman Compose. Provides NATS broker (`:4222`) configured with JetStream, Prometheus exporter (`:7777`), and Grafana OTEL-LGTM stack (`:3000`) for metrics and Tempo distributed traces.

---

## Architecture Diagram

```text
+-----------------------------------------------------------------------------------+
|                                 REACT DASHBOARD                                   |
|                             (http://localhost:5173)                               |
+-----------------------------------------------------------------------------------+
        |                                                   |
   Business Calls                                     Demo Inspection
  (POST /jobs, validate)                             (Status, Activities,
        |                                            Replay, Consumer Lab)
        v                                                   v
+-------------------------------+             +-------------------------------------+
|      Job Service (:8081)      |             |     Demo Control Service (:8080)    |
|  - Pure Business REST API     |             |  - UI Gateway & Status Scraper      |
|  - Domain Job Validation      |             |  - In-memory Activity Ring Buffer   |
|  - W3C OTEL Trace Injection   |             |  - Addressing Demo Observer (Exact, |
|  - Publishes to NATS          |             |    Single-level, Multi-level)       |
|  - NO demo harness code       |             |  - Ephemeral JetStream Replay Engine|
+-------------------------------+             +-------------------------------------+
        |                                                   |
     Publishes                                        Passive Taps &
     jobs.submitted                                    Wildcard Sub
     jobs.validate                                       jobs.>
        |                                                   |
        +-------------------------+-------------------------+
                                  |
                                  v
+-----------------------------------------------------------------------------------+
|                                   NATS SERVER                                     |
|  - Core NATS (Transient Pub/Sub & Request/Reply)                                  |
|  - JetStream (JOBS Stream, job-processor Consumer)                                |
+-----------------------------------------------------------------------------------+
                                  |
                              Delivers
                            jobs.submitted
                                  |
                                  v
+-----------------------------------------------------------------------------------+
|                               Processor Service                                   |
|  - Pure Worker Daemon (competing worker routines)                                 |
|  - Executes tasks, simulates workloads, ACKs / NACKs                              |
|  - Responds to jobs.validate                                                      |
|  - Subscribes to control.* over NATS for pause/resume and worker scaling          |
+-----------------------------------------------------------------------------------+
```

---

## Local Setup and Run Guide

### 1. Start NATS and NATS UI

Launch the NATS broker and NATS UI container using either **Docker Compose** or **Podman Compose**:

*   **Using Docker Compose**:
    ```bash
    docker compose -f deploy/docker-compose.yaml up -d
    ```

*   **Using Podman Compose**:
    ```bash
    podman-compose -f deploy/docker-compose.yaml up -d
    ```

Endpoints:
*   **NATS Broker**: `nats://localhost:4222`
*   **NATS Prometheus Exporter**: http://localhost:7777/metrics
*   **Grafana & Tempo Dashboard (OTEL-LGTM)**: http://localhost:3000 (admin / admin)

---

### 2. Start Backend Services

#### Configure Environment Variables
Copy the template configuration file in `backend/src`:

```bash
cp backend/src/.env.example backend/src/.env
```

The defaults in `.env`:
```env
PORT=8080
JOB_SERVICE_PORT=8081
NATS_URL=nats://localhost:4222
```

#### Run Demo Control Service (Terminal 1)
```bash
cd backend/src
go run cmd/demo-control-service/main.go
```
The demo control gateway will start listening on `:8080`.

#### Run Job Service (Terminal 2)
```bash
cd backend/src
go run cmd/job-service/main.go
```
The pure business Job Service will start listening on `:8081`.

#### Run Processor Service (Terminal 3)
```bash
cd backend/src
go run cmd/processor-service/main.go
```
The background processor worker will connect to NATS and begin handling jobs and validation requests.

---

### 3. Start Frontend Dashboard

In a new terminal window, start the React SPA dashboard:

```bash
cd frontend
npm install
npm run dev
```

Open http://localhost:5173 in your web browser to access the dashboard.

---

## How to Test the Demo

### Option A: Using the React Dashboard (Recommended)

1. Open http://localhost:5173 in your browser.
2. **Topology Visualizer**: Confirm all 4 components are displayed as active.
3. **Submit Jobs**: Use the **Job Submission** panel to send jobs in `CORE` or `JETSTREAM` mode (calls `POST http://localhost:8081/jobs`).
4. **Request/Reply**: Use the **Request / Reply** panel to validate jobs on `jobs.validate` (calls `POST http://localhost:8081/jobs/validate`).
5. **Demonstrate Timeout**: Toggle the Processor state to **OFF** in the Status panel, then send a validation request to observe a natural NATS timeout (HTTP 504) and `REQUEST_TIMEOUT` timeline entry.
6. **Replay & Addressing**: Use the **Stream Replay** and **Subject Addressing** panels to observe stream rewinds and wildcard routing.

---

### Option B: Using `curl` CLI

#### 1. Submit a Job (Calls Job Service :8081)
```bash
curl -X POST http://localhost:8081/jobs \
  -H "Content-Type: application/json" \
  -H "X-Correlation-Id: test-corr-1" \
  -d '{"job_id": "job-100", "type": "image-processing", "payload": {"file": "image.jpg"}, "delivery_mode": "CORE"}'
```

#### 2. Validate a Job (Calls Job Service :8081)
```bash
curl -X POST http://localhost:8081/jobs/validate \
  -H "Content-Type: application/json" \
  -H "X-Correlation-Id: test-corr-val-1" \
  -d '{"job_id": "job-val-100", "type": "image-processing", "payload": {"file": "image.jpg"}}'
```

#### 3. Inspect Demo Platform Status (Calls Demo Control :8080)
```bash
curl http://localhost:8080/status
```

#### 4. Fetch Live Activity Stream (Calls Demo Control :8080)
```bash
curl http://localhost:8080/activities
```
