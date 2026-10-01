# NATS Subscription Guide

## Purpose

This guide defines application-level practices for subscribing to and consuming messages using Core NATS.

It covers the subscriber lifecycle from subscription prerequisites through subscription patterns, execution models, failure and retry semantics, and graceful shutdown.

---

## 1. Subscription Prerequisites & Execution Models

### 1.1 Prerequisites

Subscribing to NATS messages requires an active, initialized NATS connection. Connection creation, reconnect configuration, and client lifecycle management are handled as prerequisites (refer to the NATS Client Connectivity Guide).

Conceptually:

```text
Active NATS Connection (TCP Socket)
      |
      +-- Core NATS Async Subscription (Callback / Event Dispatcher)
      |
      +-- Core NATS Sync Subscription (Polling Loop / Iterator)
      |
      +-- Core NATS Queue Group Subscription (Load-Balanced Worker Pool)
      |
      +-- Core NATS Channel / Dispatch Subscription (Buffered Pipeline)
```

---

### 1.2 Core NATS Subscription Models

Core NATS provides several subscription mechanisms depending on the execution model required by the application:

| Execution Model | Concept / Mechanism | Behavior | Typical Use Case |
| :--- | :--- | :--- | :--- |
| **Asynchronous Callback** | Callback Handler / Event Dispatcher | Client SDK invokes an application handler for each incoming message in background routines. | Real-time event handling, event-driven services, request-reply responders. |
| **Synchronous Pull / Polling** | Explicit Fetch / Iteration | Application thread explicitly pulls messages using a timeout parameter in its own loop. | Controlled processing rate, custom worker pools, batching. |
| **Channel / Queue Dispatch** | Buffered Channel / Pipeline | Client SDK pushes incoming messages into an application channel or queue. | Integration with worker pipelines and async select loops. |
| **Queue Group Subscription** | Server Load Balancer | NATS server load-balances messages across members of the same named group. | Horizontal scaling of worker nodes, distributed task distribution. |

---

## 2. Subscription Patterns & Buffer Options

### 2.1 Asynchronous Callback Subscriptions

An asynchronous subscription registers an event handler function that is automatically invoked by the NATS client SDK whenever a matching message arrives on a subject.

Standard Core NATS subscriptions operate in **Fan-Out (Broadcast)** mode: every active subscriber listening on the subject receives a copy of every published message.

```text
NATS Server -------> Client TCP Reader -------> Subscription Queue -------> Callback Routine
                                                                                    |
                                                                                    v
                                                                           Process / Respond
```

#### Basic Asynchronous Subscription

```go
// Go SDK Example
sub, err := nc.Subscribe("orders.created", func(msg *nats.Msg) {
    log.Printf("Received message on subject [%s]: %s", msg.Subject, string(msg.Data))

    // Inspect headers if present
    contentType := msg.Header.Get("Content-Type")
    correlationID := msg.Header.Get("X-Correlation-ID")

    log.Printf("Headers: Content-Type=%s, Correlation-ID=%s", contentType, correlationID)
})
if err != nil {
    return err
}
```

#### Callback Execution & Threading Considerations

* **Sequential Execution**: By default, the SDK creates a single background processing routine per subscription queue. Messages delivered to that subscription invoke the callback function **one at a time sequentially**.
* **Impact of Heavy Blocking Work**: If a callback performs slow synchronous operations (e.g. 5-second database queries or HTTP calls), the single subscription thread blocks. Incoming messages sent by the server accumulate in the client's internal ring buffer (`sub.Pending()`).
* **Slow Consumer Risk**: If incoming messages arrive faster than the blocked callback can return, the client buffer fills up and eventually triggers a **Slow Consumer Error** (`nats.ErrSlowConsumer`), dropping subsequent messages.
* **Remediation**: For heavy workloads, hand off message processing to a background worker pool or dispatch execution to a separate thread (`go process(msg)` in Go) so the subscription callback returns immediately and keeps reading incoming messages from the client queue.

#### Handling Requests & Replying to Request Subjects

When a client sends a request (e.g. via `nc.Request`), NATS attaches a unique inbox reply subject (`msg.Reply`). The subscriber acts as a **Responder** by processing the request and publishing a response back to `msg.Reply`.

The SDK provides helper methods (`msg.Respond` for raw payloads and `msg.RespondMsg` for structured messages with headers) which automatically use `msg.Reply` as the target subject, eliminating the need to manually format or publish to the inbox subject.

```text
Requester                                 NATS Server                              Responder
    |                                          |                                       |
    |-- 1. Request (Reply="_INBOX.123") ------>|-- 2. Delivers Request --------------->|
    |                                          |                                       |
    |                                          |<-- 3. msg.Respond() to "_INBOX.123" --|
    |<-- 4. Receives Response -----------------|                                       |
```

