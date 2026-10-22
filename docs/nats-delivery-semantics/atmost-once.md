# At Most Once Delivery — NATS Core

## 1. Purpose and Use Cases

At Most Once delivery is a best-effort messaging pattern using NATS Core. A message may be delivered zero or one times to a given active subscription. Messages are not persisted for later delivery, and failed processing does not trigger automatic redelivery.

Use this pattern when **low latency and transient messaging are more important than guaranteed delivery**.

**Suitable use cases**
- Live status updates where a missed update is acceptable or superseded by a newer update.
- Non-critical notifications where occasional message loss is acceptable.
- Real-time telemetry and transient events.

**Not suitable when**
- Every message must be processed.
- Consumers must recover messages after downtime.
- Failed processing must trigger automatic redelivery.
- The application requires durable message storage or replay.

For these requirements, evaluate JetStream-based delivery patterns. If the publisher must know that business processing completed, an application-level confirmation mechanism is also required.

## 2. Message Flow

The example uses `Order-Service` as the publisher and `Processor-Service` as the subscriber.

```sequence
title At Most Once Delivery - NATS Core

participant "order-service" as os
participant "nats" as nats
participant "processing-service" as ps

== Publishing ==

os -> nats: Publish()
note over os, nats: Subject: delivery.at-most-once

nats -> os: Publish completed

== Consuming ==

nats -> ps: Deliver NatsMsg
ps -> ps: Process(NatsMsg)
ps -> ps: Done with processing
```

**Important:** `Publish completed` represents completion of the publisher-side operation. It does not confirm that `Processor-Service` received or processed the message.

## 3. Implementation

### Publisher

`Order-Service` publishes messages using the standard NATS Core publish operation. Publication does not wait for the subscriber to complete processing.

**Go**

```go
err := nc.Publish(subject, payload)
if err != nil {
    // Handle the publish error.
}
```

**Java**

```java
try {
    nc.publish(subject, payload);
} catch (Exception e) {
    // Handle the publish error.
}
```

**Python**

```python
try:
    await nc.publish(subject, payload)
except Exception as exc:
    # Handle the publish error.
    print(f"Publish failed: {exc}")
```

Handle client-reported errors, but do not assume that an error or timeout always means the message was not received. Retrying an uncertain publication may result in duplicate business effects.

### Subscriber

`Processor-Service` subscribes to the subject and processes messages delivered to its active subscription. Processing failures are the application's responsibility; NATS Core does not automatically redeliver failed messages.

**Go**

```go
_, err := nc.Subscribe(subject, func(msg *nats.Msg) {
    if err := process(msg.Data); err != nil {
        // Handle the processing failure.
    }
})
if err != nil {
    // Handle the subscription error.
}
```

**Java**

```java
Dispatcher dispatcher = nc.createDispatcher(msg -> {
    try {
        process(msg.getData());
    } catch (Exception e) {
        // Handle the processing failure.
    }
});

dispatcher.subscribe(subject);
```

**Python**

```python
async def handler(msg):
    try:
        await process(msg.data)
    except Exception as exc:
        # Handle the processing failure.
        print(f"Processing failed: {exc}")

await nc.subscribe(subject, cb=handler)
```

In these examples, `process` represents application-specific business logic. The snippets illustrate the Core NATS pattern, not complete service implementations.

### Pattern Constraints

- **Transient delivery:** Messages are not persisted for offline subscribers.
- **No processing acknowledgement:** The publisher cannot confirm that the subscriber completed processing.
- **No automatic redelivery:** Messages missed during disconnection or interrupted during processing are not replayed.
- **Application-owned recovery:** If message loss is unacceptable or processing must be recoverable, evaluate a JetStream-based pattern.

**Pattern contract:** The publisher sends a message to NATS Core, which delivers it to matching active subscriptions on a best-effort basis. NATS Core does not persist messages for later delivery, confirm business processing, or automatically redeliver failed messages. Applications must tolerate possible message loss and handle uncertain outcomes when deciding whether to retry.

## 4. Failure Handling

Message loss is an expected property of At Most Once delivery, not an exceptional condition that NATS Core automatically recovers from.

| Failure scenario | Expected behavior | Application implication |
|---|---|---|
| No active subscriber at publication time | The message is not retained for a future subscription. | The event is lost to the intended subscriber; the application must tolerate the loss. |
| Publisher or subscriber loses connectivity | Publisher connectivity loss around publication can leave the outcome uncertain. Messages published while the subscriber is disconnected are not replayed after reconnection. | Publisher retries may duplicate business effects; subscriber reconnection does not recover missed events. |
| Subscriber crashes before or during processing | NATS Core does not automatically redeliver the message. | Processing may remain incomplete, and recovery must be handled by the application. |
| Application processing fails | NATS Core does not detect application-level processing failures or trigger redelivery. | The application must decide whether to retry, log, discard, or route the failed event for recovery. |
| Business processing succeeds, but completion is not recorded | The business effect may have succeeded even though the application cannot confirm completion. | Recovery may repeat the business operation; idempotency or durable processing state may be required. |
| Subscriber cannot keep up with message throughput | Slow-consumer conditions may cause messages to be dropped or the subscriber connection to be closed, depending on client and server behavior. | Ensure the application can tolerate loss under load, or evaluate JetStream if messages must be retained for consumption. |
| NATS server becomes unavailable or restarts | Core NATS does not persist messages for replay across outages. | Assess whether transient delivery meets the availability requirements; evaluate JetStream if persistence and replay are required. |

### Recommended Response

- Accept potential message loss when the business use case permits it.
- Handle processing errors within the application.
- Make retries deliberate when the outcome of a previous attempt is uncertain.
- Use application-level idempotency when repeated business effects are unacceptable.
- Evaluate JetStream when message persistence, replay, or redelivery is required.

## 5. Decision Signal

**Choose NATS Core At Most Once when message loss is acceptable.**

If the application requires messages to survive subscriber downtime, recover after processing failures, or be redelivered, At Most Once is not the appropriate pattern. Evaluate JetStream and select a delivery pattern that meets those requirements.

If the publisher requires confirmation of business processing, implement an application-level confirmation mechanism in addition to selecting an appropriate messaging pattern.