# NATS Consumer Guide

## Purpose

This guide establishes enterprise standards for designing, naming, filtering, and consuming messages from NATS JetStream Consumers.

In this platform architecture, **application developers do not create or update Consumers in application code**. JetStream Consumers are platform-governed infrastructure resources provisioned by Platform/SRE. Application services only bind to ("get") existing provisioned Consumers to consume and acknowledge messages.

This guide provides:
1. **Design & Configuration Standards:** Naming conventions, subject filtering rules, durable lifecycles, acknowledgement policies, and delivery start positions.
2. **Developer Consumption Patterns:** How application code gets a provisioned consumer and processes messages across Go, Java, and Python.
3. **Provisioning & Access Workflow:** How to request consumer access and configuration changes through the [Consumer Intake Form](nats-resources/consumer-intake.md).

For stream storage models, retention policies, and cluster-level persistence architecture, refer to the [JetStream Guide](nats-jetstream-guide.md).

---

## 1. Consumer Naming Standards

Consumer names must describe the owning application and its processing purpose.

### 1.1 Standard Format

```text
<application>-<processing-purpose>
```

| Token | Purpose | Example |
|---|---|---|
| `<application>` | Identifies the owning service, application, or business domain | `order-service`, `payment-service` |
| `<processing-purpose>` | Identifies the business responsibility or processing role | `fulfillment`, `audit`, `reconciliation` |

### 1.2 Naming Standards

| Rule | Requirement | Operational Rationale |
|---|---|---|
| Character Set | Lowercase alphanumeric characters and hyphens only (`kebab-case`). | Prevents casing mismatches and CLI parsing issues across operating systems. |
| Stability | Use stable, meaningful names for Durable Consumers across deployments. | Preserves server-side delivery sequence and acknowledgement state across restarts. |
| Uniqueness | Consumer names must be unique within the associated parent Stream. | Prevents naming collisions and unintended message routing between consumers. |
| No Instance IDs | Do not include pod IDs, UUIDs, hostnames, or container hashes. | Prevents abandoned durable state and resource leaks on the NATS cluster. |
| No Timestamps | Avoid timestamps or date strings in Durable Consumer names. | Breaks delivery state continuity; forces consumers to re-read or skip messages. |
| Rename as Migration | Treat any rename as a formal migration, not an in-place edit. | A new name initializes a new consumer with its own delivery state. |
| Workload Sharing | Multiple worker replicas must share the same Consumer name. | Enables competing consumer patterns and predictable horizontal scale-out. |

### 1.3 Compliant and Non-Compliant Naming Examples

| Compliant | Non-compliant | Reason |
|---|---|---|
| `order-service-fulfillment` | `order_service_fulfillment` | Uses underscores instead of hyphens. |
| `order-service-audit` | `ORDER-SERVICE-AUDIT` | Uses uppercase characters; must be lowercase kebab-case. |
| `payment-service-reconciliation` | `payment-service-reconciliation-20261027` | Contains a timestamp; breaks delivery state continuity across deployments. |
| `payment-service-worker` | `payment-service-worker-pod-8f2a1` | Appends pod/instance ID; creates abandoned durable consumers on pod restart. |
| `invoice-service-indexer` | `invoice-service-indexer-temp-uuid-9a8b` | Uses temporary/random UUIDs in a durable consumer name. |
| `order-service-fulfillment` | `prod-order-service-fulfillment` | Embeds environment prefix; environment isolation belongs to accounts/streams. |

---

## 2. Subject Filtering Standards

Consumer subject filters determine which messages captured by the parent Stream are eligible for delivery to the Consumer.

### 2.1 Filtering Standards

| Rule | Requirement | Operational Rationale |
|---|---|---|
| Stream Boundary | Filter must fall strictly within the parent Stream subject space. | Out-of-boundary subjects are rejected by the server or receive no messages. |
| Principle of Least Scope | Use the narrowest specific subject filter required. | Avoids delivering unwanted messages and consuming unnecessary network bandwidth. |
| Controlled Wildcards | Use wildcards (`*`, `>`) only when multi-subject matching is intentional. | Overly broad wildcards deliver unintended schemas and cause consumer contention. |
| Independent Roles | Separate distinct processing responsibilities into dedicated Consumers. | Ensures independent delivery positions, sequence tracking, and scaling characteristics. |
| Security Boundary | Enforce permissions via NATS authorization, not consumer filters. | Filters are client routing criteria, not an access control mechanism. |

### 2.2 Compliant and Non-Compliant Subject Filter Examples