```go
// Go SDK Example: Responding to a Request Message inside Subscribe Handler
sub, err := nc.Subscribe("orders.validate", func(msg *nats.Msg) {
    log.Printf("Received validation request: %s", string(msg.Data))

    // Verify requester provided a reply subject
    if len(msg.Reply) == 0 {
        log.Printf("Warning: received message without reply subject")
        return
    }

    // Basic payload response using msg.Respond
    if string(msg.Data) == "ping" {
        _ = msg.Respond([]byte("pong")) // Publishes "pong" directly to msg.Reply
        return
    }

    // Structured response with headers using msg.RespondMsg
    replyMsg := nats.NewMsg(msg.Reply)
    replyMsg.Data = []byte(`{"valid": true, "reason": "Order approved"}`)
    replyMsg.Header.Set("Status-Code", "200")
    replyMsg.Header.Set("Content-Type", "application/json")

    if err := msg.RespondMsg(replyMsg); err != nil {
        log.Printf("Failed to send structured response: %v", err)
    }
})
```

---

### 2.2 Synchronous Pull / Polling Subscriptions

A synchronous subscription allows an application thread to explicitly pull messages using a polling method with a timeout parameter. 

#### Differences Between Asynchronous and Synchronous Subscriptions

| Dimension | Asynchronous Callback (`Subscribe`) | Synchronous Pull (`SubscribeSync`) |
| :--- | :--- | :--- |
| **Control Model** | **Push (SDK-Driven)**: SDK background thread automatically invokes callback as messages arrive. | **Pull (Application-Driven)**: Application explicitly calls `sub.NextMsg(timeout)` when ready to process. |
| **Threading** | Managed internally by NATS client SDK. | Managed entirely by application threads. |
| **Rate Control** | Rate determined by inbound server arrival rate. | Rate controlled by application polling loop, avoiding callback buffer overflows. |
| **Use Cases** | Real-time event handling, instant responders. | Controlled polling loops, batching, custom worker thread pools. |

```go
// Go SDK Example: Synchronous Polling Loop
sub, err := nc.SubscribeSync("orders.created")
if err != nil {
    return err
}

// Processing loop owned by application thread
for {
    msg, err := sub.NextMsg(1 * time.Second)
    if err != nil {
        if errors.Is(err, nats.ErrTimeout) {
            // Timeout reached with no messages available; continue polling
            continue
        }
        if errors.Is(err, nats.ErrBadSubscription) {
            // Subscription closed or unsubscribed
            log.Printf("Subscription closed, exiting pull loop")
            break
        }
        log.Printf("Error fetching next message: %v", err)
        break
    }

    // Process message
    log.Printf("Synchronously processed order message: %s", string(msg.Data))
}
```

---

### 2.3 Channel / Dispatch Subscriptions

Channel or queue dispatch subscriptions deliver incoming messages into an application-owned thread-safe queue or channel. Channel subscriptions operate in **Fan-Out** mode.

#### Multi-Language Pipeline Concepts

* **Go SDK**: Uses native Go CSP channels (`chan *nats.Msg`).
* **Java SDK**: Uses `Dispatcher` managing an internal thread-safe `BlockingQueue<Message>`.
* **Python SDK**: Uses `asyncio.Queue` or asynchronous coroutine iterators (`async for msg in sub:`).

```go
// Go SDK Example: Channel Subscription Pipeline
msgChan := make(chan *nats.Msg, 64)

sub, err := nc.ChanSubscribe("orders.created", msgChan)
if err != nil {
    return err
}

go func() {
    for msg := range msgChan {
        log.Printf("Received via channel: %s", string(msg.Data))
    }
}()
```

> **Warning:** Ensure the application buffer capacity is adequate and the consumer thread processes items quickly. If the channel buffer fills up, the NATS client driver will drop incoming messages and report a slow consumer error.

---

### 2.4 Queue Group Subscriptions

Core NATS provides distributed work-queue semantics using **Queue Groups**.

Unlike standard Pub/Sub subscriptions which broadcast every message to all active subscribers (fan-out), a Queue Group operates in **Point-to-Point (Load-Balanced)** mode. When multiple service instances or worker threads subscribe to the same subject with the same queue group name, the NATS server distributes each published message to **exactly one** worker instance in the group.

```text
                                                 +--> Worker 1 (Queue Group "order-workers")
                                                 |
Publisher --------> NATS Server (Load Balancer) -+--> Worker 2 (Queue Group "order-workers")
                                                 |
                                                 +--> Worker 3 (Queue Group "order-workers")
```

#### Worker Pool Example (Multiple Concurrent Queue Group Workers)

