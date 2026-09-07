# NATS Terminology & Architecture Reference

## 1. Purpose

This document provides a terminology reference for **NATS**, with emphasis on understanding:

* What each NATS term means
* Whether it is a **deployable component**, **server-side object**, or **application/client role**
* Where it physically/logically exists
* Whether it is persistent
* How it relates to other NATS concepts

The intent is to provide a common vocabulary for **Solution Architects, Tech Leads, Developers, and Operations teams**.

---

# 2. NATS Conceptual Model

At a high level, NATS can be viewed as:

```text
Application / Worker Processes
        |
        | NATS Client Connection
        v
+-----------------------------+
|        NATS Server           |
|                             |
|  Core NATS                  |
|  - Subjects                 |
|  - Subscriptions            |
|  - Queue Groups             |
|                             |
|  JetStream                  |
|  - Streams                  |
|  - Consumers                |
|  - Message Storage          |
|  - Replicas                 |
+-----------------------------+
        |
        | Cluster / Replication
        v
+-----------------------------+
|      NATS Server Cluster     |
|                             |
|  Server 1  Server 2 ...     |
|     \       |       /       |
|       Replication            |
+-----------------------------+
```

The most important distinction is:

| Layer        | What it represents                                                         |
| ------------ | -------------------------------------------------------------------------- |
| Application  | Your publishers, subscribers, processors/workers                           |
| NATS Client  | Library/process connection used by applications                            |
| NATS Server  | Deployable NATS runtime                                                    |
| NATS Cluster | Multiple NATS servers working together                                     |
| Core NATS    | Real-time messaging capability                                             |
| JetStream    | Persistence and message-streaming capability                               |
| Stream       | Server-side persisted message collection                                   |
| Consumer     | Server-side view/state describing how messages are delivered from a stream |
| Queue Group  | Message distribution mechanism among competing subscribers                 |

---

# 3. Terminology Classification

The following table is the primary terminology reference.

