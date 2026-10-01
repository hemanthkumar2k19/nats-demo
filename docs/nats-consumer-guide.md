# NATS JetStream Consumer Guide

## Purpose

This guide defines application-level practices for consuming persistent messages from NATS JetStream streams.

It covers the consumer lifecycle from stream prerequisites through consumer creation, configuration (mandatory vs updatable vs immutable fields, default values), consumption execution models (Pull, Ordered, Worker Pools), message acknowledgement semantics (`Ack`, `Nak`, `Term`, `InProgress`), redelivery backoff, message replay mechanisms, consumer pointer manipulation use cases, and graceful shutdown.

---

## 1. JetStream Consumer Architecture & Prerequisites

### 1.1 Prerequisites

Consuming messages from JetStream requires:
1. An active NATS connection (`*nats.Conn`).
2. A JetStream API context (`jetstream.JetStream`).
3. An existing target Stream configured on the NATS server.

---

### 1.2 Resource Hierarchy & Stream vs. Consumer Relationship

In NATS JetStream, resources follow a strict top-down parent-child hierarchy:

```text
NATS Server
  └── Account
       └── Stream (Parent Data Store: Ingests, Sequences, & Retains Raw Messages)
            ├── Child Consumer A ("order-worker-v1") [Read Pointer & ACK State Machine]
            └── Child Consumer B ("audit-logger")    [Read Pointer & ACK State Machine]
```

#### Hierarchy Rules & Signals

* **Parent Data Store vs. Child View**: The **Stream** is the physical message storage engine. A **Consumer** is a lightweight, stateful child view / read-pointer attached to that stream.
* **1:N Cardinality**: One parent Stream can host multiple ($N$) independent child Consumers simultaneously.
* **Lifespan Dependency**: Deleting a parent Stream automatically cascades and deletes all child Consumers attached to it. Conversely, deleting a Consumer has no effect on stored messages in the parent Stream.
* **State Tracking**: The Consumer tracks delivery sequence counters (`NumPending`, `NumAckPending`, `NumDelivered`) per subscriber without duplicating stream storage.

#### Fan-Out Delivery Across Consumers vs. Load-Balanced Workers

* **Multiple Consumers on Same Stream (Fan-Out / Broadcast)**:
  If Consumer A (`"order-worker-v1"`) and Consumer B (`"audit-logger"`) are both configured on stream `"ORDERS"` listening to subject `"orders.created"`, **every published message is delivered to BOTH consumers**. Each consumer maintains its own independent sequence pointer and pending ACK state.
* **Multiple Workers on the Same Consumer (Point-to-Point / Load Balanced)**:
  If multiple worker nodes subscribe to the **same consumer** (`"order-worker-v1"`), JetStream **load-balances** in-flight messages across those workers so each message is processed by only one worker node.

---

## 2. Consumer Configuration & Lifecycle Management

### 2.1 Durable vs. Ephemeral Consumers

* **Durable Consumer**: A named, persistent consumer configuration stored by the JetStream server. It retains its delivery state and sequence position across client restarts or network disconnects.
* **Ephemeral Consumer**: A temporary consumer created dynamically by a client. When the client disconnects or becomes inactive, the JetStream server automatically deletes the consumer. Ideal for real-time tailing or short-lived diagnostic tasks.

---

### 2.2 Core Consumer Configuration Parameters

The table below categorizes JetStream consumer configuration parameters by their operational update capabilities and default values:

| Parameter | Type / Option | Modifiability | Default Value (If Omitted) | Purpose & Operational Behavior |
| :--- | :--- | :--- | :--- | :--- |
| **Name / Durable** | `Name`, `Durable` | **Mandatory & Immutable** | *None* | Unique identifier for the consumer. Required for durable state tracking. Cannot be renamed once created. |
| **Ack Policy** | `AckPolicy` | **Mandatory & Immutable** | `AckExplicit` | `AckExplicit` (requires explicit ACK per message; required for work queues), `AckNone` (server auto-ACKs), `AckAll` (ACK N ACKs all prior messages). |
| **Deliver Policy** | `DeliverPolicy` | **Immutable** | `DeliverAll` | Starting stream position (`DeliverAll`, `DeliverLast`, `DeliverNew`, `DeliverByStartSequence`, `DeliverByStartTime`). Immutable after creation. |
| **OptStartSeq / Time** | `OptStartSeq`, `OptStartTime` | **Immutable** | *None* | Specific sequence number or timestamp offset when `DeliverPolicy` is sequence/time based. |
| **Replay Policy** | `ReplayPolicy` | **Immutable** | `ReplayInstant` | `ReplayInstant` (deliver pending messages as fast as possible) vs `ReplayOriginal` (replay matching original timestamps). |
| **Ack Wait** | `AckWait` | **Updatable** | `30s` (30 Seconds) | Duration server waits for an ACK before assuming delivery failure and triggering redelivery. Can be updated dynamically. |
| **Max Deliver** | `MaxDeliver` | **Updatable** | `-1` (Unlimited) | Maximum delivery attempts before JetStream halts redeliveries for a message. Can be updated dynamically. |
| **Backoff Schedule** | `Backoff` | **Updatable** | `nil` (Fixed `AckWait`) | List of explicit redelivery delay durations (e.g. `[]time.Duration{1s, 5s, 30s}`). Can be updated dynamically. |
| **Max Ack Pending** | `MaxAckPending` | **Updatable** | `1000` | Maximum in-flight unacknowledged messages allowed per consumer for flow control. Can be updated dynamically. |
| **Filter Subjects** | `FilterSubject`, `FilterSubjects` | **Updatable** | `""` (All Stream Subjects) | Subject pattern filter(s) for the consumer. Can be updated dynamically. |
| **Description** | `Description` | **Updatable** | `""` | Human-readable metadata description for the consumer. |

