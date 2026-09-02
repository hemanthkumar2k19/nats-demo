# NATS Platform Demo - API Specification

This document details the HTTP endpoints and NATS subject/message contracts for the **NATS Platform Demo**. The system consists of two services:
1. **Job Service**: Exposes an HTTP API for job submission, validation, status checks, and replay triggers.
2. **Processor Service**: Consumes jobs, validates requests, processes jobs (with redelivery/deduplication support), and publishes lifecycle events.

---

## 1. HTTP API (Job Service)

All endpoints below are served by the **Job Service** (defaulting to `http://localhost:8080`).

### 1.1. Submit Job
Submit a job for processing. This is a **fire-and-forget** operation. The Job Service publishes a NATS message and immediately returns `202 Accepted`.

* **Endpoint**: `POST /jobs`
* **Content-Type**: `application/json`
* **Request Headers**:
  * `X-Correlation-Id`: (Optional) A unique string for tracing. If not provided, the Job Service will generate one.
* **Request Body**:
  ```json
  {
    "job_id": "job-101",
    "type": "image-processing",
    "payload": {
      "file": "image-101.jpg",
      "simulate_failure": false,
      "simulate_failure_count": 0
    },
    "delivery_mode": "CORE"
  }
  ```
  *(Note: `simulate_failure` and `simulate_failure_count` can be passed in the payload to test NACKs and redeliveries in Phase 7).*
* **Response**: `202 Accepted`
* **Response Body**:
  ```json
  {
    "job_id": "job-101",
    "status": "SUBMITTED",
    "correlation_id": "8f32a1c2",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  }
  ```
* **NATS Action**: Publishes message to subject `jobs.submitted`.

---

### 1.2. Validate Job
Synchronously validates a job configuration using NATS **Request/Reply**. The Job Service blocks waiting for the Processor Service's response.

* **Endpoint**: `POST /jobs/validate`
* **Content-Type**: `application/json`
* **Request Headers**:
  * `X-Correlation-Id`: (Optional) Custom correlation ID string for tracing.
* **Request Body**:
  ```json
  {
    "job_id": "job-101",
    "type": "image-processing",
    "payload": {
      "file": "image-101.jpg"
    }
  }
  ```
* **Response (Success)**: `200 OK`
* **Response Body**:
  ```json
  {
    "valid": true,
    "message": "Job configuration is valid.",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
  }
  ```
* **Response (Timeout - Processor OFF)**: `504 Gateway Timeout`
* **Response Body**:
  ```json
  {
    "error": "request timed out",
    "message": "No response received from processor service"
  }
  ```
* **NATS Action**: Sends a request to subject `jobs.validate` with a 2-second timeout. If the Processor is OFF, the handler declines to respond and `POST /jobs/validate` returns `504 Gateway Timeout`.

---

### 1.3. Get Job Status
Retrieve the status of a specific job. Initially, this status is tracked in-memory by the Job Service by listening to NATS lifecycle events.

* **Endpoint**: `GET /jobs/{job_id}`
* **Response**: `200 OK` (if found) or `404 Not Found`
* **Response Body**:
  ```json
  {
    "job_id": "job-101",
    "status": "COMPLETED",
    "delivery_count": 1,
    "correlation_id": "8f32a1c2",
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    "history": [
      {
        "status": "SUBMITTED",
        "timestamp": "2026-08-31T12:00:00Z"
      },
      {
        "status": "PROCESSING",
        "timestamp": "2026-08-31T12:00:01Z"
      },
      {
        "status": "COMPLETED",
        "timestamp": "2026-08-31T12:00:03Z"
      }
    ]
  }
  ```

---

### 1.4. List Jobs
List all jobs currently stored in the Job Service's in-memory store. Essential for observing overall progress during a demo.

* **Endpoint**: `GET /jobs`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  [
    {
      "job_id": "job-101",
      "status": "COMPLETED",
      "delivery_count": 1
    },
    {
      "job_id": "job-102",
      "status": "PROCESSING",
      "delivery_count": 2
    }
  ]
  ```

---

### 1.5. Replay Jobs
Trigger a JetStream replay. The Job Service creates an ephemeral replay consumer to replay historical events from the stream.

* **Endpoint**: `POST /jobs/replay`
* **Content-Type**: `application/json`
* **Request Body** (Option A: Sequence-based):
  ```json
  {
    "from_sequence": 100,
    "to_sequence": 120
  }
  ```
  *(or Option B: Time-based)*:
  ```json
  {
    "from_time": "2026-08-31T10:00:00Z",
    "to_time": "2026-08-31T11:00:00Z"
  }
  ```
* **Response**: `202 Accepted`
* **Response Body**:
  ```json
  {
    "status": "REPLAY_STARTED",
    "consumer": "job-replay-001"
  }
  ```

---

### 1.6. Get Subscriptions
Retrieve the active subscriptions configured for demonstrating NATS subject addressing.

* **Endpoint**: `GET /messaging/subscriptions`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  {
    "subscriptions": [
      {
        "name": "exact",
        "subject": "jobs.submitted"
      },
      {
        "name": "single-level",
        "subject": "jobs.*"
      },
      {
        "name": "multi-level",
        "subject": "jobs.>"
      }
    ]
  }
  ```

