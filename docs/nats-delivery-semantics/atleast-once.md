# At Least Once Delivery - NATS JetStream

## 1. Purpose and Use Cases

At Least Once delivery uses NATS JetStream to persist published messages and redeliver unacknowledged messages according to the consumer's acknowledgement and delivery configuration. When JetStream accepts a publication, it returns a publisher acknowledgement (`PubAck`). Consumers explicitly acknowledge successful processing; if an acknowledgement is not received within the configured timeout, or a negative acknowledgement requests redelivery, JetStream can deliver the message again.

At Least Once delivery prioritizes **message durability and recoverability while allowing duplicate deliveries**. Applications must handle duplicates safely, typically through idempotent processing.

**Suitable use cases**
- Order lifecycle events where message loss is unacceptable and duplicate processing can be handled safely.
- Asynchronous task queues and background jobs that must survive worker interruptions, subject to stream retention and delivery configuration.
- State synchronization across distributed services where missed events must be recoverable.
- Audit and event-processing workflows requiring durable message storage and replay, subject to stream retention policies.
- Payment-related workflows where durable event delivery is required and the application independently protects against duplicate financial operations.

**Not suitable when**
- Duplicate business operations cannot be safely handled and the application has no deduplication or idempotency mechanism.
- Lowest possible latency is prioritized over persistence and acknowledgement overhead, and message loss is acceptable. Consider NATS Core At Most Once.
- The workload consists of transient telemetry, heartbeats, or live metrics where missed updates are acceptable or quickly superseded.

For workloads requiring stronger duplicate-delivery protections, evaluate JetStream's Exactly Once capabilities. Application-level idempotency may still be required to prevent repeated business effects.

## 2. Message Flow

The example uses `Order-Service` as the publisher and `Processor-Service` as the consumer.

```sequence
title At Least Once Delivery - NATS JetStream

participant "order-service" as os
participant "nats / JetStream" as nats
participant "processing-service" as ps

== Publishing ==

os -> nats: PublishMsg()
note over os, nats: Subject: delivery.at-least-once

nats -> nats: Store message in Stream
nats -> os: PubAck{Stream, Sequence, ...}

== Consuming ==

ps -> nats: Consume()
note over ps, nats: Durable Consumer: at-least-once\nAckPolicy: Explicit\nAckWait: configured timeout\nMaxDeliver: configured limit

nats -> ps: Deliver NatsMsg
ps -> ps: Process(NatsMsg)

alt Processing succeeds
    ps -> nats: Ack()
    nats -> nats: Record acknowledgement
else Processing fails, retryable
    ps -> nats: Nak()
    nats -> nats: Schedule redelivery
    nats -> ps: Redeliver NatsMsg
else Processing fails permanently
    ps -> nats: Term()
    nats -> nats: Stop further redelivery
else No acknowledgement received before AckWait expires
    nats -> nats: AckWait expires
    nats -> ps: Redeliver NatsMsg
end

note over ps, nats: While processing continues, the consumer can send InProgress()\nto reset the acknowledgement timer.

note over ps, nats: Duplicate processing is possible if business processing succeeds\nbut JetStream does not record the acknowledgement.
```

**Important**: `PubAck` confirms that JetStream accepted the publication and stored the message according to the stream's configured storage and replication behavior. On the consumer side, `Ack()` signals successful processing, `Nak()` requests redelivery, and `Term()` terminates further redelivery for that message. If processing completes but the acknowledgement is not recorded, JetStream may redeliver the message, resulting in duplicate processing. Use `AckSync()` when the consumer needs confirmation that JetStream processed its acknowledgement.

`MaxDeliver` limits delivery attempts when configured. Reaching this limit does not automatically move the message to a dead-letter stream; handling that scenario is a separate recovery concern.

## 3. Implementation

### Publisher

`Order-Service` publishes messages to a JetStream-backed subject. The operation is synchronous and waits for the server to persist the message and return a `PubAck` containing the stream name and sequence number.

**Go**

```go
ack, err := js.Publish(ctx, subject, payload)
if err != nil {
    // Handle the publish error.
}
// ack.Stream, ack.Sequence confirm durable persistence
```

