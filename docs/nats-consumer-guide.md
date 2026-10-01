# NATS JetStream Consumer Guide

## Purpose

This guide defines application-level practices for consuming persistent messages from NATS JetStream streams.

It covers the consumer lifecycle from stream prerequisites through consumer creation, configuration, consumption execution models (Pull, Ordered, Worker Pools), message acknowledgement semantics (`Ack`, `Nak`, `Term`, `InProgress`), redelivery backoff strategies, and graceful shutdown.

---

## 1. JetStream Consumer Architecture & Prerequisites

### 1.1 Prerequisites

Consuming messages from JetStream requires:
1. An active NATS connection (`*nats.Conn`).
2. A JetStream API context (`jetstream.JetStream`).
3. An existing target Stream configured on the NATS server.

### 1.2 Stream vs. Consumer Relationship

* **Stream**: Stores and retains published messages sequentially according to configured retention policies (`Limits`, `WorkQueue`, `Interest`).
* **Consumer**: A stateful read-pointer and state tracking mechanism attached to a stream. A consumer tracks message delivery state, sequence numbers, and acknowledgements per client subscriber.

Conceptually:

```text
JetStream Context
      |
      +--> Stream ("ORDERS") [Stores Messages 1..N]
               |
               +--> Durable Consumer ("order-worker-v1") [Tracked Sequence & Acks]
               |
               +--> Ephemeral Consumer ("monitoring-tail") [Auto-cleaned on exit]
```

---

## 2. Consumer Configuration & Lifecycle Management

### 2.1 Durable vs. Ephemeral Consumers

* **Durable Consumer**: A named, persistent consumer configuration stored by the JetStream server. It retains its delivery state and sequence position across client restarts or network disconnects.
* **Ephemeral Consumer**: A temporary consumer created dynamically by a client. When the client disconnects or becomes inactive, the JetStream server automatically deletes the consumer. Ideal for real-time tailing or short-lived diagnostic tasks.

---

### 2.2 Core Consumer Configuration Parameters

| Parameter Concept | JetStream Config Option | Purpose & Operational Behavior |
| :--- | :--- | :--- |
| **Name / Durable** | `Name`, `Durable` | Unique identifier for durable tracking across client restarts. |
| **Deliver Policy** | `DeliverPolicy` | Specifies where in the stream to start consuming (`DeliverAll`, `DeliverLast`, `DeliverNew`, `DeliverByStartSequence`, `DeliverByStartTime`, `DeliverLastPerSubject`). |
| **Ack Policy** | `AckPolicy` | `AckExplicit` (requires explicit client ACK per message; recommended for work queues), `AckNone` (server auto-ACKs on delivery), `AckAll` (ACKing sequence N ACKs all prior messages). |
| **Ack Wait** | `AckWait` | Duration server waits for an ACK before assuming delivery failure and triggering redelivery (e.g. `30s`). |
| **Max Deliver** | `MaxDeliver` | Maximum delivery attempts before JetStream halts redelivery for a message (e.g. `5`). |
| **Backoff Schedule** | `Backoff` | List of explicit redelivery delay durations between retry attempts (e.g. `1s`, `5s`, `30s`). |
| **Replay Policy** | `ReplayPolicy` | `ReplayInstant` (deliver pending messages as fast as possible), `ReplayOriginal` (replay messages matching original publication timestamps). |
| **Filter Subjects** | `FilterSubject`, `FilterSubjects` | Filter stream messages to specific subject patterns (e.g., `orders.created.*`). |
| **Max Ack Pending** | `MaxAckPending` | Maximum unacknowledged in-flight messages allowed per consumer for flow control and backpressure (e.g., `100`). |

---

### 2.3 Consumer Lifecycle Operations

#### Creating or Updating a Consumer