| Term                              | NATS Meaning                                                                        | Type                                       | Where Does It Live?                   | Persistent?                                             | Key Relationship                                              |
| --------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------------ | ------------------------------------- | ------------------------------------------------------- | ------------------------------------------------------------- |
| **NATS Client**                   | Library/API used by an application to connect to NATS                               | Application component                      | Application process                   | No                                                      | Connects application to NATS Server                           |
| **Publisher**                     | Application that publishes messages                                                 | Application role                           | Application process                   | No                                                      | Publishes to a Subject                                        |
| **Subscriber**                    | Application that receives messages through a subscription                           | Application role                           | Application process                   | No                                                      | Subscribes to a Subject                                       |
| **Processor / Worker / Receiver** | Application component that consumes a message and performs business processing      | Application role                           | Application process                   | No                                                      | Usually acts as a Subscriber and/or JetStream Consumer client |
| **NATS Server**                   | NATS runtime responsible for connections, routing and messaging                     | **Deployable component**                   | VM/container/Kubernetes pod           | Depends on JetStream configuration                      | Core runtime                                                  |
| **NATS Cluster**                  | Multiple NATS servers operating together                                            | **Deployable/runtime topology**            | Multiple servers/nodes                | Depends on JetStream                                    | Provides scalability/HA                                       |
| **Subject**                       | Address/topic used for publishing and subscribing                                   | Server-managed messaging concept           | NATS server routing layer             | No in Core NATS; messages can be persisted by JetStream | Publisher sends to Subject; Subscriber subscribes             |
| **Subscription**                  | Interest registered by a client for a Subject                                       | Runtime object                             | NATS server + client connection       | No                                                      | Determines which client receives messages                     |
| **Queue Group**                   | Group of Core NATS subscribers where one subscriber receives each message           | Server-side/runtime subscription construct | NATS server routing state             | No                                                      | Implements competing consumers                                |
| **JetStream**                     | NATS persistence and streaming subsystem                                            | Server capability                          | NATS server/cluster                   | Yes, when configured with storage                       | Provides Streams and Consumers                                |
| **Stream**                        | Server-side configuration/object representing persisted messages matching subjects  | **NATS object**                            | NATS server / JetStream storage       | Yes                                                     | Stores messages                                               |
| **Stream Subject**                | Subject pattern configured on a Stream                                              | Stream configuration                       | Stream metadata                       | Yes as configuration                                    | Determines which published messages enter the Stream          |
| **Consumer**                      | Server-side state/configuration describing how messages are delivered from a Stream | **NATS object**                            | NATS server / JetStream state         | Depends on consumer type                                | Reads messages from a Stream                                  |
| **Durable Consumer**              | Consumer whose state survives client disconnection/reconnection                     | **NATS object**                            | JetStream server state                | Yes                                                     | Maintains delivery/acknowledgement state                      |
| **Ephemeral Consumer**            | Consumer intended for temporary use                                                 | **NATS object**                            | NATS server                           | No long-term persistence                                | Removed when no longer active                                 |
| **Consumer Group**                | Not a primary NATS object/term; usually refers to competing consumers               | Conceptual term                            | Depends on implementation             | Depends                                                 | In Core NATS this is normally a **Queue Group**               |
| **Deliver Policy**                | Defines where a Consumer starts receiving messages                                  | Consumer configuration                     | Consumer configuration                | Yes for durable consumer                                | Controls initial message position                             |
| **Ack Policy**                    | Defines whether/how delivered messages must be acknowledged                         | Consumer configuration                     | Consumer configuration                | Yes for durable consumer                                | Controls message acknowledgement                              |
| **Ack**                           | Client response indicating successful processing of a message                       | Protocol/runtime operation                 | NATS server/client interaction        | Ack state may be persisted                              | Used by JetStream Consumers                                   |
| **Redelivery**                    | Re-delivery of a message that was not acknowledged appropriately                    | Runtime behavior                           | NATS server                           | Depends on stream/consumer                              | Driven by consumer acknowledgement state                      |
| **Storage**                       | Mechanism used by JetStream to persist messages                                     | Server capability/configuration            | NATS server filesystem or memory      | Yes for file storage                                    | Used by Streams                                               |
| **Memory Storage**                | Stream messages stored in memory                                                    | Storage mode                               | NATS server memory                    | No across server restart                                | Fast but non-durable across restart                           |
| **File Storage**                  | Stream messages stored on disk                                                      | Storage mode                               | NATS server filesystem                | Yes                                                     | Normal durable JetStream storage                              |
| **Replication**                   | Maintaining multiple copies of JetStream data across servers                        | Cluster capability                         | NATS cluster                          | Yes                                                     | Provides high availability                                    |
| **Replica**                       | A copy of Stream data maintained on another NATS server                             | Server-side data copy                      | NATS server                           | Yes                                                     | Part of Stream replication                                    |
| **Stream Leader**                 | NATS server currently responsible for coordinating a Stream                         | Runtime role                               | One NATS server in cluster            | No separate object                                      | Coordinates replicated Stream                                 |
| **Consumer Leader**               | Server responsible for coordinating a replicated Consumer                           | Runtime role                               | NATS cluster                          | No separate deployable                                  | Manages Consumer state                                        |
| **Cluster**                       | Group of connected NATS servers                                                     | Runtime topology                           | Multiple NATS server instances        | Depends                                                 | Enables HA and scaling                                        |
| **Route**                         | Server-to-server connection inside a NATS cluster                                   | Runtime connection                         | NATS servers                          | No                                                      | Connects cluster members                                      |
| **Client Connection**             | Connection between an application and NATS                                          | Runtime connection                         | Client ↔ NATS Server                  | No                                                      | Carries publish/subscribe operations                          |
| **NATS Account**                  | Logical security/administrative boundary                                            | Server-side configuration object           | NATS server/cluster configuration     | Configuration-dependent                                 | Used for isolation and permissions                            |
| **NATS Server Configuration**     | Configuration controlling server behavior                                           | Configuration                              | Server filesystem / deployment config | Yes                                                     | Defines server runtime                                        |
| **JetStream Configuration**       | Configuration controlling JetStream behavior/storage                                | Configuration                              | NATS server configuration             | Yes                                                     | Enables/configures persistence                                |

---

# 4. Deployable Components vs NATS Objects

This distinction is important for architecture discussions.

## 4.1 Deployable Components

These are things that actually run as processes/containers/pods.

