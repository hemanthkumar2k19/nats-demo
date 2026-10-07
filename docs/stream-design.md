# Stream Design

## 1. Relationship Between Subjects and Streams

In NATS, **Subjects, Streams, and Consumers serve different purposes**:

| Resource | Purpose |
|---|---|
| **Subject** | Defines the message routing namespace — *where a message is published and what it represents*. |
| **Stream** | Defines the persistence boundary — *which subjects are persisted and how their messages are retained and stored*. |
| **Consumer** | Defines the consumption state — *how an application reads and tracks messages from a Stream*. |

The relationship is:

```text
Publisher → Subject → Stream → Consumer → Application
              │          │          │
           Routing    Persistence  Consumption
```

### 1.1 Subjects and Streams

A Stream can persist multiple subjects when those subjects have compatible persistence and operational requirements.

For example:

```text
ORDERS_STREAM
 ├── orders.created
 ├── orders.completed
 └── orders.cancelled
```

The subjects remain independent NATS subjects; the Stream simply provides a common persistence boundary.

> **A Stream is a persistence and operational boundary, not merely a container or folder for subjects.**

### 1.3 Stream and Consumer Are Separate Decisions

A Stream represents persisted data; Consumers represent application-specific consumption requirements.

```text
             ORDERS_STREAM
             /     |     \
            /      |      \
       Consumer A  Consumer B  Consumer C
           │           │           │
       Service A   Service B   Service C
```

### 1.4 Design Principle

> **Subjects define the messaging namespace, Streams define the persistence boundary, and Consumers define the consumption state.**


# Stream Design

## 2. Guide to Decide When to Create a New Stream

When an application requests persistence for one or more subjects, our primary decision is whether those subjects can be accommodated by an **existing Stream** or require a **new Stream**.

We should capture the usage and persistence requirements and evaluate them against the existing Stream configuration to determine the appropriate Stream boundary.

Our objective is to establish a Stream boundary where the subjects have sufficiently compatible **persistence, workload, storage, lifecycle, and consumption requirements**.

### 2.1 Decision Flow

```text
Proposed Subject(s)
        │
        ▼
Persistence Required?
    │          │
   No         Yes
    │          │
Core NATS      ▼
        Evaluate Existing Streams
                │
        ┌───────┴────────┐
        ▼                ▼
   Compatible       Not Compatible
        │                │
        ▼                ▼
  Reuse Stream       New Stream
```

Before making the decision, we should understand both the **subject requirements** and the **expected consumption behaviour**.

---

### 2.2 Information Required

Before designing the Stream, we should capture the requirements that influence persistence and consumption.

| Area | Information Required |
|---|---|
| **Purpose** | What business or technical purpose does the subject serve? |
| **Persistence** | Why must the messages be persisted? |
| **Replay / Recovery** | Is historical replay or recovery required? |
| **Retention** | How long must messages remain available? |
| **Volume** | Expected average and peak message rate / data volume |
| **Message Size** | Expected message size and variation |
| **Subject Cardinality** | Expected number and growth of unique subjects |
| **Consumers** | Who consumes the data and how many independent consumers are expected? |
| **Consumption Pattern** | Continuous processing, fan-out, replay, batch, snapshot, etc. |
| **Consumer Lifecycle** | Long-lived, temporary, or dynamically created consumers |
| **Recovery Behaviour** | What should happen when consumers are unavailable or restart? |
| **Isolation** | Any requirement for independent capacity, lifecycle, or failure isolation? |

The application provides the business and usage requirements; the Stream design should be derived from those requirements rather than from application ownership alone.

---

### 2.3 Persistence and Retention

We should first determine whether the proposed subjects require a persistence model compatible with the existing Stream.

We should evaluate:

- Retention duration
- Retention policy
- Maximum message / byte limits
- Whether historical replay is required
- Whether only recent state is required
- Whether messages need to remain available independently of consumer acknowledgement

#### Example

Consider two proposed subjects:

```text
customer.profile.updated
customer.notification.requested
```

If profile updates need to remain available for 7 days for recovery, while notification requests need to be removed after successful processing, they have materially different persistence requirements.

**Decision:** We should evaluate them as separate persistence workloads rather than assuming they should share a Stream because both belong to the same application.

---

