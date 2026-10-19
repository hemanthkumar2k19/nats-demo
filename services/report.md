# NATS Delivery Semantics & Failure Handling Evaluation Report

This report summarizes the empirical evaluation of NATS Core and JetStream delivery semantics, failure handling, graceful/ungraceful shutdown patterns, prefetching behavior, and poison message lifecycles conducted on the NATS evaluation platform.

---

## 1. Test Summary Matrix

| Test Scenario | Mechanism & Subject | Trigger / Condition | Observed Behavior | Outcome |
| :--- | :--- | :--- | :--- | :--- |
| **At-Most-Once Delivery** | Core NATS (`nc.Publish`)<br>`delivery.atmost-once` | Publish while consumer is online vs. offline | Delivered in real-time when consumer is active; permanently dropped if consumer is offline. No ACK, no storage, no retries. | Verified (Fire-and-forget) |
| **At-Least-Once Delivery (Happy Path)** | JetStream Pull Consumer<br>`delivery.atleast-once` | Publish, countdown completes, `msg.Ack()` sent | Message received, processed, and explicitly ACKed. Consumer floor advances cleanly (`Ack Pending: 0`, `Unprocessed: 0`). | Verified (Guaranteed delivery) |
| **In-Flight Graceful Shutdown** | JetStream Consumer + `sync.WaitGroup` | Single `Ctrl+C` (`SIGINT`) during active countdown | `cc.Stop()` halts new fetches; `wg.Wait()` pauses exit until in-flight callback completes and sends `msg.Ack()`. Clean shutdown, no redelivery on restart. | Verified (No duplicate execution) |
| **Sudden Process Crash & Redelivery** | JetStream Consumer + Double `Ctrl+C` | Double `Ctrl+C` (`os.Exit(1)`) mid-countdown | Process terminates immediately before `msg.Ack()`. Server holds message in `Ack Pending: > 0`. After `AckWait: 30s`, message is automatically redelivered on restart. | Verified (Zero message loss on crash) |
| **Consumer Prefetching & Batch Control** | JetStream `PullMaxMessages(1)` | Publish batch of 3 messages | Default batching prefetches downstream messages into client RAM buffer (`Ack Pending: 2`). Setting `PullMaxMessages(1)` forces strictly 1-by-1 pull from server. | Verified (1-by-1 stream processing) |
| **Poison Message & Retry Exhaustion** | JetStream `MaxDeliver: 5`<br>`AckWait: 30s` | Repeatedly crash worker across 5 successive attempts | Redelivered exactly 5 times. On 5th unacknowledged failure, consumer abandons message and advances cursor to next sequence. Original message remains in stream. | Verified (Pipeline unblocked) |
| **Dead Letter Advisory Notification** | System Advisory Subject<br>`$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>` | Message exhausts all 5 delivery attempts | NATS server broadcasts JSON advisory event containing `stream`, `consumer`, `stream_seq: 12`, and `deliveries: 5` in real time. | Verified (Native DLQ event) |

---

## 2. Key Observations & Architectural Standards

### Observation 1: Subject-to-Stream Namespace Binding ("Persistence Tap")
* In NATS, there is a **single unified subject namespace**. There are no separate "Core NATS subjects" versus "JetStream subjects".
* If a stream is configured with subject `delivery.>`, the JetStream engine attaches an internal persistence tap to that subject pattern on the server.
* **Critical Rule:** Any message published to that subject—whether sent via `nc.Publish()` (Core NATS client) or `js.Publish()` (JetStream client)—is automatically captured and committed to stream storage.
* **The Only Difference in Publishing:**
  - `nc.Publish()`: Fire-and-forget over TCP. Server still captures the message if a stream matches, but the publisher receives no confirmation or sequence number.
  - `js.Publish()`: Synchronous `PubAck` returned to the publisher containing `stream`, `sequence`, and deduplication status.

### Observation 2: Strict Isolation Between Persistent and Ephemeral Subjects
* Because streams capture any matching subject, teams must enforce strict subject prefix boundaries:
  - **Persistent Subjects (Stream Backed):** `events.<domain>.<action>` (e.g., `events.delivery.created`).
  - **Ephemeral Subjects (Core NATS Only):** `live.<domain>.<action>` or `telemetry.<domain>.<action>` (e.g., `live.delivery.ping`).
* Never allow ephemeral or throwaway test messages to share subjects with a production JetStream stream.

### Observation 3: Application-Level In-Flight Tracking vs. Protocol Draining
* `sub.Drain()` and `cc.Stop()` operate at the **NATS protocol and client-buffer layer**: they stop new incoming messages from being requested or delivered.
* `sync.WaitGroup` (`wg.Add(1)` / `defer wg.Done()`) operates at the **application execution layer**: it tracks whether user business logic (database operations, RPCs) is currently executing.
* To achieve true zero-loss graceful shutdown, both must be combined:
  1. `cc.Stop()` / `sub.Drain()` to halt ingress.
  2. `wg.Wait()` (bounded by a timeout context) to allow in-flight callbacks to finish and send `msg.Ack()`.
  3. `nc.Drain()` to cleanly close the physical transport.

### Observation 4: Raft Consistency on Consumer Replication
* For streams using `Interest` or `WorkQueue` retention, NATS strictly requires:
  `Consumer Replicas == Stream Replicas`
* In `WorkQueue` and `Interest` retention, acknowledging a message **deletes it immediately** from the stream.
* If a consumer had fewer replicas than the stream's Raft cluster, a node partition could acknowledge and delete messages without consensus, causing split-brain data corruption. NATS enforces replica parity to protect cluster consistency.

### Observation 5: Poison Message DLQ Strategy
* NATS does not move retry-exhausted messages to a separate bucket by default; it skips them to keep consumer pipelines flowing.
* The standard enterprise pattern for Dead Letter Queues in NATS is:
  1. Build a lightweight DLQ worker that subscribes to `$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`.
  2. Read the referenced `stream_seq` via `js.GetMsg(ctx, stream, seq)`.
  3. Archive the message into a dedicated DLQ stream (`DELIVERY_DLQ`).