The following examples assume a parent Stream `ORDERS` capturing `orders.>`:

| Compliant | Non-compliant | Reason |
|---|---|---|
| `orders.created` | `orders.*` (when only processing creation events) | Overly broad wildcard; delivers cancellation, payment, and refund events unnecessarily. |
| `orders.us.fulfilled` | `orders.>` (when only handling US fulfillment) | Captures all downstream tokens instead of the specific regional operation. |
| `orders.created` | `payments.received` (on Stream `ORDERS`) | Falls outside parent Stream subject space; messages will never arrive. |
| `orders.service.audit.*` | `*` | Single token wildcard at the root ignores namespace hierarchy and parent stream scope. |

---

## 3. Durable and Ephemeral Consumer Management

NATS JetStream supports Durable and Ephemeral Consumers. The choice depends on the workload's lifecycle and requirements for retaining Consumer state.

### 3.1 Consumer Lifecycle Matrix

| Dimension | Durable Consumer | Ephemeral Consumer |
|---|---|---|
| Primary Use Case | Ongoing business workloads, microservices, background workers. | Diagnostic inspection, ad hoc debugging, developer testing. |
| State Persistence | Preserved on server across client disconnects and restarts. | Deleted by server after client disconnects or inactivity timeout. |
| Naming Standard | Explicit, stable, meaningful kebab-case name. | Auto-generated or temporary session identifier. |
| Lifecycle Management | Managed via [Consumer Intake Form](nats-resources/consumer-intake.md) and formal change review. | Created dynamically; cleaned up immediately upon task completion. |
| Reconnection Behavior | Resumes from last acknowledged stream sequence. | Loses delivery state; creates new sequence tracking upon recreation. |
| Production Policy | Mandatory standard for production microservices. | Prohibited for persistent production message processing. |

### 3.2 Management Standards

| Rule | Requirement | Operational Rationale |
|---|---|---|
| Production Standard | Durable Consumers are mandatory for ongoing business workloads. | Retains delivery tracking and unacknowledged messages during service restarts. |
| Change Governance | Manage Durable Consumer lifecycle through the approved [Consumer Intake Form](nats-resources/consumer-intake.md). | Prevents accidental delivery sequence resets and uncoordinated consumer deletions. |
| Ephemeral Cleanup | Explicitly remove Ephemeral Consumers immediately after task completion. | Avoids lingering consumer state; does not rely solely on inactive threshold timers. |
| State Evaluation | Assess pending acknowledgements and stream backlog before modifying. | Prevents message re-reads, duplicate processing, and downstream overload. |
| Implementation Base | Understand persistence mechanics and stream configurations prior to consumer rollout. | Refer to the [JetStream Guide](nats-jetstream-guide.md) for stream setup and consumer pull/push models. |

### 3.3 Compliant and Non-Compliant Lifecycle Examples

| Compliant | Non-compliant | Reason |
|---|---|---|
| Durable Consumer for order fulfillment worker | Ephemeral Consumer for order fulfillment worker | Application restart causes loss of delivery tracking and missed messages. |
| Ephemeral Consumer for local CLI debugging tool | Durable Consumer without cleanup for ad-hoc debugging | Leaves orphaned durable state and unacknowledged message backlog on the server. |
| Explicit consumer deletion after diagnostic session | Relying solely on `InactiveThreshold` for ephemeral cleanup | Delays server resource reclamation until threshold timer expires. |
| Sharing durable consumer across scaled worker replicas | Creating unique durable consumer per pod replica | Creates split delivery sequences and multiplies stream message retention burden. |

---

## 4. Acknowledgement Policy Standards

The acknowledgement policy defines how the Consumer records message-processing acknowledgements and how acknowledgement state affects redelivery.

### 4.1 Acknowledgement Policies Matrix

| Policy | Server Behavior | Typical Use Case | Operational Risk / Trade-off |
|---|---|---|---|
| `AckExplicit` (Enterprise Default) | Server requires individual ACK per message before marking it complete. | Business-critical processing, financial workflows, order pipelines. | Client must manage ACK latency and handle timeouts (`AckWait`). |
| `AckAll` | Acknowledging a message acknowledges that message and all preceding unacked messages. | Strictly ordered, single-threaded batch processing. | An accidental ACK prematurely confirms prior failed or unprocessed messages. |
| `AckNone` | Server considers message acknowledged the instant it is delivered to client. | Best-effort metrics, high-throughput non-critical telemetry. | Unacknowledged messages are never redelivered on client crash; high loss risk. |

