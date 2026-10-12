# NATS JetStream Guide

## 1. Overview and When to Use

NATS JetStream is the built-in persistence engine for the NATS platform. Layered transparently on top of Core NATS subject-based addressing, JetStream provides at-least-once message delivery, distributed message storage, deduplication, message replay, and durable consumer state tracking.

### When to Use JetStream

JetStream is designed for enterprise workloads requiring:

- **Guaranteed At-Least-Once Delivery:** Messages must not be lost if consumer services are temporarily offline, crashing, or scaling.
- **Message Persistence and Historical Replay:** Subscribers require the ability to replay historical data from a specific sequence number, start time, or token.
- **Publish Deduplication:** Upstream publishers require broker-enforced deduplication windows using unique message identifiers (`Nats-Msg-Id`).
- **Flow Control and Rate Limiting:** High-volume event streams must be consumed at a pace controlled by the consumer, avoiding slow-consumer drops.
- **Work Queue and Task Processing:** Competing worker consumers require explicit acknowledgment, automatic redelivery on failure, and poison-message termination.

### When to Use Core NATS Instead

JetStream is not required for:

- High-frequency, ephemeral telemetry or metrics where loss of individual samples is acceptable.
- Sub-millisecond, memory-only pub/sub broadcasts with no replay requirements.
- Lightweight request-reply RPC calls where persistence overhead is unnecessary.