| Component              | Deployable? | Typical Deployment                  |
| ---------------------- | ----------: | ----------------------------------- |
| NATS Server            |     **Yes** | VM, container, Kubernetes Pod       |
| NATS Cluster Node      |     **Yes** | Individual NATS Server instance     |
| Application Publisher  |     **Yes** | Application/service                 |
| Application Subscriber |     **Yes** | Application/service                 |
| Processing Worker      |     **Yes** | Application/service                 |
| NATS Client Library    |          No | Embedded inside application         |
| Stream                 |          No | NATS server-side object             |
| Consumer               |          No | NATS server-side object             |
| Queue Group            |          No | NATS subscription/routing construct |
| Subject                |          No | Messaging namespace                 |
| Replica                |          No | Data copy maintained by JetStream   |
| Stream Leader          |          No | Runtime role assigned to a server   |

Therefore, an architecture diagram should normally show:

```text
Application Pod
     |
     | NATS Client
     |
     v
NATS Server Pod
```

rather than showing a Stream or Consumer as an independently deployed service.

---

# 5. Core NATS vs JetStream

One of the most important architectural distinctions is between **Core NATS** and **JetStream**.

| Capability                       | Core NATS                        | JetStream                          |
| -------------------------------- | -------------------------------- | ---------------------------------- |
| Messaging                        | Yes                              | Yes                                |
| Publish/Subscribe                | Yes                              | Yes                                |
| Subjects                         | Yes                              | Yes                                |
| Queue Groups                     | Yes                              | Yes, depending on consumer pattern |
| Persistence                      | No                               | Yes                                |
| Message replay                   | No                               | Yes                                |
| Durable message history          | No                               | Yes                                |
| Acknowledgement-based processing | No application-level persistence | Yes                                |
| Consumer state                   | Subscription state               | Persistent/managed Consumer state  |
| Redelivery                       | No persistent redelivery         | Yes                                |
| Streams                          | No                               | Yes                                |
| Consumers                        | Core subscriptions               | JetStream Consumers                |
| Replication of stored messages   | No                               | Yes                                |
| Retention policies               | No                               | Yes                                |

**Rule of thumb:**

> **Core NATS = live messaging.**
> **JetStream = persistent messaging and streaming.**

---

# 6. NATS Client

A **NATS Client** is the client library used by an application to communicate with a NATS server.

Examples:

```text
Go application
    |
    +-- NATS Go Client
              |
              v
          NATS Server
```

The client is **not a separate NATS server component**.

| Aspect         | Description                                       |
| -------------- | ------------------------------------------------- |
| Type           | Application library                               |
| Runs where     | Inside application process                        |
| Connects to    | NATS Server                                       |
| Creates        | Connections, publishers, subscriptions, consumers |
| Persistent     | No                                                |
| Responsibility | Protocol communication with NATS                  |

---

# 7. Publisher

A **Publisher** is an application role.

The publisher sends a message to a NATS **Subject**.

```text
Publisher
    |
    | publish("orders.created", message)
    v
Subject: orders.created
```

The publisher does not normally publish directly to a Stream.

Instead:

```text
Publisher
     |
     v
Subject
     |
     +------------------+
     |                  |
     v                  v
Core Subscriber      JetStream Stream
```

If the Subject is configured as part of a Stream, the message can be persisted by JetStream.

| Aspect              | Description         |
| ------------------- | ------------------- |
| Type                | Application role    |
| Lives in            | Application process |
| Sends to            | Subject             |
| Knows about Stream? | Not necessarily     |
| Persistent itself?  | No                  |

---

# 8. Subscriber

A **Subscriber** registers interest in one or more Subjects.

```text
Subject
   |
   v
Subscription
   |
   v
Subscriber Application
```

A Core NATS subscriber receives messages that are currently being published.

If the subscriber is disconnected:

> Core NATS does not retain those messages for later delivery.

JetStream changes this behavior by introducing **Streams and Consumers**.

---

# 9. Processor / Worker / Receiver

Terms such as:

* Processor
* Worker
* Receiver
* Message Handler

are **application terminology**, not distinct NATS objects.

For example:

```text
NATS Consumer
       |
       v
Application Worker
       |
       v
Business Processing
```