### 4.2 Operational Standards

| Consideration | Enterprise Standard | Implementation Guidance |
|---|---|---|
| Default Policy | `AckExplicit` is mandatory for business workloads. | Alternative policies require documented platform exception approval. |
| Redelivery Tolerance | Consumers must expect and tolerate duplicate deliveries. | Set explicit `AckWait` and configure `MaxDeliver` with dead-letter streams. |
| Processing Idempotency | Business logic must be idempotent. | Use unique transaction IDs, DB constraints, or idempotency keys. |
| Flow Control | Align `MaxAckPending` with worker processing capacity. | Avoids overwhelming worker memory and triggering unintended redelivery timeouts. |

### 4.3 Compliant and Non-Compliant Acknowledgement Examples

| Compliant | Non-compliant | Reason |
|---|---|---|
| `AckExplicit` for payment processing worker | `AckNone` for payment processing worker | Network glitch or pod crash leads to silent, unrecoverable payment data loss. |
| `AckExplicit` with idempotent deduplication key | `AckExplicit` assuming exactly-once delivery without deduplication | NATS guarantees at-least-once delivery; downstream duplicate side effects will occur. |
| `AckAll` in single-threaded strictly sequential stream | `AckAll` with concurrent out-of-order worker routines | A later message worker acknowledging completes earlier messages that are still failing. |

---

## 5. Delivery Policy Standards

The delivery policy determines the starting point for a Consumer when it is created. It should be selected according to the application's historical-data and replay requirements.

### 5.1 Delivery Policies Matrix

| Policy | Server Behavior | Typical Use Case |
|---|---|---|
| `DeliverAll` | Starts with all retained messages available in the Stream, from the beginning of history. | Rebuilding read models or processing retained historical events. |
| `DeliverNew` | Starts with messages published after the Consumer is created. | Live processing when earlier messages can safely be skipped. |
| `DeliverLast` | Starts with the most recent message in the Stream. | Reading the latest message when that is sufficient for the application. |
| `DeliverLastPerSubject` | Starts with the latest available message for each matching subject. | Initializing views that require the latest message across multiple subjects. |
| `DeliverByStartSequence` | Starts at a specified Stream sequence using `OptStartSeq`. | Replaying from a known sequence or recovery checkpoint. |
| `DeliverByStartTime` | Starts from a specified timestamp using `OptStartTime`. | Replaying messages from a defined time window. |

### 5.2 Delivery Standards

| Rule | Requirement | Operational Consideration |
|---|---|---|
| Requirements-Driven | Choose policy based on data contract, not convenience. | Prevents silent data omission or unexpected multi-gigabyte re-reads. |
| Historical Processing | Use `DeliverAll` when rebuilding state or event-sourcing. | Ensure consumers can handle initial delivery load from stream start. |
| Live Consumption | Use `DeliverNew` only when historical records are safe to skip. | Earlier unconsumed messages in the stream will never be delivered to this consumer. |
| Latest Message | Use `DeliverLast` only when a single latest event represents full state. | Intermediate state transitions are skipped. |
| Replay Safety | Dedicated replay consumer for ad hoc investigations. | Never alter the starting sequence or replay over an active production consumer. |
| Bounded Time Replay | Use `DeliverByStartTime` / `DeliverByStartSequence` for targeted recovery. | Isolate to the affected failure window using defined timestamps or sequences. |

### 5.3 Compliant and Non-Compliant Delivery Policy Examples

| Compliant | Non-compliant | Reason |
|---|---|---|
| `DeliverAll` for new audit log service populating datastore | `DeliverNew` for new audit service expecting complete historical audit trail | Omits all pre-existing events, creating permanent gaps in the audit database. |
| `DeliverNew` for real-time live notification alert banner | `DeliverAll` for real-time banner on high-volume stream | Deluges newly connected clients with years of obsolete historical alerts. |
| Creating dedicated temporary consumer for replay debugging | Modifying delivery sequence on active production consumer | Corrupts active consumer sequence tracking and causes duplicate processing across all workers. |
| `DeliverByStartTime` specifying exact incident window | `DeliverAll` to recover a 15-minute outage on a 90-day stream | Unnecessarily replays 90 days of traffic instead of targeting the 15-minute failure window. |

---

## 6. How Application Developers Consume Messages

### 6.1 Enterprise Governance: Get, Do Not Create

