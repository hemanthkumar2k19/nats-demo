# NATS Demo - Backend Specification

## 1. Purpose

This project is a small backend demo to evaluate core NATS capabilities using the Go NATS client.

The demo uses a simple **Job Processing** domain. Business logic is intentionally minimal. The primary objective is to demonstrate NATS capabilities through a small, flexible, and extendable system.

### NATS capabilities in scope

1. Addressing
2. Fire-and-forget Pub/Sub
3. Durable Streaming - JetStream
4. Consumer Model
5. Competing Consumers
6. Ordering
7. Delivery Semantics
8. Replay / Rewind
9. Request / Reply

---

# 2. High-Level Architecture

```text
                         +----------------------+
                         |    Demo Service      |
                         |                      |
                         | HTTP API             |
                         | Job Management       |
                         | NATS Publisher       |
                         | NATS Requester       |
                         +----------+-----------+
                                    |
                                    |
                                    v
                         +----------------------+
                         |        NATS          |
                         |                      |
                         |    Core NATS         |
                         |    JetStream         |
                         +----------+-----------+
                                    |
                         jobs.submitted
                                    |
                                    v
                         +----------------------+
                         |  Processor Service   |
                         |                      |
                         | NATS Consumer        |
                         | Request/Reply        |
                         | Job Processing       |
                         +----------------------+

```

## Services

### Demo Service

Responsibilities:

* Expose HTTP APIs for the demo
* Accept jobs
* Publish job events
* Perform NATS Request/Reply operations
* Provide basic job status

### Processor Service

Responsibilities:

* Consume submitted jobs
* Process jobs
* Send ACK/NACK as appropriate
* Demonstrate failure and redelivery
* Handle Request/Reply requests
* Publish job lifecycle events

### NATS

Responsibilities:

* Core Pub/Sub
* Request/Reply
* Subject-based addressing
* JetStream persistence
* Consumer management
* Message acknowledgement
* Redelivery
* Replay

---

# 3. Subject Design

Use the following subject hierarchy.

```text
jobs.validate
jobs.submitted
jobs.processing
jobs.completed
jobs.failed
```

### Wildcard examples

```text
jobs.*
```

Matches direct subjects such as:

```text
jobs.validate
jobs.submitted
jobs.completed
```

```text
jobs.>
```

Matches all subjects below `jobs`.

This can later be used by an Observer service.

---

# 4. NATS Message Structure

A NATS message consists of:

1. Subject
2. Headers
3. Payload

Example:

```text
Subject:
jobs.submitted

Headers:
Content-Type: application/json
Nats-Msg-Id: job-101
X-Correlation-Id: 8f32a1c2
X-Source: demo-service

Payload:
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

The business payload should not contain NATS-specific metadata such as subject or correlation ID.

---

# 5. Models

## 5.1 Job

Represents a job submitted to the platform.

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

Fields:

| Field     | Type   | Description           |
| --------- | ------ | --------------------- |
| `job_id`  | string | Unique job identifier |
| `type`    | string | Type of job           |
| `payload` | object | Job-specific data     |

---

## 5.2 Job Status

```json
{
  "job_id": "job-101",
  "status": "COMPLETED"
}
```

Possible statuses:

```text
SUBMITTED
PROCESSING
COMPLETED
FAILED
```

---

## 5.3 Job Validation Request

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

---

## 5.4 Job Validation Response

```json
{
  "valid": true,
  "message": "Job is valid"
}
```

---

# 6. Demo Service API

Base path:

```text
/
```

## 6.1 Submit Job

### Endpoint

```http
POST /jobs
```

### Purpose

Accept a new job and publish it to NATS using fire-and-forget messaging.

### Request

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

### Response

```http
202 Accepted
```

```json
{
  "job_id": "job-101",
  "status": "SUBMITTED"
}
```

### NATS Operation

```text
Publish
    |
    v