The Worker typically uses a NATS client to:

1. Receive a message
2. Process it
3. Acknowledge it when successful

Therefore:

> **Consumer = NATS server-side delivery state**
> **Worker/Processor = application-side business processing**

This distinction is useful when designing architecture diagrams.

---

# 10. Subject

A **Subject** is the addressing mechanism in NATS.

Examples:

```text
orders.created
orders.updated
orders.cancelled

jobs.validation
jobs.processing
jobs.completed
```

Subjects are hierarchical strings and support wildcard subscriptions.

For example:

```text
jobs.*
```

can receive:

```text
jobs.validation
jobs.processing
jobs.completed
```

A Subject is **not a deployable component** and is not equivalent to a physical queue.

| Aspect                | Description                 |
| --------------------- | --------------------------- |
| Type                  | Messaging namespace/address |
| Deployable            | No                          |
| Stored independently? | No in Core NATS             |
| Used by               | Publishers and Subscribers  |
| Used by JetStream     | Yes                         |
| Physical queue?       | No                          |

---

# 11. Subscription

A Subscription represents a client's interest in a Subject.

```text
Application
    |
    +-- subscribe("orders.created")
             |
             v
        NATS Server
```

A subscription belongs to a client connection.

If the client disconnects, the subscription disappears.

Therefore:

> A Core NATS Subscription is primarily a **runtime connection-level construct**, not a durable messaging object.

---

# 12. Queue Group

A **Queue Group** is NATS terminology for **competing consumers in Core NATS**.

Example:

```text
Subject: jobs.process

             NATS
              |
       +------+------+
       |             |
   Worker-1       Worker-2
   group=workers  group=workers
```

If three workers belong to the same Queue Group:

```text
Worker-1 ─┐
Worker-2 ─┼── Queue Group: processors
Worker-3 ─┘
```

each message is delivered to **one member** of the group rather than every member.

Important:

> **Queue Group ≠ JetStream Consumer**

A Core NATS Queue Group:

* Does not persist messages
* Does not maintain durable acknowledgement state
* Does not provide replay
* Is primarily a load-balancing/competing-consumer mechanism

---

# 13. JetStream

**JetStream** is the persistence and streaming subsystem of NATS.

It provides capabilities such as:

* Persistent message storage
* Streams
* Consumers
* Message replay
* Acknowledgements
* Redelivery
* Retention policies
* Replication

Conceptually:

```text
                  JetStream
                     |
          +----------+----------+
          |                     |
       Streams              Consumers
          |                     |
       Messages             Delivery State
```

JetStream runs as part of the NATS Server; it is **not normally deployed as a separate application/service**.

---

# 14. Stream

A **Stream** is a server-side JetStream object that stores messages.

Example:

```text
Stream: JOBS

Subjects:
    jobs.>

Messages:
    jobs.validation
    jobs.processing
    jobs.completed
```

The Stream defines:

* Which Subjects it captures
* How messages are retained
* Storage type
* Number of replicas
* Maximum age/size/count
* Other stream-level policies

Conceptually:

```text
Publisher
    |
    v
Subject: jobs.processing
    |
    v
+----------------------+
| Stream: JOBS         |
|                      |
| Message 1            |
| Message 2            |
| Message 3            |
+----------------------+
```

A Stream is **not a separate process**.

It is a **server-side JetStream object**.

---

# 15. Stream Storage

JetStream Streams can use different storage mechanisms.

| Storage | Where data lives                      | Survives server restart? | Typical purpose                  |
| ------- | ------------------------------------- | -----------------------: | -------------------------------- |
| Memory  | NATS server memory                    |                       No | High-speed/non-durable use cases |
| File    | NATS server filesystem/storage volume |                      Yes | Durable messaging                |

For production durable messaging, **File Storage** is generally the important architecture concept.

Example:

```text
NATS Server Pod
      |
      +-- JetStream
            |
            +-- Stream
                  |
                  +-- File Storage
                        |
                        v
                   Persistent Volume
```

The Stream is the logical object.

The filesystem/Persistent Volume is the physical storage.

---

# 16. Consumer

A **JetStream Consumer** is a server-side object that defines how messages from a Stream are delivered to an application.

