# Exactly Once Delivery — NATS JetStream

## 1. Purpose and Use Cases

Exactly Once delivery in NATS JetStream combines **publisher-side message deduplication** with **synchronous consumer acknowledgement using `DoubleAck`**. These mechanisms suppress duplicate stream entries for repeated publications within the configured duplicate window and allow consumers to confirm that JetStream processed their acknowledgement.

On the publishing side, the publisher attaches a stable message identifier using the `Nats-Msg-Id` header. JetStream tracks message identifiers within the stream's configured duplicate window. When the same identifier is published again within that window, JetStream returns a `PubAck` containing the existing stream sequence and `Duplicate=true`, without storing another stream entry.

On the consuming side, the consumer uses explicit acknowledgements and calls `DoubleAck` after successful processing. This request-reply interaction confirms that JetStream processed the acknowledgement. If confirmation is not received, the consumer cannot assume that the acknowledgement was recorded or that it was not recorded.

**Use Exactly Once delivery when publisher deduplication and confirmed consumer acknowledgement are important, while recognizing that external business operations still require appropriate idempotency controls.**

**Suitable use cases**

- Financial, billing, and payment-related events where repeated publication must not create duplicate stream entries, with separate safeguards against duplicate financial operations.
- API webhooks and ingestion gateways where clients may retry publications after network timeouts.
- Business lifecycle events where duplicate publication must be suppressed and downstream processing must be recoverable.
- Distributed services communicating over unreliable network links where publication outcomes may be uncertain.
- Event-driven workflows where consumers need confirmation that JetStream processed their acknowledgement.

**Not suitable when**

- The workload prioritizes minimum latency and maximum throughput over publisher deduplication and synchronous acknowledgement confirmation.
- The publisher cannot assign a stable, unique identifier to each logical message and reuse it when retrying that publication.
- Publication retries may occur after the configured duplicate window and the application has no additional deduplication mechanism.
- The application expects NATS to provide an atomic transaction across JetStream and an external database, payment provider, or other service.

For workloads that can tolerate duplicate deliveries and handle them through application-level idempotency, consider JetStream At Least Once delivery. For transient messaging where message loss is acceptable and persistence is unnecessary, consider NATS Core At Most Once delivery.

## 2. Message Flow

The example uses `order-service` as the publisher and `processing-service` as the consumer.

```sequence
title Exactly Once Delivery - NATS JetStream

participant "order-service" as os
participant "nats / JetStream" as nats
participant "processing-service" as ps

== Publishing ==

os -> nats: PublishMsg()
note over os, nats: Subject: delivery.exactly-once\nHeader: Nats-Msg-Id = order-event-101

nats -> nats: Check message ID within duplicate window

alt New message ID
    nats -> nats: Store message in Stream
    nats -> os: PubAck{Stream, Sequence, Duplicate=false}
else Duplicate message ID
    nats -> os: PubAck{Stream, Sequence, Duplicate=true}
    note over nats, os: Existing stream entry acknowledged;\nno new entry stored
end

== Consuming ==

ps -> nats: Consume()
note over ps, nats: Durable Consumer: exactly-once\nAckPolicy: Explicit

nats -> ps: Deliver NatsMsg
ps -> ps: Process(NatsMsg)
ps -> nats: DoubleAck()
nats -> ps: Ack confirmation

note over ps, nats: Confirms JetStream processed the acknowledgement
note over ps: External business effects may still require\napplication-level idempotency
```

The publisher receives a `PubAck` confirming that JetStream accepted the publication or recognized it as a duplicate within the configured duplicate window.

The consumer processes the message and calls `DoubleAck` to request confirmation that JetStream processed the acknowledgement. This confirmation concerns the consumer acknowledgement, not the success or uniqueness of external business side effects.

**Important:** Publisher deduplication and consumer acknowledgement confirmation address different failure scenarios. Publisher deduplication suppresses repeated stream entries within the duplicate window. `DoubleAck` confirms acknowledgement processing. Neither mechanism atomically coordinates JetStream with an external database or service.

## 3. Implementation

### Publisher

`order-service` assigns a stable identifier to each logical message through the `Nats-Msg-Id` header. The identifier must be reused when retrying the same publication. Different logical messages must have different identifiers.

The publish operation returns a `PubAck` containing the stream name, sequence number, and duplicate indicator.

**Go**

```go
msg := &nats.Msg{
    Subject: subject,
    Data:    payload,
    Header:  nats.Header{
        nats.MsgIdHdr: []string{"order-event-101"},
    },
}

ack, err := js.PublishMsg(ctx, msg)
if err != nil {
    // Handle the publish error.
    // If the outcome is uncertain, retry the same logical
    // publication with the same message ID.
    return err
}

if ack.Duplicate {
    // JetStream recognized a duplicate publication.
    // No new stream entry was stored.
}
```

**Java**

```java
Headers headers = new Headers();
headers.put("Nats-Msg-Id", "order-event-101");

Message msg = NatsMessage.builder()
    .subject(subject)
    .headers(headers)
    .data(payload)
    .build();

try {
    PublishAck ack = js.publish(msg);

    if (ack.isDuplicate()) {
        // JetStream recognized a duplicate publication.
    }
} catch (Exception e) {
    // Handle the publish error.
    // Reuse the same message ID when retrying
    // the same logical publication.
}
```

