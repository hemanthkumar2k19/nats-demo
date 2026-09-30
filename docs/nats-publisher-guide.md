# NATS Publisher Guide

## Purpose

This guide defines application-level practices for publishing messages to NATS using Core NATS and JetStream.

It covers the publisher lifecycle from initialization through message construction, publishing, failure and retry handling, and graceful shutdown.

---

## 1. Publisher Initialization

### 1.1 NATS Connection

A publisher should use an active, reusable NATS connection.

The connection is responsible for communication with the NATS server and can be safely reused by concurrent publishing operations.

```go
nc, err := nats.Connect(
    "nats://localhost:4222",
    nats.Name("my-service"),
)
if err != nil {
    return err
}
```

### 1.2 JetStream Context

A JetStream context is created from an existing NATS connection when the application requires JetStream publishing.

Creating the JetStream context does not create another network connection.

```go
js, err := jetstream.New(nc)
if err != nil {
    return err
}
```

Conceptually:

```text
NATS Connection
      |
      +-- Core NATS Publisher
      |
      +-- JetStream Context
               |
               +-- JetStream Publisher
```

---

## 2. Message Construction

A publisher constructs a message using:

* Subject
* Payload
* Optional headers
* Optional reply subject (`msg.Reply`)

```go
msg := nats.NewMsg("orders.created")
msg.Data = []byte(`{"orderId":"12345"}`)

// Optional headers
msg.Header.Set("Content-Type", "application/json")

// Optional reply subject (used by responders to send responses back)
msg.Reply = "orders.reply.inbox"
```

The message structure is independent of whether it is subsequently published through Core NATS or JetStream.

Publisher-specific metadata such as a JetStream message ID (`Nats-Msg-Id`) or a Core NATS reply subject (`msg.Reply`) should be added according to the publishing pattern being used.

---

## 3. Core NATS Publishing

Core NATS provides ephemeral publish semantics. A Core NATS publish does not wait for a server-side publish acknowledgement or subscriber acknowledgement.

### 3.1 Basic Publish

Use `Publish` when only a subject and payload are required.

```go
err := nc.Publish(
    "orders.created",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}
```

The client queues the message for transmission and the publish operation returns without waiting for subscriber confirmation.

Core NATS does not provide JetStream-style publish acknowledgements or persistence.

---

### 3.2 Structured Message Publish

Use `PublishMsg` when headers or an optional reply subject are required on the message.

```go
msg := nats.NewMsg("orders.created")
msg.Data = []byte(`{"orderId":"12345"}`)
msg.Header.Set("Content-Type", "application/json")

// Optional reply subject attached to the structured message
msg.Reply = "orders.reply.inbox"

err := nc.PublishMsg(msg)
if err != nil {
    return err
}
```

---

### 3.3 Request-Reply Publish

A publisher can create a request message containing an explicitly supplied reply subject using `PublishRequest`.

```go
err := nc.PublishRequest(
    "orders.validate",
    "_INBOX.response",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}
```

`PublishRequest` publishes the request with the specified reply subject (`_INBOX.response`). It does **not** wait for the response.

Response subscription and response handling belong to the request/reply interaction rather than the publishing operation itself.

---

### 3.4 Flush

Use `Flush` when the application needs confirmation that pending client operations have been processed by the server.

```go
if err := nc.Flush(); err != nil {
    return err
}
```

`Flush` is useful when:

* validating connectivity
* coordinating shutdown
* confirming that pending client operations have reached the server

`Flush` does not provide message persistence or subscriber acknowledgement.

---

### 3.5 Concurrent Publishing

A shared NATS connection can be used by concurrent publisher routines.

```go
go func() {
    _ = nc.Publish("orders.created", payload)
}()

go func() {
    _ = nc.Publish("orders.updated", payload)
}()
```

A separate connection should not be created for every publish operation.

---

## 4. JetStream Publishing

JetStream publishing provides server-side acknowledgement and persistence according to the configured stream and storage policy.

### 4.1 Synchronous Publish

Use synchronous publishing when the application needs the JetStream publish acknowledgement before continuing.

```go
ack, err := js.Publish(
    context.Background(),
    "orders.created",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}

fmt.Println(ack.Stream, ack.Sequence)
```

The publish operation waits for the JetStream publish acknowledgement.

The acknowledgement can provide information such as:

* Stream
* Sequence
* Duplicate detection result

---

### 4.2 Asynchronous Publish

Use asynchronous publishing when the application wants to continue processing without waiting for every individual publish acknowledgement.

```go
future, err := js.PublishAsync(
    "orders.created",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}

ack, err := future.Ok()
if err != nil {
    return err
}
```

Applications using asynchronous publishing should account for outstanding publishes before shutdown.