This is one of the most important concepts to understand.

```text
                 Stream
                   |
            +------+------+
            |             |
       Consumer A     Consumer B
            |             |
            v             v
        Processor A   Processor B
```

The same Stream can therefore support multiple independent consumers.

For example:

```text
Stream: ORDERS

Consumer: billing
Consumer: notification
Consumer: analytics
```

Each Consumer can have its own:

* Delivery position
* Acknowledgement policy
* Delivery policy
* Redelivery behavior
* Consumer state

Therefore:

> **Stream = stored message history**
> **Consumer = a view/stateful delivery mechanism over that history**

---

# 17. Durable Consumer

A **Durable Consumer** retains its Consumer state beyond the lifetime of an individual client connection.

Example:

```text
Stream: JOBS
       |
       v
Durable Consumer: job-processors
       |
       +---- Worker 1
       +---- Worker 2
       +---- Worker 3
```

If Worker 1 disconnects, another worker can continue consuming using the same durable Consumer.

The Consumer retains information such as delivery/acknowledgement state.

This is particularly important for:

* Long-running processing
* Worker restarts
* Failure recovery
* Reliable processing
* Replay/recovery scenarios

---

# 18. Ephemeral Consumer

An **Ephemeral Consumer** is intended for temporary consumption.

Conceptually:

```text
Application
     |
     +-- create consumer
     |
     +-- consume
     |
     +-- disconnect
     |
     v
Consumer eventually removed
```

Unlike a Durable Consumer, it is not intended to maintain long-lived consumer state.

---

# 19. Consumer vs Queue Group

This is a common source of confusion.

| Concept            | Queue Group                  | JetStream Consumer             |
| ------------------ | ---------------------------- | ------------------------------ |
| NATS area          | Core NATS                    | JetStream                      |
| Purpose            | Competing consumers          | Stateful message delivery      |
| Server-side object | No standalone durable object | Yes                            |
| Persistence        | No                           | Can be durable                 |
| ACK state          | No JetStream ACK state       | Yes                            |
| Replay             | No                           | Yes                            |
| Redelivery         | No persistent redelivery     | Yes                            |
| Delivery position  | Live only                    | Managed by Consumer            |
| Typical use        | Simple worker load balancing | Reliable/persistent processing |

A useful mental model:

```text
Core NATS:

Subject
   |
   +-- Queue Group
         |
         +-- Worker 1
         +-- Worker 2
         +-- Worker 3


JetStream:

Stream
   |
   +-- Consumer
         |
         +-- Worker 1
         +-- Worker 2
         +-- Worker 3
```

---

# 20. "Consumer Group" Terminology

**Consumer Group is not the primary NATS term.**

In architecture discussions, people may use "consumer group" generically to mean:

> Multiple consumers/workers sharing message-processing responsibility.

In NATS, the correct terminology depends on the mechanism being used:

| Requirement                                        | NATS terminology                                       |
| -------------------------------------------------- | ------------------------------------------------------ |
| Core NATS competing subscribers                    | **Queue Group**                                        |
| JetStream persistent delivery state                | **Consumer**                                           |
| Multiple workers sharing a JetStream pull Consumer | Multiple clients using the same Consumer               |
| Push Consumer load balancing                       | Consumer with appropriate delivery/group configuration |

Therefore, architecture documentation should preferably use **Queue Group** for Core NATS and **Consumer** for JetStream.

---

# 21. Deliver Policy

**Deliver Policy** is a JetStream Consumer configuration.

It determines **where in the Stream the Consumer should start receiving messages**.

Common policies include:

| Policy                   | Meaning                                        |
| ------------------------ | ---------------------------------------------- |
| `DeliverAll`             | Start from the earliest available message      |
| `DeliverNew`             | Deliver only new messages                      |
| `DeliverLast`            | Start with the latest message                  |
| `DeliverByStartSequence` | Start from a specified sequence                |
| `DeliverByStartTime`     | Start from a specified timestamp               |
| `DeliverLastPerSubject`  | Start with the latest message for each subject |

Example:

```text
Stream:

1 ---- 2 ---- 3 ---- 4 ---- 5 ---- 6
                         ^
                         |
                 Consumer starts here
```