jobs.submitted
```

The API does not wait for the Processor Service to finish processing the job.

---

# 6.2 Validate Job

### Endpoint

```http
POST /jobs/validate
```

### Purpose

Validate a job using NATS Request/Reply.

### Request

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

### Response

```http
200 OK
```

```json
{
  "valid": true,
  "message": "Job is valid"
}
```

### NATS Operation

```text
Demo Service
     |
     | Request
     v
jobs.validate
     |
     v
Processor Service
     |
     | Reply
     v
Demo Service
```

---

# 6.3 Get Job Status

### Endpoint

```http
GET /jobs/{job_id}
```

### Purpose

Return the current status of a job.

### Example

```http
GET /jobs/job-101
```

### Response

```json
{
  "job_id": "job-101",
  "status": "COMPLETED"
}
```

For the initial demo, status can be maintained in memory.

Persistent job storage is not required.

---

# 6.4 Replay Jobs

This API is part of the target design but can be implemented later.

### Endpoint

```http
POST /jobs/replay
```

### Purpose

Start replaying previously stored JetStream messages.

### Request

```json
{
  "from_sequence": 100,
  "to_sequence": 120
}
```

### Response

```http
202 Accepted
```

```json
{
  "status": "REPLAY_STARTED",
  "consumer": "job-replay-001"
}
```

The implementation can later support replay based on timestamp.

Example:

```json
{
  "from_time": "2026-08-31T10:00:00Z",
  "to_time": "2026-08-31T11:00:00Z"
}
```

---

# 7. Processor Service NATS APIs

The Processor Service does not require HTTP APIs initially.

Its primary interface is NATS.

---

## 7.1 Job Submission Consumer

### Subject

```text
jobs.submitted
```

### Message

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

### Processing

```text
Receive
   |
   v
Process Job
   |
   +---- Success ----> ACK
   |
   +---- Failure ----> NACK / no ACK
```

The consumer will eventually be configured as a durable JetStream consumer.

---

# 7.2 Job Validation Request/Reply

### Subject

```text
jobs.validate
```

### Request

```json
{
  "job_id": "job-101",
  "type": "image-processing",
  "payload": {
    "file": "image-101.jpg"
  }
}
```

### Response

```json
{
  "valid": true,
  "message": "Job is valid"
}
```

---

# 7.3 Job Lifecycle Events

The Processor Service publishes lifecycle events.

### Processing

```text
jobs.processing
```

Payload:

```json
{
  "job_id": "job-101",
  "status": "PROCESSING"
}
```

### Completed

```text
jobs.completed
```

Payload:

```json
{
  "job_id": "job-101",
  "status": "COMPLETED"
}
```

### Failed

```text
jobs.failed
```

Payload:

```json
{
  "job_id": "job-101",
  "status": "FAILED"
}
```

---

# 8. JetStream Design

Initially use one stream.

## Stream

```text
Name:
JOBS
```

### Subjects

```text
jobs.>
```

### Purpose

Persist job-related events and enable:

* Durable processing
* Consumer state
* ACK
* Redelivery
* Replay
* Ordering evaluation

Conceptually:

```text
                     +----------------+
jobs.> ------------> |     JOBS       |
                     |   JetStream    |
                     +-------+--------+
                             |
               +-------------+-------------+
               |             |             |
               v             v             v
          Processor       Observer       Replay
           Consumer       Consumer       Consumer
```

---

# 9. Consumer Design

The primary Processor consumer should be:

```text
Type:
Durable

Mode:
Pull

Purpose:
Job processing
```

Conceptually:

```text
JOBS Stream
     |
     v
Durable Consumer
     |
     | Fetch
     v
Processor
     |
     | Process
     v
ACK
```

Later evaluate:

* Durable vs ephemeral
* Pull vs push
* Consumer acknowledgement
* Pending messages
* Redelivery
* Consumer restart

---

# 10. Competing Consumers

Multiple Processor Service instances should use the same consumer / queue configuration.

Example:

```text
              jobs.submitted
                    |
             JOBS JetStream
                    |
             job-processors
                    |
       +------------+------------+
       |            |            |
       v            v            v
 processor-1   processor-2   processor-3
