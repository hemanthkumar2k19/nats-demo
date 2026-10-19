# NATS Delivery Semantics Guide: At-Most-Once and At-Least-Once

## 1. Overview

NATS supports two delivery patterns for distributed messaging:

- **At-Most-Once Delivery:** Uses **Non-Persistent Subjects**. Messages are routed purely in-memory from publisher to active subscribers. If a subscriber is offline, crashes during processing, or experiences network disconnects, the message is not retained or retransmitted.
- **At-Least-Once Delivery:** Uses **Persistent Subjects** (subjects bound to a storage Stream). Messages are written to storage before acknowledgment to the publisher. The broker tracks consumer delivery state, expects explicit message acknowledgments, and automatically redelivers unacknowledged messages upon timeout or worker crash.

### Delivery Pattern Summary

| Delivery Pattern | Target Subject Type | Broker Storage | Publisher Receipt | Subscriber Acknowledgment | Failure / Crash Behavior |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **At-Most-Once** | Non-Persistent Subjects | In-memory only | None (Fire-and-forget) | None (Implicit receipt) | Message dropped; no redelivery |
| **At-Least-Once** | Persistent Subjects | Stream storage (File/Memory) | Explicit `PubAck` receipt | Explicit `Ack` required | Automatic redelivery after `AckWait` expires |

> **Developer Rule: Subject Isolation and Publishing Methods**
>
> Subject isolation between Persistent and Non-Persistent Subjects should be enforced through organizational naming governance (e.g., prefixing persistent subjects with `events.>` and non-persistent subjects with `live.>`).
>
> Developers must observe two critical rules to avoid publishing mistakes:
> 1. **Accidental Persistence Tap:** Publishing to a Persistent Subject using standard publish (`nc.Publish`) still causes NATS to persist the message into the Stream, but the publisher receives no `PubAck` or sequence number to confirm storage.
> 2. **Publish Failure on Unbound Subjects:** Publishing using JetStream publish (`js.Publish`) to a Non-Persistent Subject (one not bound to any stream) will fail with an error (`no responders` or `stream not found`).

---

## 2. At-Most-Once Delivery (Non-Persistent Subjects)

### 2.1 Mechanics

At-Most-Once delivery provides ultra-low latency, real-time message routing:
1. The publisher publishes directly to a **Non-Persistent Subject**.
2. The broker routes the message in-memory directly to matching, currently connected subscribers.
3. If no subscribers are connected, or if a subscriber drops connection mid-stream, the message is discarded immediately.
4. Neither the publisher nor the subscriber exchanges delivery acknowledgment receipts with the broker.

### 2.2 Publisher Settings

For At-Most-Once publishing, configure the publisher with the following settings:

| Setting | Value / Type | Description |
| :--- | :--- | :--- |
| **Subject Type** | Non-Persistent Subject | Must not match any stream subject pattern (e.g., `live.telemetry.v1`) |
| **Publish Type** | Standard Publish | Direct socket write without awaiting a storage acknowledgment |
| **Flush Policy** | Optional Flush | Forces immediate TCP socket write for low-frequency messages |
| **Client Buffer Limit** | `65536` messages or `8 MB` | Publisher-side memory buffer during transient network latency |

### 2.3 Subscriber Settings

Ensure the subscriber establishes a subscription with the following parameters:

| Setting | Value / Type | Description |
| :--- | :--- | :--- |
| **Subject Pattern** | Non-Persistent Subject | Subject pattern with optional wildcards (e.g., `live.telemetry.>`) |
| **Queue Group** | Optional (e.g., `telemetry-workers`) | Distributes messages round-robin across worker instances; omit for broadcast |
| **Pending Message Limit** | `65536` messages | Subscriber-side buffer capacity before triggering slow consumer alerts |
| **Pending Byte Limit** | `67108864` bytes (64 MB) | Memory allocation limit for in-flight subscriber buffer |

### 2.4 Graceful Shutdown and Drainage

To ensure in-flight messages in the socket buffer are processed before service termination:
- **Protocol Action:** Call subscription `Drain()` upon receiving termination signals (`SIGINT`, `SIGTERM`).
- **Effect:** The client unregisters from receiving new messages, processes all messages already buffered in client memory, and cleanly closes the subscription before disconnecting.

### 2.5 Implementation Checklist (At-Most-Once)

- [ ] **Ensure** the target subject is a Non-Persistent Subject (not bound to any stream).
- [ ] **Ensure** the publisher sends messages asynchronously without expecting persistence receipts.
- [ ] **Ensure** the subscriber creates a subscription with a `QueueGroup` if load-balancing across worker instances is required.
- [ ] **Ensure** subscriber termination handlers call subscription `Drain()` to flush buffered messages prior to application exit.

---

## 3. At-Least-Once Delivery (Persistent Subjects)

### 3.1 Mechanics

