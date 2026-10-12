# NATS Core Messaging Guide

## 1. Overview and When to Use

Core NATS is the high-performance, in-memory messaging engine at the foundation of the NATS platform. It operates on an "at-most-once" (fire-and-forget) delivery model with sub-millisecond latencies, minimal memory footprint, and high message throughput.

### When to Use Core NATS

Core NATS is ideal for:

- **Ephemeral Notifications & Real-Time Telemetry:** Broadcasting high-volume event streams (IoT sensor metrics, market data feeds, presence notifications) where the latest state matters more than historical replay.
- **Low-Latency Service Communication:** Fast request-reply RPC interactions where services need quick point-in-point responses without broker persistence overhead.
- **Dynamic Fan-out & Broadcast:** Distributing messages across an arbitrary number of active subscribers without maintaining per-subscriber state on the broker.
- **Workload Load Balancing:** Horizontal load distribution across stateless worker pools using Queue Groups.

### When NOT to Use Core NATS (Use JetStream Instead)

Do NOT use Core NATS if your application requires:

- Guaranteed delivery (at-least-once or exactly-once).
- Message persistence across broker restarts or network disconnections.
- Message replay or time-travel consumption for offline/recovering subscribers.
- Message deduplication, consumer acknowledgements, or dead-letter queues.