```

Expected behavior:

```text
job-101 -> processor-1
job-102 -> processor-2
job-103 -> processor-3
job-104 -> processor-1
```

The number of Processor instances can be increased without changing the application API.

---

# 11. Delivery Semantics

The demo should explicitly test message failure.

Example:

```text
Job received
     |
     v
Processing
     |
     X
   Failure
     |
     X
  No ACK
     |
     v
Redelivery
     |
     v
Processing succeeds
     |
     v
   ACK
```

The demo should record the delivery count so that redelivery can be observed.

Example:

```json
{
  "job_id": "job-101",
  "delivery_count": 2,
  "status": "COMPLETED"
}
```

---

# 12. Message Deduplication

When publishing a message, use a unique message ID.

Example header:

```text
Nats-Msg-Id: job-101
```

If the same message is published again within the configured deduplication window, JetStream should detect it as a duplicate.

This capability should be tested separately from normal processing.

---

# 13. Ordering

Use job sequence numbers to evaluate ordering.

Example:

```text
job-001
job-002
job-003
job-004
job-005
```

The demo should record:

```text
Received sequence:
1
2
3
4
5
```

Ordering should be tested under different conditions:

1. Single consumer
2. Multiple consumers
3. Concurrent processing
4. Message failure and redelivery

The demo should document the observed behavior rather than assuming ordering across all scenarios.

---

# 14. Replay / Rewind

Previously stored messages should be replayable from JetStream.

Supported demo scenarios:

### Replay from sequence

```text
From: 100
To:   120
```

### Replay from timestamp

```text
From: 10:00
To:   11:00
```

Replay should use a separate consumer so that normal processing is not affected.

```text
JOBS Stream
     |
     +---- Normal Consumer
     |
     +---- Replay Consumer
```

---

# 15. Correlation and Message Headers

Use headers for infrastructure metadata.

Example:

```text
Content-Type: application/json
Nats-Msg-Id: job-101
X-Correlation-Id: 8f32a1c2
X-Source: demo-service
```

`X-Correlation-Id` should be propagated across the request and event flow.

Example:

```text
HTTP Request
     |
     | correlation-id = 8f32a1c2
     v
Demo Service
     |
     | jobs.submitted
     | correlation-id = 8f32a1c2
     v
Processor
```

This will also make the system easier to observe later through the UI.

---

# 16. Go Project Structure

```text
nats-demo/
│
├── cmd/
│   ├── demo-service/
│   │   └── main.go
│   │
│   └── processor-service/
│       └── main.go
│
├── internal/
│   │
│   ├── jobs/
│   │   ├── model.go
│   │   └── service.go
│   │
│   ├── messaging/
│   │   ├── subjects.go
│   │   ├── publisher.go
│   │   ├── requester.go
│   │   └── consumer.go
│   │
│   └── config/
│       └── config.go
│
├── api/
│   └── http/
│       ├── handler.go
│       └── routes.go
│
├── go.mod
├── go.sum
└── README.md
```

---

# 17. Package Responsibilities

## `cmd/`

Application entry points only.

### `demo-service`

Responsible for starting:

* Configuration
* NATS connection
* HTTP server
* Publisher
* Requester

### `processor-service`

Responsible for starting:

* Configuration
* NATS connection
* Job consumer
* Request/Reply handler
* Job processor

---

## `internal/jobs/`

Contains business/domain models and job processing logic.

Example:

```go
type Job struct {
    JobID   string                 `json:"job_id"`
    Type    string                 `json:"type"`
    Payload map[string]interface{} `json:"payload"`
}
```

This package should not depend heavily on NATS-specific concepts.

---

## `internal/messaging/`

Contains NATS-specific functionality.

Responsibilities:

* Publish
* Request/Reply
* Subscribe
* JetStream streams
* JetStream consumers
* ACK/NACK
* Headers
* Message handling

---

## `internal/messaging/subjects.go`

Central location for subject definitions.

```go
const (
    SubjectJobValidate   = "jobs.validate"
    SubjectJobSubmitted  = "jobs.submitted"
    SubjectJobProcessing = "jobs.processing"
    SubjectJobCompleted  = "jobs.completed"
    SubjectJobFailed     = "jobs.failed"
)
```

---

## `internal/config/`

Contains application configuration such as:

```text
NATS_URL
NATS_USER
NATS_PASSWORD
NATS_STREAM
NATS_CONSUMER
```

Keep configuration simple initially.

---

## `api/http/`

Contains HTTP handlers and routes for the Demo Service.

The HTTP layer should call the job/service layer rather than directly implementing NATS operations.

---

# 18. Implementation Sequence

Build incrementally in this order.

## Phase 1 - Core Pub/Sub

```text
POST /jobs
     |
     v
