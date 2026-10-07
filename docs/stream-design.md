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


## 2. Guide to Decide When to Create a New Stream

When persistence is required for a subject or set of subjects, we should determine whether an **existing Stream can accommodate the workload** or whether an **independent Stream boundary** is required.

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

The Application Team provides the usage requirements; whoever owns the Stream design uses those requirements to determine the appropriate topology.

### 2.1 Information We Should Capture

Before making the decision, we should understand:

| Area | What we need to know |
|---|---|
| **Purpose** | What does the subject represent? |
| **Persistence** | Why must the messages be persisted? |
| **Retention** | How long must they remain available? |
| **Replay / Recovery** | Is historical replay or recovery required? |
| **Volume** | Expected average and peak message rate / data volume |
| **Message Size** | Expected message size and variation |
| **Subject Cardinality** | Expected number of distinct subjects and their growth |
| **Consumers** | Number and type of expected consumers |
| **Consumption Pattern** | Continuous processing, fan-out, replay, batch, snapshot, etc. |
| **Consumer Lifecycle** | Long-lived, temporary, or dynamically created |
| **Isolation** | Any requirement for independent capacity, lifecycle, or failure isolation? |

### 2.2 Stream Compatibility

We should evaluate whether the proposed subjects are compatible with the existing Stream across the following dimensions.

#### Persistence and Retention

We should evaluate:

- Retention policy and duration
- Message / byte limits
- Replay and recovery requirements
- Message lifecycle

**Example:**

```text
customer.profile.updated
customer.notification.requested
```

If one requires seven days of recovery while the other is removed after successful processing, they have materially different persistence requirements.

---

#### Workload and Capacity

We should evaluate:

- Publish rate
- Peak traffic
- Message size
- Aggregate retained data
- Storage growth
- Subject cardinality

For example:

```text
inventory.stock.updated
inventory.reservation.updated
```

may have a predictable workload, while:

```text
device.telemetry.status.<device-id>
```

may generate substantially higher and continuously growing traffic.

We should determine whether these workloads require independent Stream boundaries.

---

#### Storage and Availability

We should evaluate compatibility of:

- Storage characteristics
- Durability
- Replication
- Failure tolerance
- Recovery requirements

Materially different requirements may justify independent Streams.

---

#### Lifecycle and Operational Isolation

We should determine whether the subjects need to be independently:

- Created or decommissioned
- Purged
- Scaled
- Configured
- Operated
- Isolated from failures in other workloads

---

#### Consumer Requirements

Different Consumer requirements **do not automatically require different Streams**.

We should consider:

- Number of Consumers
- Durable vs ephemeral
- Long-lived vs short-lived
- Filter requirements
- Replay behaviour
- Consumer creation rate
- Expected lag
- Acknowledgement and redelivery behaviour

For example:

```text
                CUSTOMER_EVENTS
                 /      |      \
          Consumer A Consumer B Consumer C
```

Three independent services consuming the same persisted workload normally require three Consumers, not three Streams.

A separate Stream should be considered only when Consumer behaviour creates a **materially different workload or operational requirement**.

### 2.3 Reuse or Create?

We should ask:

> **Can the proposed subjects and their expected Consumers operate within the existing Stream without introducing incompatible persistence, workload, storage, lifecycle, or operational requirements?**

**Yes → Reuse the existing Stream.**

**No → Identify the requirement that justifies a new Stream.**

Possible justifications include:

```text
New Stream
   │
   ├── Retention isolation
   ├── Workload / capacity isolation
   ├── Storage / replication isolation
   ├── Consumer workload isolation
   ├── Lifecycle isolation
   └── Failure / operational isolation
```

## 3. Impact of Subjects and Consumers on Stream Design

Stream boundaries should account for both **subject-side workload** and **consumer-side workload**.

Poor boundaries create two opposite problems:

- **Too Broad** → unrelated workloads become operationally coupled.
- **Too Narrow** → excessive Stream count creates operational fragmentation.

### 3.1 Subject-Side Anti-Patterns

| Anti-Pattern | Impact |
|---|---|
| **Catch-All Stream** | Unrelated workloads share retention, storage, capacity, and operational changes. |
| **One Stream per Subject** | Stream proliferation, repeated configuration, and increased operational overhead. |
| **Group by Subject Prefix Only** | Similar naming does not guarantee compatible persistence or workload requirements. |
| **Group by Application Only** | Application ownership can create unnecessary retention, capacity, and lifecycle coupling. |
| **Ignore Subject Cardinality** | Large or continuously growing subject populations can increase resource and operational complexity. |
| **Mix Very Different Volumes** | High-volume workloads can dominate shared Stream capacity and affect other subjects. |
| **Mix Incompatible Retention** | A common retention policy may over-retain one workload or under-retain another. |

