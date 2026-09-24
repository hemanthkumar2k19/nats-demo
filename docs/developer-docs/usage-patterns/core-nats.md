# Core NATS Usage Patterns Guide

This guide details Core NATS features across three core categories: Publishing Patterns, Subscribing Patterns & Lifecycle, and Architectural Messaging Patterns.

---

## Part 1: Publishing Patterns

### 1. Simple Publish (`Publish`)
- **Fire-and-Forget**: Sends raw byte payload asynchronously to a subject.
- **At-Most-Once**: Message is delivered to active subscribers; unhandled messages are discarded.

```go
// Simple Publish (fire-and-forget)
err := nc.Publish("orders.created", []byte(`{"order_id": "ORD-1001"}`))
```

---

### 2. Publish with Reply Subject (`PublishRequest`)
- **Explicit Reply Routing**: Attaches a custom reply subject string to the published message payload.
- **Custom Response Paths**: Allows consumers to reply to a designated subject specified by publisher.

```go
// Publish with custom reply-to subject
err := nc.PublishRequest("orders.query", "orders.responses.custom", []byte(`{"order_id": "ORD-1001"}`))
```

---

### 3. Structured Message Publish (`PublishMsg`)
- **Metadata Support**: Sends a structured `nats.Msg` object containing custom headers (key-value metadata).
- **Control Headers**: Enables tracing, correlation IDs, and content-type metadata.

```go
// Publish structured message with headers
msg := &nats.Msg{
	Subject: "orders.created",
	Header:  nats.Header{"Correlation-ID": []string{"CORR-9988"}},
	Data:    []byte(`{"order_id": "ORD-1001"}`),
}
err := nc.PublishMsg(msg)
```

---

### 4. Synchronous Request (`Request`)
- **Synchronous Request-Reply**: Sends request payload and blocks waiting for a single response.
- **Automatic Inbox**: Automatically handles inbox creation, timeout enforcement, and cleanup.

```go
// Send request and block for response up to 2 seconds
reply, err := nc.Request("service.inventory.check", []byte(`{"item_id": "ITEM-42"}`), 2*time.Second)
if err == nil {
	log.Printf("Received reply: %s", string(reply.Data))
}
```

---

### 5. Structured Request (`RequestMsg`)
- **Headers in Requests**: Sends a structured `nats.Msg` with custom headers as a request.
- **Response Headers**: Returns response message including headers sent back by responder.

```go
// Send request with headers and wait for response
reqMsg := &nats.Msg{
	Subject: "service.inventory.check",
	Header:  nats.Header{"Trace-ID": []string{"TRACE-1234"}},
	Data:    []byte(`{"item_id": "ITEM-42"}`),
}
replyMsg, err := nc.RequestMsg(reqMsg, 2*time.Second)
```

---

### 6. Outbound Buffer Flush (`Flush`)
- **Socket Synchronization**: Flushes outbound client message buffer to network socket.
- **Delivery Verification**: Blocks until server acknowledges receipt of buffered data.

```go
// Flush outbound buffer to server with 2-second timeout
err := nc.FlushTimeout(2 * time.Second)
```

---

## Part 2: Subscribing - Patterns and Lifecycle

### 1. Asynchronous Callback Subscription (`Subscribe`)
- **Non-Blocking Callback**: Spawns an internal background goroutine to execute callback for each message.
- **Event-Driven**: Ideal for continuous event processing.

```go
// Asynchronous subscription using callback
sub, err := nc.Subscribe("orders.*", func(msg *nats.Msg) {
	log.Printf("Received on [%s]: %s", msg.Subject, string(msg.Data))
})
```

---

### 2. Synchronous Pull Subscription (`SubscribeSync`)
- **Blocking Pull Model**: Returns a subscription where application manually pulls messages via `NextMsg`.
- **Flow Control**: Application controls message consumption rate using timeout limits.

```go
// Synchronous subscription with manual pulling
syncSub, err := nc.SubscribeSync("orders.updates")
if err == nil {
	msg, err := syncSub.NextMsg(2 * time.Second) // Pull next message
	if err == nil {
		log.Printf("Pulled message: %s", string(msg.Data))
	}
}
```

---

### 3. Channel Subscription (`ChanSubscribe`)
- **Channel Delivery**: Delivers incoming `*nats.Msg` directly into a Go channel.
- **Concurrency Integration**: Integrates directly with Go worker pools and `select` blocks.

```go
// Channel-based subscription with buffered channel
msgChan := make(chan *nats.Msg, 64)
sub, err := nc.ChanSubscribe("orders.*", msgChan)

// Consume from channel
msg := <-msgChan
log.Printf("Channel received: %s", string(msg.Data))
```

---

### 4. Subscription Lifecycle & Flow Control

- **Auto-Unsubscribe (`AutoUnsubscribe`)**: Automatically unsubscribes after receiving `maxMsgs` messages.
- **Unsubscribe (`Unsubscribe`)**: Immediately cancels subscription interest on server and client.
- **Subscription Drain (`Drain`)**: Gracefully processes in-flight buffered messages before closing subscription.
- **Slow Consumer Protection (`SetPendingLimits`)**: Binds maximum pending message/byte buffers to prevent memory growth.