The Deliver Policy determines the starting point.

Important distinction:

> **Deliver Policy controls where consumption starts.**

It is not the same as acknowledgement behavior.

---

# 22. Ack Policy

**Ack Policy** determines how a JetStream Consumer handles acknowledgements.

Common policies:

| Policy        | Meaning                                                               |
| ------------- | --------------------------------------------------------------------- |
| `AckNone`     | No acknowledgement expected                                           |
| `AckAll`      | Acknowledgement can acknowledge all pending messages up to that point |
| `AckExplicit` | Each message must be explicitly acknowledged                          |

Typical reliable processing:

```text
Message
   |
   v
Consumer
   |
   v
Worker
   |
   +---- Processing successful
             |
             v
           ACK
```

If the message is not acknowledged according to the Consumer configuration, JetStream can make it eligible for redelivery.

---

# 23. ACK vs Processing Success

An ACK is not the same thing as business processing itself.

The application decides **when** to ACK.

For example:

```text
Receive Message
      |
      v
Validate
      |
      v
Business Processing
      |
      v
Persist Result
      |
      v
ACK Message
```

If the application ACKs before the business operation completes:

```text
Receive
  |
 ACK
  |
Processing fails
```

the message may no longer be available for normal redelivery.

Therefore, ACK placement is an important application design decision.

---

# 24. Redelivery

JetStream can redeliver messages that remain unacknowledged according to the Consumer configuration.

Example:

```text
Message 101
     |
     v
Worker A
     |
     X  Worker crashes
     
Message remains unacknowledged
     |
     v
Redelivery
     |
     v
Worker B
```

This is one of the major differences between Core NATS and JetStream.

---

# 25. Stream Replication

JetStream can maintain multiple replicas of a Stream across NATS servers.

Example:

```text
                 Stream
                    |
          +---------+---------+
          |         |         |
          v         v         v
       Server-1  Server-2  Server-3
       Replica   Replica   Replica
```

The replication factor determines how many copies of the Stream data are maintained.

For example:

```text
Replicas = 3

Server 1 -> Replica
Server 2 -> Replica
Server 3 -> Replica
```

Replication provides resilience against server/node failure.

---

# 26. Stream Leader

For a replicated Stream, one NATS server acts as the **Stream Leader**.

Conceptually:

```text
             Stream Leader
                 |
          +------+------+
          |             |
       Replica        Replica
       Server 2       Server 3
```

The leader coordinates Stream operations and replication.

If the leader fails, JetStream can elect another suitable server to become leader.

Important:

> **Stream Leader is a runtime role, not a separately deployed component.**

---

# 27. Leadership Lifecycle

At a high level:

```text
             NATS Cluster
                  |
             Stream Leader
                  |
          +-------+-------+
          |               |
       Replica           Replica
          |               |
          +-------+-------+
                  |
            Leader fails
                  |
                  v
             Election
                  |
                  v
        New Stream Leader
```

The architecture should therefore distinguish:

* **NATS Server** → physical/deployable component
* **Stream** → logical server-side object
* **Replica** → copy of Stream data
* **Leader** → temporary runtime role assigned to one server

---

# 28. Consumer State

A Consumer maintains state associated with message delivery.

Conceptually:

```text
Stream
  |
  +-- Message 1
  +-- Message 2
  +-- Message 3
  +-- Message 4
        |
        v
Consumer State
  |
  +-- Delivery position
  +-- Pending messages
  +-- Acknowledgement state
  +-- Redelivery state
```

This state allows JetStream to provide reliable consumption behavior.

For durable Consumers, this state is intended to survive application/client restarts.

---

# 29. Consumer and Stream Relationship

A Stream and Consumer have different responsibilities.

| Stream                        | Consumer                              |
| ----------------------------- | ------------------------------------- |
| Stores messages               | Delivers messages                     |
| Defines message retention     | Defines consumption behavior          |
| Defines captured Subjects     | Defines delivery position             |
| Has storage configuration     | Has ACK configuration                 |
| Has replication configuration | Has delivery/redelivery configuration |
| Represents message history    | Represents a consumer's view/state    |

Mental model:

```text
                  STREAM
        "What messages do we retain?"
                    |
                    v
        +-----------------------+
        | Message 1             |
        | Message 2             |
        | Message 3             |
        | Message 4             |
        | Message 5             |
        +-----------------------+
                    |
          +---------+---------+
          |                   |
          v                   v
     Consumer A          Consumer B
     "Where do I          "Where do I
      start/read?"         start/read?"
```

---

# 30. NATS Cluster

A NATS Cluster is a group of NATS Server instances connected together.

Example:

```text
                NATS Cluster
        +-------------------------+
        |                         |
        |  Server 1 <-> Server 2  |
        |      \          /       |
        |       \        /        |
        |        Server 3         |
        |                         |
        +-------------------------+
```

The individual NATS Servers are the deployable components.

The Cluster is the logical/runtime topology.

| Term    | Meaning                                    |
| ------- | ------------------------------------------ |
| Server  | Individual NATS process                    |
| Cluster | Group of NATS servers                      |
| Route   | Server-to-server cluster connection        |
| Leader  | Runtime role for replicated JetStream data |
| Replica | Copy of replicated JetStream data          |

---

# 31. Route

A **Route** is a server-to-server connection used by NATS servers within a cluster.

```text
NATS Server 1
      |
    Route
      |
NATS Server 2
```

Routes allow NATS servers to communicate and participate in the cluster.

A Route is:

* Not an application connection
* Not a Subject
* Not a Stream
* Not a Consumer

It is part of the **NATS server topology**.

---

# 32. Client Connection vs Route

|                           | Client Connection         | Route                     |
| ------------------------- | ------------------------- | ------------------------- |
| Connects                  | Application → NATS Server | NATS Server → NATS Server |
| Purpose                   | Application messaging     | Cluster communication     |
| Created by                | NATS Client               | NATS Server               |
| Part of application?      | Yes                       | No                        |
| Part of cluster topology? | No                        | Yes                       |

---

# 33. Physical vs Logical View

For architecture discussions, it helps to separate physical deployment from logical NATS objects.

### Physical deployment

```text
Kubernetes

+-------------------+
| NATS Server Pod 1 |
|                   |
| JetStream         |
+-------------------+

+-------------------+
| NATS Server Pod 2 |
|                   |
| JetStream         |
+-------------------+

+-------------------+
| NATS Server Pod 3 |
|                   |
| JetStream         |
+-------------------+
```

### Logical NATS model

```text
NATS Cluster
     |
     +-- Stream: JOBS
     |      |
     |      +-- Messages
     |
     +-- Consumer: PROCESSORS
     |
     +-- Consumer: ANALYTICS
     |
     +-- Subjects
     |
     +-- Queue Groups
```

The logical objects are hosted and managed by the NATS servers; they are **not separate Kubernetes deployments**.

---

# 34. Complete Example

Consider a job-processing system.

```text
                    Application
                        |
                  Publisher
                        |
                        v
                jobs.processing
                        |
                        v
              +----------------+
              | NATS Cluster   |
              |                |
              | Stream: JOBS   |
              +----------------+
                        |
                        v
             Consumer: PROCESSORS
                        |
              +---------+---------+
              |         |         |
              v         v         v
           Worker-1  Worker-2  Worker-3
              |         |         |
              +---------+---------+
                        |
                  Business Logic
                        |
                        v
                       ACK
```

Here:

| Element           | What it is                      |
| ----------------- | ------------------------------- |
| Publisher         | Application role                |
| `jobs.processing` | Subject                         |
| NATS Cluster      | Runtime topology                |
| NATS Server       | Deployable component            |
| `JOBS`            | Stream                          |
| `PROCESSORS`      | Durable Consumer                |
| Worker-1/2/3      | Application processes           |
| ACK               | Consumer acknowledgement        |
| File Storage      | Physical JetStream storage      |
| Replicas          | Copies of Stream data           |
| Stream Leader     | Runtime role on one NATS Server |

---

# 35. Recommended Architecture Vocabulary

For consistency across architecture documents, use the following terminology.

