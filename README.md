# NATS Reference Platform & Microservices Architecture

This repository is a production-like **NATS evaluation platform and reference implementation** built with Go, Gin, and Zerolog. It demonstrates Core NATS messaging patterns, JetStream persistence, request-reply semantics, and dynamic subscription lifecycle management across segregated microservices.

---

## 1. System Architecture

```
                                  +-------------------+
                                  |    NATS Server    |
                                  |  nats://localhost |
                                  +---------+---------+
                                            ^
                       +--------------------+--------------------+
                       |                                         |
                       v                                         v
        +-----------------------------+           +-----------------------------+
        |        Order Service        |           |     Processing Service      |
        |     (Publisher Service)     |           |    (Subscriber Service)     |
        |    http://localhost:8080    |           |    http://localhost:8081    |
        +--------------+--------------+           +--------------+--------------+
                       |                                         |
     +-----------------+-----------------+       +---------------+---------------+
     |                 |                 |       |               |               |
     v                 v                 v       v               v               v
  Publish       Publish/Reply     Publish/JS  Subscribe      Unsubscribe       Drain
```

- **Order Service (`services/cmd/order-service`)**: Exposed on port `:8080`. Handles Core NATS and JetStream message publishing.
- **Processing Service (`services/cmd/processing-service`)**: Exposed on port `:8081`. Manages dynamic subscription lifecycles (create, list, unsubscribe, drain) and background message processing.

---

## 2. JetStream Stream Setup

Before publishing to JetStream subjects (e.g. `orders.created`), create the target stream using the NATS CLI:

```bash
nats stream add ORDERS \
  --subjects "orders.>" \
  --storage file \
  --retention limits
```

Alternatively, streams can be initialized programmatically via the `jetstream.CreateOrUpdateStream` Go SDK API.

---

## 3. Running the Services

### Prerequisites
- Go 1.22+
- Running NATS Server with JetStream enabled (`nats-server -js`)

### Start NATS Server
```bash
nats-server -js
```

### Start Order Service (Publisher)
```bash
go run ./services/cmd/order-service
```

### Start Processing Service (Subscriber / Consumer)
```bash
go run ./services/cmd/processing-service
```

---

## 4. API Reference & Sample Requests

### A. Publisher APIs (Order Service - `http://localhost:8080`)

#### 1. Standard Publish (`POST /api/v1/publish`)
```bash
curl -X POST http://localhost:8080/api/v1/publish \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "orders.created",
    "data": "Order #1001 payload",
    "headers": {
      "Correlation-ID": "c8f9a2b1-4e20-411a-b33c-58e39f72a123",
      "Source-System": "checkout-service"
    }
  }'
```

#### 2. Synchronous Request-Reply (`POST /api/v1/publish/request`)
```bash
curl -X POST http://localhost:8080/api/v1/publish/request \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "orders.status",
    "data": "Get status for Order #1001",
    "headers": {
      "Correlation-ID": "c8f9a2b1-4e20-411a-b33c-58e39f72a123"
    }
  }'
```

#### 3. JetStream Synchronous Publish (`POST /api/v1/publish/js`)
Returns native `PubAck` object (`stream`, `sequence`, `domain`, `duplicate`).

```bash
curl -X POST http://localhost:8080/api/v1/publish/js \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "orders.created",
    "data": "JetStream synchronous order #1001"
  }'
```

#### 4. JetStream Asynchronous Publish (`POST /api/v1/js/publish/async`)
```bash
curl -X POST http://localhost:8080/api/v1/js/publish/async \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "orders.created",
    "data": "JetStream async order #1002"
  }'
```

---

### B. Subscription Lifecycle APIs (Processing Service - `http://localhost:8081`)

#### 1. Create & Start Subscription (`POST /api/v1/subscriptions`)
```bash
curl -X POST http://localhost:8081/api/v1/subscriptions \
  -H "Content-Type: application/json" \
  -d '{
    "subject": "orders.>",
    "handlerType": "order"
  }'
```

#### 2. List Active Subscriptions (`GET /api/v1/subscriptions`)
```bash
curl -X GET http://localhost:8081/api/v1/subscriptions
```

#### 3. Graceful Drain Subscription (`POST /api/v1/subscriptions/drain`)
```bash
curl -X POST http://localhost:8081/api/v1/subscriptions/drain \
  -H "Content-Type: application/json" \
  -d '{
    "id": "sub-1"
  }'
```

#### 4. Stop / Unsubscribe Immediately (`POST /api/v1/subscriptions/unsubscribe`)
```bash
curl -X POST http://localhost:8081/api/v1/subscriptions/unsubscribe \
  -H "Content-Type: application/json" \
  -d '{
    "id": "sub-1"
  }'
```

---

## 5. Logging & Observability

Logging is managed by **Zerolog** (`github.com/rs/zerolog`).
- Human-friendly colorful console logs are configured by default in development (`logger.Init(false)`).
- Formatted structured JSON logging can be toggled via `logger.Init(true)` for production.

---

## 6. Repository Layout

```text
.
├── README.md                           # Platform Overview & API Guide
├── docs/                               # Developer Guides & Changelog
│   ├── CHANGELOG.md
│   ├── nats-publisher-guide.md
│   ├── nats-subscription-guide.md
│   └── nats-consumer-guide.md
└── services/                           # Go Application Backend
    ├── api/                            # Gin REST Handlers
    │   ├── publisher_handler.go
    │   └── subscription_handler.go
    ├── cmd/                            # Application Entrypoints
    │   ├── order-service/main.go       # Publisher Service (:8080)
    │   └── processing-service/main.go  # Consumer Service (:8081)
    ├── config/                         # Environment Configuration
    ├── logger/                         # Zerolog Console Formatting
    ├── messaging/                      # NATS Core & JetStream Drivers
    ├── model/                          # Shared Models & DTOs
    ├── natsclient/                     # Connection Lifecycle & Recovery
    ├── service/                        # Service Layer Orchestration
    └── subscription/                   # Thread-safe Subscription Registry
```