**Python**

```python
try:
    ack = await js.publish(
        subject=subject,
        payload=payload,
        headers={"Nats-Msg-Id": "order-event-101"},
    )

    if ack.duplicate:
        # JetStream recognized a duplicate publication.
        pass

except Exception as exc:
    # Handle the publish error.
    # Reuse the same message ID when retrying
    # the same logical publication.
    print(f"Publish failed: {exc}")
```

Inspect the duplicate indicator in the returned `PubAck`. When it is `true`, JetStream has recognized the publication as a duplicate and returns the existing stream sequence rather than creating another stream entry.

A successful publication acknowledgement does not confirm that a consumer has processed the message.

### Subscriber

`processing-service` creates or binds to a durable consumer configured with explicit acknowledgements (`AckPolicy: Explicit`). After successfully processing a message, the consumer calls `DoubleAck` using the synchronous acknowledgement API supported by its SDK.

If business processing fails, the application should classify the failure and choose an appropriate response. Retryable failures can use `Nak()`; unrecoverable messages can use `Term()`. If `DoubleAck` fails or times out, treat the acknowledgement outcome as uncertain rather than automatically classifying the business operation as failed.

**Go**

```go
consumer, err := js.CreateOrUpdateConsumer(
    ctx,
    streamName,
    jetstream.ConsumerConfig{
        Durable:       consumerName,
        FilterSubject: subject,
        AckPolicy:     jetstream.AckExplicitPolicy,
        AckWait:       30 * time.Second,
        MaxDeliver:    5,
    },
)
if err != nil {
    // Handle consumer configuration error.
    return err
}

cc, err := consumer.Consume(func(msg jetstream.Msg) {
    if err := process(msg.Data()); err != nil {
        if isRetryable(err) {
            _ = msg.Nak()
        } else {
            _ = msg.Term()
        }
        return
    }

    // Confirm that JetStream processed the acknowledgement.
    if err := msg.DoubleAck(ctx); err != nil {
        // Confirmation was not received.
        // The acknowledgement outcome is uncertain.
        // Do not assume the business operation failed.
        return
    }
})
if err != nil {
    // Handle consume error.
    return err
}

_ = cc
```

**Java**

The Java client uses its SDK-specific synchronous acknowledgement method. The example below keeps processing failures separate from acknowledgement-confirmation failures.

```java
ConsumerConfiguration config = ConsumerConfiguration.builder()
    .durable(consumerName)
    .filterSubject(subject)
    .ackPolicy(AckPolicy.Explicit)
    .ackWait(Duration.ofSeconds(30))
    .maxDeliver(5)
    .build();

ConsumerContext consumer =
    js.getStreamContext(streamName).createOrUpdateConsumer(config);

MessagesContext messages = consumer.consume(msg -> {
    try {
        process(msg.getData());
    } catch (RetryableException e) {
        msg.nak();
        return;
    } catch (Exception e) {
        msg.term();
        return;
    }

    try {
        // Use the SDK's synchronous acknowledgement API
        // to perform the DoubleAck interaction.
        msg.ackSync(Duration.ofSeconds(5));
    } catch (Exception ackError) {
        // Confirmation was not received.
        // The acknowledgement outcome is uncertain.
        // Do not treat this as a business-processing failure.
    }
});
```

**Python**

The Python client also uses its SDK-specific synchronous acknowledgement method. Keep acknowledgement-confirmation errors separate from exceptions raised by business processing.

```python
consumer = await js.create_or_update_consumer(
    stream=stream_name,
    config=ConsumerConfig(
        durable_name=consumer_name,
        filter_subject=subject,
        ack_policy=AckPolicy.EXPLICIT,
        ack_wait=30,
        max_deliver=5,
    ),
)

async def handler(msg):
    try:
        await process(msg.data)
    except RetryableError:
        await msg.nak()
        return
    except Exception:
        await msg.term()
        return

    try:
        # Use the SDK's synchronous acknowledgement API
        # to perform the DoubleAck interaction.
        await msg.ack_sync(timeout=5)
    except Exception:
        # Confirmation was not received.
        # The acknowledgement outcome is uncertain.
        # Do not treat this as a business-processing failure.
        return

await consumer.consume(handler)
```

**Implementation notes**

- `process` represents application-specific business logic.
- `DoubleAck` requests confirmation that JetStream processed the consumer acknowledgement. SDK method names may differ.
- For long-running work, send `InProgress()` periodically while processing continues. A single call does not protect an arbitrarily long operation from acknowledgement timeout.
- `Nak()` requests redelivery, generally immediately unless a delay is specified.
- `Term()` terminates further redelivery for that message. It does not indicate successful processing.
- `MaxDeliver` limits automatic delivery attempts; reaching the limit does not automatically move a message to a dead-letter stream.
- If acknowledgement confirmation is uncertain, application-level idempotency is essential because the business operation may already have completed.

### Pattern Constraints