In production and staging environments, application services **MUST NOT** execute consumer creation or mutation calls (such as `CreateConsumer`, `CreateOrUpdateConsumer`, or `AddConsumer`).

- **Platform Invariant:** Streams and Consumers are provisioned via Terraform, GitOps, or SRE tooling following an approved [Consumer Intake Form](nats-resources/consumer-intake.md).
- **Application Responsibility:** Application code is given the `stream` and `consumer` names via configuration (e.g., environment variables). The code simply obtains a reference to the existing provisioned consumer ("get" operation) and starts consuming.

### 6.2 Binding to a Provisioned Consumer

#### Go (jetstream package)

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
)

func main() {
    nc, err := nats.Connect(os.Getenv("NATS_URL"))
    if err != nil {
        log.Fatalf("NATS connect error: %v", err)
    }
    defer nc.Close()

    js, err := jetstream.New(nc)
    if err != nil {
        log.Fatalf("JetStream init error: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    streamName := os.Getenv("NATS_STREAM_NAME")     // e.g. "ORDERS"
    consumerName := os.Getenv("NATS_CONSUMER_NAME") // e.g. "order-service-fulfillment"

    // Bind to the existing platform-provisioned Consumer (DO NOT call CreateConsumer)
    cons, err := js.Consumer(ctx, streamName, consumerName)
    if err != nil {
        log.Fatalf("Failed to bind to provisioned consumer %q on stream %q: %v", consumerName, streamName, err)
    }

    // Start consuming messages
    cc, err := cons.Consume(func(msg jetstream.Msg) {
        if err := processOrder(msg.Data()); err != nil {
            log.Printf("Processing error, sending NAK: %v", err)
            msg.Nak()
            return
        }
        msg.Ack()
    })
    if err != nil {
        log.Fatalf("Consume start failed: %v", err)
    }
    defer cc.Stop()

    select {} // Block until shutdown
}

func processOrder(data []byte) error {
    // Idempotent business processing logic
    return nil
}
```

#### Java (io.nats.client)

```java
package com.example.service;

import io.nats.client.*;
import io.nats.client.api.ConsumerInfo;

public class OrderWorker {
    public static void main(String[] args) throws Exception {
        String natsUrl = System.getenv().getOrDefault("NATS_URL", Options.DEFAULT_URL);
        String streamName = System.getenv("NATS_STREAM_NAME");     // e.g. "ORDERS"
        String consumerName = System.getenv("NATS_CONSUMER_NAME"); // e.g. "order-service-fulfillment"

        Connection nc = Nats.connect(natsUrl);
        JetStream js = nc.jetStream();

        // Bind to existing platform-provisioned Consumer (DO NOT call createConsumer)
        ConsumerContext cons = js.getConsumerContext(streamName, consumerName);

        // Consume messages continuously
        ConsumerContext.Consumer consumer = cons.consume(msg -> {
            try {
                processOrder(msg.getData());
                msg.ack();
            } catch (Exception e) {
                msg.nak();
            }
        });

        Thread.currentThread().join();
    }

    private static void processOrder(byte[] data) {
        // Idempotent business processing logic
    }
}
```

#### Python (nats.aio.client)

```python
import asyncio
import os
import nats

async def main():
    nc = await nats.connect(os.getenv("NATS_URL", "nats://localhost:4222"))
    js = nc.jetstream()

    stream_name = os.getenv("NATS_STREAM_NAME", "ORDERS")
    consumer_name = os.getenv("NATS_CONSUMER_NAME", "order-service-fulfillment")

    # Bind to existing platform-provisioned Pull Consumer
    sub = await js.pull_subscribe("", durable=consumer_name, stream=stream_name)

    print(f"Connected to provisioned consumer {consumer_name} on stream {stream_name}")

    while True:
        try:
            # Batch fetch from provisioned consumer
            msgs = await sub.fetch(batch=10, timeout=2.0)
            for msg in msgs:
                try:
                    await process_order(msg.data)
                    await msg.ack()
                except Exception:
                    await msg.nak()
        except nats.errors.TimeoutError:
            continue

async def process_order(data: bytes):
    # Idempotent business processing logic
    pass

if __name__ == "__main__":
    asyncio.run(main())
```

### 6.3 Message Acknowledgement Signals

When consuming from a Consumer configured with `AckExplicit`, the application must signal message completion using one of four supported verbs:

| Method | Signal | Server Action | When to Use |
|---|---|---|---|
| `Ack()` | Positive Ack | Server records successful completion and will not redeliver this message sequence. | Message processing succeeded and downstream side effects are safely committed. |
| `Nak()` | Negative Ack | Server marks message for redelivery according to consumer backoff configuration. | Transient, recoverable errors (e.g. database connection timeout, rate limit exceeded). |
| `Term()` | Terminate | Server permanently marks message as failed; no further redeliveries will occur. | Poison messages, corrupted payloads, or unrecoverable business rule violations. |
| `InProgress()` | In Progress | Server resets the `AckWait` timer, granting more time before redelivery triggers. | Long-running operations exceeding standard `AckWait` duration. |

### 6.4 Strict Sequential Ordering with Durable Consumers

When an application requires strict message-by-message ordered processing alongside durable state retention and failure recovery:

1. Configure the Durable Consumer with `AckExplicit` and `MaxAckPending = 1`.
2. This ensures the NATS server delivers exactly one unacknowledged message at a time.
3. The application must process and `Ack()` (or retry and resolve) the message before the next message in the sequence will be released.
4. Scale only as a single active worker instance (or active-passive standby) for that specific partition/subject.

---

## 7. Resource Provisioning & Access via Consumer Intake

Because application services do not create or modify Consumers directly, teams follow a formal intake and governance workflow for all consumer requirements.

### 7.1 How to Request Consumer Access

```text
[ Application Team ]
         |
         | 1. Fill Intake Form (nats-resources/consumer-intake.md)
         v
[ Platform / SRE Team ]
         |
         | 2. Validate Stream Subject Mapping & Quotas
         | 3. Provision Consumer via IaC / SRE Tooling
         v
[ NATS JetStream Cluster ]
         |
         | 4. Consumer is Active and Bound to Stream
         v
[ Application Service ]
         |
         | 5. Configures NATS_CONSUMER_NAME in Environment
         | 6. Calls js.Consumer(ctx, stream, consumer) ("Get")
         v
   [ Message Processing Begins ]
```

1. **Review Standards:** Check sections 1 through 5 of this guide for naming, filtering, lifecycle, acknowledgement, and delivery start positions.
2. **Submit Form:** Complete the [Consumer Intake Form](nats-resources/consumer-intake.md) with:
   - Proposed Consumer Name (e.g., `order-service-fulfillment`)
   - Owning Team & Service Name
   - Required Subject Filter(s) (e.g., `orders.created`)
   - Delivery Model (Pull or Push)
   - Acknowledgement Policy (`AckExplicit`) and `MaxAckPending`
   - Replay / Start Policy (`DeliverNew`, `DeliverAll`, etc.)
3. **Platform Provisioning:** Platform/SRE verifies stream capacity, provisions the Consumer via GitOps/Terraform, and attaches the approved configuration.
4. **Service Configuration:** The application team injects the provisioned Consumer name and Stream name into service configuration (`NATS_STREAM_NAME`, `NATS_CONSUMER_NAME`).
5. **Runtime Binding:** The service starts up and retrieves the consumer using `js.Consumer(...)` or equivalent SDK call.

---

## 8. Summary & Best Practices

| Phase | Developer Responsibility | Anti-Pattern to Avoid |
|---|---|---|
| **Design** | Select kebab-case purpose name, narrowest subject filter, and `AckExplicit`. | Appending pod IDs, random UUIDs, or timestamps to consumer names. |
| **Provisioning** | Submit [Consumer Intake Form](nats-resources/consumer-intake.md) to Platform/SRE. | Calling `CreateConsumer` or `AddConsumer` in application initialization code. |
| **Consumption** | Bind to provisioned consumer using `js.Consumer(...)` ("get" only). | Modifying consumer configuration dynamically at runtime. |
| **Processing** | Design idempotent handlers; explicitly signal `Ack()`, `Nak()`, or `Term()`. | Assuming exactly-once delivery without duplicate protection. |
| **Maintenance** | Request consumer modifications or retirement through formal intake. | Abandoning unused durable consumers on the cluster. |

---

## 9. References

- [NATS JetStream Guide (Stream & Persistence Architecture)](nats-jetstream-guide.md)
- [NATS Consumer Intake Form (Lifecycle & Provisioning)](nats-resources/consumer-intake.md)
- [NATS Subject Design Guide](subject-guide.md)
- [NATS Resource Lifecycle Management](nats-resources/resource-management.md)
- [NATS JetStream Consumers Documentation](https://docs.nats.io/concepts/jetstream/consumers)
- [Synadia: NATS 101 - JetStream Consumers](https://www.synadia.com/videos/nats-101-jetstream-consumers)
