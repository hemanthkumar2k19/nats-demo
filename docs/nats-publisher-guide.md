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

A NATS message consists of a destination subject, binary payload data, optional headers, and an optional reply subject.

### 2.1 Subject & Target Namespace

Every published message must have a destination subject that complies with subject naming standards (`<domain>.<entity>.<action>`).

```go
msg := nats.NewMsg("orders.created")
```

### 2.2 Payload & Serialization

Application domain objects must be serialized into a binary byte slice (e.g., JSON) before being set on `msg.Data`.

```go
type OrderEvent struct {
    OrderID string `json:"order_id"`
}

event := OrderEvent{OrderID: "12345"}
payload, err := json.Marshal(event)
if err != nil {
    return err
}

msg.Data = payload
```

### 2.3 Headers & Metadata

Use NATS headers (`msg.Header`) for metadata such as content types, correlation IDs, or JetStream deduplication keys (`Nats-Msg-Id`).

```go
msg.Header.Set("Content-Type", "application/json")
msg.Header.Set("X-Correlation-ID", correlationID)
```

### 2.4 Optional Reply Subject

A message can specify an optional reply subject (`msg.Reply`) indicating where a responder can publish a response.

```go
msg.Reply = "orders.reply.inbox"
```

The message structure is independent of whether it is subsequently published through Core NATS or JetStream.

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

### 3.3 Publishing with Reply Subject (`PublishRequest`)

A publisher can send a message with an explicitly attached reply subject using `PublishRequest`.

```go
err := nc.PublishRequest(
    "orders.validate",
    "orders.reply.inbox",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}
```

> **Important Distinction:** `PublishRequest` is an asynchronous publish operation that simply attaches the reply subject to the message. It does **NOT** wait for a response or create a response subscription.

---

### 3.4 Synchronous Request-Reply Pattern (`Request` / `RequestMsg`)

Unlike `PublishRequest`, the true **Request-Reply pattern** (`nc.Request` / `nc.RequestMsg`) is a synchronous, blocking query:

```text
Publisher                                       NATS Server                                      Responder
    |                                                |                                               |
    |-- 1. Creates ephemeral Inbox (_INBOX.xxx) ---->|                                               |
    |-- 2. Publishes Request msg (Reply=_INBOX.xxx)->|---------------- 3. Delivers Request --------->|
    |                                                |                                               |
    |                                                |<--------------- 4. Responds to _INBOX.xxx ----|
    |<-- 5. Receives Response (or timeout) ----------|                                               |
```

1. The client SDK automatically creates an ephemeral inbox subscription (`_INBOX.xxx`).
2. The client attaches `_INBOX.xxx` as `msg.Reply` and publishes the request.
3. The client **blocks** waiting for a responder to publish a response to `_INBOX.xxx` until the specified timeout expires.

```go
// Synchronous Request-Reply: publishes request AND waits for response
resp, err := nc.Request(
    "orders.validate",
    []byte(`{"orderId":"12345"}`),
    2*time.Second,
)
if err != nil {
    return fmt.Errorf("request-reply failed or timed out: %w", err)
}

log.Printf("Response received: %s", string(resp.Data))
```

For structured request messages with headers:

```go
msg := nats.NewMsg("orders.validate")
msg.Data = []byte(`{"orderId":"12345"}`)
msg.Header.Set("Content-Type", "application/json")

resp, err := nc.RequestMsg(msg, 2*time.Second)
if err != nil {
    return err
}
```

---

### 3.5 Flush

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

### 3.6 Concurrent Publishing

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

The acknowledgement provides information such as:

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

select {
case ack := <-future.Ok():
    log.Printf("Published to stream=%s sequence=%d", ack.Stream, ack.Sequence)
case err := <-future.Err():
    return fmt.Errorf("async publish failed: %w", err)
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

Publish Expectations provide **Optimistic Concurrency Control (OCC)** and atomic conditional publishing in JetStream.

When multiple publisher workers or microservices attempt to append updates concurrently, publish expectations prevent out-of-order writes, lost updates, and state corruption without requiring distributed locks.

#### How Expectations Work

Before appending a message to a stream, the JetStream server evaluates the requested expectations against current stream metadata:

1. **Assertion Match**: If the stream state matches all expectations, JetStream appends the message atomically and returns an `Ack`.
2. **Assertion Mismatch**: If any expectation fails (for example, another worker wrote to the stream first), JetStream rejects the message, appends nothing, and returns a JetStream expectation error to the client.

```text
Publisher                       JetStream Server                   Stream State
    |                                   |                                |
    |-- Publish (Expect Seq 100) ------>| Check Last Seq = 100?          |
    |                                   |   |                            |
    |                                   |   +-- MATCH ------------------>| Append Msg (Seq 101)
    |<-- ACK (Seq 101) -----------------|                                |
    |                                   |                                |
    |-- Publish (Expect Seq 100) ------>| Check Last Seq = 100?          |
    |                                   |   |                            |
    |                                   |   +-- MISMATCH (Current 101) --| Reject Msg (No Store)
    |<-- ERR (Wrong Last Sequence) -----|                                |
```

#### Available Expectation Options

JetStream supports several expectation options passed as `jetstream.PublishOpt`:

* **`WithExpectStream(name)`**: Guarantees that the target subject maps to the specified stream name, preventing misrouted messages if subject configurations shift.
* **`WithExpectLastSequence(seq)`**: Asserts that the stream's absolute last sequence number exactly equals `seq`. Useful when strict single-stream linear sequence is required.
* **`WithExpectLastSubjectSequence(seq)`**: Asserts the last sequence number recorded specifically for the message subject. Useful for per-entity ordering (e.g. order `ORD-123`) across a shared multi-entity stream.
* **`WithExpectLastMsgID(id)`**: Asserts that the last message appended to the stream had the specific `Nats-Msg-Id`. Useful when chain-linking transactions.

#### Example: Conditional Publish with Sequence Assertion

The following example demonstrates publishing an event only if the stream sequence matches the caller's expected last sequence. If another process modified the stream concurrently, the publish is safely rejected.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/nats-io/nats.go/jetstream"
)

func publishWithExpectation(js jetstream.JetStream, expectedSeq uint64, payload []byte) error {
    ctx := context.Background()

    // Assert that the stream's last sequence is exactly expectedSeq
    ack, err := js.Publish(
        ctx,
        "orders.updated",
        payload,
        jetstream.WithExpectLastSequence(expectedSeq),
        jetstream.WithExpectStream("ORDERS"),
    )
    if err != nil {
        var jsErr jetstream.JetStreamError
        if errors.As(err, &jsErr) && jsErr.APIError() != nil {
            // Check for expectation mismatch (NATS JetStream ErrCode 10071)
            if jsErr.APIError().ErrorCode == 10071 {
                return fmt.Errorf("concurrency conflict: stream sequence moved beyond %d: %w", expectedSeq, err)
            }
        }
        return fmt.Errorf("publish failed: %w", err)
    }

    log.Printf("Published message to stream %s at sequence %d", ack.Stream, ack.Sequence)
    return nil
}
```

#### Use Cases for Publish Expectations

* **State Machine Transitions**: Ensure state events (`ORDER_SUBMITTED` -> `ORDER_PAID`) are only stored if the predecessor event sequence is intact.
* **Optimistic Locking**: Multi-tenant or partitioned applications can write updates to dedicated subjects (`orders.ORD-123.events`) using `WithExpectLastSubjectSequence` without locking the global stream.
* **Preventing Race Conditions**: Multiple workers reading work from a stream can commit updates back only if no competing worker committed first.

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
