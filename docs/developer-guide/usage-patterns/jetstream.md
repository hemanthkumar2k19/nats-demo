# NATS JetStream Usage Patterns Guide

This guide details JetStream operations across three core categories: Publishing Patterns, Subscribing Patterns & Consumer Lifecycle, and Message Acknowledgment & Redelivery Patterns.

---

## 1. Publishing Patterns

### 1.1 Synchronous Publish
- **Stream Persistence**: Sends a message payload to a JetStream subject and blocks for server stream acknowledgment.
- **Publish Acknowledgment**: Returns metadata containing target stream name, sequence number, and duplicate status.

```go
// Synchronous publish to JetStream stream
ack, err := js.Publish(ctx, "orders.created", []byte(`{"order_id": "ORD-1001"}`))
if err == nil {
	log.Printf("Published to stream %s [seq=%d]", ack.Stream, ack.Sequence)
}
```

---

### 1.2 Publish with Headers & Deduplication
- **Deduplication**: Attaches a unique message ID header (`Nats-Msg-Id`) to eliminate duplicate message processing during retries or network failures.
- **Expectation Assertions**: Supports sequence and stream assertions (`ExpectStream`, `ExpectLastMsgID`, `ExpectLastSequence`) for optimistic concurrency.

```go
// Publish with deduplication ID and custom headers
msg := &nats.Msg{
	Subject: "orders.created",
	Header: nats.Header{
		"Nats-Msg-Id":    []string{"MSG-UNIQUE-1001"},
		"Correlation-ID": []string{"CORR-9988"},
	},
	Data: []byte(`{"order_id": "ORD-1001"}`),
}
ack, err := js.PublishMsg(ctx, msg)
```

---

### 1.3 Asynchronous Publish
- **High Throughput**: Non-blocking publish returning an asynchronous promise/future for high-concurrency publishing.
- **Background Acknowledgment**: Allows the application to continue work while the JetStream server processes acknowledgments in the background.

```go
// Non-blocking async publish
futureAck, err := js.PublishAsync("orders.created", []byte(`{"order_id": "ORD-1002"}`))

// Await async acknowledgment
select {
case ack := <-futureAck.Ok():
	log.Printf("Async Ack: stream=%s seq=%d", ack.Stream, ack.Sequence)
case err := <-futureAck.Err():
	log.Printf("Async Publish Error: %v", err)
}
```

---

### 1.4 Async Publish Completion
- **Buffer Flushing**: Blocks until all in-flight asynchronous publishes are processed and acknowledged by the server.
- **Zero Data Loss**: Ensures all pending async publishes complete before application shutdown.

```go
// Flush and await all pending async publish ACKs
select {
case <-js.PublishAsyncComplete():
	log.Println("All async publishes completed successfully.")
case <-time.After(5 * time.Second):
	log.Println("Timeout waiting for async publish completion.")
}
```

---

## 2. Subscribing Patterns & Consumer Lifecycle

### 2.1 Consumer Provisioning & Lifecycle

#### 2.1.1 Durable vs Ephemeral Consumer
- **Durable Consumer**: Persists consumer state and sequence progress on the server across client restarts.
- **Ephemeral Consumer**: Temporary consumer automatically deleted when the client disconnects or becomes inactive.
- **Delivery & Ack Policies**: Configures delivery policy (e.g. deliver all, deliver new, deliver last) and acknowledgment policy (e.g. explicit ack).

```go
// Create or update a durable JetStream consumer
cons, err := js.CreateOrUpdateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{
	Durable:       "order-processor",
	DeliverPolicy: jetstream.DeliverAllPolicy,
	AckPolicy:     jetstream.AckExplicitPolicy,
	AckWait:       30 * time.Second,
	MaxDeliver:    5,
})
```

---

#### 2.1.2 Ordered Consumer
- **In-Order Guarantee**: Creates a lightweight, auto-recreating consumer for strictly ordered message processing.
- **Simplified Lifecycle**: Eliminates manual sequence tracking and automatic consumer cleanup.

```go
// Create an ordered consumer for strict sequential processing
cons, err := js.OrderedConsumer(ctx, "ORDERS", jetstream.OrderedConsumerConfig{
	FilterSubjects: []string{"orders.created"},
})
```

---

#### 2.1.3 Consumer Lifecycle Management
- **State Query**: Inspects consumer state, pending message count, and redelivery stats.
- **Pause & Resume**: Temporarily halts message delivery to consumer.
- **Deletion**: Deletes consumer state from server.