At-Least-Once delivery guarantees message persistence, consumer tracking, and automated redelivery:
1. The publisher transmits a message to a **Persistent Subject**.
2. The broker writes the message to stream storage and returns a `PubAck` receipt with the assigned stream sequence.
3. A consumer pulls messages from the stream according to configured delivery rules.
4. The consumer processes the message and transmits an explicit acknowledgment (`Ack`).
5. If the consumer crashes or fails to acknowledge within `AckWait`, the broker automatically redelivers the message up to `MaxDeliver` times.

### 3.2 Stream Configuration Settings

Ensure a Stream exists with the following configuration:

| Stream Parameter | Configuration Value | Purpose / Operational Context |
| :--- | :--- | :--- |
| **Name** | `DELIVERY` | Unique stream identifier |
| **Subjects** | `[ "events.delivery.>" ]` | Persistent Subjects bound to this storage stream |
| **Storage** | `File` | Persistent on-disk storage (`Memory` for transient queues) |
| **Retention** | `Limits` | Retains messages until age, size, or count limits are reached |
| **Replicas** | `1` (or `3` for HA Clustered) | Storage replication copies across cluster nodes |
| **MaxAge** | `86400s` (24 Hours) | Retention window for stream messages |
| **MaxBytes** | `10737418240` (10 GB) | Storage capacity limit |
| **Duplicates** | `120s` (2 Minutes) | Deduplication window tracking `Nats-Msg-Id` |
| **Discard** | `Old` | Overwrites oldest messages when storage capacity is reached |

### 3.3 Publisher Configuration Settings

Ensure the publisher uses the following settings when publishing to Persistent Subjects:

| Publisher Setting | Configuration Value | Purpose / Operational Context |
| :--- | :--- | :--- |
| **Target Subject** | `events.delivery.order-created` | Persistent Subject bound to the target stream |
| **Publish Acknowledgment** | Synchronous `PubAck` | Verifies stream sequence, stream name, and duplicate status |
| **Deduplication Header** | `Nats-Msg-Id: <UUID>` | Server rejects duplicate submissions within the stream deduplication window |
| **Publish Timeout** | `5s` | Maximum time to await storage commit acknowledgment before retrying |

### 3.4 Consumer Configuration Settings

Ensure a Durable Pull Consumer exists with the following configuration:

| Consumer Parameter | Configuration Value | Purpose / Operational Context |
| :--- | :--- | :--- |
| **Durable_Name** | `delivery-processor` | Consumer state identity surviving client restarts |
| **FilterSubject** | `events.delivery.>` | Persistent Subject pattern for message consumption |
| **AckPolicy** | `Explicit` | **Mandatory:** Every message must be explicitly acknowledged by the worker |
| **AckWait** | `30s` | Processing timeout window before broker triggers redelivery |
| **MaxDeliver** | `5` | Maximum delivery attempts before the message is abandoned as poison |
| **DeliverPolicy** | `All` (or `New`) | `All` replays entire stream; `New` starts from new incoming messages |
| **ReplayPolicy** | `Instant` | Messages delivered at maximum rate without honoring original timing |
| **MaxAckPending** | `1000` | Limits unacknowledged in-flight messages across all consumer workers |
| **BackOff** | `[ 1s, 5s, 15s, 30s ]` | Optional exponential delay intervals between redelivery attempts |

### 3.5 Subscriber Pull and Flow Control Settings

Ensure the subscriber application pulls messages using bounded fetch parameters:

| Pull Parameter | Configuration Value | Purpose / Operational Context |
| :--- | :--- | :--- |
| **Batch Size (`PullMaxMessages`)** | `1` (for serial) or `10-100` (for batch) | Number of messages pulled into client memory per fetch cycle |
| **Pull Timeout (`Expires`)** | `5s` | Maximum duration a pull request waits for messages before timing out |

> **Prefetching Note:** Setting `Batch Size: 1` prevents consumer prefetching, ensuring unacknowledged messages are not held in client RAM buffers when workers crash.

### 3.6 Acknowledgment Protocol Reference

Ensure the consumer transmits the appropriate acknowledgment signal based on processing results:

| Acknowledgment Signal | Result | Broker Action |
| :--- | :--- | :--- |
| **`Ack()`** | Success | Commits consumer progress; message will not be redelivered. |
| **`Nak()`** | Processing Failure | Instructs broker to initiate immediate redelivery without waiting for `AckWait`. |
| **`InProgress()`** | Long-running Task | Resets the `AckWait` timer on the broker, granting extra processing time. |
| **`Term()`** | Unrecoverable Error | Ceases all redelivery attempts immediately without waiting for `MaxDeliver`. |

### 3.7 Graceful Shutdown and In-Flight Execution Tracking

To achieve zero message loss and prevent duplicate executions during planned restarts:

1. **Stop Consumer Pulls:** Ensure the consumer fetch loop is stopped (`Stop()`), halting new message requests from the broker.
2. **Drain Subscription:** Ensure the consumer subscription is drained (`Drain()`) to clear buffered client transport frames.
3. **Wait for In-Flight Work:** Ensure an application-level synchronization barrier (e.g., worker pool or WaitGroup) blocks process exit until all active tasks finish processing and transmit their final `Ack()`.
4. **Drain Connection:** Ensure the NATS client connection executes `Drain()` before process exit.