```go
// Go SDK Example: Create or Update a Durable Consumer Configuration
cfg := jetstream.ConsumerConfig{
    Durable:       "order-worker-v1",
    DeliverPolicy: jetstream.DeliverAllPolicy,
    AckPolicy:     jetstream.AckExplicit,
    AckWait:       30 * time.Second,
    MaxDeliver:    5,
    Backoff:       []time.Duration{1 * time.Second, 5 * time.Second, 15 * time.Second},
    FilterSubject: "orders.created",
    MaxAckPending: 100,
}

cons, err := js.CreateOrUpdateConsumer(ctx, "ORDERS", cfg)
if err != nil {
    return err
}
```

#### Inspecting Consumer Status

```go
// Go SDK Example: Inspect Consumer State
info, err := cons.Info(ctx)
if err == nil {
    log.Printf("Consumer %s on Stream %s: Pending=%d, AckPending=%d, Redelivered=%d",
        info.Name, info.Stream, info.NumPending, info.NumAckPending, info.NumRedelivered)
}
```

#### Pausing and Resuming Consumers

Consumers can be paused to halt message delivery during maintenance or downstream dependency outages, then resumed.

```go
// Go SDK Example: Pause Consumer for 5 Minutes
pauseResp, err := js.PauseConsumer(ctx, "ORDERS", "order-worker-v1", time.Now().Add(5*time.Minute))
if err == nil {
    log.Printf("Consumer paused until: %v", pauseResp.PauseUntil)
}

// Go SDK Example: Manually Resume Consumer
resumeResp, err := js.ResumeConsumer(ctx, "ORDERS", "order-worker-v1")
if err == nil {
    log.Printf("Consumer resumed: %v", resumeResp.Paused)
}
```

#### Deleting a Consumer

```go
// Go SDK Example: Delete Consumer
err := js.DeleteConsumer(ctx, "ORDERS", "order-worker-v1")
if err != nil {
    return err
}
```

---

## 3. Consumption Patterns & Execution Models

### 3.1 Pull Consumers (`Consume` / `Fetch`)

Pull consumers give client applications complete control over message delivery speed and backpressure. The client explicitly requests or streams batches of messages from JetStream.

#### Continuous Pull Consumption (`cons.Consume`)

The SDK continuously pulls message batches in the background and executes the application handler callback.

```go
// Go SDK Example: Continuous Pull Consumption
consCtx, err := cons.Consume(func(msg jetstream.Msg) {
    log.Printf("Received msg seq %d: %s", msg.Sequence(), string(msg.Data()))

    // Acknowledge message processing completion
    if err := msg.Ack(); err != nil {
        log.Printf("Failed to ACK message: %v", err)
    }
})
if err != nil {
    return err
}
defer consCtx.Stop()
```

#### Batch Pull Fetch (`cons.Fetch`)

Pull a specific maximum batch size or wait up to a maximum duration timeout.

```go
// Go SDK Example: Batch Fetch
batch, err := cons.Fetch(10, jetstream.FetchMaxWait(2*time.Second))
if err != nil {
    return err
}

for msg := range batch.Messages() {
    log.Printf("Fetched msg seq %d: %s", msg.Sequence(), string(msg.Data()))
    _ = msg.Ack()
}
```

#### Non-Blocking Batch Fetch (`cons.FetchNoWait`)

Fetches whatever pending messages are currently available on the server without waiting if fewer than the requested count exist.

---

### 3.2 Ordered Consumers (`OrderedConsumer`)

An **Ordered Consumer** guarantees strict, in-order sequential delivery without gaps or duplicate processing.

If a message delivery gap occurs or network disruption happens, JetStream automatically resets and recreates an ephemeral consumer starting precisely at the last successfully acknowledged sequence number.

* Configured with `AckNone` (server tracks sequence position automatically).
* Single concurrent processing model.
* Ideal for event-sourcing streams and ordered transaction logs.

```go
// Go SDK Example: Ordered Consumer Creation & Consumption
cons, err := js.OrderedConsumer(ctx, "ORDERS", jetstream.OrderedConsumerConfig{
    FilterSubjects: []string{"orders.created"},
})
if err != nil {
    return err
}

consCtx, err := cons.Consume(func(msg jetstream.Msg) {
    log.Printf("Ordered event seq %d: %s", msg.Sequence(), string(msg.Data()))
})
if err != nil {
    return err
}
defer consCtx.Stop()
```

