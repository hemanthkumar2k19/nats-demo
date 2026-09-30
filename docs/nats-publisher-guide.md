# NATS Publisher Guide

This guide provides software engineers and platform developers with design patterns, operational principles, and Go implementations for publishing messages using NATS.

The guide separates **common publisher concerns** from the publishing semantics specific to **Core NATS** and **JetStream**.

---

## Architectural Overview & Mental Model

The publisher lifecycle consists of common message-processing concerns followed by transport-specific publishing semantics.

```text
                         NATS Publisher
                              |
              +---------------+---------------+
              |                               |
      Common Publisher Concerns       Publishing Semantics
              |                               |
      Message Construction          +---------+---------+
      Payload & Serialization       |                   |
      Headers & Metadata       Core NATS          JetStream
      Connection Reuse          Publish              Publish
      Concurrency               Semantics            Semantics
      Buffer Management
      Error Handling
      Retry & Failure Handling
      Graceful Shutdown
```

The common publisher layer applies regardless of whether the application ultimately publishes through Core NATS or JetStream.

---

# 1. Message Construction

## 1.1 Subject Selection

Publishers must use subjects that comply with the platform's **Subject Naming Guidelines**.

A subject identifies the destination of the message and should represent the messaging contract rather than an implementation detail.

Example:

```go
func BuildSubject(domain, entity, event string) string {
    // Example: "orders.fulfillment.created"
    return fmt.Sprintf("%s.%s.%s", domain, entity, event)
}
```

The complete subject namespace and naming rules are defined in the **NATS Semantics & Naming Guidelines**.

---

## 1.2 Payload & Message Construction

Construct a NATS message with:

* Destination subject
* Payload
* Message headers

```go
func ConstructMessage(subject string, payload []byte) *nats.Msg {
    return &nats.Msg{
        Subject: subject,
        Data:    payload,
        Header:  make(nats.Header),
    }
}
```

Message construction should be completed before the message is submitted to the publishing layer.

---

## 1.3 Headers & Metadata

Use NATS headers for message metadata that should not be embedded directly into the business payload.

Common header categories include:

* Content type
* Correlation identifier
* Message identifier
* Trace context
* Application metadata
* Schema or contract metadata

Example:

```go
func AttachStandardHeaders(
    msg *nats.Msg,
    contentType string,
    correlationID string,
) {
    msg.Header.Set("Content-Type", contentType)
    msg.Header.Set("X-Correlation-ID", correlationID)
}
```

Enterprise-standard message envelope and schema metadata are defined in the **Message Envelope & Schema Management** guide.

Transport-specific headers, such as JetStream deduplication headers, are covered in the JetStream Publishing section.

---

# 2. Payload & Serialization

## 2.1 Payload Serialization

Applications should serialize domain objects into the agreed message format before publishing.

The publishing layer should receive the serialized payload rather than being responsible for application-domain serialization.

Example using JSON:

```go
type OrderCreatedEvent struct {
    OrderID    string    `json:"order_id"`
    CustomerID string    `json:"customer_id"`
    Amount     float64   `json:"amount"`
    Timestamp  time.Time `json:"timestamp"`
}

func CreateOrderPayload(event OrderCreatedEvent) ([]byte, error) {
    data, err := json.Marshal(event)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal event payload: %w", err)
    }

    return data, nil
}
```

JSON is used here as an implementation example. The enterprise message format and schema requirements are defined separately.

---

## 2.2 Payload Size & Large Messages

NATS servers enforce a maximum message payload size.

Publishers should:

* Validate payload sizes where appropriate
* Avoid unnecessarily large messages
* Avoid embedding large binary objects directly in messages
* Use an external storage reference for large objects when appropriate

The server's configured maximum payload can be inspected through the client connection:

```go
maxAllowed := nc.MaxPayload()

if int64(len(msg.Data)) > maxAllowed {
    return fmt.Errorf(
        "payload size (%d bytes) exceeds maximum (%d bytes)",
        len(msg.Data),
        maxAllowed,
    )
}
```

Applications should treat the server-configured limit as the authoritative constraint rather than assuming a fixed universal value.

---

# 3. Connection Reuse & Multiplexing

## 3.1 Long-Lived Connection