**Official reference:** [JetStream Documentation](https://docs.nats.io/learn/jetstream)

### API Documentation References

#### Go
1. **JetStream Package:** [nats.go/jetstream Package API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
2. **Context Initialization:** [jetstream.New API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#New)

#### Java
1. **JetStream Interface:** [JetStream API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html)
2. **Context Initialization:** [Connection.jetStream() API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#jetStream())

---

## 2. Publishing

Publishing in JetStream transmits messages over standard NATS subjects bound to a persistent stream. Unlike Core NATS fire-and-forget publishing, JetStream publishing provides an acknowledgment receipt (`PubAck`) from the storage engine confirming that the message has been written to the stream.

### Key Publishing Characteristics

- **Stream Ingestion & PubAck:** When publishing through the JetStream client, the server commits the message to storage and returns an acknowledgment containing the stream name, assigned 64-bit stream sequence number, and duplicate indicator.
- **Synchronous Publishing:** By default, publish operations block until the acknowledgment arrives from the broker, ensuring confirmed persistence before the call returns.
- **Subject Matching:** The target subject must match at least one subject bound to an active stream; otherwise, the broker rejects the write or falls back to Core NATS depending on server configuration.

**Official references:**
- [JetStream - Publishing](https://docs.nats.io/learn/jetstream/publishing)
- [JetStream - Headers Reference](https://docs.nats.io/reference/jetstream/api/headers)

### 2.1 Advanced Publishing

For high-throughput applications, data integrity guarantees, and fine-grained lifecycle management, JetStream provides advanced publishing paradigms that extend basic synchronous writes.

- **Message Deduplication (`Nats-Msg-Id`):**
  - Upstream publishers include a unique identifier in the `Nats-Msg-Id` header.
  - The stream storage layer tracks these IDs within a deduplication window. If a network blip causes a retry of an already-stored message, JetStream suppresses the duplicate write and returns the existing sequence number in the `PubAck` receipt.
- **Per-Message TTL (`Nats-TTL`):**
  - From a developer perspective, individual messages can have their own expiration lifespan independent of the stream's global `MaxAge` retention.
  - Setting the `Nats-TTL` header (e.g. `30s`, `5m`, `1h`) or passing an SDK publish option (`jetstream.WithMsgTTL`) instructs the storage engine to automatically purge that specific message once its duration elapses.
- **Asynchronous Publishing:**
  - Standard synchronous publishing incurs round-trip network latency on each message.
  - Async publishing pipelines multiple publish operations without blocking for each acknowledgment up front, returning futures or promises that are collected in batches.
- **The Order Trap and Optimistic Concurrency:**
  - In async publishing, if message 3 fails or times out while messages 4, 5, and 6 succeed, resending message 3 later assigns it a higher sequence number than messages 4, 5, and 6 (inverting order).
  - Mitigate order inversion using optimistic concurrency headers:
    - `Nats-Expected-Last-Subject-Sequence` (`Nats-Expected-Last-Subject-Seq`): Server accepts the write only if the subject's last sequence exactly matches expectations, failing fast on reorder attempts.
    - `Nats-Expected-Last-Sequence` (`Nats-Expected-Last-Seq`): Verifies stream sequence integrity before committing.
    - `Nats-Expected-Last-Msg-Id`: Guarantees causality against the preceding message ID.
    - `Nats-Expected-Stream`: Guards against publishing to an unexpected stream.
- **Atomic Batch Publishing:**
  - Commits a logical group of messages (such as the line items of an order) on an all-or-nothing basis.
  - Messages in the batch are tagged with `Nats-Batch-Id` and `Nats-Batch-Sequence`, committing only when the final item carries `Nats-Batch-Commit`.
  - A sequence gap, limit overrun, or timeout abandons the entire batch without committing partial state.
- **Fast Ingest:**
  - High-frequency telemetry publishers can send standard Core NATS messages to stream subjects without awaiting `PubAck` receipts when ingestion throughput is prioritized over write confirmations.

**Official references:**
- [JetStream - Advanced publishing](https://docs.nats.io/learn/jetstream/advanced-publishing)
- [JetStream - Avoiding duplicate writes](https://docs.nats.io/learn/jetstream/publishing#avoiding-duplicate-writes)
- [JetStream - Per-message TTL](https://docs.nats.io/learn/jetstream/message-ttl)

### API Documentation References

#### Go
1. **Publishing Methods (Sync & Async):** [jetstream.Publisher API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Publisher)
2. **Publish Acknowledgment Model:** [jetstream.PubAck Model](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#PubAck)
3. **Async Publish Future:** [jetstream.PubAckFuture API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#PubAckFuture)
4. **Per-Message TTL Option:** [jetstream.WithMsgTTL API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithMsgTTL)
5. **Deduplication Option:** [jetstream.WithMsgID API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithMsgID)
6. **Optimistic Concurrency Options:** [jetstream.PublishOpt API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#PublishOpt), [WithExpectLastSequence](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithExpectLastSequence), and [WithExpectLastSequenceForSubject](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#WithExpectLastSequenceForSubject)

#### Java
1. **Synchronous Publishing:** [JetStream.publish(Message) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html#publish(io.nats.client.Message))
2. **Asynchronous Publishing:** [JetStream.publishAsync(Message) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/JetStream.html#publishAsync(io.nats.client.Message))
3. **Publish Options Model:** [PublishOptions Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/PublishOptions.html)
4. **Publish Options Builder (Concurrency & Deduplication via messageId):** [PublishOptions.Builder API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/PublishOptions.Builder.html)
5. **Message Interface & Headers (TTL & Metadata):** [Message API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Message.html#getHeaders()) and [Headers Model Source](https://github.com/nats-io/nats.java/blob/main/src/main/java/io/nats/client/impl/Headers.java)
6. **Publish Acknowledgment Model:** [PublishAck Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/api/PublishAck.html)

---

## 3. Consumers

JetStream consumers manage the read cursor, delivery state, subject filtering, and acknowledgment tracking for reading messages back from a stream.

### 3.1 Subject Filters

Filtering allows a consumer to restrict its view of a stream to a specific subset of subjects, rather than receiving every message stored in that stream.

- **Wildcard Matching:** Consumer filter subjects support standard NATS wildcard tokens (`*` matching a single token, and `>` matching one or more trailing tokens).
- **Multiple Filter Subjects:** Modern JetStream consumers support multi-subject filtering, allowing a single consumer to subscribe to multiple distinct subject patterns within the same stream.
- **Stream Sharing:** Filtering enables multiple independent microservices and workers to share a single stream while processing only the specific event types or tenant subjects relevant to their domain.

**Official reference:** [JetStream - Filtering](https://docs.nats.io/learn/jetstream/filtering)

### 3.2 Pull Consumers and Push Consumers

JetStream supports two consumption delivery mechanisms:

- **Push Consumers:**
  - The server proactively delivers messages to a client-specified delivery subject (often a Core NATS inbox).
  - Suitable for simple, single-subscriber listeners where broker flow control is not critical.
  - **Limitation:** Can lead to slow-consumer message dropping if the consumer cannot keep pace with server push rates.
- **Pull Consumers (Standard Enterprise Pattern):**
  - Inverts the control model: the application asks the server for work when ready.
  - Provides native backpressure, eliminates slow-consumer dropped messages, supports batching, and enables horizontal scaling across competing worker pools.
- **Two Pull Consumption Patterns:**
  - **Fetch (Batch):** Requests a discrete batch of up to *N* messages. The call blocks until either the full batch arrives or a timeout expires, whichever comes first. Ideal for periodic jobs, batch drains, or stepped workflows.
  - **Consume (Continuous Flow):** Establishes an asynchronous, ongoing processing loop. The client library automatically manages background pull requests and flow-control windows, invoking a registered message handler as messages arrive. Ideal for long-running service workers.
- **Pull Bounding Parameters:**
  - **Batch Message Count (`MaxMessages`):** Limits the maximum number of messages delivered per pull request.
  - **Batch Byte Size (`MaxBytes`):** Sets an aggregate byte ceiling per pull, preventing excessive client memory usage.
  - **Timeout (`Expires` / `MaxWait`):** Configures how long the pull request waits for messages before returning a partial or empty response.
  - **Idle Heartbeats:** The server emits periodic heartbeats during quiet periods to verify connection liveness and detect stalled sockets.

**Official references:**
- [JetStream - Reading Back](https://docs.nats.io/learn/jetstream/reading-back)
- [JetStream - Pull consumers in depth](https://docs.nats.io/learn/jetstream/pull-consumers)

### 3.3 Ordered Consumers

Ordered consumers are specialized ephemeral consumers engineered for fast, strictly sequential reads across a stream.

- **Guaranteed In-Order Processing:** Guarantees that messages arrive strictly in the order they were recorded in the stream, with a maximum of one in-flight message at a time.
- **Transparent Gap Recovery:** If a message delivery stalls, fails, or encounters a sequence gap, the client library automatically detects the mismatch, tears down the underlying ephemeral consumer, and transparently recreates a new consumer starting at the next expected stream sequence.
- **Zero Consumer State Maintenance:** Applications do not need to assign consumer names or issue manual message acknowledgments. When the read finishes or the application terminates, the underlying ephemeral consumer cleans itself up automatically.

**Official reference:** [JetStream - Ordered consumers](https://docs.nats.io/learn/jetstream/ordered-consumer)

### 3.4 Delivery Policy and Tracking

JetStream tracks the lifecycle of every message delivered to a consumer to guarantee at-least-once processing semantics.

- **Delivery Policy Configuration:** Consumers specify where in the stream to begin reading:
  - `DeliverAll`: Replays all messages from the beginning of the stream.
  - `DeliverLast`: Starts at the most recent message stored in the stream.
  - `DeliverNew`: Delivers only messages stored after consumer creation.
  - `DeliverByStartSequence`: Starts at a specific 64-bit stream sequence number.
  - `DeliverByStartTime`: Starts at a specified UTC timestamp.
  - `DeliverLastPerSubject`: Replays the latest message stored for each distinct subject.
- **Delivery Lifecycle & Retry Mechanics:**
  - **In-Flight Tracking:** When a message is delivered to a client, JetStream marks it as in-flight and starts an acknowledgment timer (`AckWait`).
  - **Acknowledgment Timeout (`AckWait`):** If the client fails to acknowledge or reset the timer before `AckWait` expires, JetStream marks the delivery attempt as timed out and reschedules the message for redelivery.
  - **Maximum Deliveries (`MaxDeliver`):** To prevent malformed or crashing payloads from causing infinite retry loops, consumers define a maximum delivery attempt threshold. When `MaxDeliver` is reached without a successful acknowledgment, JetStream halts redelivery for that message and can emit a consumer advisory.
  - **Backoff Policies:** Instead of fixed-interval retries, consumers can specify custom linear or exponential backoff durations between redelivery attempts.
- **Message Metadata:** Every delivered JetStream message carries metadata detailing:
  - Consumer delivery sequence and stream sequence numbers.
  - Delivery attempt count (number of times this message has been delivered to this consumer).
  - Stream timestamp when the message was originally stored.
  - Number of pending messages remaining for the consumer.

**Official reference:** [JetStream - Delivery and Acknowledgment](https://docs.nats.io/learn/jetstream/delivery-and-acknowledgment)

### 3.5 Acknowledgment Policy and Signals

Explicit acknowledgment is the protocol mechanism by which an application signals the outcome of message processing back to JetStream.

- **Acknowledgment Policies:**
  - `AckExplicit` (Standard and Recommended): Every single message must be acknowledged individually.
  - `AckNone`: Fire-and-forget; the server considers the message acknowledged as soon as it is transmitted.
  - `AckAll`: Acknowledging a message implicitly acknowledges all unacknowledged messages up to that sequence number.
- **Core Acknowledgment Signals (`AckExplicit`):**
  - **Ack (`+ACK`):**
    - **Meaning:** Processing succeeded completely.
    - **Server Action:** The message delivery is marked successful, consumer sequence progress advances, and the message will not be redelivered.
  - **Nak (`-NAK`):**
    - **Meaning:** Negative acknowledgment; processing failed due to a transient error (e.g., downstream database temporarily unreachable).
    - **Server Action:** JetStream immediately reschedules the message for redelivery (or after an optional delay if specified via delay-nak), without waiting for the `AckWait` timeout to expire.
  - **Term (`+TERM`):**
    - **Meaning:** Terminate delivery; processing encountered an unrecoverable error (e.g., corrupt payload, schema validation failure).
    - **Server Action:** JetStream permanently cancels further redeliveries of this message to this consumer, avoiding wasted retry cycles and poison-pill crashes.
  - **InProgress (`+WPI`):**
    - **Meaning:** Work in progress; the consumer is still actively processing a long-running task.
    - **Server Action:** Resets the consumer's `AckWait` timer on the server, granting additional processing time before a timeout-triggered redelivery occurs.

**Official reference:** [JetStream - Acknowledgment](https://docs.nats.io/learn/jetstream/acknowledgment)

### 3.6 Scaling Consumers with Worker Pools

Scaling consumer throughput in JetStream is achieved by attaching multiple concurrent worker instances or processes to a single durable pull consumer.

- **One Consumer, Many Workers:** Multiple worker processes or threads independently bind to the identical durable pull consumer name on a stream.
- **Dynamic Load Distribution:** JetStream automatically distributes stored messages across the active workers as each worker issues pull requests. Each stored message is handed to exactly one worker at a time.
- **No Consumer Group Rebalancing:** Unlike partition-based messaging systems that require expensive cluster-wide rebalancing stops whenever worker instances join or leave, JetStream pull workers scale dynamically and independently. Workers pull messages at their individual processing rate.
- **Worker Crash & Failover Recovery:** If a worker crashes, drops its network connection, or stalls mid-message without issuing an acknowledgment, JetStream tracks the unacknowledged message. Once `AckWait` expires, the broker automatically redelivers that message to another surviving worker in the pool.

**Official reference:** [JetStream - Scaling a consumer (Worker pool)](https://docs.nats.io/learn/jetstream/worker-pool)

### API Documentation References

#### Go
1. **Binding to Existing Consumer:** [jetstream.JetStream API (Consumer)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#JetStream)
2. **Consumer Interface (Fetch & Consume):** [jetstream.Consumer API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Consumer)
3. **Fetch Bounding Options:** [FetchMaxWait](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#FetchMaxWait) and [FetchBytes](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#FetchBytes)
4. **Consumer Configuration Model (Filters, Deliver & Ack Policies):** [jetstream.ConsumerConfig Model](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#ConsumerConfig)
5. **Ordered Consumer Configuration Model:** [jetstream.OrderedConsumerConfig Model](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#OrderedConsumerConfig)
6. **Message Interface (Metadata & Ack Signals):** [jetstream.Msg API](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#Msg)
7. **Message Metadata Model:** [jetstream.MsgMetadata Model](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream#MsgMetadata)

#### Java
1. **Stream Context Lookup:** [Connection.getStreamContext(String) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#getStreamContext(java.lang.String))
2. **Consumer Context Lookup:** [StreamContext.getConsumerContext(String) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/StreamContext.html#getConsumerContext(java.lang.String))
3. **Consumer Configuration Model (Filters, Deliver & Ack Policies):** [ConsumerConfiguration Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/api/ConsumerConfiguration.html)
4. **Continuous Consuming & Options:** [ConsumerContext.consume(MessageHandler) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/ConsumerContext.html#consume(io.nats.client.MessageHandler)) and [ConsumeOptions Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/ConsumeOptions.html)
5. **Batch Fetching & Options:** [ConsumerContext.fetch(FetchConsumeOptions) API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/ConsumerContext.html#fetch(io.nats.client.FetchConsumeOptions)) and [FetchConsumeOptions Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/FetchConsumeOptions.html)
6. **Ordered Consumer:** [StreamContext.createOrderedConsumer API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/StreamContext.html#createOrderedConsumer(io.nats.client.api.OrderedConsumerConfiguration)), [OrderedConsumerContext API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/OrderedConsumerContext.html), and [OrderedConsumerConfiguration Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/api/OrderedConsumerConfiguration.html)
7. **Message Interface (Ack Signals):** [Message API](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Message.html)
8. **Message Metadata Model:** [NatsJetStreamMetaData Model](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/impl/NatsJetStreamMetaData.html)