---

### 3.3 Consumer Worker Pools (Horizontal Load Balancing)

Multiple consumer instances or worker threads can pull messages concurrently from the **same durable consumer** on a stream. JetStream automatically load-balances in-flight unacknowledged messages across the active worker pool up to the `MaxAckPending` limit.

```text
JetStream Consumer ("order-worker-v1")
       |
       +--> Worker Node 1 (cons.Consume) ---> Processes Msg Seq 101 (Ack)
       |
       +--> Worker Node 2 (cons.Consume) ---> Processes Msg Seq 102 (Ack)
       |
       +--> Worker Node 3 (cons.Consume) ---> Processes Msg Seq 103 (Ack)
```

```go
// Go SDK Example: Worker Pool Consuming from a Shared Durable Consumer
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func StartConsumerWorkerPool(ctx context.Context, js jetstream.JetStream, stream string, consumerName string, workerCount int) error {
	cons, err := js.Consumer(ctx, stream, consumerName)
	if err != nil {
		return fmt.Errorf("failed to bind consumer: %w", err)
	}

	for i := 1; i <= workerCount; i++ {
		workerID := i
		go func() {
			consCtx, err := cons.Consume(func(msg jetstream.Msg) {
				log.Printf("[Worker-%d] Processing msg seq %d on subject %s", workerID, msg.Sequence(), msg.Subject())
				// Simulate processing workload
				time.Sleep(100 * time.Millisecond)
				_ = msg.Ack()
			})
			if err != nil {
				log.Printf("[Worker-%d] Consumption error: %v", workerID, err)
				return
			}
			<-ctx.Done()
			consCtx.Stop()
		}()
	}

	log.Printf("Started %d workers for durable consumer %s", workerCount, consumerName)
	return nil
}
```

---

## 4. Message Acknowledgement & Failure Semantics

### 4.1 Message Acknowledgement Types

When `AckPolicy` is set to `AckExplicit`, JetStream requires explicit client feedback. Consumers notify JetStream of message processing outcomes using 4 primary ACK signals:

| ACK Method | Concept | Behavior & Server Action | Typical Use Case |
| :--- | :--- | :--- | :--- |
| `msg.Ack()` | Positive ACK | Confirms successful message processing. Server removes message from consumer pending list. | Successful processing complete. |
| `msg.Nak()` / `msg.NakWithDelay(d)` | Negative ACK | Signals processing failure. Server reschedules message for immediate or delayed redelivery. | Transient failure (e.g. database timeout, rate limit). |
| `msg.Term()` | Terminate Message | Signals permanent unprocessable failure. Server halts all redeliveries for this message immediately. | Poison pill payload, schema corruption, fatal validation error. |
| `msg.InProgress()` | Heartbeat / Extend Wait | Resets the server `AckWait` timer for this specific message. | Long-running tasks (e.g. video processing, batch report generation). |

```go
// Go SDK Example: Handling ACK Signals Based on Error Classification
consCtx, err := cons.Consume(func(msg jetstream.Msg) {
    err := processOrder(msg.Data())
    if err == nil {
        _ = msg.Ack() // Successful processing
        return
    }

    if errors.Is(err, ErrPermanentSchemaCorruption) {
        log.Printf("Poison pill detected at seq %d, terminating redelivery: %v", msg.Sequence(), err)
        _ = msg.Term() // Permanent failure; do not redeliver
        return
    }

    if errors.Is(err, ErrTransientDatabaseTimeout) {
        log.Printf("Transient error at seq %d, requesting delayed retry in 10s: %v", msg.Sequence(), err)
        _ = msg.NakWithDelay(10 * time.Second) // Reschedule retry after 10 seconds
        return
    }

    // Default immediate negative acknowledgement
    _ = msg.Nak()
})
```

---

### 4.2 Redelivery, MaxDeliver & DLQ System Advisories

#### Redelivery Limits (`MaxDeliver`)

When a message is NAKed or its `AckWait` timer expires without receiving an ACK, JetStream increments the delivery count (`msg.Metadata().NumDelivered`).

