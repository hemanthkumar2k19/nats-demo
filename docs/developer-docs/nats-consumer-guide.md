# NATS JetStream Consumer Usage Guide

## 1. Overview

This guide explains how application developers consume messages from NATS JetStream using provisioned consumers. It covers pull and push delivery, message acknowledgement, processing concurrency, ordering, and common issues.

Streams and Consumers are provisioned and managed through the approved platform process. Subject usage and filtering follow the platform's naming and configuration conventions. Developers should request new resources or configuration changes through the approved [Consumer Intake Form](nats-resources/consumer-intake.md) and use the resulting configuration in their applications.

Read this guide first to understand how to consume messages. Then review the **Delivery Semantics** guide to understand At-Most-Once, At-Least-Once, and Exactly-Once delivery guarantees.

### Coverage Matrix

| Topic | Go | Java | Python 3 |
|---|---|---|---|
| Pull consumer (Continuous) | `consumer.Consume` | `consumerContext.consume` | Loop with `fetch` |
| Pull consumer (Batch) | `consumer.Fetch` | `consumerContext.fetchMessages` | `psub.fetch` |
| Push consumer | Core `Subscribe` to deliver subject | Core `subscribe` to deliver subject | Core `subscribe` to deliver subject |
| Acknowledgements | `Ack`, `Nak`, `Term`, `InProgress` | `ack`, `nak`, `term`, `inProgress` | `ack`, `nak`, `term`, `in_progress` |
| Sequential processing | Sequential message loop | Sequential message loop | Sequential async loop |
| Concurrent processing | Goroutines with bounded workers | Executor or worker pool | Bounded `asyncio` workers |
| Ordered Consumer | `OrderedConsumer` | Ordered subscription | Ordered subscription (`ordered_consumer=True`) |

The code snippets focus on consumer operations and processing behaviour. They assume that the NATS connection and JetStream client are already available, and that the consumer has been provisioned or is an Ordered Consumer created through the supported client API.

SDK APIs differ by version. Confirm the signatures against the versions approved by your platform before using the snippets as copy-and-run examples.

## 2. Before You Start

Obtain the following details from the platform team:

- NATS endpoint and authentication method.
- Stream name and provisioned consumer name.
- Subject filter, where applicable.
- Delivery mode: pull or push.
- Deliver subject, where applicable (required for Push Consumers).
- Acknowledgement policy and retry expectations.
- Approved SDK and runtime versions.

If the required consumer is unavailable, submit a request through the approved [Consumer Intake Form](nats-resources/consumer-intake.md). Do not create, update, reset, or delete shared resources from application code.

## 3. Pull and Push Consumers

The provisioned consumer determines the delivery mode your application should use.

### 3.1 Pull Consumers

A pull consumer lets the application request messages when it is ready to process them. It has no server-configured delivery subject. Modern SDKs support two pull consumption models:

1. **Continuous processing (`Consume`)**: An asynchronous client-managed loop that pulls messages and invokes a callback handler with client-side backpressure and flow control.
2. **On-demand batch retrieval (`Fetch`)**: Explicitly pulls a bounded batch of messages on schedule or demand.

**When to use pull**

- The application needs control over batch size and request timing.
- Processing capacity varies or depends on downstream services.
- The application needs backpressure or controlled concurrency.
- The workload requires explicit retry and failure handling.

Pull consumers are the recommended starting point for almost all modern application workloads.

#### Continuous Processing (Callback-driven Pull)

**Go — Continuous consumption (`Consume`)**

```go
consumer, err := js.Consumer(ctx, "ORDERS", "orders-pull")
if err != nil {
    return err
}

consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
    if err := process(msg.Data()); err != nil {
        _ = msg.Nak()
        return
    }
    _ = msg.Ack()
})
if err != nil {
    return err
}
defer consumeCtx.Stop()
```

**Java — Continuous consumption (`consume`)**

```java
StreamContext streamContext = nc.getStreamContext("ORDERS");
ConsumerContext consumerContext = streamContext.getConsumerContext("orders-pull");

MessageConsumer consumer = consumerContext.consume(msg -> {
    try {
        process(msg.getData());
        msg.ack();
    } catch (Exception e) {
        msg.nak();
    }
});
```

**Python 3 — Continuous consumption (`asyncio` loop with `fetch`)**

In Python (`nats-py`), continuous pull consumption is driven via an explicit `asyncio` task loop rather than a background thread callback, ensuring execution remains cooperative, non-blocking, and cancellation-safe within the application's event loop:

```python
sub = await js.pull_subscribe(
    "orders.created",
    durable="orders-pull",
    stream="ORDERS",
)

while True:
    try:
        messages = await sub.fetch(batch=10, timeout=2)
        for msg in messages:
            try:
                await process(msg.data)
                await msg.ack()
            except Exception:
                await msg.nak()
    except asyncio.TimeoutError:
        # Timeout occurs when no messages are available in the window; continue loop
        continue
```