NATS client connections are designed to support concurrent operations over a long-lived connection.

Applications should:

* Establish connections during application initialization
* Reuse the connection across publishers and other NATS operations
* Avoid creating a new connection for every publish operation
* Close or drain the connection during application shutdown

```text
Application
     |
     +-------------------+
     |                   |
 Publisher A         Publisher B
     |                   |
     +---------+---------+
               |
       Shared NATS Connection
               |
           NATS Server
```

A single connection can multiplex publishing, subscribing, request/reply, and JetStream API operations.

---

## 3.2 Publisher Dependency Management

Publisher components should receive the shared NATS connection as a dependency rather than creating their own connection.

```go
type EventPublisher struct {
    nc *nats.Conn
}

func NewEventPublisher(nc *nats.Conn) *EventPublisher {
    return &EventPublisher{
        nc: nc,
    }
}

func (p *EventPublisher) Publish(
    subject string,
    data []byte,
) error {
    msg := nats.NewMsg(subject)
    msg.Data = data

    return p.nc.PublishMsg(msg)
}
```

This keeps connection lifecycle management separate from business-level publishing logic.

---

# 4. Publish Concurrency

## 4.1 Concurrent Publishing

Applications can perform concurrent publishing operations using a shared NATS connection.

```go
func ConcurrentPublish(
    nc *nats.Conn,
    subject string,
    workerCount int,
) {
    var wg sync.WaitGroup

    for i := 0; i < workerCount; i++ {
        wg.Add(1)

        go func(workerID int) {
            defer wg.Done()

            payload := []byte(
                fmt.Sprintf("worker-%d payload", workerID),
            )

            _ = nc.Publish(subject, payload)
        }(i)
    }

    wg.Wait()
}
```

Applications should avoid introducing unnecessary connection-per-worker designs.

---

## 4.2 Ordering Considerations

Concurrent publishers should not assume application-level ordering across independent publishing workers.

If message ordering is a business requirement, the application should design its publishing workflow accordingly.

Ordering requirements may differ between Core NATS and JetStream and are covered in their respective publishing sections.

---

# 5. Buffer Management

Publishing systems may contain multiple buffering layers. These should not be treated as equivalent.

```text
Application Producer
       |
       v
Application Buffer
       |
       v
NATS Client
       |
       +--> Reconnect Buffer
       |
       v
NATS Server
```

## 5.1 NATS Client Reconnect Buffer

The NATS client can buffer outbound messages while a connection is reconnecting.

The buffer size is configurable:

```go
func ConfigurePublishBuffer() []nats.Option {
    return []nats.Option{
        nats.ReconnectBufSize(16 * 1024 * 1024),
    }
}
```

Applications should understand the memory implications of increasing this buffer.

The reconnect buffer should not be treated as durable message storage.

---

## 5.2 Application-Side Buffering

Applications that require controlled producer buffering may implement an explicit bounded queue.

```go
var ErrBufferFull = errors.New(
    "application publish buffer is full",
)

type BoundedPublisher struct {
    nc    *nats.Conn
    queue chan *nats.Msg
}

func NewBoundedPublisher(
    nc *nats.Conn,
    bufferSize int,
) *BoundedPublisher {
    publisher := &BoundedPublisher{
        nc:    nc,
        queue: make(chan *nats.Msg, bufferSize),
    }

    go publisher.startWorker()

    return publisher
}

func (bp *BoundedPublisher) Submit(msg *nats.Msg) error {
    select {
    case bp.queue <- msg:
        return nil

    default:
        return ErrBufferFull
    }
}
```

Applications should use bounded rather than unbounded queues to prevent uncontrolled memory growth during prolonged publishing failures.

---

## 5.3 Backpressure & Buffer Limits

Applications should explicitly define behavior when the publishing buffer reaches capacity.

Possible strategies include:

* Block the producer
* Reject the new message
* Apply backpressure to the upstream component
* Drop the message when explicitly permitted
* Persist the message through another mechanism

The appropriate strategy depends on the application's delivery requirements.

A message removed from an application buffer but not successfully published must have an explicit failure-handling policy. Applications should not silently discard such messages.

---

# 6. Publish Error Handling

## 6.1 Synchronous Publish Errors