> **Updating Consumer Configuration:** Updatable fields can be modified on an existing consumer using `js.CreateOrUpdateConsumer` or `js.UpdateConsumer`. Attempting to modify **Immutable** fields on an existing consumer will return a server error (`10012 Consumer Delivery Policy Mutated`). Immutable fields require deleting and recreating the consumer.

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

### 3.1 Pull Consumers (`Consume` / `Fetch` / `FetchNoWait`)

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

Fetches whatever pending messages are currently available on the server without waiting if fewer than the requested batch size exist. Returns immediately.

```go
// Go SDK Example: Non-Blocking Batch Fetch (FetchNoWait)
batch, err := cons.FetchNoWait(10)
if err != nil {
    return err
}

for msg := range batch.Messages() {
    log.Printf("Non-blocking fetched msg seq %d: %s", msg.Sequence(), string(msg.Data()))
    _ = msg.Ack()
}
```

---

### 3.2 Ordered Consumers (`OrderedConsumer`)

An **Ordered Consumer** guarantees strict, in-order sequential delivery without gaps or duplicate processing.

> **Ephemeral Nature:** **Ordered Consumers are ALWAYS Ephemeral.** An Ordered Consumer cannot be durable. Because JetStream manages the underlying ephemeral consumer automatically — automatically destroying and recreating a new ephemeral consumer at sequence N+1 whenever a network gap, timeout, or server failover occurs — durable tracking is not permitted.

Key Characteristics:
* Automatically created as ephemeral consumers by the SDK driver.
* Configured with `AckNone` (server tracks sequence position automatically).
* Single worker concurrent processing model.
* Ideal for event-sourcing streams and ordered transaction logs.

