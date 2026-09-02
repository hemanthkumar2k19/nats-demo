# NATS Demo - Deployment Guide

This guide describes how to deploy, configure, run, and verify the services of the **NATS Platform Demo**.

---

## 1. Required Runtime Components

The workspace relies on the following runtime components:
- **Go Runtime**: Version 1.22+ to run the backend services.
- **NodeJS & npm/pnpm**: To build and serve the React dashboard.
- **Docker & Docker Compose**: To orchestrate the NATS Broker and NATS UI dashboard locally.

---

## 2. NATS & Observability Infrastructure Deployment (Docker Compose)

The NATS message broker and observability stack are run via Docker.

### Configuration (`deploy/docker-compose.yaml`)
The compose configuration starts:
1. **NATS Server (`nats`)**: Runs the official NATS image, exposes port `4222` (client connection), port `8222` (monitoring server), and maps a local volume `nats-data` to `/data` for JetStream persistence.
2. **NATS Prometheus Exporter (`nats-exporter`)**: Exposes NATS monitoring metrics on port `7777` (`/metrics`), scraping NATS port `8222` with `-jsz`, `-connz`, `-subz`, `-varz`.
3. **Grafana OTEL-LGTM (`otel-lgtm`)**: Unified local observability stack combining OpenTelemetry Collector (`:4317` gRPC / `:4318` HTTP), Prometheus (`:9090`), Tempo (`:3200` / OTLP `:4317`), and Grafana (`:3000`). Pre-provisioned with the `NATS Platform Demo - Metrics` dashboard and Tempo distributed trace explorer.

### Command to Start:
From the project root directory, run:
```bash
docker compose -f deploy/docker-compose.yaml up -d
```

---

## 3. Backend Deployment

The backend consists of three Go services located under `backend/src`.

### Configuration (`backend/src/.env`)
The Go services load settings from environment variables or a local `.env` file. Initialize it by copying `.env.example`:
```bash
cp backend/src/.env.example backend/src/.env
```
Default parameters:
- `PORT=8080` (HTTP port for `demo-control-service`)
- `JOB_SERVICE_PORT=8081` (HTTP port for `job-service`)
- `NATS_URL=nats://localhost:4222` (connection string for NATS client)

### Starting Backend Services
In separate terminal windows, run the following commands:
1. **Demo Control Service** (UI Gateway & Observability Tap):
   ```bash
   cd backend/src
   go run cmd/demo-control-service/main.go
   ```
2. **Job Service** (Pure Business REST API):
   ```bash
   cd backend/src
   go run cmd/job-service/main.go
   ```
3. **Processor Service** (Worker Daemon):
   ```bash
   cd backend/src
   go run cmd/processor-service/main.go
   ```

---

## 4. Frontend Deployment

The React dashboard is run locally using Vite.

### Dependencies Installation
Install dependencies in the frontend workspace:
```bash
cd frontend
npm install
```

### Running Frontend Local Server
Start Vite in development mode:
```bash
npm run dev
```
The console will expose the local URL (e.g., `http://localhost:5173`). Open it in your web browser.

---

## 5. Basic Verification Steps

Once all services are running, verify the setup with these steps:

### Connection Checks
1. Access Grafana at `http://localhost:3000` (credentials: `admin` / `admin`) and confirm the NATS metrics dashboard and Tempo traces are operational.
2. Confirm the main header status bar in the React UI displays:
   - **NATS Server: Connected**
   - **Demo Control: Active (:8080)**
   - **Job Service: Active (:8081)**
   - **Processor Service: Active**

### Flow Verification
1. **Submit Core Job**: Submit a job in `CORE` mode. Ensure the activity log displays:
   - `PUBLISHED` -> `RECEIVED` -> `COMPLETED`
2. **Offline Durable Queue Check**:
   - Turn off the processor in the UI (Processor state shows `OFF`).
   - Submit a job in `JETSTREAM` mode.
   - Verify the stream pending count increases by 1.
   - Turn the processor back `ON`.
   - Verify the job is immediately pulled and processed, bringing the pending count back to 0.
3. **Request/Reply Validation**:
   - With Processor state `ON`, use the Request/Reply panel to send a validation request.
   - Verify the result status is `SUCCESS` (`valid: true`) and the interaction timeline shows:
     - `REQUEST_SENT` -> `REQUEST_RECEIVED` -> `REPLY_SENT` -> `REPLY_RECEIVED`
   - Turn Processor state `OFF` and send a validation request.
   - Verify after ~2 seconds the result status shows `TIMEOUT` (`No response received from processor service`), HTTP 504 is returned, and the interaction timeline records `REQUEST_SENT` -> `REQUEST_TIMEOUT`.
4. **Metrics Observability in Grafana**:
   - Access Grafana at `http://localhost:3000` (credentials: `admin` / `admin`).
   - Open the provisioned dashboard: `NATS Platform Demo - Metrics`.
   - Submit jobs and send validation requests in the UI; observe live metrics update across the 5 sections: NATS Health, Messaging, JetStream, Consumers, and JetStream Activity.
5. **Distributed Tracing in Tempo**:
   - Access Grafana Explore at `http://localhost:3000/explore`.
   - Select the `Tempo` data source.
   - Click **[ View in Tempo -> ]** from any inspected job in the Job Details Inspector in the React UI (or paste the Trace ID into the Tempo query bar).
   - Observe the full distributed trace waterfall spanning `job-service` HTTP handlers, NATS context-injected headers, and `processor-service` consumer and execution spans.