---

### 4.3 Managing Pending Publishes

JetStream provides APIs to inspect and wait for asynchronous publishing activity.

```go
pending := js.PublishAsyncPending()

if pending > 0 {
    <-js.PublishAsyncComplete()
}
```

The application should define an appropriate shutdown boundary so that pending publishing work is either completed or explicitly abandoned.

---

### 4.4 Publish Deduplication

JetStream supports duplicate publish detection using the `Nats-Msg-Id` header.

```go
msg := nats.NewMsg("orders.created")
msg.Header.Set("Nats-Msg-Id", "order-12345")
msg.Data = []byte(`{"orderId":"12345"}`)

ack, err := js.PublishMsg(context.Background(), msg)
if err != nil {
    return err
}

if ack.Duplicate {
    // Server detected that this message ID was already processed
}
```

When retrying the same logical message, retain the same `Nats-Msg-Id`.

The server can then detect a duplicate publish within the applicable duplicate-detection window.

---

### 4.5 Publish Expectations

JetStream allows publishers to specify expectations about the stream state before accepting a publish.

Examples include:

* Expected last sequence
* Expected last message ID

```go
ack, err := js.Publish(
    context.Background(),
    "orders.created",
    payload,
    jetstream.WithExpectLastSequence(100),
)
```

These expectations can be used when publishing depends on a known stream state.

---

## 5. Failure Handling

Publish failures should be classified before deciding whether to retry.

```text
Publish
   |
   +-- Success
   |
   +-- Failure
        |
        +-- Permanent -> Fail
        |
        +-- Transient -> Retry
        |
        +-- Unknown Outcome -> Apply Duplicate Strategy
```

Typical failure categories include:

| Failure | Typical Handling |
| :--- | :--- |
| Invalid subject / request | Fail |
| Authentication / authorization failure | Fail and correct configuration |
| Connection unavailable | Allow reconnect / retry where appropriate |
| JetStream API timeout | Evaluate and retry if operation outcome is safe |
| Stream or publish expectation failure | Re-evaluate application state |
| Storage/server failure | Retry according to operational policy |

For JetStream publishing, an error does not always mean that the server definitely did not process the logical message. Retry design should therefore consider duplicate handling.

---

## 6. Retry Handling

Retries should be used for failures that are potentially transient.

The retry policy should define:

* Maximum retry attempts
* Retry delay / backoff
* Retryable errors
* Maximum retry duration
* Duplicate handling

Use bounded retries rather than retrying indefinitely.

```text
Publish
   |
   v
Failure
   |
   v
Is failure retryable?
   +-- No -> Fail
   |
   +-- Yes
        |
        v
     Wait / Backoff
        |
        v
      Retry
```

For JetStream, when retrying the same logical message, retain the same `Nats-Msg-Id` where duplicate detection is part of the retry strategy.

---

## 7. Graceful Shutdown

Publishers should stop accepting new work before shutting down the NATS connection.

A typical shutdown sequence is:

```text
Stop New Publishing
       |
       v
Complete Application-Level Pending Work
       |
       v
Wait for JetStream Async Publishes (if applicable)
       |
       v
Flush Core NATS Operations (if applicable)
       |
       v
Drain NATS Connection
       |
       v
Close
```

For JetStream asynchronous publishing:

```go
<-js.PublishAsyncComplete()
```

For Core NATS pending operations:

```go
if err := nc.Flush(); err != nil {
    // handle shutdown error
}
```

The connection can then be drained:

```go
if err := nc.Drain(); err != nil {
    // handle shutdown error
}
```

`Drain` provides graceful connection shutdown by allowing pending client work to be processed before the connection closes.

When the connection and server remain healthy, this gives in-flight publications an opportunity to reach the server before shutdown.

`Drain` does not by itself guarantee message durability or delivery.

Applications should use a bounded shutdown timeout rather than waiting indefinitely.

---

## 8. Publisher Capability Summary

| Capability | Core NATS | JetStream |
| :--- | :--- | :--- |
| Basic publish | Yes | Yes |
| Structured message | Yes | Yes |
| Headers | Yes | Yes |
| Reply subject | Yes | Yes |
| Server publish acknowledgement | - | Yes |
| Synchronous publish | - | Yes |
| Asynchronous publish | Client-side | Yes |
| Publish acknowledgement | - | Yes |
| Duplicate detection | - | Yes |
| Publish expectations | - | Yes |
| Persistent storage | - | Yes |
| Flush | Yes | Yes |
| Connection drain | Yes | Yes |

---

## 9. Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client API Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go)
* [NATS Go JetStream Package Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
* [Official NATS Developer Documentation](https://docs.nats.io/using-nats/developer)