A publish operation can return an error immediately.

Examples include:

* Connection closed
* Connection draining
* Invalid subject
* Maximum payload exceeded
* Client outbound buffer limitations
* Other client-side publishing errors

Applications should inspect and handle returned errors rather than assuming that a successful function call always represents successful business-level delivery.

```go
func PublishMessage(
    nc *nats.Conn,
    msg *nats.Msg,
) error {
    if err := nc.PublishMsg(msg); err != nil {
        return fmt.Errorf("NATS publish failed: %w", err)
    }

    return nil
}
```

---

## 6.2 Connection & Reconnection Events

Publishing behavior can be affected by connection state changes.

Applications should monitor relevant connection lifecycle events such as:

* Disconnect
* Reconnect
* Reconnect failure
* Connection closure

These events should be used together with publish errors when diagnosing publishing failures.

Connection lifecycle management is described in the **NATS Client & SDK Developer Guide**.

---

## 6.3 Asynchronous Client Errors

NATS clients can expose asynchronous errors through client error handlers for errors that are not returned directly by the publishing operation.

Applications should configure asynchronous error handling where required and ensure that such errors are surfaced through application logging and observability.

Asynchronous error handling should not be treated as a replacement for checking the error returned by a publish operation.

---

# 7. Retry & Failure Handling

## 7.1 Retry Strategy

Publish retries should be used only for failures that may be transient.

Applications should distinguish between:

```text
Publish Failure
      |
      +--> Transient
      |      |
      |      +--> Retry
      |
      +--> Permanent
             |
             +--> Fail / Reject / Escalate
```

Applications should not blindly retry every publishing error.

---

## 7.2 Retry Limits & Backoff

When retries are required, applications should define:

* Maximum retry attempts
* Retry delay
* Backoff strategy
* Maximum retry duration
* Failure handling after retry exhaustion

Avoid tight retry loops that can increase load during a NATS outage.

---

## 7.3 Duplicate & Idempotency Considerations

A retry may result in a duplicate message if the publisher cannot determine whether the previous publish was successfully received.

Therefore, applications should consider:

* Message identifiers
* Idempotent processing
* Duplicate detection
* Business-level deduplication

The mechanism used to establish delivery confirmation and deduplication differs between Core NATS and JetStream.

Those mechanisms are covered in the respective publishing sections rather than being treated as generic NATS behavior.

---

# 8. Graceful Shutdown

## 8.1 Publisher Shutdown

Application shutdown should follow a controlled sequence:

```text
Stop Accepting New Messages
          |
          v
Process / Resolve Application Buffer
          |
          v
Drain NATS Connection
          |
          v
Close Application
```

Applications should define a shutdown timeout so that shutdown cannot block indefinitely.

---

## 8.2 Draining the Connection

Use `nc.Drain()` when gracefully shutting down a NATS connection.

```go
func ShutdownPublisher(nc *nats.Conn) {
    log.Println("Draining NATS connection...")

    if err := nc.Drain(); err != nil {
        log.Printf(
            "Error during NATS connection drain: %v",
            err,
        )
    }
}
```

Draining gives pending client work and publications an opportunity to complete before the connection closes.

It should not be treated as an unconditional guarantee that every application-buffered or pending message will be successfully delivered.

Applications should handle drain timeout or failure explicitly.

---

# 9. Publisher Lifecycle

The common publisher lifecycle is:

```text
Message Construction
        |
        v
Serialization
        |
        v
Header / Metadata Attachment
        |
        v
Application Buffer
        |
        v
NATS Client
        |
        v
Transport-Specific Publish
        |
        +------------------+
        |                  |
      Success            Failure
                           |
                    Retry / Requeue /
                    Reject / Escalate
```

The final publishing behavior depends on the selected NATS messaging model.

---

# 10. Core NATS Publishing

Core NATS publishing provides lightweight messaging without JetStream persistence or publish acknowledgements.

This section covers Core NATS-specific topics:

### 10.1 Basic Publish

```go
err := nc.Publish(
    "orders.fulfillment.created",
    payload,
)

if err != nil {
    return fmt.Errorf("publish failed: %w", err)
}
```

### 10.2 Publish Message