### 2.4 Workload and Capacity

Subjects sharing a Stream contribute to the same Stream-level workload.

We should evaluate:

- Publish rate
- Peak traffic
- Message size
- Aggregate retained data
- Storage growth
- Subject cardinality
- Expected growth over time

A large number of subjects does not automatically require multiple Streams. The concern is whether their combined workload creates an undesirable operational or capacity boundary.

#### Example

```text
inventory.stock.updated
inventory.reservation.updated
```

may generate moderate and predictable traffic.

Another proposed subject:

```text
device.telemetry.status.<device-id>
```

may generate substantially higher and continuously growing traffic.

If both are placed in one Stream, the telemetry workload can dominate the Stream's storage and operational characteristics.

**Decision:** We should evaluate whether these workloads require independent Stream boundaries.

---

### 2.5 Subject Cardinality

We should consider both the **number of subjects** and the **rate at which new subjects are introduced**.

We should determine whether subjects are:

- Fixed and well-defined
- Dynamically generated
- Created per tenant, device, user, session, or other entity
- Potentially unbounded
- Frequently created and retired

High cardinality is not itself a reason to create one Stream per subject.

Instead, we should ask:

> **Does the subject population create a workload that should be independently managed?**

#### Example

```text
shipment.status
shipment.location
shipment.exception
```

has a relatively predictable subject model.

A subject structure such as:

```text
device.<device-id>.diagnostics
```

can grow with the number of devices.

The latter requires explicit capacity and lifecycle consideration before deciding how it should be mapped to Streams.

---

### 2.6 Storage and Replication

We should evaluate whether the proposed subjects are compatible with the existing Stream's storage and availability requirements.

We should consider:

- Storage type
- Durability requirements
- Replication factor
- Failure tolerance
- Recovery requirements
- Expected storage growth

#### Example

Suppose an application has:

```text
payment.transaction.completed
```

requiring durable replicated storage for business recovery, while:

```text
application.debug.snapshot
```

is retained only temporarily for operational troubleshooting.

Although both originate from the same application, their storage and durability requirements may justify different Stream boundaries.

---

### 2.7 Consumer Requirements

Consumer requirements must be evaluated as part of Stream design, but **a different Consumer requirement does not automatically mean a different Stream**.

A single Stream can support multiple independent Consumers:

```text
                CUSTOMER_EVENTS
                 /      |      \
                /       |       \
          Consumer A Consumer B Consumer C
              │          │          │
          Service A  Service B  Service C
```

We should evaluate:

- Number of Consumers
- Durable vs ephemeral Consumers
- Long-lived vs short-lived Consumers
- Consumer filter requirements
- Replay requirements
- Consumer creation rate
- Expected consumer lag
- Acknowledgement and redelivery behaviour
- Whether consumers access the same workload or substantially different workloads

Consumer-specific behaviour should normally remain at the **Consumer level**.

A separate Stream should be considered only when consumer behaviour creates a **materially different workload or operational requirement**.

#### Example

A Stream contains:

```text
order.created
order.shipped
order.cancelled
```

and is consumed by:

- Order Processing
- Customer Notification
- Reporting

These consumers have different acknowledgement and delivery requirements but consume the same persisted event workload.

**Decision:** Multiple Consumers on the same Stream are appropriate.

Conversely, if another application requires large-scale historical replay of the data for batch processing, we should evaluate the additional storage, replay, and I/O workload before deciding whether that workload should share the same Stream.

---

### 2.8 Lifecycle and Operational Isolation

We should determine whether the proposed subjects need to be managed independently.

We should consider:

- Independent ownership
- Different creation/decommissioning lifecycle
- Independent purge requirements
- Independent capacity growth
- Different operational criticality
- Failure isolation
- Independent configuration changes

#### Example

An application may have:

```text
fraud.alert.created
```

as a business-critical event and:

```text
fraud.model.debug
```

as temporary diagnostic data.

If the diagnostic workload is regularly purged, recreated, or scaled independently, coupling it with the business-critical workload may create unnecessary operational dependency.

---

### 2.9 Reuse Existing Stream vs Create New Stream

After evaluating the above factors, we should ask:

> **Can the proposed subjects operate within the existing Stream without introducing incompatible persistence, workload, consumer, storage, or lifecycle requirements?**

