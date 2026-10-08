```mermaid
flowchart TD

    subgraph PHASE1 ["Phase 1: Environment & Setup"]
        START(["NATS Developer Walkthrough"])
        START --> UNDERSTAND["1. Understand NATS<br/><br/>Ref: NATS Documentation"]
        UNDERSTAND --> ENV{"2. Select Environment"}
        ENV -->|"Local"| LOCAL["Local Development<br/><br/>Ref: NATS Local Setup Guide"]
        ENV -->|"Org Dev"| ACCESS["Dev Environment Access<br/><br/>Ref: NATS Dev Environment Access Guide"]
        ACCESS --> DEVREADY["Dev Environment Ready"]
        LOCAL --> CONNECT
        DEVREADY --> CONNECT["3. Verify NATS Connectivity<br/><br/>Ref: NATS Connectivity Guide"]
    end

    subgraph PHASE2 ["Phase 2: Design & Client SDK"]
        SUBJECT["4. Understand Subject Design<br/><br/>Ref: NATS Subject Design"]
        CLIENT["5. Use NATS Client SDK<br/><br/>Ref: NATS Client Guide"]
        MESSAGING["6. Implement Messaging"]
        SUBJECT --> CLIENT --> MESSAGING
    end

    subgraph PHASE3 ["Phase 3: Messaging Execution"]
        PUBLISH["Publish Messages<br/><br/>Ref: NATS Publisher Guide"]
        RECEIVE["Receive Messages"]
        PERSIST{"Requires Persistence or<br/>Durable Processing?"}
        SUBSCRIBE["Core NATS Subscription<br/><br/>Ref: NATS Subscription Guide"]
        CONSUMER["JetStream Consumer<br/><br/>Ref: NATS Consumer Guide"]

        RECEIVE --> PERSIST
        PERSIST -->|"No"| SUBSCRIBE
        PERSIST -->|"Yes"| CONSUMER
    end

    subgraph PHASE4 ["Phase 4: Usage Patterns & Integration"]
        PATTERNS["7. Explore Usage Patterns"]
        ASYNC["Async Inter-Service Comm<br/><br/>Ref: NATS Async Inter-Service Communication"]
        SAGA["Event-Driven Saga<br/><br/>Ref: NATS Event-Driven Saga"]
        COMPLETE(["NATS Integrated into Application"])

        PATTERNS --> ASYNC --> COMPLETE
        PATTERNS --> SAGA --> COMPLETE
    end

    CONNECT --> SUBJECT
    MESSAGING --> PUBLISH
    MESSAGING --> RECEIVE
    PUBLISH --> PATTERNS
    SUBSCRIBE --> PATTERNS
    CONSUMER --> PATTERNS

    classDef start fill:#0B5FFF,color:#fff,stroke:#084BCC,stroke-width:2px
    classDef step fill:#E8F1FF,color:#111,stroke:#5B8DEF,stroke-width:1.5px
    classDef decision fill:#FFF4CC,color:#111,stroke:#D6A700,stroke-width:2px
    classDef guide fill:#F5F5F5,color:#111,stroke:#888,stroke-width:1.5px
    classDef pattern fill:#F0E8FF,color:#111,stroke:#805AD5,stroke-width:2px
    classDef endpoint fill:#E7F7ED,color:#111,stroke:#35A66F,stroke-width:2px

    class START start
    class UNDERSTAND,CONNECT,SUBJECT,CLIENT,MESSAGING,PATTERNS step
    class ENV,PERSIST decision
    class LOCAL,ACCESS,DEVREADY,PUBLISH,RECEIVE,SUBSCRIBE,CONSUMER guide
    class ASYNC,SAGA pattern
    class COMPLETE endpoint
```

---

## Guide Index & Reference Matrix

| Step | Document Brief / Purpose | Document Link |
| :--- | :--- | :--- |
| **1. Understand NATS** | Overview of NATS concepts, messaging models, subjects, streams, and JetStream consumers. | |
| **2. Development Environment** | Setup instructions for local NATS testing and organization dev environment access procedures. | |
| **3. Verify NATS Connectivity** | Protocol connectivity validation, DNS resolution, CLI commands, and network troubleshooting. | |
| **4. Understand Subject Design** | Subject naming hierarchy (`<domain>.<entity>.<action>`), validation rules, and wildcard matching semantics. | |
| **5. NATS Client SDK** | Connection initialization, cluster failover, lifecycle handlers, liveness options, and JetStream API context. | |
| **6. Publish Messages** | Core NATS publishing, JetStream persistent publishing, headers, deduplication, and publish expectations (OCC). | |
| **6. Core NATS Subscription** | Ephemeral pub/sub subscriptions, async callbacks, sync pull loops, queue group worker pools, and draining. | |
| **6. JetStream Consumer** | Persistent durable/ephemeral consumer CRUD, pull batching (`fetch`), ordered consumers, and ACK/NAK signals. | |
| **7. Async Inter-Service Communication** | Architecture reference for replacing synchronous HTTP/gRPC calls with asynchronous NATS messaging patterns. | |
| **7. Event-Driven Saga** | Distributed transaction coordination across microservices using event-driven saga orchestration over NATS. | |