To demonstrate true queue group load balancing, the example below spins up a **pool of 3 worker routines** sharing the same queue group `"order-workers"`. When messages are published to `"orders.created"`, the NATS server load-balances messages across the 3 workers.

```go
// Go SDK Example: Queue Group Worker Pool (3 Concurrent Workers)
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

func StartWorkerPool(nc *nats.Conn, workerCount int) ([]*nats.Subscription, error) {
	subs := make([]*nats.Subscription, 0, workerCount)

	for i := 1; i <= workerCount; i++ {
		workerID := i
		// Each worker subscribes to "orders.created" under the SAME queue group name "order-workers"
		sub, err := nc.QueueSubscribe("orders.created", "order-workers", func(msg *nats.Msg) {
			log.Printf("[Worker-%d] Handling load-balanced order: %s", workerID, string(msg.Data))
		})
		if err != nil {
			return nil, fmt.Errorf("failed to start worker %d: %w", workerID, err)
		}
		subs = append(subs, sub)
	}

	log.Printf("Successfully started worker pool with %d workers on queue group 'order-workers'", workerCount)
	return subs, nil
}
```

#### Synchronous Queue Group Worker Loop

```go
// Go SDK Example: Synchronous Worker Pulling from Queue Group
sub, err := nc.QueueSubscribeSync("orders.created", "order-workers")
if err != nil {
    return err
}

msg, err := sub.NextMsg(2 * time.Second)
if err == nil {
    log.Printf("Worker pulled load-balanced message: %s", string(msg.Data))
}
```

---

### 2.5 Subscription Buffer Management & Control

#### Managing Subscription Pending Limits

Each subscription maintains an internal client-side buffer to store messages prior to handler execution. If processing is slower than the inbound network arrival rate, the buffer can fill up.

Application limits for message counts and memory byte sizes can be configured per subscription:

```go
// Go SDK Example
sub, err := nc.Subscribe("orders.created", handler)
if err != nil {
    return err
}

// Set maximum 5,000 pending messages or 10 MB buffer limit for this subscription
err = sub.SetPendingLimits(5000, 10*1024*1024)
if err != nil {
    return err
}
```

#### Inspecting Dropped Messages & Buffer Usage

```go
// Go SDK Example
msgs, bytes, err := sub.Pending()
if err == nil {
    log.Printf("Current pending: msgs=%d, bytes=%d", msgs, bytes)
}

dropped, err := sub.Dropped()
if err == nil && dropped > 0 {
    log.Printf("Warning: subscription has dropped %d messages due to slow consumption", dropped)
}
```

#### Auto-Unsubscribe

Auto-Unsubscribe configures a subscription to automatically unregister itself after receiving a target number of messages.

```go
// Go SDK Example
sub, err := nc.Subscribe("orders.notifications", handler)
if err != nil {
    return err
}

// Automatically unsubscribe after processing exactly 10 messages
err = sub.AutoUnsubscribe(10)
if err != nil {
    return err
}
```

---

## 3. Failure Handling & Resilience

### 3.1 Core NATS Messaging Semantics & Limitations

Core NATS provides **at-most-once delivery** semantics:

1. **No Storage or Retention**: If no subscriber is connected or actively listening on a subject when a message is published, the message is dropped by the server.
2. **No Application Acknowledgements**: Core NATS does not track message delivery, subscriber acknowledgements, or redelivery timeouts.
3. **At-Most-Once Delivery**: If a subscriber thread or handler panics/fails while processing a message, Core NATS will **not** redeliver the message.

> **Note:** Applications requiring persistent delivery, explicit message acknowledgements, redelivery backoff, or dead-letter queues should evaluate **JetStream Consumers**.

---

### 3.2 Common Core NATS Subscriber Error Conditions

| Error Condition | Origin | Description & Cause | Remediation / Handling Strategy |
| :--- | :--- | :--- | :--- |
| **Slow Consumer** | Client Driver | Subscriber processing rate is slower than inbound network arrival; pending client buffer limit reached. | Increase pending limits, optimize handler execution, or scale worker pool. |
| **Poll Timeout** | Client SDK | Synchronous pull method timed out waiting for a message. | Expected operational behavior in polling loops; continue loop. |
| **Invalid Subscription** | Client SDK | Operation attempted on an unregistered or closed subscription handle. | Terminate consumer loop and release resources. |
| **Connection Closed** | Client Driver | The underlying client NATS connection was closed. | Terminate subscriber routines or await reconnect notification. |
| **Stale Connection** | Server / Client | Connection ping/pong heartbeat timed out. | Client driver automatically attempts connection recovery. |

---

### 3.3 Client Reconnection Behavior for Subscriptions