**Official reference:** [Core NATS Documentation](https://docs.nats.io/learn/core-nats/)

---

## 2. Publish-Subscribe

Publish-Subscribe (Pub/Sub) is the fundamental communication pattern in Core NATS. It provides one-to-many message broadcast where publishers publish messages to a subject, and any subscriber listening to matching subject patterns receives a copy in real time.

- Publishers and subscribers are completely decoupled: publishers send messages regardless of whether subscribers exist.
- If no subscribers are active on a subject when a message is published, the message is immediately dropped by the broker without error.

**Official reference:** [Core NATS - Publish-Subscribe](https://docs.nats.io/learn/core-nats/publish-subscribe)

### Reference Links (Go & Java Standards)

#### Go
1. **Examples:** [NATS by Example - Publish-Subscribe](https://natsbyexample.com/examples/messaging/pub-sub/cli) (Use the language selector to switch to Go)
2. **API References:** [nats.Conn.Publish API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Publish) and [nats.Conn.Subscribe API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Subscribe)
3. **Code Files References:** [nats-pub on GitHub](https://github.com/nats-io/nats.go/blob/main/examples/nats-pub/main.go), [nats-sub on GitHub](https://github.com/nats-io/nats.go/blob/main/examples/nats-sub/main.go), and [nats.go Basic Usage](https://github.com/nats-io/nats.go#basic-usage)

#### Java
1. **Examples:** [NATS by Example - Publish-Subscribe](https://natsbyexample.com/examples/messaging/pub-sub/cli) (Use the language selector to switch to Java when available)
2. **API References:** [Connection.publish Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#publish-java.lang.String-byte:A-) and [Dispatcher.subscribe Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Dispatcher.html#subscribe-java.lang.String-)
3. **Code Files References:** [nats.java Examples Directory](https://github.com/nats-io/nats.java/tree/main/examples) and [nats.java Basic Usage](https://github.com/nats-io/nats.java#basic-usage)

---

## 3. Request-Reply

Request-Reply enables synchronous or coordinated request-response communication between two endpoints.

Under the hood, NATS accomplishes request-reply over standard pub/sub:
1. The requester creates an ephemeral, unique reply subject (inbox) beginning with `_INBOX.`.
2. The requester publishes the message with the `Reply` field set to that inbox.
3. The requester awaits a response on that inbox until a configured timeout occurs.
4. The replier receives the message, processes it, and publishes the response directly to the `msg.Reply` subject.

**Official reference:** [Core NATS - Request-Reply](https://docs.nats.io/learn/core-nats/request-reply)

### Reference Links (Go & Java Standards)

#### Go
1. **Examples:** [NATS by Example - Request-Reply](https://natsbyexample.com/examples/messaging/request-reply/go) (Use the language selector to switch to Go)
2. **API References:** [nats.Conn.Request API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.Request) and [nats.Msg.Respond API](https://pkg.go.dev/github.com/nats-io/nats.go#Msg.Respond)
3. **Code Files References:** [nats-req on GitHub](https://github.com/nats-io/nats.go/blob/main/examples/nats-req/main.go), [nats-rply on GitHub](https://github.com/nats-io/nats.go/blob/main/examples/nats-rply/main.go), and [nats.go Request-Reply Usage](https://github.com/nats-io/nats.go#basic-usage)

#### Java
1. **Examples:** [NATS by Example - Request-Reply](https://natsbyexample.com/examples/messaging/request-reply/java)
2. **API References:** [Connection.request Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Connection.html#request-java.lang.String-byte:A-java.time.Duration-) and [Message.getReplyTo Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Message.html#getReplyTo--)
3. **Code Files References:** [nats.java Request](https://github.com/nats-io/nats.java/blob/main/examples/src/main/java/io/nats/examples/NatsReq.java) and [nats.Java Replay](https://github.com/nats-io/nats.java/blob/main/examples/src/main/java/io/nats/examples/NatsReply.java)

### 3.1 Scatter-Gather

Scatter-Gather is a specialized variant of Request-Reply where a single request is broadcast to multiple responders, and the requester gathers multiple replies over a defined time window.

- The requester creates a single inbox and subscribes to it.
- The request is published to a subject where multiple worker instances or services are listening.
- Each replier independently sends its response to the shared inbox.
- The requester collects responses until a deadline expires or a target number of responses is received.

**Official reference:** [Core NATS - Scatter-Gather](https://docs.nats.io/learn/core-nats/scatter-gather)

---

## 4. Queue Groups

Queue Groups provide automatic, broker-managed load balancing across a pool of subscribers listening to the same subject.

- Multiple subscriber instances join a shared queue group by specifying a group name (e.g., `order-workers`).
- When a message is published to the subject, NATS delivers it to **exactly one** subscriber in the queue group chosen pseudo-randomly.
- Other queue groups on the same subject each receive their own copy of the message, enabling pub/sub across logical services and load balancing within each service.
- If queue subscribers fail or drop off, the broker immediately re-distributes incoming traffic across remaining active members without manual rebalancing.

**Official reference:** [Core NATS - Queue Groups](https://docs.nats.io/learn/core-nats/queue-groups)

### Reference Links (Go & Java Standards)

#### Go
1. **Examples:** [NATS by Example](https://natsbyexample.com/) (Core messaging patterns)
2. **API References:** [nats.Conn.QueueSubscribe API](https://pkg.go.dev/github.com/nats-io/nats.go#Conn.QueueSubscribe)
3. **Code Files References:** [nats-qsub on GitHub](https://github.com/nats-io/nats.go/blob/main/examples/nats-qsub/main.go)

#### Java
1. **Examples:** [NATS by Example](https://natsbyexample.com/) (Core messaging patterns)
2. **API References:** [Dispatcher.subscribe (Queue Group) Javadoc](https://javadoc.io/doc/io.nats/jnats/latest/io/nats/client/Dispatcher.html#subscribe-java.lang.String-java.lang.String-)
3. **Code Files References:** [nats.java Basic Usage](https://github.com/nats-io/nats.java/blob/main/examples/src/main/java/io/nats/examples/NatsSubQueue.java)

---

## 5. Best Practices

| Suggestion | Reason |
|---|---|
| Use JetStream instead of Core NATS whenever delivery guarantees, persistence, or message replay are required. | Core NATS operates on an at-most-once (fire-and-forget) delivery model; messages published while a subscriber is offline or disconnected are permanently discarded. |
| Keep subscription callbacks lightweight and offload heavy processing to internal worker pools or bounded worker queues (`chan` in Go, `BlockingQueue` in Java). | Executing long-running I/O or database calls inside subscription callbacks causes client-side buffers to overflow, triggering `ErrSlowConsumer` and dropping incoming messages. |
| Design request-reply service handlers to be strictly idempotent. | A client-side timeout on `nc.Request` indicates only that a response was not received within the window; the replier may still be processing the message, so non-idempotent retries cause duplicate execution. |
| Handle `ErrNoResponders` errors on request operations. | Enables clients to fail fast when no service is listening on the request subject, avoiding blocking until the full timeout duration expires. |
| Reuse a single, long-lived `nats.Conn` across all application routines and threads. | NATS client connections are thread-safe and multiplex all operations over a single TCP socket; creating connections per request or thread causes resource exhaustion and socket churn. |
| Ensure all worker instances in a load-balanced tier register with the identical queue group name. | Any subscriber listening to the subject without specifying the queue group receives an independent copy of every message, leading to unintended duplicate execution. |
| Always use connection and subscription draining (`nc.Drain()`) during controlled application shutdown (such as on SIGTERM). | Abruptly calling `nc.Close()` terminates connections immediately, dropping in-flight messages and buffered outbound publishes that draining safely flushes. |

---

## 6. Next Steps