```go
// Go SDK Example: Ordered Consumer Creation & Consumption (Always Ephemeral)
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

### 4.2 Redelivery, Message Replay & SDK Inspection APIs

#### Redelivery Limits (`MaxDeliver`)

When a message is NAKed or its `AckWait` timer expires without receiving an ACK, JetStream increments the delivery count (`msg.Metadata().NumDelivered`).

Once `NumDelivered >= MaxDeliver`, JetStream stops redelivering the message to consumers.

#### Message Replay Mechanisms & Replay Policies

JetStream allows applications to replay historical messages stored in a stream. Replay behavior is controlled by two configuration parameters:

1. **Replay Speed Policy (`ReplayPolicy`)**:
   * **`ReplayInstant` (Default)**: Replays historical messages as fast as the network and client processing allow. Ideal for rapid catch-up or bootstrapping state caches.
   * **`ReplayOriginal`**: Replays historical messages maintaining the exact relative time gaps (delays) between their original publication timestamps. Ideal for realistic workload simulation, time-series playback, or debugging past production traffic patterns.

2. **Historical Replay Start Offset (`DeliverPolicy`)**:
   * **`DeliverAll`**: Replay all stored messages from sequence 1.
   * **`DeliverLast`**: Start at the most recent message in the stream.
   * **`DeliverNew`**: Skip historical data; consume only new messages published after consumer creation.
   * **`DeliverByStartSequence`**: Start replay at exact sequence $N$ (`OptStartSeq`).
   * **`DeliverByStartTime`**: Start replay at exact timestamp $T$ (`OptStartTime`).
   * **`DeliverLastPerSubject`**: Start by delivering the single latest message for each filtered subject.

#### SDK Inspection APIs for Redelivery & Sequences

The NATS Client SDK provides rich metadata inspection capabilities on every received message (`jetstream.Msg`) and consumer handle (`jetstream.Consumer`):

1. **Inbound Message Metadata (`msg.Metadata()`)**:
   ```go
   // Go SDK Example: Inspect Message Redelivery & Sequence Metadata
   meta, err := msg.Metadata()
   if err == nil {
       log.Printf("Stream Sequence: %d, Consumer Sequence: %d", meta.Sequence.Stream, meta.Sequence.Consumer)
       log.Printf("Delivery Attempt Count: %d", meta.NumDelivered) // 1 = first attempt, >1 = REDELIVERY
       log.Printf("Remaining Pending Stream Messages: %d", meta.NumPending)
       log.Printf("Original Publication Timestamp: %v", meta.Timestamp)
   }

   // Detect if current message is a redelivery
   if meta.NumDelivered > 1 {
       log.Printf("WARNING: Processing redelivered message (attempt %d of %d)", meta.NumDelivered, maxDeliver)
   }
   ```

2. **Consumer-Level State Inspection (`cons.Info()`)**:
   ```go
   // Go SDK Example: Consumer Redelivery Statistics
   info, err := cons.Info(ctx)
   if err == nil {
       log.Printf("Total Consumer Redeliveries: %d", info.NumRedelivered)
       log.Printf("Unprocessed Messages Pending: %d", info.NumPending)
       log.Printf("In-Flight Unacknowledged Messages: %d", info.NumAckPending)
   }
   ```

---

### 4.3 Consumer Pointer Manipulation & Reset Use Cases

#### Do Developers Need to Manipulate Consumer Pointers?

**Yes.** A consumer pointer represents the consumer's current sequence position (`AckPending`, delivered stream sequence). While JetStream automatically advances the consumer pointer as messages are ACKed, developers occasionally need to **manually manipulate or reset consumer sequence pointers**.

#### Key Use Cases for Pointer Manipulation

1. **Time Travel / Bugfix Reprocessing**:
   When a bug is discovered in subscriber business logic, historical messages may have been processed incorrectly or NAKed. Developers fix the bug, then recreate or update the consumer pointer (`DeliverByStartSequence` or `DeliverByStartTime`) to rewind and re-process the exact range of historical messages.
2. **Skipping Poison Sequences / Fast-Forwarding**:
   If a corrupted sequence is stalling a consumer and cannot be handled, developers can fast-forward the consumer sequence pointer past the bad sequence to resume downstream processing.
3. **Disaster Recovery & Microservice Database Resync**:
   When restoring a downstream microservice database from a backup taken at timestamp $T$, the developer resets the consumer pointer using `DeliverByStartTime(T)` to re-sync event state precisely from the backup checkpoint.
4. **Automated Ephemeral Gap Recovery**:
   For Ordered Consumers, JetStream client SDK drivers manipulate consumer pointers under the hood automatically — recreating ephemeral consumers at sequence $N+1$ whenever network gaps or failovers occur.

```go
// Go SDK Example: Rewinding Consumer Pointer to Sequence 1000 for Reprocessing
func ResetConsumerPointer(ctx context.Context, js jetstream.JetStream, stream string, consumerName string, startSeq uint64) error {
	// Delete existing consumer pointer
	_ = js.DeleteConsumer(ctx, stream, consumerName)

	// Recreate consumer with DeliverByStartSequence pointing to startSeq
	cfg := jetstream.ConsumerConfig{
		Durable:       consumerName,
		DeliverPolicy: jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:   startSeq,
		AckPolicy:     jetstream.AckExplicit,
	}

	_, err := js.CreateOrUpdateConsumer(ctx, stream, cfg)
	if err != nil {
		return fmt.Errorf("failed to reset consumer pointer: %w", err)
	}

	log.Printf("Consumer %s sequence pointer successfully reset to start at sequence %d", consumerName, startSeq)
	return nil
}
```

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

| Capability | Pull Consumer (`Consume`) | Batch Fetch (`Fetch` / `FetchNoWait`) | Ordered Consumer |
| :--- | :--- | :--- | :--- |
| **Execution Model** | Continuous Push/Pull Callback | On-Demand Batch Pull | Continuous Sequential Callback |
| **ACK Policy** | `AckExplicit` | `AckExplicit` | `AckNone` (Auto-tracked) |
| **Flow Control** | `MaxAckPending` | Batch Size Limit | Single Stream Sequence |
| **Consumer Nature** | Durable or Ephemeral | Durable or Ephemeral | **Always Ephemeral** |
| **Horizontal Scaling** | Multi-Worker Load Balanced | Multi-Worker Load Balanced | Single Worker Instance |
| **Redelivery Support** | `Nak`, `NakWithDelay`, `MaxDeliver` | `Nak`, `NakWithDelay`, `MaxDeliver` | Auto-recreate on sequence gap |
| **Order Guarantee** | Parallel / Unordered across workers | Parallel / Unordered across workers | Strict In-Order Sequential |

---

## 7. Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client JetStream Package (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go/jetstream)
* [Official NATS JetStream Consumer Documentation](https://docs.nats.io/nats-concepts/jetstream/consumers)