If **yes**, the existing Stream should normally be reused.

If **no**, we should identify the requirement that justifies an independent Stream.

A new Stream should therefore have a **documented reason**, such as:

```text
New Stream
   │
   └── Reason
        ├── Retention isolation
        ├── Workload isolation
        ├── Storage / replication isolation
        ├── Consumer workload isolation
        ├── Lifecycle isolation
        └── Failure / operational isolation
```

---

### 2.10 Enterprise Standard

Based on the above evaluation, the following standards apply to Stream management:

1. **Existing compatible Streams must be reused by default.**

2. **A new Stream must have a specific persistence or operational justification.**

3. **A new subject does not by itself justify a new Stream.**

4. **Application ownership does not determine Stream boundaries.** Subjects from the same application may require separate Streams, while compatible subjects from different applications may share a Stream.

5. **Subjects with compatible persistence, workload, storage, lifecycle, and consumption requirements should be grouped into the same Stream where practical.**

6. **Materially incompatible requirements must be isolated into separate Streams.**

7. **Consumer differences alone must not be used as a reason to create a new Stream.** Consumer-level requirements should be handled through Consumers unless they create a materially different workload or operational boundary.

8. **High subject cardinality must be explicitly evaluated but must not automatically result in one Stream per subject.**

9. **The number of Streams should be minimised without creating inappropriate coupling between workloads.**

10. **Every new Stream must have an explicit justification recorded as part of the Maker–Checker approval.**

### 2.11 Core Decision Principle

> **Create the minimum number of Streams necessary to maintain appropriate persistence, workload, consumer, storage, and operational boundaries.**

The objective is therefore neither:

```text
One Subject -> One Stream
```

nor:

```text
One Application -> One Stream
```

but rather:

```text
Compatible Requirements -> Shared Stream

Materially Different Requirements -> Separate Streams
```

## 3. Impact of Subjects and Consumers on Stream Design

The way we group subjects into Streams, and the way we create Consumers against those Streams, directly affects **storage, retention, resource utilisation, workload isolation, operational complexity, and scalability**.

The following anti-patterns should be considered when designing a Stream.

### 3.1 Anti-Pattern: One Stream for Unrelated Subjects

A Stream should not become a catch-all persistence boundary for subjects with materially different requirements.

```text id="4f0d3d"
APPLICATION_STREAM
 ├── payment.completed
 ├── audit.security
 ├── device.telemetry
 ├── user.notification
 └── debug.event
```

These subjects may have different:

- Message volumes
- Retention requirements
- Message sizes
- Subject cardinality
- Consumer behaviour
- Lifecycle
- Operational criticality

**Impact:**

- Retention and storage become coupled.
- One workload can dominate Stream resource consumption.
- Capacity planning becomes difficult.
- Troubleshooting becomes harder.
- Changes to the Stream can affect unrelated workloads.
- Operational blast radius increases.

---

### 3.2 Anti-Pattern: One Stream Per Subject Without Justification

The opposite extreme is creating a separate Stream for every persisted subject.

```text id="4f2r3k"
orders.created     → ORDERS_CREATED_STREAM
orders.updated     → ORDERS_UPDATED_STREAM
orders.cancelled   → ORDERS_CANCELLED_STREAM
```

when all three have the same persistence and operational requirements.

**Impact:**

- Unnecessary Stream proliferation
- Increased configuration and lifecycle management
- More resources to monitor
- Repeated configuration for equivalent workloads
- Fragmented operational visibility
- Increased platform overhead

A new subject alone is not sufficient justification for a new Stream.

---

### 3.3 Anti-Pattern: Grouping Subjects Only by Naming Hierarchy

Subjects with the same prefix do not necessarily have compatible persistence requirements.

```text id="yq4f7m"
customer.profile.updated
customer.audit.created
customer.notification.sent
```

Grouping them solely because they begin with `customer` can introduce unnecessary coupling.

**Impact:**

- Incompatible retention requirements
- Different workload characteristics
- Different consumer behaviour
- Difficult operational management

Subject hierarchy defines the messaging namespace; it does not by itself define the persistence boundary.

---

### 3.4 Anti-Pattern: Grouping Subjects Only by Application

All subjects owned by one application do not necessarily belong in one Stream.