```go
sub, _ := nc.Subscribe("telemetry.*", func(msg *nats.Msg) {})

// 1. Auto-unsubscribe after receiving 100 messages
sub.AutoUnsubscribe(100)

// 2. Set slow consumer buffer limits (max 1000 msgs, 8MB)
sub.SetPendingLimits(1000, 8*1024*1024)

// 3. Graceful subscription drain
sub.Drain()

// 4. Immediate unsubscribe
sub.Unsubscribe()
```

---

## Part 3: Architectural Messaging Patterns

### 1. Publish / Subscribe (PubSub)
- **Asynchronous 1-to-N Pattern**: Publishers send messages to subject names without knowing active subscribers.
- **Real-Time Delivery**: Active subscribers receive messages published to matching subjects in real time.
- **At-Most-Once Delivery**: Messages published without active subscribers are not stored or replayed.

```go
package main

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Drain()

	// 1. Subscribe to subject
	sub, _ := nc.Subscribe("orders.created", func(msg *nats.Msg) {
		log.Printf("Received: %s", string(msg.Data))
	})
	defer sub.Unsubscribe()

	// 2. Publish message to subject
	nc.Publish("orders.created", []byte(`{"order_id": "ORD-1001"}`))
	nc.Flush()
}
```

---

### 2. Request / Response
- **Point-to-Point Pattern**: Requester sends a message and waits for a response from a service.
- **Inbox Routing**: Requester attaches a unique reply subject (`_INBOX.<id>`) to the request.
- **Direct Reply**: Responders publish replies directly to the inbox subject for delivery back to requester.

```go
package main

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Drain()

	// 1. Responder service
	nc.Subscribe("service.inventory.check", func(msg *nats.Msg) {
		if msg.Reply != "" {
			// Helper to reply directly to msg.Reply inbox
			msg.Respond([]byte(`{"item_id": "ITEM-42", "in_stock": true}`))
		}
	})

	// 2. Requester (sends request and waits for reply)
	reply, err := nc.Request("service.inventory.check", []byte(`{"item_id": "ITEM-42"}`), 2*time.Second)
	if err == nil {
		log.Printf("Reply received: %s", string(reply.Data))
	}
}
```

---

### 3. Scatter and Gather
- **1-to-N Query Pattern**: Requester broadcasts a request to multiple worker services simultaneously.
- **Inbox Aggregation**: Requester attaches a single reply inbox to aggregate incoming responses.
- **Bounded Response Collection**: Requester gathers responses until expected count or timeout deadline is reached.

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Drain()

	subject := "quote.compute"

	// Simulate 3 worker nodes
	for i := 1; i <= 3; i++ {
		workerID := fmt.Sprintf("worker-%d", i)
		nc.Subscribe(subject, func(msg *nats.Msg) {
			if msg.Reply != "" {
				msg.Respond([]byte(fmt.Sprintf(`{"worker": "%s", "price": %d}`, workerID, 100+i*5)))
			}
		})
	}

	// 1. Create inbox & sync subscription
	replyInbox := nc.NewRespInbox()
	sub, _ := nc.SubscribeSync(replyInbox)
	defer sub.Unsubscribe()

	// 2. Broadcast scatter request
	nc.PublishMsg(&nats.Msg{Subject: subject, Reply: replyInbox, Data: []byte(`{"req_id": "R1"}`)})

	// 3. Gather responses within 1 second deadline
	expected := 3
	deadline := time.Now().Add(1 * time.Second)
	var gathered [][]byte

	for len(gathered) < expected {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		msg, err := sub.NextMsg(remaining)
		if err != nil {
			break
		}
		gathered = append(gathered, msg.Data)
		log.Printf("Gathered response #%d: %s", len(gathered), string(msg.Data))
	}
}
```

---

### 4. Queue Groups (Load Balancing)
- **Load Balanced Delivery**: Subscribers sharing a Queue Group Name form a competing consumer pool.
- **1-of-N Distribution**: NATS server routes each message on the subject to exactly one group member.
- **Coexistence**: Non-queue subscribers on the same subject still receive their own copy of every message.

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer nc.Drain()

	subject := "tasks.process"
	queueGroup := "payment-processors"

	// 1. Register 3 workers under the same queue group
	for i := 1; i <= 3; i++ {
		workerID := fmt.Sprintf("worker-%d", i)
		nc.QueueSubscribe(subject, queueGroup, func(msg *nats.Msg) {
			log.Printf("[%s] Processed task: %s", workerID, string(msg.Data))
		})
	}

	// 2. Publish 6 tasks (balanced across workers)
	for t := 1; t <= 6; t++ {
		nc.Publish(subject, []byte(fmt.Sprintf(`{"task_id": "TASK-%d"}`, t)))
	}

	nc.Flush()
	time.Sleep(500 * time.Millisecond)
}
```