Demo Service
     |
     | Publish
     v
jobs.submitted
     |
     v
Processor
```

Learn:

* NATS connection
* Publish
* Subscribe
* Subjects

---

## Phase 2 - Subject Hierarchy

Add:

```text
jobs.*
jobs.>
```

Test wildcard subscriptions.

---

## Phase 3 - Request/Reply

Implement:

```text
POST /jobs/validate
```

using:

```text
jobs.validate
```

---

## Phase 4 - JetStream

Create:

```text
JOBS
```

with:

```text
jobs.>
```

Move job processing to JetStream.

---

## Phase 5 - Durable Pull Consumer

Implement:

```text
JOBS
  |
  v
Durable Consumer
  |
  v
Processor
```

Test restart and pending messages.

---

## Phase 6 - Competing Consumers

Run multiple Processor instances.

Test load distribution.

---

## Phase 7 - Delivery Semantics

Test:

* ACK
* NACK
* No ACK
* Redelivery
* Delivery count
* Message deduplication

---

## Phase 8 - Ordering

Test ordering with:

* Single consumer
* Multiple workers
* Concurrent processing
* Redelivery

---

## Phase 9 - Replay

Implement:

```text
POST /jobs/replay
```

and experiment with JetStream delivery policies.

---

# 19. Initial Scope Boundaries

The first version should **not** include:

* Database
* Authentication/authorization
* Kubernetes
* External job processing
* Complex business logic
* UI
* Observer service
* Multi-tenancy
* Distributed deployment
* Production-grade configuration management

These can be added later if required.

The initial objective is:

```text
2 Go Services
       +
    1 NATS
       +
9 NATS Capabilities
```

The implementation should remain small enough that each NATS capability can be independently tested and demonstrated.


# Status Management

The backend is responsible for providing platform and service status to the frontend.

The frontend should **not connect directly to NATS**. NATS connectivity and platform status should be exposed through backend APIs.

```text
React UI
   |
   | HTTP
   v
Demo Service
   |
   | NATS Go Client
   v
NATS Server
```

## Status API

### Endpoint

```http
GET /status
```

### Purpose

Return the current status of the demo platform and its services.

### Response

```json
{
  "status": "UP",
  "nats": {
    "status": "CONNECTED"
  },
  "services": [
    {
      "name": "demo-service",
      "status": "ACTIVE"
    },
    {
      "name": "processor-service",
      "status": "ACTIVE"
    }
  ]
}
```

The exact response can evolve as additional services and status information are introduced.

## Responsibilities

The backend should provide status information for:

* NATS connectivity
* Demo Service
* Processor Service
* Processor instances, when applicable
* Future backend components

NATS status should be determined using the backend's NATS connection/client rather than by exposing NATS connectivity directly to the browser.

## Future Extension

As the UI grows, status management may be separated into a dedicated Observer Service. This is not required for the initial implementation.

The status API should therefore remain loosely coupled to the rest of the application so that additional status information can be added without changing the frontend architecture.

```text
Current:

UI -> Demo Service -> NATS

Future:

UI -> Observer Service -> NATS
                   |
                   +-> Service Status
                   +-> Stream Status
                   +-> Consumer Status
                   +-> Activity
```