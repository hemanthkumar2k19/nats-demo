# NATS Reference Platform & Microservices Architecture

This repository is a production-like **NATS evaluation platform and reference implementation** built with Go, Gin, and Zerolog. It demonstrates JetStream stream persistence, pull and push consumer workers, and message publishing across segregated microservices.

---

## 1. System Architecture

```text
                             +-------------------+
                             |    NATS Server    |
                             |  nats://localhost |
                             +---------+---------+
                                       ^
            +--------------------------+--------------------------+
            |                          |                          |
            v                          v                          v
+-----------------------+  +-----------------------+  +-----------------------+
|     Order Service     |  |    Payment Worker     |  |  Notification Worker  |
| http://localhost:8080 |  |  (JetStream Pull)     |  |  (JetStream Push)     |
+-----------+-----------+  +-----------------------+  +-----------------------+
            |
    +-------+-------+
    |       |       |
    v       v       v
 Streams Consumers Publish
 (CRUD)   (CRUD)    (JS)
```

- **Order Service (`services/cmd/order-service`)**: Exposed on port `:8080`. Provides REST APIs for publishing messages to JetStream and managing stream and consumer lifecycles.
- **Payment Worker (`services/cmd/payment-worker`)**: Background worker that pulls and processes payment events from the `PAYMENTS` stream (`payments-pull` consumer).
- **Notification Worker (`services/cmd/notification-worker`)**: Background worker that consumes payment notifications from the `PAYMENTS` stream (`payments-push` consumer).

---

## 2. JetStream Stream Setup

Before running the workers or publishing messages, ensure the target stream exists. You can create the stream via the REST API or using the NATS CLI:

**Option A: Via REST API**
`POST http://localhost:8080/api/v1/streams` with body `{"name": "PAYMENTS", "subjects": ["payments.>"], "storage": "file"}`

**Option B: Via NATS CLI**
```bash
nats stream add PAYMENTS --subjects "payments.>" --storage file --default
```

---

## 3. Running the Services

### Prerequisites
- Go 1.22+
- NATS Server with JetStream enabled (`nats-server -js`)

### 1. Start NATS Server
```bash
nats-server -js
```

### 2. Start Order Service (API Publisher, Stream & Consumer Management)
```bash
go run ./services/cmd/order-service
```

### 3. Start Payment Worker (Pull Consumer)
```bash
go run ./services/cmd/payment-worker
```

### 4. Start Notification Worker (Push Consumer)
```bash
go run ./services/cmd/notification-worker
```

---

## 4. API Reference

The Order Service exposes REST endpoints under `/api/v1` on port `:8080`.

All endpoints, pre-configured variables, and sample payloads can be imported directly into Postman via [nats_platform.postman_collection.json](file:///Users/mulukahemanthkumar/Documents/dev/poc/NATS/demo-1/nats_platform.postman_collection.json).

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `POST` | `/api/v1/publish/js` | Publishes a message to JetStream synchronously and returns `PubAck`. |
| `POST` | `/api/v1/streams` | Creates or retrieves a JetStream stream. |
| `DELETE` | `/api/v1/streams` | Deletes a JetStream stream using a JSON body (`name`). |
| `POST` | `/api/v1/consumers` | Creates or retrieves a JetStream consumer (pull or push). |
| `DELETE` | `/api/v1/consumers` | Deletes a JetStream consumer using a JSON body (`streamName`, `consumerName`). |

---

## 5. Logging & Observability

Logging is managed by **Zerolog** (`github.com/rs/zerolog`).
- Development: Human-friendly colorful console output (`logger.Init(false)`).
- Production: Formatted structured JSON output (`logger.Init(true)`).

---

## 6. Repository Layout

```text
.
|-- README.md                           # Platform Overview and Run Guide
|-- nats_platform.postman_collection.json # Ready-to-import Postman Collection v2.1.0
|-- docs/                               # Developer Guides and Changelog
|   |-- CHANGELOG.md
|   |-- nats-publisher-guide.md
|   `-- nats-consumer-guide.md
`-- services/                           # Go Application Backend
    |-- api/                            # Gin REST Handlers
    |   |-- consumer_handler.go         # Consumer management endpoints
    |   |-- publisher_handler.go        # JetStream publish endpoint
    |   `-- stream_handler.go           # Stream management endpoints
    |-- cmd/                            # Application Entrypoints (3 Services)
    |   |-- notification-worker/main.go # Push Consumer Worker
    |   |-- order-service/main.go       # REST API Publisher, Stream & Consumer Manager (:8080)
    |   `-- payment-worker/main.go      # Pull Consumer Worker
    |-- config/                         # Environment Configuration
    |-- logger/                         # Zerolog Console/JSON Formatting
    |-- messaging/                      # NATS JetStream Drivers & Utilities
    |   |-- consumer.go                 # Consumer create/delete logic
    |   |-- js.go                       # JetStream publish logic
    |   |-- stream.go                   # Stream create/delete logic
    |   `-- util.go                     # Policy resolvers and handlers
    |-- model/                          # Shared Models and DTOs
    |-- natsclient/                     # Connection Lifecycle & Recovery
    `-- service/                        # Service Layer Orchestration
```