> **Crash Behavior:** If a worker terminates abruptly (crash, SIGKILL, power loss), active messages remain in `Ack Pending` on the broker and are automatically redelivered to surviving workers after `AckWait` expires.

### 3.8 Implementation Checklist (At-Least-Once)

- [ ] **Ensure** a Stream exists with Persistent Subjects bound, `Storage: File`, and a `Duplicates` window.
- [ ] **Ensure** a Consumer exists with `AckPolicy: Explicit`, defined `AckWait`, and `MaxDeliver`.
- [ ] **Ensure** the publisher verifies `PubAck` and includes a unique `Nats-Msg-Id`.
- [ ] **Ensure** the consumer pulls messages with bounded batch size and emits `Ack()` only after processing completes.
- [ ] **Ensure** application shutdown coordinates consumer fetch stoppage with in-flight task completion before connection closure.

---

## 4. Poison Message and Dead Letter Handling

### 4.1 Poison Message Lifecycle

A poison message is a message that cannot be processed successfully (e.g., malformed payload or downstream database failure) and causes repeated worker failures.

When an unacknowledged message reaches its `MaxDeliver` threshold:
1. The consumer fails or times out on its final attempt (`deliveries == MaxDeliver`).
2. The broker marks delivery exhausted for that specific consumer.
3. The broker automatically advances the consumer's sequence cursor to the next message. The failed message is bypassed, preventing head-of-line blocking for the consumer pipeline.
4. The message remains securely persisted in the stream for historical inspection or replay.

### 4.2 System Advisory Event Notification

Upon exhausting `MaxDeliver`, NATS automatically broadcasts an advisory event to the system subject:

```
$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.<STREAM>.<CONSUMER>
```

#### Advisory Payload Schema

```json
{
  "type": "io.nats.jetstream.advisory.v1.max_deliver",
  "id": "EcuhDtcFD8sBXHbE96Km3C",
  "timestamp": "2026-10-19T09:30:09.475668352Z",
  "stream": "DELIVERY",
  "consumer": "at-least-once",
  "stream_seq": 12,
  "deliveries": 5
}
```

### 4.3 Dead Letter Queue (DLQ) Implementation Standard

NATS does not move failed messages into a secondary bucket by default; it skips them to keep the pipeline moving. To establish a Dead Letter Queue pattern:

1. **Ensure** an administrative or supervisor service subscribes to `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`.
2. **Ensure** the supervisor extracts the `stream` and `stream_seq` fields from the advisory event.
3. **Ensure** the supervisor loads the original payload from the stream using direct sequence lookup (`GetMsg`).
4. **Ensure** the supervisor publishes the failed payload to a designated Dead Letter Subject (e.g., `dlq.delivery.order-processing`) bound to a DLQ stream, and alerts operators.

---

## 5. Official References and SDK Standards

### 5.1 Official NATS Documentation

- [NATS Publish-Subscribe](https://docs.nats.io/learn/core-nats/publish-subscribe)
- [NATS Queue Groups](https://docs.nats.io/learn/core-nats/queue-groups)
- [NATS Consumers and Delivery Guarantees](https://docs.nats.io/learn/jetstream/consumers)
- [NATS Advisories and Monitoring](https://docs.nats.io/reference/jetstream/advisories)
- [NATS Message Headers Reference](https://docs.nats.io/reference/jetstream/api/headers)

### 5.2 NATS by Example Tutorials

- [Publish-Subscribe Messaging](https://natsbyexample.com/examples/messaging/pub-sub/cli)
- [Queue Groups](https://natsbyexample.com/examples/messaging/queue-groups/cli)
- [Stream Publishing](https://natsbyexample.com/examples/jetstream/publish/cli)
- [Pull Consumers](https://natsbyexample.com/examples/jetstream/pull-consumer/cli)

### 5.3 Client SDK Repositories and Reference Guides

- **Go SDK (`nats.go`):**
  - Repository: [github.com/nats-io/nats.go](https://github.com/nats-io/nats.go)
  - Connection & Subscriptions: [pkg.go.dev/github.com/nats-io/nats.go](https://pkg.go.dev/github.com/nats-io/nats.go)
  - JetStream v2 Consumer & Stream API: [pkg.go.dev/github.com/nats-io/nats.go/jetstream](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
- **Java SDK (`jnats`):**
  - Repository: [github.com/nats-io/nats.java](https://github.com/nats-io/nats.java)
  - Javadoc API: [javadoc.io/doc/io.nats/jnats/latest](https://javadoc.io/doc/io.nats/jnats/latest/index.html)
- **Python SDK (`nats-py`):**
  - Repository: [github.com/nats-io/nats.py](https://github.com/nats-io/nats.py)
  - API Documentation: [nats-io.github.io/nats.py](https://nats-io.github.io/nats.py/)
- **C# / .NET SDK (`nats.net`):**
  - Repository: [github.com/nats-io/nats.net](https://github.com/nats-io/nats.net)