```text id="z3w8xu"
Application A
      │
      └── APPLICATION_A_STREAM
             ├── business.events
             ├── audit.events
             └── temporary.processing
```

**Impact:**

- Retention coupling
- Capacity coupling
- Lifecycle coupling
- Unnecessary operational dependency

Application ownership should therefore not be used as the sole Stream-boundary criterion.

---

### 3.5 Anti-Pattern: Ignoring Subject Cardinality

A subject model that creates a large or continuously growing number of unique subjects can materially change the workload of a Stream.

For example:

```text id="o6j8xv"
device.<device-id>.status
```

may produce a unique subject for every device.

**Impact:**

- Increased Stream metadata
- Increased filtering complexity
- Greater consumer workload
- More difficult capacity planning
- Increased operational complexity

High cardinality does **not** automatically mean one Stream per subject. The resulting workload and growth pattern must be evaluated.

---

### 3.6 Anti-Pattern: Mixing Significantly Different Workloads

Subjects with dramatically different traffic characteristics can create an undesirable shared workload.

```text id="2qk2m9"
application.started
```

may generate very few messages, while:

```text id="h3l6n8"
application.request.trace
```

may generate messages continuously at very high volume.

**Impact:**

- High-volume traffic determines much of the Stream's resource requirements.
- Low-volume workloads become coupled to high-volume workloads.
- Capacity planning becomes less predictable.
- Performance investigation becomes more difficult.

---

### 3.7 Anti-Pattern: Mixing Incompatible Retention Requirements

Retention is a Stream-level concern.

For example:

```text id="2qbl9s"
customer.preference.changed
```

may require only a short recovery window, while:

```text id="g2jv8h"
customer.legal.consent.updated
```

may require substantially longer retention.

**Impact:**

- Unnecessary storage consumption through over-retention
- Insufficient retention for workloads requiring longer recovery
- Difficulty changing retention independently
- Potential recovery or compliance concerns

Subjects with materially different retention requirements should therefore be evaluated for separate Stream boundaries.

---

## 3.8 Anti-Pattern: Creating Consumers Without Considering Stream Workload

A Stream can have multiple Consumers, and different applications may independently consume the same persisted data.

```text id="xk6q71"
                CUSTOMER_EVENTS
                 /      |      \
                /       |       \
          Consumer A Consumer B Consumer C
              │          │          │
          Service A  Service B  Service C
```

Multiple Consumers are not inherently a problem.

The anti-pattern is creating Consumers without considering their combined workload.

We should evaluate:

- Number of Consumers
- Durable vs ephemeral Consumers
- Long-lived vs short-lived Consumers
- Consumer creation rate
- Consumer filter requirements
- Replay behaviour
- Expected lag
- Acknowledgement and redelivery behaviour
- Number of applications consuming the Stream

**Impact:**

- Increased server-side consumer state
- Increased filtering and delivery work
- Increased replay / recovery load
- More operational resources to manage
- Increased Stream workload even when publish volume remains unchanged

Synadia's guidance specifically highlights **consumer count, filter overlap, and dynamic subscription behaviour** as important scaling considerations for JetStream workloads.

---

### 3.9 Anti-Pattern: Treating Every Consumer Requirement as a Stream Requirement

Different Consumers do not automatically require separate Streams.

For example:

```text id="x3c8t4"
ORDERS_STREAM
      │
      ├── Order Processor
      ├── Notification Service
      └── Reporting
```

These Consumers can have independent delivery state and acknowledgement behaviour while consuming the same persisted workload.

Creating three Streams solely because there are three Consumers would unnecessarily duplicate the persistence boundary.

**Impact of unnecessary Stream separation:**

- Duplicate persisted data
- Additional storage consumption
- More Stream configuration
- Increased operational complexity
- More difficult data lifecycle management

Consumer-specific requirements should normally be handled at the Consumer level.

---

### 3.10 Anti-Pattern: Allowing Consumer Behaviour to Distort an Otherwise Appropriate Stream

The opposite problem can also occur.

Suppose a Stream contains a normal operational workload:

```text id="m0x0tq"
ORDER_EVENTS
```

and a new consumer requires large-scale historical replay or repeated snapshot-style access.

The new Consumer may introduce a substantially different workload against the same persisted data.