When a network disruption occurs:

1. The client driver buffers outbound operations and retains active subscription definitions.
2. Upon successful reconnect to the NATS server, the client driver **automatically re-registers** all active subscriptions with the server.
3. Messages published to NATS while the client was disconnected are **not** replayed by Core NATS (ephemeral messaging).

---

### 3.4 In-Handler Error Recovery & Worker Safety

#### Exception / Panic Protection

An unhandled exception or panic inside a callback handler can crash the internal subscription thread or application process. Implement defensive exception handling inside message handlers:

```go
// Go SDK Example
sub, err := nc.Subscribe("orders.created", func(msg *nats.Msg) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Recovered from panic in message callback [subject=%s]: %v", msg.Subject, r)
        }
    }()

    // Process message payload safely
    processOrder(msg.Data)
})
```
---

## 4. Graceful Shutdown

Shutting down a subscriber cleanly prevents data loss for messages already buffered in client memory.

### 4.1 Unsubscribing vs. Draining

| Action | Concept / Mechanism | Behavior |
| :--- | :--- | :--- |
| **Unsubscribe** | Immediate Cancel | Immediately notifies the server to stop sending messages for this subscription and **discards** any unprocessed messages sitting in the local client buffer. |
| **Subscription Drain** | Graceful Flush | Notifies the server to stop routing new messages to this subscription, but **allows all already-buffered messages in the client queue to finish processing** before closing. |
| **Connection Drain** | Connection Flush | Drains **all** active subscriptions on the connection, waits for pending publications/flushes, and closes the connection cleanly. |

---

### 4.2 Graceful Teardown Sequence

```text
Trigger Shutdown Signal (OS Signal / Context Cancel)
       |
       v
Stop Receiving New Server Messages (Subscription Drain / Connection Drain)
       |
       v
Process All Messages Remaining in Client Ring Buffer
       |
       v
Signal Handler Completion (Wait Group / Thread Join)
       |
       v
Close Connection
```

---

### 4.3 Production Example: Graceful Subscriber Teardown

The runnable snippet below demonstrates how to construct a resilient, graceful Core NATS subscriber service using connection and subscription draining.

```go
// Go SDK Example
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
)

func main() {
	nc, err := nats.Connect(
		"nats://localhost:4222",
		nats.Name("graceful-subscriber-demo"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	var wg sync.WaitGroup

	// Subscribe to order events
	sub, err := nc.Subscribe("orders.created", func(msg *nats.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Printf("Started processing message: %s", string(msg.Data))
		// Simulate processing workload
		time.Sleep(200 * time.Millisecond)
		log.Printf("Finished processing message: %s", string(msg.Data))
	})
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	log.Printf("Subscribed to %s. Awaiting messages or shutdown signal...", sub.Subject)

	// Set up OS signal listening for graceful termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutdown signal received. Initiating graceful drain...")

	// Create a bounded shutdown context timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Begin draining subscription (stops new server messages, allows queue to flush)
	if err := sub.Drain(); err != nil {
		log.Printf("Error draining subscription: %v", err)
	}

	// Wait for in-flight goroutines and buffered messages to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("All buffered messages processed cleanly.")
	case <-shutdownCtx.Done():
		log.Printf("Shutdown context timed out before all messages finished processing.")
	}

	// Drain complete connection and close
	if err := nc.Drain(); err != nil {
		log.Printf("Error draining NATS connection: %v", err)
	}

	log.Printf("Subscriber service shutdown complete.")
}
```

---

## 5. Core NATS Subscriber Capability Summary

| Capability | Async Callback | Sync Pull | Channel / Queue | Queue Group | Responder Pattern |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Execution Model** | SDK Callback | Polling Loop | Channel / Queue Read | Callback / Polling | Callback / Polling |
| **Concurrency Control** | Sequential per Sub | Custom Pool | Channel Workers | Scaled Worker Nodes | Request Handler |
| **Delivery Model** | Fan-out (Broadcast) | Fan-out (Broadcast) | Fan-out (Broadcast) | Point-to-Point (Balanced) | Dependent on Subscription |
| **Pending Buffer Management** | Supported | Supported | Supported | Supported | Supported |
| **Auto-Unsubscribe Support** | Supported | Supported | Supported | Supported | Supported |
| **Graceful Drain Support** | Supported | Supported | Supported | Supported | Supported |

---

## 6. Official References

* [Official NATS Go Client Repository](https://github.com/nats-io/nats.go)
* [NATS Go Client API Documentation (pkg.go.dev)](https://pkg.go.dev/github.com/nats-io/nats.go)
* [Official NATS Developer Subscription Documentation](https://docs.nats.io/using-nats/developer/receive)