- **Duplicate window:** JetStream tracks publisher message IDs within a configured duplicate window. Ensure the effective window covers the expected publication retry interval. A repeated identifier submitted after the window expires may be accepted as a new message.
- **Stable message identifiers:** Use a unique identifier for each logical publication and reuse it across retries. Do not use a non-unique entity identifier for multiple distinct events belonging to the same entity.
- **Explicit acknowledgements:** Configure the consumer with `AckPolicy: Explicit` and acknowledge messages according to their processing outcome.
- **Synchronous confirmation:** `DoubleAck` adds a request-reply interaction and associated latency compared with an asynchronous acknowledgement. Actual performance depends on network conditions, replication, and server configuration.
- **Duplicate delivery:** Publisher deduplication does not eliminate all possible consumer redeliveries. A message may be delivered again when processing succeeds but acknowledgement recording or confirmation is uncertain.
- **Application boundary:** JetStream's messaging mechanisms do not provide an atomic transaction spanning message acknowledgement and external database or service mutations. Protect such operations with idempotency, transactional state changes, or application-level deduplication.

**Pattern contract**

The publisher supplies a stable message identifier for each logical publication and reuses it across retries. JetStream suppresses duplicate stream entries within the configured duplicate window. The consumer uses explicit acknowledgement and `DoubleAck` to confirm acknowledgement processing. Applications remain responsible for handling uncertain outcomes and preventing unacceptable duplicate business effects.

## 4. Failure Handling

Exactly Once delivery reduces duplicate stream entries and provides consumer acknowledgement confirmation, but applications must still handle retry windows, timeouts, and crash boundaries.

| Failure scenario | Expected behavior | Application implication |
|---|---|---|
| Publisher retries within the duplicate window | JetStream recognizes the same `Nats-Msg-Id` and returns `Duplicate=true` with the existing stream sequence. | The retry does not create another stream entry. |
| Publisher retries after the duplicate window expires | JetStream may treat the identifier as new and store another stream entry. | Set the duplicate window to cover the expected retry interval or use additional application-level deduplication. |
| Publisher loses connection before receiving `PubAck` | The message may have been stored even though the publisher did not receive confirmation. | Retry the same logical publication with the same message ID. Deduplication can reconcile the retry only while the ID remains within the duplicate window. |
| Consumer crashes before acknowledging | If no acknowledgement is recorded, JetStream can redeliver the message after `AckWait` expires, subject to consumer configuration and delivery limits. | Business processing may run again. Use idempotency where duplicate effects are unacceptable. |
| Business processing succeeds but acknowledgement confirmation is lost | JetStream may have recorded the acknowledgement even though the consumer did not receive confirmation. | Do not infer the acknowledgement outcome solely from the timeout. Protect external side effects against duplicate execution. |
| Network partition or timeout during `DoubleAck` | The consumer receives no confirmation. The acknowledgement may or may not have been recorded. | Treat the outcome as uncertain. Avoid classifying the business operation as failed solely because confirmation timed out. |
| Processing exceeds `AckWait` | JetStream may redeliver the message while the original processing attempt is still running. | Send `InProgress()` periodically for long-running work and prevent unsafe concurrent duplicate execution. |
| Poison message or unrecoverable payload | The application can issue `Term()`, or automatic delivery attempts can stop after `MaxDeliver` is reached. Reaching the delivery limit generates an advisory. | Define a recovery process. Neither `Term()` nor reaching `MaxDeliver` automatically moves the message to a DLQ. |
| JetStream leader failover during `DoubleAck` | The acknowledgement request may complete or time out during the leadership transition. | Handle uncertain confirmation and design processing to tolerate redelivery. Availability and continuity depend on replication and quorum. |

### Recommended Response

- Configure the duplicate window to exceed the expected publisher retry interval.
- Use stable, unique, domain-scoped message identifiers, such as an event UUID that remains unchanged across retries.
- Use `DoubleAck` where confirmation that JetStream processed the acknowledgement is required.
- Implement application-level idempotency using database unique constraints, upserts, transaction keys, or state-transition checks.
- Set `AckWait` based on expected processing duration and send `InProgress()` periodically during long-running work.
- Define retry behavior for retryable and permanent processing failures.
- Establish a separate recovery strategy for messages that reach `MaxDeliver`; do not assume they are automatically moved to a DLQ.
- Configure stream replication according to durability and availability requirements, and validate expected behavior during failures.

## 5. Decision Signal

**Choose NATS JetStream Exactly Once when publisher-side duplicate suppression and confirmed consumer acknowledgement are required.**

This pattern is useful for workloads where duplicate publications are costly and the consumer needs confirmation that JetStream processed its acknowledgement. Its guarantees depend on the duplicate window, stream and consumer configuration, and application behavior.

Exactly Once messaging does not guarantee that every message reaches a consumer under all conditions, nor does it guarantee that an external business operation executes exactly once. Applications must handle uncertain outcomes and protect external side effects with idempotency or other appropriate safeguards.

If the application can tolerate duplicate delivery and safely handle it through idempotent processing, consider JetStream At Least Once delivery. If message persistence is unnecessary and message loss is acceptable, consider NATS Core At Most Once delivery.
