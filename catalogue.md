# Feature Catalogue

| # | Category | Feature | Scope / Target | Status |
| :--- | :--- | :--- | :--- | :--- |
| **1.1** | NATS Setup and Client Connectivity | NATS Local Cluster | 3-node local cluster deployment via Podman Compose | Completed |
| **2.1** | Publishing | Transient Publishing | Ephemeral in-memory publishing via Core NATS (`nc.Publish`) | Completed |
| **2.2** | Publishing | Persistent Publishing | Stream-backed persistent publishing with JetStream (`js.Publish`) | Completed |
| **3.1** | Consumer | Push Consumer | Notification Worker (`payments-push` consumer) | Completed |
| **3.2** | Consumer | Pull Consumer | Payment Worker (`payments-pull` consumer) | Completed |
| **3.3** | Consumer | Consumer State | State validation, sequence tracking, and ack floor monitoring | Completed |
| **3.4** | Consumer | Consumer Parallelism | Concurrent message processing across worker pools | In Progress |
| **4.1** | Delivery Semantics | At-Most-Once Delivery | Processing Service fire-and-forget over `delivery.atmost-once` | Completed |
| **4.2** | Delivery Semantics | At-Least-Once Delivery | Processing Service persistent pull with explicit acknowledgment | Completed |
| **4.3** | Delivery Semantics | Exactly-Once Delivery | Processing Service publisher deduplication window and consumer `DoubleAck` | Completed |
| **5.1** | Message Replay | Message Replay | Stream event replay via start sequence and timestamp policies | Planned |
| **6.1** | Failure Handling and Reliability | Publisher Reliability | Disconnect handling, retry buffers, and connection reconnect options | Completed |
| **6.2** | Failure Handling and Reliability | Consumer Reliability | Graceful shutdown, crash redelivery, max delivery limits, and DLQ advisories | Completed |
| **6.3** | Failure Handling and Reliability | NATS Cluster Reliability | Cluster node failure, failover resilience, and Raft consensus validation | Planned |