We should therefore evaluate whether the additional consumption pattern creates:

- Significant replay traffic
- Increased storage I/O
- Large delivery bursts
- High consumer creation/deletion rates
- Significant filtering overhead
- Resource contention with existing Consumers

If the consumer workload becomes a materially different operational workload, Stream separation may need to be reconsidered.

The important point is:

> **Consumer requirements can influence Stream design, but should not automatically determine it.**

---

### 3.11 Anti-Pattern: Ignoring the Combined Subject and Consumer Effect

Stream design should not be evaluated only from the publishing side.

Consider:

```text id="x9b5pp"
                  STREAM
                    │
        ┌───────────┼───────────┐
        │           │           │
     Subject A   Subject B   Subject C
        │           │           │
        └───────────┼───────────┘
                    │
          ┌─────────┼─────────┐
          │         │         │
       Consumer A Consumer B Consumer C
```

The Stream workload is influenced by both:

**Subject side**

- Publish rate
- Message size
- Retained data
- Subject cardinality
- Retention

**Consumer side**

- Number of Consumers
- Filter complexity
- Replay
- Delivery rate
- Consumer lifecycle
- Acknowledgement / redelivery behaviour

A Stream that appears appropriate based only on its subjects may become unsuitable when its actual consumer workload is considered.

---

### 3.12 Anti-Pattern: Ignoring Operational Blast Radius

A Stream is an operational boundary.

Any Stream-level operation can potentially affect every subject and Consumer associated with it.

Examples include:

- Retention changes
- Storage changes
- Replication changes
- Subject configuration changes
- Purging
- Stream deletion
- Capacity changes

The more unrelated workloads we place into one Stream, the larger the potential blast radius of these operations becomes.

---

## 3.13 Enterprise Standard

Based on the above anti-patterns and their impact, we establish the following standards:

1. **A Stream should contain subjects with compatible persistence and operational requirements.**

2. **We should not create a catch-all Stream for unrelated workloads.**

3. **We should not create one Stream per subject unless an independent persistence or operational requirement justifies it.**

4. **Subject naming similarity and application ownership are not sufficient reasons to group subjects into a Stream.**

5. **Materially different retention, storage, replication, workload, lifecycle, or operational requirements should be evaluated as separate Stream boundaries.**

6. **High-cardinality and high-volume subjects must be explicitly evaluated before being grouped with other workloads.**

7. **Multiple Consumers on a Stream are supported and should not, by themselves, result in additional Streams.**

8. **Consumer-specific requirements should normally be handled at the Consumer level.**

9. **Consumer count, filtering, replay, delivery behaviour, and lifecycle must be considered when assessing the total workload of a Stream.**

10. **A Consumer workload that creates a materially different operational or capacity requirement may justify a separate Stream boundary.**

11. **We should evaluate both subject-side and consumer-side workload before finalising a Stream design.**

12. **Every Stream should represent an intentional persistence and operational boundary.**

### 3.14 Core Principle

> **We should group subjects and Consumers around compatible persistence and operational requirements, while avoiding both unnecessary coupling and unnecessary fragmentation.**

The two extremes to avoid are:

```text id="d1q6uh"
Too Broad
Unrelated Subjects + Consumers
            ↓
     Excessive Coupling
```

and:

```text id="84okzj"
Too Narrow
Every Subject / Consumer
            ↓
    Excessive Fragmentation
```

The desired design is:

```text id="t2s9ck"
Compatible Workloads
        ↓
    Shared Stream

Materially Different
Operational Requirements
        ↓
   Separate Stream
```

## References

- [Synadia - How Many Subjects for a JetStream Stream?](https://www.synadia.com/blog/how-many-subjects-jetstream-stream)
- [Synadia - Scaling Dynamic Dashboard Subscriptions with JetStream & Core NATS](https://www.synadia.com/blog/scaling-dynamic-dashboard-subscriptions-jetstream-core-nats)
- [NATS Documentation - JetStream Concepts](https://docs.nats.io/concepts/jetstream)
- [NATS Documentation - JetStream Pull Consumers](https://docs.nats.io/learn/jetstream/pull-consumers)
- [Synadia - NATS 101: JetStream Consumers](https://www.synadia.com/videos/nats-101-jetstream-consumers)