#### On-Demand Batch Retrieval (`Fetch`)

**Go — Fetch a batch**

```go
consumer, err := js.Consumer(ctx, "ORDERS", "orders-pull")
if err != nil {
    return err
}

msgs, err := consumer.Fetch(10)
if err != nil {
    return err
}

for msg := range msgs.Messages() {
    if err := process(msg.Data()); err != nil {
        _ = msg.Nak()
        continue
    }
    _ = msg.Ack()
}
```

**Java — Fetch a batch**

```java
StreamContext streamContext = nc.getStreamContext("ORDERS");
ConsumerContext consumerContext = streamContext.getConsumerContext("orders-pull");

try (FetchConsumer fetchConsumer = consumerContext.fetchMessages(10)) {
    Message msg;
    while ((msg = fetchConsumer.nextMessage()) != null) {
        try {
            process(msg.getData());
            msg.ack();
        } catch (Exception e) {
            msg.nak();
        }
    }
}
```

**Python 3 — Fetch a batch**

```python
sub = await js.pull_subscribe(
    "orders.created",
    durable="orders-pull",
    stream="ORDERS",
)

messages = await sub.fetch(batch=10, timeout=2)

for msg in messages:
    try:
        await process(msg.data)
        await msg.ack()
    except Exception:
        await msg.nak()
```

### 3.2 Push Consumers

A push consumer is configured on the NATS server with a mandatory `DeliverSubject` (for example, `orders.delivery`). JetStream proactively forwards incoming stream messages to that `DeliverSubject`.

The application handles messages by subscribing directly to the consumer's `DeliverSubject`.

> [!NOTE]
> Do not call pull methods like `consumer.Consume()` or `consumer.Fetch()` on a Push Consumer; the SDK will return an error (such as `ErrNotPullConsumer`). For callback-driven event handling with modern flow control, use a **Pull Consumer** with `consumer.Consume()`.

**When to use push**

- The platform has provisioned an existing push consumer with a designated `DeliverSubject`.
- The application needs real-time delivery without initiating client pull requests.
- The existing push configuration meets the application's flow-control and delivery requirements.

**Go — Subscribe to DeliverSubject**

```go
// In Go, subscribe directly to the push consumer's configured DeliverSubject.
// Use nc.QueueSubscribe when running multiple worker instances for load balancing.
sub, err := nc.Subscribe("orders.delivery", func(msg *nats.Msg) {
    if err := process(msg.Data); err != nil {
        _ = msg.Nak()
        return
    }
    _ = msg.Ack()
})
if err != nil {
    return err
}
defer sub.Drain()
```

**Java — Subscribe to DeliverSubject**

```java
// In Java, listen on the DeliverSubject using a Dispatcher.
Dispatcher dispatcher = nc.createDispatcher();
Subscription sub = nc.subscribe(
    "orders.delivery",
    dispatcher,
    msg -> {
        try {
            process(msg.getData());
            msg.ack();
        } catch (Exception e) {
            msg.nak();
        }
    }
);
```

**Python 3 — Subscribe to DeliverSubject**

```python
async def handle_message(msg):
    try:
        await process(msg.data)
        await msg.ack()
    except Exception:
        await msg.nak()

# In Python, subscribe directly to the push consumer's deliver_subject
sub = await nc.subscribe(
    "orders.delivery",
    cb=handle_message,
)
```

In all languages, messages received on a JetStream push consumer delivery subject are standard NATS messages that support explicit JetStream acknowledgements (`msg.Ack()`, `msg.Nak()`, `msg.Term()`).

### 3.3 Quick Comparison

| Requirement | Recommended approach |
|---|---|
| Control when and how many messages are requested | Pull |
| Apply backpressure based on processing capacity | Pull |
| Receive messages continuously through a configured delivery subject | Push |
| Observe messages in sequence without application acknowledgements | Ordered Consumer |
| Process independent messages concurrently | Pull is often a good starting point |

## 4. Message Acknowledgements

For consumers configured to require acknowledgements, acknowledge a message after the relevant business operation succeeds. An acknowledgement tells JetStream how the application handled the message; it does not automatically make the business operation exactly-once.

### 4.1 Acknowledgement Actions

| Outcome | Go | Java | Python 3 |
|---|---|---|---|
| Processing succeeded | `msg.Ack()` | `msg.ack()` | `await msg.ack()` |
| Retry the message | `msg.Nak()` | `msg.nak()` | `await msg.nak()` |
| Do not redeliver | `msg.Term()` | `msg.term()` | `await msg.term()` |
| Processing is still in progress | `msg.InProgress()` | `msg.inProgress()` | `await msg.in_progress()` |

