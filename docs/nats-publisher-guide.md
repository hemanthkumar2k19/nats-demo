# NATS Publisher Guide

## Purpose

This guide defines application-level practices for publishing messages to NATS using Core NATS and JetStream.

It covers the publisher lifecycle from initialization through message construction, publishing, failure and retry handling, and graceful shutdown.

---

## 1. Prerequisites & NATS Subject Architecture

### 1.1 Prerequisites

Publishing to NATS requires an active NATS connection (`*nats.Conn`). For JetStream publishing, a JetStream API context (`jetstream.JetStream`) initialized from the connection is required (refer to the NATS Client Connectivity Guide).

Conceptually:

```text
Active NATS Connection (TCP Socket)
      |
      +-- Core NATS Publisher (Ephemeral Publish / Request)
      |
      +-- JetStream Context (Server Ack & Persistent Stream Publish)
```

---

### 1.2 NATS Subjects Brief & Naming Architecture

#### What is a NATS Subject?

A **Subject** is a lightweight, case-sensitive ASCII string used by NATS to route messages between publishers and subscribers/streams. Subjects act as destination addresses and do not require prior creation or registration in Core NATS.

#### Subject Naming Hierarchy

Enterprise NATS subjects follow a dot-separated hierarchical naming convention:

`<domain>.<entity>.<action>` or `<region>.<service>.<resource>.<event>`

Examples:
* `orders.eu.created`
* `payments.us.processed`
* `telemetry.sensors.temp.reading`

#### Rules for Valid NATS Subjects

* **Characters**: Alphanumeric ASCII characters (`a-z`, `A-Z`, `0-9`), dots (`.`), hyphens (`-`), and underscores (`_`). Spaces and special characters are forbidden.
* **Tokens**: Substrings separated by dots (`.`). Empty tokens are invalid (e.g., `orders..created` is invalid).
* **Case Sensitivity**: Subjects are strictly case-sensitive (`ORDERS.created` and `orders.created` are separate subjects).
* **Concrete Publish Requirement**: Messages MUST be published to **concrete subjects** (wildcards are forbidden when publishing).

#### NATS Wildcards (Subscriber & Stream Filtering)

Subscribers and JetStream streams use wildcards to listen to or capture groups of subjects:

| Wildcard Symbol | Name | Scope & Behavior | Example Pattern | Matching Subjects | Non-Matching Subjects |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `*` | Single-Token Wildcard | Matches **exactly one token** at a specific level in the hierarchy. | `orders.*.created` | `orders.eu.created`, `orders.us.created` | `orders.created`, `orders.eu.123.created` |
| `>` | Multi-Token Wildcard | Matches **one or more tokens** at the end of a subject (must be final token). | `orders.>` | `orders.created`, `orders.eu.created`, `orders.eu.123.created` | `audit.orders.created` |

> **Publisher Rule:** Wildcards (`*` and `>`) are evaluated **only during subscriber matching**, not during message publishing.
> * If you execute `nats pub "order.*" "Hello"`, NATS publishes the message to the **literal 7-character string subject `"order.*"`**.
> * It will **NOT** expand or deliver the message to subscribers listening on `order.created` or `order.123`.
> * Therefore, publishers MUST always publish to explicit, concrete subject strings (e.g., `order.created`).

---

## 2. Message Construction

A NATS message consists of a destination subject, binary payload data, optional headers, and an optional reply subject.

### 2.1 Subject & Target Namespace

Every published message must have a destination subject that complies with subject naming standards (`<domain>.<entity>.<action>`).

#### Go

```go
msg := nats.NewMsg("orders.created")
```

#### Java

```java
Message msg = NatsMessage.builder()
        .subject("orders.created")
        .build();
```

### 2.2 Payload & Serialization

Application domain objects must be serialized into a binary byte slice (e.g., JSON) before being set on `msg.Data`.

#### Go

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

#### Java

```java
OrderEvent event = new OrderEvent("12345");
byte[] payload = objectMapper.writeValueAsBytes(event);

Message msg = NatsMessage.builder()
        .subject("orders.created")
        .data(payload)
        .build();
```

