# NATS Platform Demo

This repository is a lightweight demonstration and evaluation workspace for NATS capabilities. It implements a simple Job Processing workflow using Go services and the official Go NATS client.

## Overview

The demo demonstrates NATS messaging capabilities (addressing, fire-and-forget pub/sub, competing consumers, and message tracking) using a Job Processing domain.

The workspace consists of:
*   **Demo Service**: An HTTP API that accepts jobs and publishes job events to NATS.
*   **Processor Service**: A background service that consumes job events from NATS and processes them.
*   **NATS Server**: Run via Docker Compose, configured with JetStream and WebSockets.
*   **NATS UI**: A web dashboard to monitor connections, subjects, and JetStream state.

---

## Architecture Diagram

```text
+----------------------+
|    Demo Service      |
|                      |
|  HTTP API (:8080)    |
|  NATS Publisher      |
+----------+-----------+
           |
     (jobs.submitted)
           |
           v
+----------------------+
|     NATS Broker      |
|                      |
|  JetStream Enabled   |
+----------+-----------+
           |
           v
+----------------------+
|  Processor Service   |
|                      |
|  NATS Consumer       |
+----------------------+
```

---

## Local Setup and Run Guide

### 1. Start NATS and NATS UI
Launch the NATS broker and NATS UI container using Docker Compose:

```bash
docker compose -f deploy/docker-compose.yaml up -d
```

*   **NATS Broker**: Available on `localhost:4222`
*   **NATS UI Dashboard**: Open http://localhost:3000 in your browser to inspect the cluster.

### 2. Configure Environment Variables
Copy the template configuration file in the `src` directory to initialize environment variables:

```bash
cp src/.env.example src/.env
```

The defaults inside `.env` are configured to connect to your local NATS container:
```env
PORT=8080
NATS_URL=nats://localhost:4222
```

### 3. Run the Demo Service
Start the HTTP API service:

```bash
cd src
go run cmd/demo-service/main.go
```
The HTTP API will begin listening on `:8080`.

### 4. Run the Processor Service
In a new terminal window, start the processing worker service:

```bash
cd src
go run cmd/processor-service/main.go
```

---

## How to Test the Demo

### 1. Submit a Job
Send a POST request to the Demo Service API using `curl`:

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -H "X-Correlation-Id: test-correlation-id-1" \
  -d '{"job_id": "job-100", "type": "demo", "payload": {"foo": "bar"}}'
```

### 2. Observe the Execution Flow

*   **Demo Service Terminal Logs**:
    You will see logs confirming configuration load, NATS connection initialization, and the HTTP server starting:
    ```text
    Starting demo-service...
    [Init] Loaded configuration: NATS_URL=nats://localhost:4222, PORT=8080
    [Init] Connected to NATS wrapper client
    [Run] HTTP server listening on :8080
    ```

*   **Processor Service Terminal Logs**:
    You will see the consumer service connect, subscribe, receive the message, and log its completion:
    ```text
    Initializing processor-service...
    [Init] Loaded configuration: NATS_URL=nats://localhost:4222
    [Init] Connected to NATS wrapper client
    Starting execution...
    [Run] Subscribed to subject: jobs.submitted
    [Consumer] Received message on subject: jobs.submitted | Job ID: job-100 | Correlation ID: test-correlation-id-1
    [Processor] Processing job job-100 of type demo
    [Processor] Successfully processed job job-100
    ```

*   **NATS UI Dashboard**:
    Open http://localhost:3000 to view active clients, connections, and message traffic statistics.