Use `Term` only when the application has decided that the message should not be retried. Use `InProgress` for work that may exceed the configured acknowledgement wait period, where supported by the consumer configuration.

### 4.2 Example: Success, Retry, or Permanent Failure

**Go**

```go
switch {
case isValid(msg.Data()):
    if err := process(msg.Data()); err != nil {
        _ = msg.Nak()
        return
    }
    _ = msg.Ack()

case isPermanentlyInvalid(msg.Data()):
    logInvalidMessage(msg.Data())
    _ = msg.Term()

default:
    _ = msg.Nak()
}
```

**Java**

```java
if (isPermanentlyInvalid(msg.getData())) {
    logInvalidMessage(msg.getData());
    msg.term();
} else {
    try {
        process(msg.getData());
        msg.ack();
    } catch (Exception e) {
        msg.nak();
    }
}
```

**Python 3**

```python
if is_permanently_invalid(msg.data):
    log_invalid_message(msg.data)
    await msg.term()
else:
    try:
        await process(msg.data)
        await msg.ack()
    except Exception:
        await msg.nak()
```

The validation and processing functions are application-specific. Classify failures deliberately rather than treating every error as permanent or retryable.

### 4.3 Long-running Processing

If processing may exceed the acknowledgement wait period, send progress acknowledgements while work continues, where supported.

**Go**

```go
_ = msg.InProgress()

if err := process(msg.Data()); err != nil {
    _ = msg.Nak()
    return
}
_ = msg.Ack()
```

**Java**

```java
msg.inProgress();

try {
    process(msg.getData());
    msg.ack();
} catch (Exception e) {
    msg.nak();
}
```

**Python 3**

```python
await msg.in_progress()

try:
    await process(msg.data)
    await msg.ack()
except Exception:
    await msg.nak()
```

For long-running work, progress acknowledgements may need to be sent repeatedly until processing completes. Configure acknowledgement timing with the platform team if the existing settings do not fit the workload.

### 4.4 Prevent Duplicate Business Effects

Messages can be redelivered if JetStream does not receive an acknowledgement, even if the application completed its business operation. Make processing idempotent where practical, using business identifiers, database constraints, or deduplication records.

Review the **Delivery Semantics** guide for the implications of acknowledgements, redelivery, and exactly-once guarantees.

## 5. Message Ordering and Processing Concurrency

Stream sequence, delivery order, processing order, and completion order are not interchangeable. Choose a processing pattern based on the application's business requirements.

### 5.1 Sequential Processing

Sequential processing starts the next message only after the current message has finished processing.

**When to use sequential processing**

- Each operation depends on the preceding operation.
- Business state must be updated in a defined order.
- Simpler processing and failure handling are more important than maximum parallel throughput.

**Go**

```go
for msg := range msgs.Messages() {
    if err := process(msg.Data()); err != nil {
        _ = msg.Nak()
        continue
    }
    _ = msg.Ack()
}
```

**Java**

```java
for (Message msg : messages) {
    try {
        process(msg.getData());
        msg.ack();
    } catch (Exception e) {
        msg.nak();
    }
}
```

**Python 3**

```python
for msg in messages:
    try:
        await process(msg.data)
        await msg.ack()
    except Exception:
        await msg.nak()
```

These examples process each batch sequentially. Sequential processing within one application loop does not guarantee global ordering across multiple instances, and redelivery may affect the order in which business operations complete.

### 5.2 Concurrent Processing

Concurrent processing allows multiple messages to be processed at the same time. This can increase throughput, but messages may complete in a different order from the order in which they were received.

**When to use concurrent processing**

- Messages are independent.
- The application benefits from parallel processing.
- The business logic can tolerate completion order differing from delivery order.

**Go — Bounded worker pattern**

```go
workers := 4
jobs := make(chan jetstream.Msg)

for i := 0; i < workers; i++ {
    go func() {
        for msg := range jobs {
            if err := process(msg.Data()); err != nil {
                _ = msg.Nak()
                continue
            }
            _ = msg.Ack()
        }
    }()
}

for msg := range msgs.Messages() {
    jobs <- msg
}
```

**Java — Executor pattern**

```java
ExecutorService workers = Executors.newFixedThreadPool(4);

for (Message msg : messages) {
    workers.submit(() -> {
        try {
            process(msg.getData());
            msg.ack();
        } catch (Exception e) {
            msg.nak();
        }
    });
}
```

**Python 3 — Bounded worker pattern**

```python
async def worker(queue):
    while True:
        msg = await queue.get()
        try:
            await process(msg.data)
            await msg.ack()
        except Exception:
            await msg.nak()
        finally:
            queue.task_done()
```

These are illustrative processing patterns, not complete worker-pool implementations. Production code should bound queued work, propagate errors, and coordinate worker lifecycle. In particular, do not acknowledge a message before its business operation succeeds.