### 2.3 Headers & Metadata

Use NATS headers (`msg.Header`) for metadata such as content types, correlation IDs, or JetStream deduplication keys (`Nats-Msg-Id`).

#### Go

```go
msg.Header.Set("Content-Type", "application/json")
msg.Header.Set("X-Correlation-ID", correlationID)
```

#### Java

```java
Headers headers = new Headers();
headers.set("Content-Type", "application/json");
headers.set("X-Correlation-ID", correlationId);

Message msg = NatsMessage.builder()
        .subject("orders.created")
        .headers(headers)
        .build();
```

### 2.4 Optional Reply Subject

A message can specify an optional reply subject (`msg.Reply`) indicating where a responder can publish a response.

#### Go

```go
msg.Reply = "orders.reply.inbox"
```

#### Java

```java
Message msg = NatsMessage.builder()
        .subject("orders.created")
        .replyTo("orders.reply.inbox")
        .build();
```

The message structure is independent of whether it is subsequently published through Core NATS or JetStream.

---

## 3. Core NATS Publishing

Core NATS provides ephemeral publish semantics. A Core NATS publish does not wait for a server-side publish acknowledgement or subscriber acknowledgement.

### 3.1 Basic Publish

Use `Publish` when only a subject and payload are required.

#### Go

```go
err := nc.Publish(
    "orders.created",
    []byte(`{"orderId":"12345"}`),
)
if err != nil {
    return err
}
```

#### Java

```java
nc.publish("orders.created", "{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8));
```

The client queues the message for transmission and the publish operation returns without waiting for subscriber confirmation.

Core NATS does not provide JetStream-style publish acknowledgements or persistence.

---

### 3.2 Structured Message Publish

Use `PublishMsg` when headers or an optional reply subject are required on the message.

#### Go

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

#### Java

```java
Headers headers = new Headers();
headers.set("Content-Type", "application/json");

Message msg = NatsMessage.builder()
        .subject("orders.created")
        .replyTo("orders.reply.inbox")
        .headers(headers)
        .data("{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8))
        .build();

nc.publish(msg);
```

---

### 3.3 Publishing with Reply Subject (`PublishRequest`)

A publisher can send a message with an explicitly attached reply subject using `PublishRequest`.

#### Go

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

#### Java