---

### 1.7. Get Addressing Activity
Retrieve the observed routing activity displaying which subscriptions received each message.

* **Endpoint**: `GET /messaging/activity`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  {
    "events": [
      {
        "subject": "jobs.submitted",
        "received_by": [
          "exact",
          "single-level",
          "multi-level"
        ],
        "timestamp": "2026-08-31T10:30:00Z"
      },
      {
        "subject": "jobs.processing.started",
        "received_by": [
          "multi-level"
        ],
        "timestamp": "2026-08-31T10:30:01Z"
      }
    ]
  }
  ```

---

### 1.8. Get Activities
Retrieve flat chronological NATS activity logs for the dashboard.
* **Endpoint**: `GET /activities`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  [
    {
      "timestamp": "17:07:34",
      "job_id": "job-772",
      "event": "PUBLISHED",
      "subject": "jobs.submitted",
      "worker": "job-service",
      "delivery_count": 1,
      "delivery_mode": "CORE",
      "correlation_id": "corr-job-772",
      "msg_id": "job-772",
      "job_type": "image-processing",
      "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
    },
    {
      "timestamp": "17:07:35",
      "job_id": "job-772",
      "event": "RECEIVED",
      "subject": "jobs.received",
      "worker": "processor-1",
      "delivery_count": 1,
      "delivery_mode": "CORE",
      "correlation_id": "corr-job-772",
      "msg_id": "job-772",
      "job_type": "image-processing",
      "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
    }
  ]
  ```

---