**Java**

```java
try {
    PublishAck ack = js.publish(subject, payload);
    // ack.getStream(), ack.getSeqno() confirm durable persistence
} catch (Exception e) {
    // Handle the publish error.
}
```

**Python**

```python
try:
    ack = await js.publish(subject, payload)
    # ack.stream, ack.seq confirm durable persistence
except Exception as exc:
    # Handle the publish error.
    print(f"Publish failed: {exc}")
```

Handle client-reported publish errors. A publish failure or timeout means the message was not confirmed as persisted; retrying without publisher deduplication identifiers may result in duplicate entries stored in the stream.

### Subscriber

`Processor-Service` creates or binds to a durable pull consumer configured with explicit acknowledgments (`AckPolicy: Explicit`), an acknowledgment timeout (`AckWait`), and a maximum delivery limit (`MaxDeliver`).

**Go**

```go
consumer, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
    Durable:       consumerName,
    FilterSubject: subject,
    AckPolicy:     jetstream.AckExplicitPolicy,
    AckWait:       30 * time.Second,
    MaxDeliver:    5,
})
if err != nil {
    // Handle consumer configuration error.
}

cc, err := consumer.Consume(func(msg jetstream.Msg) {
    if err := process(msg.Data()); err != nil {
        if isRetryable(err) {
            _ = msg.Nak() // Request immediate redelivery
        } else {
            _ = msg.Term() // Unrecoverable: stop redelivery
        }
        return
    }
    _ = msg.Ack() // Successfully processed
})
if err != nil {
    // Handle consume error.
}
```

**Java**

```java
ConsumerConfiguration config = ConsumerConfiguration.builder()
    .durable(consumerName)
    .filterSubject(subject)
    .ackPolicy(AckPolicy.Explicit)
    .ackWait(Duration.ofSeconds(30))
    .maxDeliver(5)
    .build();

ConsumerContext cc = js.getStreamContext(streamName).createOrUpdateConsumer(config);

MessagesContext mc = cc.consume(msg -> {
    try {
        process(msg.getData());
        msg.ack();
    } catch (RetryableException e) {
        msg.nak();
    } catch (Exception e) {
        msg.term();
    }
});
```

**Python**

```python
consumer = await js.create_or_update_consumer(
    stream=stream_name,
    config=ConsumerConfig(
        durable_name=consumer_name,
        filter_subject=subject,
        ack_policy=AckPolicy.EXPLICIT,
        ack_wait=30,
        max_deliver=5,
    )
)

async def handler(msg):
    try:
        await process(msg.data)
        await msg.ack()
    except RetryableError:
        await msg.nak()
    except Exception:
        await msg.term()

await consumer.consume(handler)
```

In these examples:
- `process` represents application-specific business logic.
- `Ack()` signals successful completion; JetStream marks the message as acknowledged.
- `Nak()` signals a retryable error; JetStream reschedules delivery.
- `InProgress()` resets the `AckWait` timer during long-running processing to prevent premature redelivery.
- `Term()` signals an unrecoverable failure; JetStream ceases further delivery attempts for that message.

### Pattern Constraints

- **Stream requirement:** The subject must be covered by an appropriately configured JetStream stream with a defined storage type (`file` or `memory`), retention policy, and capacity limits.
- **Explicit acknowledgement:** The consumer should use `AckPolicy: Explicit` and acknowledge messages according to their processing outcome. `Ack()` acknowledges successful processing, `Nak()` requests redelivery, `InProgress()` signals ongoing processing and resets the acknowledgement timer, and `Term()` terminates further redelivery for that message.
- **Duplicate delivery possibility:** Network failures, worker crashes, or lost acknowledgements can cause a message to be delivered more than once, even if the previous processing attempt succeeded.
- **Consumer idempotency:** Applications should make processing idempotent where duplicate business effects are unacceptable. Techniques include database uniqueness constraints, transaction keys, and deduplication records.
- **Bounded delivery attempts:** When `MaxDeliver` is configured, JetStream limits delivery attempts. Reaching the limit generates a delivery-limit advisory, but does not automatically move the message to a dead-letter stream or remove it from the stream. Recovery and DLQ handling require a separate strategy.
- **Acknowledgement confirmation:** A regular `Ack()` does not confirm that JetStream received and processed the acknowledgement. Use `AckSync()` when confirmation of acknowledgement receipt is required.