```go
msg := nats.NewMsg("orders.fulfillment.created")
msg.Data = payload

msg.Header.Set("Content-Type", "application/json")

if err := nc.PublishMsg(msg); err != nil {
    return fmt.Errorf("publish failed: %w", err)
}
```

### 10.3 Flush & Server Acknowledgement

Applications that require confirmation that pending client operations have been processed by the server can use the appropriate NATS client flush mechanism.

```go
if err := nc.Flush(); err != nil {
    return fmt.Errorf("NATS flush failed: %w", err)
}
```

A successful client publish operation and a successful flush should not be interpreted as JetStream persistence or a JetStream publish acknowledgement.

### 10.4 Core NATS Retry Considerations

Core NATS does not provide a JetStream-style publish acknowledgement.

Applications requiring stronger confirmation or persistence semantics should use JetStream.

---

# 11. JetStream Publishing

JetStream publishing provides persistence-oriented messaging semantics and publish acknowledgements.

This section covers JetStream-specific publisher behavior.

## 11.1 JetStream Context

Create a JetStream API context using the existing NATS connection:

```go
js, err := jetstream.New(nc)
if err != nil {
    return fmt.Errorf(
        "failed to initialize JetStream: %w",
        err,
    )
}
```

Creating the JetStream context does not create another NATS network connection.

---

## 11.2 Publish with Acknowledgement

JetStream publishing returns a publish acknowledgement.

```go
ack, err := js.Publish(
    ctx,
    "orders.fulfillment.created",
    payload,
)

if err != nil {
    return fmt.Errorf(
        "JetStream publish failed: %w",
        err,
    )
}

log.Printf(
    "Message stored in stream %s, sequence %d",
    ack.Stream,
    ack.Sequence,
)
```

The application can use the acknowledgement to determine whether the JetStream publish operation was accepted successfully.

---

## 11.3 Message Deduplication

JetStream supports message deduplication using the `Nats-Msg-Id` message header.

```go
msg := nats.NewMsg(
    "orders.fulfillment.created",
)

msg.Data = payload
msg.Header.Set(
    "Nats-Msg-Id",
    messageID,
)

ack, err := js.PublishMsg(ctx, msg)

if err != nil {
    return fmt.Errorf(
        "JetStream publish failed: %w",
        err,
    )
}
```

Applications should generate stable message identifiers when retrying the same logical message.

The identifier should represent the logical message rather than the individual retry attempt.

---

## 11.4 JetStream Retry

JetStream publish retries should use the same message identity when the application is retrying the same logical message.

```go
func PublishWithRetry(
    ctx context.Context,
    js jetstream.JetStream,
    subject string,
    data []byte,
    messageID string,
) (*jetstream.PubAck, error) {

    msg := nats.NewMsg(subject)
    msg.Data = data
    msg.Header.Set("Nats-Msg-Id", messageID)

    maxAttempts := 3

    var lastErr error

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        ack, err := js.PublishMsg(ctx, msg)

        if err == nil {
            return ack, nil
        }

        lastErr = err

        time.Sleep(
            time.Duration(attempt) * 100 * time.Millisecond,
        )
    }

    return nil, fmt.Errorf(
        "JetStream publish failed after %d attempts: %w",
        maxAttempts,
        lastErr,
    )
}
```

Applications should still use bounded retries and backoff.

JetStream deduplication reduces duplicate persistence when the same logical message is retried with the same message identifier, but applications should still design downstream processing to be idempotent.

---

# 12. Publisher Decision Model

The common publisher flow can be summarized as:

```text
                 Build Message
                       |
                       v
              Serialize Payload
                       |
                       v
              Attach Metadata
                       |
                       v
             Validate Message
                       |
                       v
              Select Transport
                 /           \
                /             \
               v               v
        Core NATS          JetStream
           |                   |
        Publish             Publish
           |                   |
        No PubAck            PubAck
           |                   |
       Application        Persistence /
       Error Handling     Deduplication
```

Applications should select the publishing model based on their delivery and persistence requirements rather than implementing transport-specific behavior in the common publisher layer.

---

## Official References

* NATS Go Client
* NATS Go Client Documentation
* NATS Go JetStream Package
* NATS JetStream Documentation