### 5.3 Ordered Consumers

An Ordered Consumer is intended for applications that need to observe a Stream sequentially, such as inspecting or analysing events in order. It is not a general replacement for an acknowledged work-processing consumer.

**When to use an Ordered Consumer**

- You need sequential observation of a Stream.
- You are inspecting, replaying, or analysing event history.
- You do not need application-controlled acknowledgements or load balancing across workers.

Ordered Consumers are ephemeral, do not require acknowledgements, and are not designed to distribute work across multiple workers. Client libraries can recreate the underlying consumer when they detect a sequence gap.

**Go — Create and consume from an Ordered Consumer**

```go
consumer, err := js.OrderedConsumer(
    ctx,
    "ORDERS",
    jetstream.OrderedConsumerConfig{},
)
if err != nil {
    return err
}

consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
    inspect(msg.Data())
})
if err != nil {
    return err
}
defer consumeCtx.Stop()
```

**Java — Ordered push subscription**

```java
PushSubscribeOptions options =
    PushSubscribeOptions.builder()
        .stream("ORDERS")
        .ordered(true)
        .build();

js.subscribe(
    "orders.created",
    dispatcher,
    msg -> inspect(msg.getData()),
    false,
    options
);
```

Ordered push subscriptions have specific restrictions: they are ephemeral, do not use normal acknowledgements, and cannot be used as queue-group load-balanced subscriptions. Confirm the supported options for the Java SDK version in use.

**Python 3 — Ordered Consumer**

```python
sub = await js.subscribe(
    "orders.created",
    ordered_consumer=True,
)

while True:
    try:
        msg = await sub.next_msg(timeout=2)
        inspect(msg.data)
    except asyncio.TimeoutError:
        break
```

In `nats-py`, an Ordered Consumer is configured directly via `ordered_consumer=True` on `js.subscribe()`, which transparently manages ephemeral consumer lifecycle, heartbeats, flow control, and sequence recovery under the hood.

Ordered Consumers are suitable for ordered observation, not for application-controlled retries or acknowledged work distribution. If business operations require acknowledgements, retry handling, or multiple workers, use the appropriate provisioned application consumer.

## 6. Common Consumer Issues

| Symptom | What to check |
|---|---|
| No messages received | Verify Stream, consumer, subject filter, and delivery mode. |
| Messages are redelivered | Check acknowledgement handling and processing duration. |
| Duplicate business effects | Make processing idempotent and review acknowledgement failure scenarios. |
| Messages complete out of order | Check concurrent handlers, multiple instances, and redelivery. |
| Processing falls behind | Review processing latency, worker limits, pending messages, and downstream dependencies. |
| Push subscription does not work | Confirm that the provisioned consumer is push-based and that the SDK binds to the correct configuration. |
| Consumer configuration appears incorrect | Contact the platform team through the approved process; do not modify shared consumers directly. |

## 7. Related Documentation

- **JetStream Guide:** [JetStream Guide](nats-jetstream-guide.md)
- **Consumer Intake Form:** [Consumer Intake Form](nats-resources/consumer-intake.md)
- **JetStream Consumers:** https://docs.nats.io/nats-concepts/jetstream/consumers
- **Go JetStream SDK:** https://github.com/nats-io/nats.go/tree/main/jetstream
- **Java NATS SDK:** https://github.com/nats-io/nats.java
- **Python NATS SDK:** https://github.com/nats-io/nats.py

## 8. Developer Readiness Checklist

### A. Provisioning and Configuration

- [ ] The required consumer has been provisioned through the approved platform process.
- [ ] The application uses the correct Stream, consumer, and subject configuration.
- [ ] The application uses an approved SDK and runtime version.
- [ ] Required credentials and permissions are configured.

### B. Message Consumption

- [ ] The application uses the provisioned delivery mode.
- [ ] Acknowledgements follow the consumer's configured policy.
- [ ] Messages are acknowledged only after relevant business processing succeeds.
- [ ] Retryable and permanent failures have defined handling behaviour.
- [ ] Long-running processing accounts for acknowledgement timing.

### C. Reliability and Ordering

- [ ] Processing can safely handle duplicate deliveries where required.
- [ ] Concurrent processing is not assumed to preserve completion order.
- [ ] Ordering requirements have been addressed in the application design.
- [ ] Redelivery and failure handling have been considered.

### D. Operational Readiness

- [ ] Processing and acknowledgement errors have useful diagnostic logs.
- [ ] Logs do not expose credentials or sensitive payloads.
- [ ] The team knows how to investigate common consumer issues.
- [ ] The team has reviewed the JetStream Guide and Delivery Semantics guide.

**Before release:** Confirm that the application uses the provisioned consumer as intended and that its acknowledgement, failure-handling, and ordering behaviour match the application's requirements.