### 3.2 Consumer-Side Anti-Patterns

| Anti-Pattern | Impact |
|---|---|
| **Consumer per Application Without Need** | Unnecessary consumer state and operational overhead. |
| **Treat Every Consumer as a Stream Requirement** | Unnecessary Stream proliferation and persistence fragmentation. |
| **Ignore Consumer Scale** | Large Consumer populations can increase server-side state and delivery workload. |
| **Ignore Replay Workload** | Large replay operations can introduce additional I/O and delivery load. |
| **Ignore Dynamic Consumer Lifecycle** | Frequent Consumer creation and deletion can increase operational overhead. |
| **Use Stream Separation for a Consumer-Level Problem** | Persistence becomes fragmented when Consumer configuration would have been sufficient. |

### 3.3 Combined Impact

A Stream's effective workload is influenced by both sides:

**Subject Side**
- Publish rate
- Message size
- Retained data
- Subject cardinality
- Retention requirements

**Consumer Side**
- Consumer count
- Filtering
- Replay
- Delivery rate
- Consumer lifecycle

These together determine the **Stream workload and operational characteristics**.

A Stream that appears appropriate based only on its subjects may become unsuitable once Consumer workload is considered.

However, a large number of Consumers does **not automatically** require separate Streams. Consumer-level configuration should be considered first.

### 3.4 Operational Blast Radius

A Stream is an **operational boundary**. Changes to the Stream can affect all subjects and Consumers associated with it, including:

- Retention changes
- Storage configuration
- Replication configuration
- Subject configuration
- Purging
- Stream deletion

> The more unrelated workloads we place in one Stream, the larger the potential operational blast radius.

Therefore, Stream boundaries should minimise unnecessary coupling while avoiding unnecessary Stream proliferation.

## 4. Enterprise Standards

### 4.1 Stream Boundary Standards

1. **Reuse an existing compatible Stream by default.**
2. **Create a new Stream only when a specific persistence or operational requirement justifies an independent boundary.**
3. A new **Subject, Application, Service, or Consumer does not by itself justify a new Stream.**
4. A Stream should contain subjects with compatible:
   - Persistence requirements
   - Retention requirements
   - Workload and capacity characteristics
   - Storage and replication requirements
   - Lifecycle requirements
   - Operational requirements
5. **Application ownership and subject naming hierarchy are not standalone criteria** for defining Stream boundaries.
6. **Materially incompatible requirements must be isolated** into separate Streams.

### 4.2 Subject Standards

7. **Subject cardinality and workload must be evaluated** before grouping subjects into an existing Stream.
8. High-volume or continuously growing subject populations must be assessed for their impact on shared Stream capacity and operations.
9. Avoid both:
   - **Catch-All Streams**
   - **One-Stream-per-Subject designs**
10. Stream boundaries must be based on **compatible persistence and operational requirements**, not simply on the number of subjects.

### 4.3 Consumer Standards

11. Multiple Consumers may share a Stream when they consume **compatible persisted data**.
12. Consumer-specific requirements should normally be handled at the **Consumer level**.
13. The following must be considered when assessing Stream workload:
   - Consumer count
   - Filtering
   - Replay behaviour
   - Delivery behaviour
   - Consumer lifecycle
14. Consumer behaviour may justify Stream separation **only when it creates a materially different workload or operational requirement**.

### 4.4 Governance Standards

15. Every new Stream must have an **explicit justification** recorded as part of the Maker–Checker process.
16. Every Stream must represent an **intentional persistence and operational boundary**.
17. The number of Streams should be **minimised without creating inappropriate coupling** between workloads.

### 4.5 Core Enterprise Principle

> **Create the minimum number of Streams necessary to maintain appropriate persistence and operational boundaries.**

The target model is:

**Compatible Requirements → Shared Stream**

**Materially Different Requirements → Separate Stream**

The objective is neither:

**One Subject → One Stream**

nor:

**One Application → One Stream**

Instead:

> **One Stream boundary for one compatible persistence and operational workload.**


## 5. References

- [Synadia - How Many Subjects for a JetStream Stream?](https://www.synadia.com/blog/how-many-subjects-jetstream-stream)
- [Synadia - Scaling Dynamic Dashboard Subscriptions with JetStream & Core NATS](https://www.synadia.com/blog/scaling-dynamic-dashboard-subscriptions-jetstream-core-nats)
- [NATS Documentation - JetStream Concepts](https://docs.nats.io/concepts/jetstream)
- [NATS Documentation - JetStream Pull Consumers](https://docs.nats.io/learn/jetstream/pull-consumers)
- [Synadia - NATS 101: JetStream Consumers](https://www.synadia.com/videos/nats-101-jetstream-consumers)