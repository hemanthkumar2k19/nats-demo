# Streams

## 1. Overview

A **Stream** is a JetStream persistence and operational boundary that stores messages published to one or more NATS subjects.

A Stream determines **which messages are persisted and how that persisted workload is managed**, including retention, storage, replication, and lifecycle characteristics.

A Stream can contain multiple subjects:

```text
ORDERS
├── orders.created
├── orders.confirmed
├── orders.cancelled
└── orders.completed
```

Consumers then read persisted messages from the Stream according to their own consumption requirements.

```text
Publisher
    │
    ▼
 Subject(s)
    │
    ▼
  Stream
    │
    ├── Consumer A
    ├── Consumer B
    └── Consumer C
```

### Design Principle

> **Subjects define the messaging namespace, Streams define the persistence boundary, and Consumers define the consumption state.**

A new Subject, Application, Service, or Consumer **does not automatically require a new Stream**. We should reuse an existing Stream when its persistence, workload, storage, lifecycle, and operational requirements are compatible.

For detailed Stream boundary decisions, see **Stream Design**

---

## 2. Enterprise Standard for Naming

Stream names should identify the **logical persisted workload or persistence boundary**, rather than the individual subject, application, consumer, or deployment environment.

### 2.1 Naming Principles

| Principle | Standard |
|---|---|
| **Semantic identity** | Name the logical persisted workload represented by the Stream. |
| **Subject independence** | Do not derive the Stream name directly from a single Subject. |
| **Application independence** | Do not use application/service names unless the application itself represents the intended persistence boundary. |
| **Environment independence** | Do not encode environment when it is already represented by the NATS Account or deployment boundary. |
| **Consumer independence** | Do not include Consumer names or consumption behaviour. |
| **Configuration independence** | Do not encode retention, storage, replica count, or other Stream configuration in the name. |
| **Stability** | Prefer names that remain valid when applications, deployments, or implementation details change. |

### 2.2 Naming Format

Use:

```text
UPPER_SNAKE_CASE
```

Examples:

```text
ORDER_EVENTS
CUSTOMER_EVENTS
INVENTORY_EVENTS
PAYMENT_EVENTS
```

The name should be **concise, human-readable, and semantically meaningful**.

Avoid:

```text
ORDERS_CREATED_STREAM
PAYMENT_SERVICE_PROD_STREAM
ORDER_EVENTS_RETENTION_7D
DEV_ORDER_EVENTS
ORDER_CONSUMER_STREAM
```

These encode subject details, application ownership, environment, configuration, or consumer behaviour that should be represented elsewhere.

### Core Principle

> **A Stream name identifies what persisted workload the Stream represents—not how the workload happens to be implemented.**

This gives us a clean separation:

```text
Account       → Enterprise / security boundary
Stream        → Persistence / operational boundary
Subject       → Messaging / routing namespace
Consumer      → Consumption state
```