| Instead of saying   | Prefer                                                   |
| ------------------- | -------------------------------------------------------- |
| NATS Queue          | Subject / Stream / Consumer, depending on context        |
| NATS Topic          | Subject                                                  |
| NATS Consumer Group | Queue Group for Core NATS; Consumer for JetStream        |
| NATS Worker         | Application Worker                                       |
| NATS Processor      | Application Processor / Worker                           |
| NATS Receiver       | Subscriber / Consumer client                             |
| NATS Queue Consumer | JetStream Consumer, if referring to JetStream            |
| NATS Storage        | JetStream Stream Storage                                 |
| NATS Node           | NATS Server                                              |
| NATS Instance       | NATS Server                                              |
| NATS Cluster Node   | NATS Server                                              |
| NATS Queue          | Avoid unless describing an application-level abstraction |
| Stream Service      | Stream (server-side object)                              |

---

# 36. Quick Classification Reference

This table can be used as the **one-page terminology cheat sheet**.

| NATS Term          |  Deployable? |     Server-side Object? | Application Role? | Main Purpose                   |
| ------------------ | -----------: | ----------------------: | ----------------: | ------------------------------ |
| NATS Server        |      **Yes** |                         |                   | NATS runtime                   |
| NATS Cluster       | **Topology** |                         |                   | HA/scaling                     |
| NATS Client        |              |                         |           **Yes** | Application connection library |
| Publisher          |              |                         |           **Yes** | Sends messages                 |
| Subscriber         |              |                         |           **Yes** | Receives messages              |
| Processor / Worker |              |                         |           **Yes** | Processes messages             |
| Subject            |              |       **Yes / logical** |                   | Message addressing             |
| Subscription       |              |             **Runtime** |                   | Client interest in Subject     |
| Queue Group        |              |             **Runtime** |                   | Core NATS competing consumers  |
| JetStream          |              |   **Server capability** |                   | Persistence/streaming          |
| Stream             |              |                 **Yes** |                   | Message storage/history        |
| Consumer           |              |                 **Yes** |                   | Stateful message delivery      |
| Durable Consumer   |              |                 **Yes** |                   | Persistent Consumer state      |
| Ephemeral Consumer |              |                 **Yes** |                   | Temporary Consumer             |
| Deliver Policy     |              |     **Consumer config** |                   | Starting delivery position     |
| Ack Policy         |              |     **Consumer config** |                   | ACK behavior                   |
| ACK                |              |       Runtime operation |           **Yes** | Processing acknowledgement     |
| Storage            |              | **Yes / configuration** |                   | Message persistence            |
| File Storage       |              |    **Physical storage** |                   | Durable messages               |
| Memory Storage     |              |    **Physical storage** |                   | In-memory messages             |
| Replica            |              |     **Yes / data copy** |                   | HA                             |
| Stream Leader      |              |        **Runtime role** |                   | Coordinates replicated Stream  |
| Consumer Leader    |              |        **Runtime role** |                   | Coordinates Consumer state     |
| Route              |              |  **Runtime connection** |                   | Server-to-server communication |

---

# 37. Simplified Mental Model

For most architecture discussions, the following model is sufficient:

```text
                    APPLICATIONS
                         |
               +---------+---------+
               |                   |
            Publisher           Worker
               |                   |
               |                   |
               v                   |
             Subject               |
               |                   |
        +------+-------+           |
        |              |           |
        v              v           |
   Core NATS       JetStream       |
                      |             |
                   Stream           |
                      |             |
                   Consumer <-------+
                      |
                 ACK / Redelivery
                      |
                      v
                 Stored Data
                      |
             +--------+--------+
             |        |        |
          Replica  Replica  Replica
             |        |        |
             +--------+--------+
                      |
                NATS Cluster
                      |
             +--------+--------+
             |        |        |
          Server   Server   Server
```

## The five concepts to remember

1. **Subject** → Where a message is addressed.
2. **Stream** → Where JetStream stores messages.
3. **Consumer** → How an application reads messages from a Stream.
4. **Queue Group** → How Core NATS distributes live messages among competing subscribers.
5. **NATS Server** → The actual deployable runtime hosting these capabilities and objects.

The most important architectural distinction is therefore:

> **NATS Server is deployed. Stream and Consumer are created/configured inside NATS. Publisher, Subscriber, and Worker are application components that connect to NATS.**