Once `NumDelivered >= MaxDeliver`, JetStream stops redelivering the message to consumers.

#### System DLQ Advisory Events

When a message exceeds `MaxDeliver` or receives `msg.Term()`, JetStream automatically emits a **System Advisory Event** on the subject:

`$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.<STREAM>.<CONSUMER>`

Advisories contain metadata payload details such as `stream`, `consumer`, `stream_seq`, and `deliveries`. Monitoring services or DLQ workers can subscribe to `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>` to automatically log or forward failed message sequences to a dedicated DLQ stream.

---

## 5. Graceful Shutdown & Consumer Drainage

### 5.1 Consumer Context Stop (`Stop`) vs. Drain (`Drain`)

| Shutdown Method | Action | Behavior |
| :--- | :--- | :--- |
| `consCtx.Stop()` | Stop Pulling | Immediately stops pulling new messages from JetStream; in-flight handler callbacks complete. |
| `consCtx.Drain()` | Graceful Flush | Allows all currently pulled/in-flight messages to complete processing and ACK before shutting down. |

---

### 5.2 Production Example: Resilient JetStream Consumer Service

The runnable snippet below demonstrates how to construct a resilient JetStream consumer service with durable consumer binding, worker handling, and graceful teardown.

```go
// Go SDK Example: Complete Resilient JetStream Consumer Service
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	nc, err := nats.Connect("nats://localhost:4222", nats.Name("jetstream-consumer-demo"))
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to initialize JetStream: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure durable consumer exists
	cfg := jetstream.ConsumerConfig{
		Durable:       "order-processing-consumer",
		AckPolicy:     jetstream.AckExplicit,
		AckWait:       15 * time.Second,
		MaxDeliver:    3,
		FilterSubject: "orders.created",
		MaxAckPending: 50,
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, "ORDERS", cfg)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	var wg sync.WaitGroup

	// Start continuous consumer callback
	consCtx, err := cons.Consume(func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Printf("Processing order seq %d [attempt %d]: %s",
			msg.Sequence(), msg.Metadata().NumDelivered, string(msg.Data()))

		// Simulate processing workload
		time.Sleep(100 * time.Millisecond)

		if err := msg.Ack(); err != nil {
			log.Printf("Failed to ACK msg seq %d: %v", msg.Sequence(), err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to start consume: %v", err)
	}
	log.Printf("Consumer started. Listening for orders...")

	// Listen for OS termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutdown signal received. Stopping consumer...")

	// Stop fetching new messages from JetStream
	consCtx.Stop()

	// Bounded shutdown context to wait for in-flight tasks to finish ACKing
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("All in-flight messages ACKed cleanly.")
	case <-shutdownCtx.Done():
		log.Printf("Shutdown timeout reached before all in-flight messages completed.")
	}

	log.Printf("JetStream consumer service shutdown complete.")
}
```

---

## 6. JetStream Consumer Capability Matrix

| Capability | Pull Consumer (`Consume`) | Batch Fetch (`Fetch`) | Ordered Consumer |
| :--- | :--- | :--- | :--- |
| **Execution Model** | Continuous Push/Pull Callback | On-Demand Batch Pull | Continuous Sequential Callback |
| **ACK Policy** | `AckExplicit` | `AckExplicit` | `AckNone` (Auto-tracked) |
| **Flow Control** | `MaxAckPending` | Batch Size Limit | Single Stream Sequence |
| **Horizontal Scaling** | Multi-Worker Load Balanced | Multi-Worker Load Balanced | Single Worker Instance |
| **Redelivery Support** | `Nak`, `NakWithDelay`, `MaxDeliver` | `Nak`, `NakWithDelay`, `MaxDeliver` | Auto-recreate on sequence gap |
| **Order Guarantee** | Parallel / Unordered across workers | Parallel / Unordered across workers | Strict In-Order Sequential |

---

## 7. Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client JetStream Package (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
* [Official NATS JetStream Consumer Documentation](https://docs.nats.io/nats-concepts/jetstream/consumers)
