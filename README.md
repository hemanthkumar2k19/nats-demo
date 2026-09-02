# NATS Platform Demo

This repository is a lightweight demonstration and evaluation workspace for NATS capabilities. It implements a simple Job Processing workflow using Go services and a React SPA dashboard.

## Overview

The demo demonstrates NATS messaging capabilities (addressing, fire-and-forget pub/sub, durable JetStream streaming, competing consumers, request/reply, and stream replay) using a Job Processing domain.

The workspace consists of:
*   **Job Service**: An HTTP API (`:8080`) that accepts jobs, validates payloads via Request/Reply, tracks job activity, and publishes events to NATS.
*   **Processor Service**: A background worker service that consumes jobs, handles validation requests, and publishes lifecycle events.
*   **Frontend Dashboard**: A React SPA developer dashboard (`:5173`) to submit jobs, trigger Request/Reply validation, toggle processor state, view replay, and inspect live NATS activity.
*   **NATS Server**: Run via Docker Compose or Podman Compose, configured with JetStream.
*   **NATS UI**: A web dashboard (`:3000`) to monitor connections, subjects, and JetStream state.

---

## Architecture Diagram

```text
+------------------------+       +------------------------+
|   React Dashboard      |       |  Processor Service     |
|   (http://localhost:5173) |       |                        |
+-----------+------------+       |  NATS Consumer         |
            |                    |  Request/Reply Handler |
            | HTTP               +-----------+------------+
            v                                |
+------------------------+                   |
|     Job Service        |                   |
|                        |                   |
|   HTTP API (:8080)     |                   |
|   NATS Publisher       |<------------------+
|   Lifecycle Observer   |    NATS Messages / Events
+-----------+------------+
            |
            v
+------------------------+
|      NATS Broker       |
|                        |
|   JetStream Enabled    |
+------------------------+
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
*   **NATS UI Dashboard**: http://localhost:3000

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
NATS_URL=nats://localhost:4222
```

#### Run Job Service (Terminal 1)
```bash
cd backend/src
go run cmd/job-service/main.go
```
The HTTP API will start listening on `:8080`.

#### Run Processor Service (Terminal 2)
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
2. **Submit Jobs**: Use the **Job Submission** panel to send jobs in `CORE` or `JETSTREAM` mode.
3. **Request/Reply**: Use the **Request / Reply** panel to validate jobs on `jobs.validate`.
4. **Demonstrate Timeout**: Toggle the Processor state to **OFF** in the Status panel, then send a validation request to observe a natural NATS timeout (HTTP 504) and `REQUEST_TIMEOUT` timeline entry.
5. **Replay & Addressing**: Use the **Stream Replay** and **Subject Addressing** panels to observe stream rewinds and wildcard routing.

---

### Option B: Using `curl` CLI

#### 1. Submit a Job (Fire-and-Forget / Core PubSub)
```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -H "X-Correlation-Id: test-corr-1" \
  -d '{"job_id": "job-100", "type": "image-processing", "payload": {"file": "image.jpg"}, "delivery_mode": "CORE"}'
```

#### 2. Validate a Job (Sync Request/Reply)
```bash
curl -X POST http://localhost:8080/jobs/validate \
  -H "Content-Type: application/json" \
  -H "X-Correlation-Id: test-corr-val-1" \
  -d '{"job_id": "job-val-100", "type": "image-processing", "payload": {"file": "image.jpg"}}'
```