### 1.9. Get System Status
Retrieve overall service connectivity and JetStream Stream pending stats.
* **Endpoint**: `GET /status`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  {
    "nats": {
      "status": "CONNECTED"
    },
    "services": [
      {
        "name": "processor-service",
        "status": "ACTIVE",
        "details": "Processor is active and processing messages",
        "processing": true
      }
    ],
    "jetstream": {
      "stream": "JOBS",
      "pending": 0
    }
  }
  ```

---

### 1.10. Put Processor State
Dynamically toggle whether the processor-service background processing is enabled or disabled.
* **Endpoint**: `PUT /processor/state`
* **Content-Type**: `application/json`
* **Request Body**:
  ```json
  {
    "enabled": false
  }
  ```
* **Response**: `200 OK`
* **Response Body**:
  ```json
  {
    "enabled": false,
    "status": "STOPPED"
  }
  ```

---

### 1.11. Get Consumer Status
Retrieve active JetStream consumer configuration and live metrics (pending, ack_pending, redelivered).
* **Endpoint**: `GET /consumer`
* **Response**: `200 OK`
* **Response Body**:
  ```json
  {
    "name": "job-processor",
    "type": "durable",
    "workers": 2,
    "ordering": "normal",
    "delivery": "at-least-once",
    "status": "ACTIVE",
    "pending": 0,
    "ack_pending": 0,
    "redelivered": 0
  }
  ```

---

### 1.12. Put Consumer Configuration
Dynamically configure consumer settings (Durable vs Ephemeral, worker pool size, ordering).
* **Endpoint**: `PUT /consumer`
* **Content-Type**: `application/json`
* **Request Body**:
  ```json
  {
    "type": "durable",
    "workers": 2,
    "ordering": "normal"
  }
  ```
* **Response**: `200 OK`
* **Response Body**: Returns updated `ConsumerStatusResponse`.

---

## 2. NATS Subjects & Payload Contracts

All NATS message payloads are structured as JSON. Standard metadata is passed via NATS headers to keep the payload clean.

### 2.1. Headers
The following headers are present in messages:
* `Content-Type`: `application/json`
* `Nats-Msg-Id`: Unique message ID (used for JetStream message deduplication).
* `X-Correlation-Id`: Correlation ID passed down from the client.
* `X-Source`: Identifier of the sending service (e.g., `job-service` or `processor-service`).
* `traceparent`: Standard W3C distributed tracing header (`00-<trace_id>-<span_id>-01`) for OpenTelemetry context propagation.

---

### 2.2. NATS Contracts

| Subject | Direction | Purpose | Payload Schema |
| :--- | :--- | :--- | :--- |
| `jobs.submitted` | Job -> Processor | Job submission event | `{"job_id": string, "type": string, "payload": object, "delivery_mode": string}` |
| `jobs.validate` | Job <-> Processor | Req/Rep validation | **Req**: `{"job_id": string, "type": string, "payload": object, "delivery_mode": string}`<br>**Rep**: `{"valid": boolean, "message": string}` |
| `jobs.request.received` | Processor -> Job | Validation request received event | `{"job_id": string, "status": "REQUEST_RECEIVED", "delivery_count": 1}` |
| `jobs.reply.sent` | Processor -> Job | Validation reply dispatched event | `{"job_id": string, "status": "REPLY_SENT", "delivery_count": 1}` |
| `jobs.processing.started` | Processor -> Job | Job processing started | `{"job_id": string, "status": "PROCESSING", "delivery_count": int}` |
| `jobs.processing.completed` | Processor -> Job | Processing completed successfully | `{"job_id": string, "status": "COMPLETED", "delivery_count": int}` |
| `jobs.processing.failed` | Processor -> Job | Processing failed | `{"job_id": string, "status": "FAILED", "delivery_count": int, "error": string}` |
| `jobs.completed` | Processor -> Job | Job successfully finished | `{"job_id": string, "status": "COMPLETED", "delivery_count": int}` |
| `jobs.failed` | Processor -> Job | Job execution failed | `{"job_id": string, "status": "FAILED", "delivery_count": int, "error": string}` |
| `jobs.stored` | Job -> Job/Observability | JetStream message stored ack | `{"job_id": string, "status": "STORED", "sequence": uint64}` |
| `jobs.deduplicated` | Job -> Job/Observability | JetStream deduplication ack | `{"job_id": string, "status": "DEDUPLICATED", "sequence": uint64}` |
| `consumer.config.set` | Job <-> Processor | Dynamic consumer reconfiguration | **Req**: `{"type": string, "workers": int, "ordering": string}`<br>**Rep**: `ConsumerStatusResponse` |

---

## 3. Workflow Flowchart

```mermaid
sequenceDiagram
    autonumber
    actor Client
    participant JS as Job Service (HTTP/NATS)
    participant NATS as NATS (JetStream)
    participant PS as Processor Service (NATS)

    Note over Client, PS: Core Pub/Sub Workflow (Phase 1 & 4)
    Client->>JS: POST /jobs (job-101)
    JS->>NATS: Publish jobs.submitted
    JS-->>Client: 202 Accepted (SUBMITTED)
    NATS->>PS: Deliver job-101
    PS->>NATS: Publish jobs.processing
    NATS->>JS: Deliver jobs.processing event (status updated in memory)
    PS->>PS: Process job
    PS->>NATS: Publish jobs.completed
    NATS->>JS: Deliver jobs.completed event (status updated in memory)
    PS->>NATS: ACK message

    Note over Client, PS: Request/Reply Workflow (Success Path)
    Client->>JS: POST /jobs/validate (X-Correlation-Id)
    JS->>NATS: Request jobs.validate
    NATS->>PS: Deliver validation request
    PS->>NATS: Publish jobs.request.received
    PS-->>NATS: Reply validation response
    PS->>NATS: Publish jobs.reply.sent
    NATS-->>JS: Deliver validation response
    JS-->>Client: 200 OK (valid: true)

    Note over Client, PS: Request/Reply Workflow (Timeout Path - PS OFF)
    Client->>JS: POST /jobs/validate
    JS->>NATS: Request jobs.validate (2s timeout)
    Note over PS: Processor is OFF (not responding)
    NATS--xJS: Timeout (2s elapsed, no reply)
    JS-->>Client: 504 Gateway Timeout
```

---

## 4. Proposed Phased Approach

We recommend building the services incrementally across the following stages:

1. **Phase 1 (Core)**:
   * Implement basic folder structure, `go.mod`, configurations.
   * `POST /jobs` endpoint publishing to Core NATS `jobs.submitted`.
   * Processor subscribing to `jobs.submitted` with simple logging.
2. **Phase 2 (Lifecycle & Status)**:
   * Processor publishing lifecycle events (`jobs.processing`, `jobs.completed`, `jobs.failed`).
   * Job service subscribing to wildcard `jobs.*` to track in-memory state.
   * `GET /jobs` and `GET /jobs/{job_id}` endpoints.
3. **Phase 3 (Request/Reply)**:
   * `POST /jobs/validate` HTTP handler on Job Service.
   * Sync NATS `jobs.validate` handler on Processor Service.
4. **Phase 4 (JetStream & Durability)**:
   * Setup JetStream stream `JOBS` listening to `jobs.>`.
   * Migrate processor consumer to a Pull-based Durable JetStream consumer.
5. **Phase 5 (Advanced capabilities)**:
   * Competing consumers, delivery semantics (ACK/NACK/redelivery count), and message deduplication.
   * Replay endpoint and custom consumer policies.