```go
// Query consumer status
info, _ := cons.Info(ctx)
log.Printf("Pending msgs: %d", info.NumPending)

// Pause consumer delivery for 5 minutes
_ = cons.Pause(ctx, time.Now().Add(5*time.Minute))

// Delete consumer from stream
_ = js.RemoveConsumer(ctx, "ORDERS", "order-processor")
```

---

#### 2.1.4 Advanced Consumer & Backpressure Controls
- **Multi-Subject Filtering**: Binds a single consumer across multiple distinct subject patterns.
- **Backpressure Control**: Limits total unacknowledged in-flight messages to prevent worker memory exhaustion under heavy load.
- **Dead Letter Queue (DLQ) Mechanics**: Halts delivery after max retry limit is reached to isolate poison messages.

```go
// Advanced consumer with multi-subject filtering and backpressure limits
cons, err := js.CreateOrUpdateConsumer(ctx, "ORDERS", jetstream.ConsumerConfig{
	Durable:        "order-processor-advanced",
	FilterSubjects: []string{"orders.created", "orders.updated"},
	MaxAckPending:  100, // Limit in-flight unacked messages
	MaxDeliver:     5,   // Max retries before dead-lettering
	AckWait:        30 * time.Second,
})
```

---

#### 2.1.5 Direct Stream Message Inspection
- **Sequence Retrieval**: Fetches a specific message directly by its stream sequence number.
- **Subject State Query**: Fetches the latest stored message state for a specific subject without creating a consumer.

```go
// Fetch message directly by sequence number or latest by subject
stream, _ := js.Stream(ctx, "ORDERS")
msg, err := stream.GetMsg(ctx, 42)
latestMsg, err := stream.GetLastMsgForSubject(ctx, "orders.created")
```

---

### 2.2 Message Fetching Patterns

#### 2.2.1 Continuous Push-Style Consumption
- **Event-Driven**: Registers a continuous background callback for stream processing.
- **Auto-Prefetch**: Automatically manages credits and prefetches messages from server.

```go
// Continuous background consumption
cc, err := cons.Consume(func(msg jetstream.Msg) {
	log.Printf("Consumed: %s", string(msg.Data()))
	msg.Ack()
})
defer cc.Stop() // Stop consumer callback on shutdown
```

---

#### 2.2.2 Batch Pull-Style Fetch
- **On-Demand Batching**: Pulls an explicit batch of up to N messages or B bytes with a timeout deadline.
- **Flow Control**: Application controls exact batch processing rate and memory consumption.

```go
// Fetch batch of up to 10 messages with 2-second timeout
batch, err := cons.Fetch(10, jetstream.FetchMaxWait(2*time.Second))
if err == nil {
	for msg := range batch.Messages() {
		log.Printf("Fetched batch msg: %s", string(msg.Data()))
		msg.Ack()
	}
}
```

---

#### 2.2.3 Non-Blocking Batch Fetch
- **Immediate Return**: Pulls available messages immediately without waiting if batch is incomplete.

```go
// Fetch available messages without waiting
batch, err := cons.FetchNoWait(10)
if err == nil {
	for msg := range batch.Messages() {
		msg.Ack()
	}
}
```

---

#### 2.2.4 Single Message Fetch
- **Single Pull**: Retrieves the single next available message from consumer on-demand.

```go
// Fetch single next message
msg, err := cons.Next()
if err == nil {
	log.Printf("Next msg: %s", string(msg.Data()))
	msg.Ack()
}
```

---

## 3. Message Acknowledgment & Redelivery Patterns

### 3.1 Explicit Acknowledgment
- **Successful Processing**: Confirms message was successfully processed so JetStream updates consumer sequence.

```go
// Acknowledge message completion
msg.Ack()
```

---

### 3.2 Negative Acknowledgment / Redelivery
- **Immediate Redelivery**: Signals processing failure to request immediate message redelivery from JetStream server.

```go
// Request immediate redelivery
msg.Nak()
```

---

### 3.3 Delayed Redelivery / Backoff
- **Exponential Backoff**: Signals processing failure and requests redelivery after a specific backoff delay.

```go
// Request redelivery after 10 seconds
msg.NakWithDelay(10 * time.Second)
```

---

### 3.4 In-Progress Heartbeat
- **AckWait Extension**: Resets consumer processing timer for long-running operations to prevent premature redelivery.

```go
// Reset AckWait timer during long processing tasks
msg.InProgress()
```

---

### 3.5 Termination / Dead-Lettering
- **Poison Pill Removal**: Permanently halts redelivery for unprocessable or malformed messages.

```go
// Terminate redelivery permanently
msg.Term()
```
