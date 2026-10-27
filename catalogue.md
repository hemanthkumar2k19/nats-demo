# Feature Catalogue

| # | Category | Feature | Scope / Target | Status |
| :--- | :--- | :--- | :--- | :--- |
| 1 | NATS Setup and Client Connectivity | NATS Local Cluster | 3-node local cluster deployment via Podman Compose | Completed |
| 2 | Publishing | Transient Publishing | Ephemeral in-memory publishing via Core NATS | Completed |
| 3 | Publishing | Persistent Publishing | Stream-backed persistent publishing via JetStream | Completed |
| 4 | Consumer | Push Consumer | Notification Worker (Push delivery) | Completed |
| 5 | Consumer | Pull Consumer | Payment Worker (Pull delivery) | Completed |
| 6 | Consumer | Consumer State | State validation, sequence tracking, and ack floor monitoring | Completed |
| 7 | Consumer | Consumer Parallelism | Concurrent message processing across worker pools | Completed |
| 8 | Delivery Semantics | At-Most-Once Delivery | Processing Service fire-and-forget message delivery | Completed |
| 9 | Delivery Semantics | At-Least-Once Delivery | Processing Service persistent pull with explicit acknowledgment | Completed |
| 10 | Delivery Semantics | Exactly-Once Delivery | Processing Service publisher deduplication and consumer synchronous acknowledgment | Completed |
| 11 | Message Replay | Message Replay | Stream event replay via start sequence and timestamp policies | Planned |
| 12 | Failure Handling and Reliability | Publisher Reliability | Disconnect handling, retry buffers, and connection reconnect options | Planned |
| 13 | Failure Handling and Reliability | Consumer Reliability | Graceful shutdown, crash redelivery, max delivery limits, and DLQ advisories | Planned |
| 14 | Failure Handling and Reliability | NATS Cluster Reliability | Cluster node failure, failover resilience, and Raft consensus validation | Planned |