```java
nc.publish("orders.validate", "orders.reply.inbox", "{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8));
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

#### Go

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

#### Java

```java
// Synchronous Request-Reply: publishes request AND waits for response
CompletableFuture<Message> future = nc.request("orders.validate", "{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8));
try {
    Message resp = future.get(2, TimeUnit.SECONDS);
    System.out.printf("Response received: %s%n", new String(resp.getData(), StandardCharsets.UTF_8));
} catch (Exception e) {
    System.err.printf("Request-reply failed or timed out: %s%n", e.getMessage());
}
```

For structured request messages with headers:

```java
Headers headers = new Headers();
headers.set("Content-Type", "application/json");

Message msg = NatsMessage.builder()
        .subject("orders.validate")
        .headers(headers)
        .data("{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8))
        .build();

CompletableFuture<Message> future = nc.request(msg);
try {
    Message resp = future.get(2, TimeUnit.SECONDS);
    System.out.printf("Response received: %s%n", new String(resp.getData(), StandardCharsets.UTF_8));
} catch (Exception e) {
    System.err.printf("Structured request-reply failed: %s%n", e.getMessage());
}
```

---

### 3.5 Flush

Use `Flush` when the application needs confirmation that pending client operations have been processed by the server.

#### Go

```go
if err := nc.Flush(); err != nil {
    return err
}
```

#### Java

```java
try {
    nc.flush(Duration.ofSeconds(2));
} catch (Exception e) {
    System.err.printf("Flush failed: %s%n", e.getMessage());
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

#### Go

```go
go func() {
    _ = nc.Publish("orders.created", payload)
}()

go func() {
    _ = nc.Publish("orders.updated", payload)
}()
```

#### Java

```java
ExecutorService executor = Executors.newFixedThreadPool(2);

executor.submit(() -> nc.publish("orders.created", payload));
executor.submit(() -> nc.publish("orders.updated", payload));
```

A separate connection should not be created for every publish operation.

---

## 4. JetStream Publishing

JetStream publishing provides server-side acknowledgement and persistence according to the configured stream and storage policy.

### 4.1 Synchronous Publish

Use synchronous publishing when the application needs the JetStream publish acknowledgement before continuing.

#### Go

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

#### Java

```java
JetStream js = nc.jetStream();
PublishAck ack = js.publish("orders.created", "{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8));

System.out.printf("Stream: %s, Sequence: %d%n", ack.getStream(), ack.getSequence());
```

The publish operation waits for the JetStream publish acknowledgement.

The acknowledgement provides information such as:

* Stream
* Sequence
* Duplicate detection result

---

### 4.2 Asynchronous Publish

Use asynchronous publishing when the application wants to continue processing without waiting for every individual publish acknowledgement.

#### Go

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

#### Java

```java
JetStream js = nc.jetStream();
CompletableFuture<PublishAck> future = js.publishAsync(
    "orders.created",
    "{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8)
);

try {
    PublishAck ack = future.get(5, TimeUnit.SECONDS);
    System.out.printf("Published to stream=%s sequence=%d%n", ack.getStream(), ack.getSequence());
} catch (Exception e) {
    System.err.printf("Async publish failed: %s%n", e.getMessage());
}
```

Applications using asynchronous publishing should account for outstanding publishes before shutdown.

---

### 4.3 Managing Pending Publishes

JetStream provides APIs to inspect and wait for asynchronous publishing activity.

#### Go

```go
pending := js.PublishAsyncPending()

if pending > 0 {
    <-js.PublishAsyncComplete()
}
```

#### Java

```java
// In Java NATS SDK, async publish futures (CompletableFuture<PublishAck>) track pending status
CompletableFuture<PublishAck> future = js.publishAsync("orders.created", payload);
if (!future.isDone()) {
    PublishAck ack = future.get(5, TimeUnit.SECONDS);
}
```

The application should define an appropriate shutdown boundary so that pending publishing work is either completed or explicitly abandoned.

---

### 4.4 Publish Deduplication

JetStream supports duplicate publish detection using the `Nats-Msg-Id` header.

#### Go

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

#### Java

```java
Headers headers = new Headers();
headers.set("Nats-Msg-Id", "order-12345");

Message msg = NatsMessage.builder()
        .subject("orders.created")
        .headers(headers)
        .data("{\"orderId\":\"12345\"}".getBytes(StandardCharsets.UTF_8))
        .build();

PublishAck ack = js.publish(msg);
if (ack.isDuplicate()) {
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

##### Go

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

##### Java

```java
import io.nats.client.JetStream;
import io.nats.client.PublishAck;
import io.nats.client.api.PublishOptions;
import io.nats.client.JetStreamApiException;

public class ConditionalPublish {
    public static void publishWithExpectation(JetStream js, long expectedSeq, byte[] payload) throws Exception {
        try {
            PublishOptions opts = PublishOptions.builder()
                    .expectedLastSequence(expectedSeq)
                    .stream("ORDERS")
                    .build();

            PublishAck ack = js.publish("orders.updated", payload, opts);
            System.out.printf("Published message to stream %s at sequence %d%n", ack.getStream(), ack.getSequence());
        } catch (JetStreamApiException e) {
            // Check for expectation mismatch (NATS JetStream ErrCode 10071)
            if (e.getErrorCode() == 10071) {
                System.err.printf("Concurrency conflict: stream sequence moved beyond %d: %s%n", expectedSeq, e.getMessage());
            } else {
                throw e;
            }
        }
    }
}
```

#### Use Cases for Publish Expectations

* **State Machine Transitions**: Ensure state events (`ORDER_SUBMITTED` -> `ORDER_PAID`) are only stored if the predecessor event sequence is intact.
* **Optimistic Locking**: Multi-tenant or partitioned applications can write updates to dedicated subjects (`orders.ORD-123.events`) using `WithExpectLastSubjectSequence` without locking the global stream.
* **Preventing Race Conditions**: Multiple workers reading work from a stream can commit updates back only if no competing worker committed first.

---

## 5. Failure Handling & Retry

### 5.1 Publish Errors

Publish errors in NATS vary by messaging model: Core NATS publishing, Core NATS Request-Reply, and JetStream publishing.

Developers should understand how errors are surfaced by the NATS client, which failures can be classified directly, and when a publish outcome may remain unknown.

#### Error Manifestation by Messaging Pattern

| Messaging Pattern      | API                            | Error / Outcome                            | Typical Causes                                                                                                                           |
| :--------------------- | :----------------------------- | :----------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------- |
| **Core NATS**          | `nc.Publish`                   | Client-side publish error                  | Invalid subject, connection state, outbound buffer limit, payload exceeds `max_payload`                                                  |
| **Core Request-Reply** | `nc.Request`                   | Responder / timeout error                  | No active responder, responder unavailable, response not received before deadline                                                        |
| **JetStream**          | `js.Publish` / `js.PublishMsg` | Publish error / timeout / `PubAck` outcome | Stream configuration, publish expectation failure, authorization, resource limits, server/storage conditions, or acknowledgement timeout |

#### Publish Outcome and Error Classification

Not all publish failures provide enough information to determine whether the server processed the message.

For example, a JetStream publish may time out while waiting for its `PubAck`:

1. **Network Disconnect** — The publish request may not have reached the server.
2. **Server-Side Processing Delay** — The server may still be processing the publish.
3. **JetStream Cluster Transition** — A temporary leader or cluster transition may delay the acknowledgement.
4. **Acknowledgement Lost** — The server may have successfully stored the message, but the `PubAck` may not have reached the client.

Therefore, a timeout does **not necessarily mean that the message was not stored**. Retrying an operation with an unknown outcome can result in a duplicate publish unless an appropriate idempotency mechanism is used.

Where the SDK exposes a specific semantic error, applications should classify the error accordingly. Where the outcome cannot be determined, applications should treat it as an **unknown publish outcome** rather than assuming failure.

#### Common NATS Error Reference

The table below provides a comprehensive reference of Core NATS client errors and JetStream API error codes encountered during publishing, along with their status codes, classifications, and recommended remediation:

| Error / Symbol | Source / API | Error / Status Code | Classification | Root Cause & Remediation Guidance |
| :--- | :--- | :--- | :--- | :--- |
| `nats.ErrNoResponders` | Core Request-Reply | Err 503 | Transient / App | No active subscribers are listening on the request subject. Ensure the service is running and subscribed. |
| `nats.ErrTimeout` / `context.DeadlineExceeded` | Core / JetStream | Client Timeout | Unknown Outcome | Request or publish Ack was not received before deadline. For JetStream, outcome is unknown (message MAY be stored). |
| `nats.ErrMaxPayload` | Core NATS | Err 400 | Permanent | Message payload size exceeds the server `max_payload` limit. Reduce payload size or use Object Store. |
| `nats.ErrAuthorization` | Core NATS | Err 403 | Permanent | Client credentials lack publish permission for the subject. Correct account ACLs or subject permissions. |
| `nats.ErrConnectionClosed` | Core NATS | Client State | Permanent | The client connection was explicitly closed or unrecoverable. Re-establish NATS connection. |
| `nats.ErrReconnectBufExceeded` | Core NATS | Client Buffer | Transient Limit | Outbound client buffer (`ReconnectBufSize`) filled while disconnected. Increase buffer or slow publishing. |
| `jetstream.ErrStreamNotFound` | JetStream | Code `10005` (HTTP 404) | Configuration | Target stream does not exist or subject is not bound to a stream. Verify stream config and subject mapping. |
| `ErrStreamLimits` | JetStream | Code `10054` (HTTP 400) | Resource Limit | Stream limits (`max_msgs`, `max_bytes`, `max_msg_size`) exceeded. Increase stream limits or purge expired data. |
| `ErrStreamWrongLastSequence` | JetStream | Code `10071` (HTTP 400) | State Conflict | `WithExpectLastSequence` assertion failed (OCC failure). Re-read stream head sequence before publishing. |
| `ErrStreamWrongLastSubjectSequence` | JetStream | Code `10072` (HTTP 400) | State Conflict | `WithExpectLastSubjectSequence` assertion failed for subject. Re-evaluate subject event history before retry. |
| `ErrStreamWrongLastMsgID` | JetStream | Code `10073` (HTTP 400) | State Conflict | `WithExpectLastMsgID` assertion failed. Verify logical message ordering. |
| `ErrStreamWrongStream` | JetStream | Code `10075` (HTTP 400) | Configuration | Subject maps to a stream name different from `WithExpectStream`. Check stream subject bindings. |
| `ack.Duplicate == true` (`ErrDuplicate`) | JetStream | Code `10077` (HTTP 200) | Duplicate | Message `Nats-Msg-Id` was already accepted within `duplicate_window`. Success outcome; message not duplicated. |
| `ErrNoStreamResponse` | JetStream | Code `10014` (HTTP 503) | Cluster Transient | Server timed out waiting for Raft quorum consensus across stream replicas. Retry publish. |
| `ErrClusterUnavail` | JetStream | Code `10023` (HTTP 503) | Cluster Transient | JetStream cluster is unavailable or undergoing leader re-election. Retry publish after short backoff. |

---

### 5.2 Retry Policy

The NATS Client SDK (`nats.go`) and JetStream package (`jetstream`) provide built-in transport resilience and configurable client-side retry options. 

Developers should leverage SDK client options to configure connection retries, backoff strategies, reconnect buffers, and async publish error handling directly through the NATS driver.

#### SDK Connection Retry & Reconnect Options

When establishing a connection, the NATS client SDK provides built-in reconnect options that handle network disruptions automatically:

| SDK Option | Purpose | Default Value | Usage / Behavior |
| :--- | :--- | :--- | :--- |
| `nats.RetryOnFailedConnect(true)` | Initial Connection Retry | `false` | Enables SDK to retry connection on startup if initial server dial fails. |
| `nats.MaxReconnects(n)` | Reconnect Attempts Limit | `60` (`-1` = infinite) | Maximum number of reconnect attempts before client closes connection. |
| `nats.ReconnectWait(d)` | Fixed Reconnect Delay | `2s` | Fixed wait duration between reconnect attempts. |
| `nats.CustomReconnectDelay(fn)` | SDK-Managed Backoff | `nil` | Function hook executed by SDK on each reconnect attempt to apply custom exponential backoff/jitter. |
| `nats.ReconnectJitter(j, tlsJ)` | Reconnect Jitter | `100ms` / `1s` | Random duration added to reconnect delay to prevent thundering herd. |
| `nats.ReconnectBufSize(bytes)` | Outbound Reconnect Buffer | `8MB` (`8 * 1024 * 1024`) | Outbound memory buffer storing publications while disconnected, flushed on reconnect. |

#### JetStream Publisher Resilience Options

JetStream provides client-level options for managing asynchronous publish buffering, error callbacks, and server-side deduplication:

| JetStream Mechanism | Config / API | Description |
| :--- | :--- | :--- |
| **Pending Buffer Limit** | `jetstream.WithPublishAsyncMaxPending(n)` | Sets maximum unacknowledged async publish count to enforce backpressure. |
| **Async Error Handler** | `jetstream.WithPublishAsyncErrHandler(cb)` | Configures SDK callback invoked when an async publish fails or times out. |
| **Async Completion Signal** | `js.PublishAsyncComplete()` | Returns channel signaling when all in-flight async publications receive Acks or fail. |
| **Server Deduplication** | Header `Nats-Msg-Id` | Enables server-side duplicate suppression within `duplicate_window`. |

#### Configuring NATS SDK Retry & Resilience Options in Go

The example below demonstrates how to configure NATS client SDK options for automated connection reconnect backoff, buffer management, and JetStream async error handling:

##### Go

```go
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func ConnectWithSDKRetry(serverURL string) (jetstream.JetStream, error) {
	// Configure NATS Client SDK options for automated connection reconnect & backoff
	nc, err := nats.Connect(
		serverURL,
		// Retry connection on initial startup dial
		nats.RetryOnFailedConnect(true),
		// Unlimited reconnect attempts during network outages
		nats.MaxReconnects(-1),
		// SDK-managed exponential backoff with jitter
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			base := 100 * time.Millisecond
			max := 10 * time.Second
			delay := base * time.Duration(1<<uint(attempts))
			if delay > max {
				delay = max
			}
			// Add randomized jitter
			jitter := time.Duration(rand.Int63n(int64(delay / 2)))
			return delay + jitter
		}),
		// Increase client outbound buffer for publications while disconnected
		nats.ReconnectBufSize(16*1024*1024), // 16MB
		// Event handlers for logging connection state changes
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v. Reconnecting in background...", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected successfully to %s", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Initialize JetStream context with async publish error handling options
	js, err := jetstream.New(nc,
		jetstream.WithPublishAsyncMaxPending(256),
		jetstream.WithPublishAsyncErrHandler(func(js jetstream.JetStream, msg *nats.Msg, err error) {
			log.Printf("Async publish failed for subject %s (Msg-ID: %s): %v",
				msg.Subject, msg.Header.Get("Nats-Msg-Id"), err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	return js, nil
}
```

##### Java

```java
import io.nats.client.Connection;
import io.nats.client.ConnectionListener;
import io.nats.client.ErrorListener;
import io.nats.client.JetStream;
import io.nats.client.Nats;
import io.nats.client.Options;

import java.time.Duration;

public class SDKRetrySetup {
    public static JetStream connectWithSDKRetry(String serverUrl) throws Exception {
        Options options = new Options.Builder()
                .server(serverUrl)
                .maxReconnects(-1) // Unlimited reconnect attempts
                .reconnectWait(Duration.ofSeconds(2)) // SDK reconnect wait
                .reconnectBufferSize(16 * 1024 * 1024) // 16MB outbound buffer
                .connectionListener((conn, type) -> {
                    if (type == ConnectionListener.Events.DISCONNECTED) {
                        System.out.println("NATS disconnected. Reconnecting in background...");
                    } else if (type == ConnectionListener.Events.RECONNECTED) {
                        System.out.printf("NATS reconnected successfully to %s%n", conn.getConnectedUrl());
                    }
                })
                .errorListener(new ErrorListener() {
                    @Override
                    public void errorOccurred(Connection conn, String error) {
                        System.out.printf("Async error: %s%n", error);
                    }
                })
                .build();

        Connection nc = Nats.connect(options);
        return nc.jetStream();
    }
}
```


## 6. Graceful Shutdown

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

### Go

```go
<-js.PublishAsyncComplete()
```

### Java

```java
// Wait for in-flight publish futures to complete
future.get(5, TimeUnit.SECONDS);
```

For Core NATS pending operations:

### Go

```go
if err := nc.Flush(); err != nil {
    // handle shutdown error
}
```

### Java

```java
try {
    nc.flush(Duration.ofSeconds(2));
} catch (Exception e) {
    System.err.printf("Flush error: %s%n", e.getMessage());
}
```

The connection can then be drained:

### Go

```go
if err := nc.Drain(); err != nil {
    // handle shutdown error
}
```

### Java

```java
try {
    nc.drain(Duration.ofSeconds(5)).get();
} catch (Exception e) {
    System.err.printf("Drain error: %s%n", e.getMessage());
}
```

`Drain` provides graceful connection shutdown by allowing pending client work to be processed before the connection closes.

When the connection and server remain healthy, this gives in-flight publications an opportunity to reach the server before shutdown.

`Drain` does not by itself guarantee message durability or delivery.

Applications should use a bounded shutdown timeout rather than waiting indefinitely.

---

## 7. Publisher Capability Summary

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

## 8. Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client API Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go)
* [NATS Go JetStream Package Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
* [Official NATS Java Client Repository](https://github.com/nats-io/nats.java)
* [NATS Java Client Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/index.html)
* [Official NATS Developer Documentation](https://docs.nats.io/using-nats/developer)