**Pattern contract:** JetStream durably stores accepted messages according to the stream's storage and replication configuration and supports redelivery of unacknowledged messages according to consumer configuration. Applications must account for duplicate deliveries, handle processing failures, and implement idempotency or deduplication where repeated business effects are unacceptable.

## 4. Failure Handling

Redelivery of unacknowledged messages is a fundamental characteristic of At Least Once delivery. Actual behavior depends on stream retention, consumer acknowledgement settings, delivery limits, and storage replication configuration.

| Failure scenario | Expected behavior | Application implication |
|---|---|---|
| Consumer is offline when a message is published | If the stream captures and retains the message, it remains available according to the stream's retention policy. | A durable consumer can resume delivery according to its configured delivery policy and acknowledgement state. |
| Consumer crashes before or during processing | If no acknowledgement is received, JetStream can redeliver the message after `AckWait` expires, subject to delivery limits. | Processing may run again. Make business operations idempotent where duplicate effects are unacceptable. |
| Processing succeeds, but the acknowledgement is not recorded | JetStream may redeliver the message because it cannot confirm that processing was acknowledged. | Duplicate business operations are possible. Use idempotency or application-level deduplication. |
| Transient dependency failure (database or RPC timeout) | The application can send `Nak()` to request redelivery. Redelivery is generally immediate unless a delay is specified. | Classify transient failures and use controlled retry behavior to avoid overwhelming unavailable dependencies. |
| Processing duration exceeds `AckWait` | JetStream may redeliver the message while the original processing attempt is still running. | Send `InProgress()` periodically for long-running work, and prevent concurrent duplicate execution where necessary. |
| Poison message or unrecoverable data corruption | The application can send `Term()` to stop further redelivery, or JetStream can stop automatic delivery attempts after `MaxDeliver` is reached. A delivery-limit advisory is generated when the limit is reached. | Define a remediation strategy for failed messages. Reaching `MaxDeliver` does not automatically move a message to a DLQ or remove it from the stream. |
| Publisher loses connection before receiving `PubAck` | The publication outcome is uncertain: JetStream may have stored the message even though the publisher did not receive the acknowledgement. | Retrying may create a duplicate stream message. Use `Nats-Msg-Id` deduplication where appropriate, noting that deduplication is limited to the configured duplicate window. |
| NATS server node crashes in a clustered environment | JetStream can continue through failover if the stream's replica group retains a quorum and the required replicas are available. | Configure suitable stream replication and validate failure scenarios. Do not assume every server failure is lossless or that the cluster can continue without quorum. |

### Recommended Response

- Make consumer processing idempotent using natural unique keys, database constraints, upserts, or transaction-state checks.
- Configure `AckWait` based on expected processing duration, and use `InProgress()` for long-running tasks.
- Set `MaxDeliver` according to the application's retry policy and define how messages that reach the limit will be handled.
- Make retry behavior deliberate: use `Nak()` for retryable failures and `Term()` for messages that should not be redelivered.
- Use `Nats-Msg-Id` when publisher-side deduplication is needed, and understand the configured deduplication window.
- Configure stream replicas according to availability and durability requirements.

## 5. Decision Signal

**Choose NATS JetStream At Least Once when durable message storage and recoverable delivery are required, and the application can handle duplicate deliveries.**

JetStream can retain messages and redeliver unacknowledged messages according to stream and consumer configuration. It does not, by itself, guarantee that every business operation completes successfully or executes only once.

If stronger protection against duplicate publication and acknowledgement uncertainty is required, evaluate JetStream Exactly Once capabilities, including publisher message-ID deduplication and synchronous consumer acknowledgement (`DoubleAck`). Application-level idempotency may still be required to protect business operations from duplicate effects.