# Feature Catalogue

| # | Category | Feature | Scope / Target | Status |
| :--- | :--- | :--- | :--- | :--- |
| 1 | NATS Setup and Client Connectivity | NATS Local Cluster | 3-node local cluster deployment via Podman Compose | Completed |
| 2 | Publishing | Transient Publishing | Ephemeral in-memory publishing via Core NATS | Completed |
| 3 | Publishing | Persistent Publishing | Stream-backed persistent publishing via JetStream | Completed |
| 4 | Publishing | Rich Message Publishing | Runtime short ID generation, header enrichment, and unescaped JSON data formatting | Completed |
| 5 | Consumer | Push Consumer | Notification Worker (Push delivery) | Completed |
| 6 | Consumer | Pull Consumer | Payment Worker (Pull delivery) | Completed |
| 7 | Consumer | Consumer State | State validation, sequence tracking, and ack floor monitoring | Completed |
| 8 | Consumer | Consumer Parallelism | Concurrent message processing across worker pools | Completed |
| 9 | Delivery Semantics | At-Most-Once Delivery | Processing Service fire-and-forget message delivery | Completed |
| 10 | Delivery Semantics | At-Least-Once Delivery | Processing Service persistent pull with explicit acknowledgment | Completed |
| 11 | Delivery Semantics | Exactly-Once Delivery | Processing Service publisher deduplication and consumer synchronous acknowledgment | Completed |
| 12 | Stream Inspection | Stream Sequence Discovery | Discovery of sequence and timestamp boundaries to guide replay and reprocess workflows | Completed |
| 13 | Message Replay | Historical Audit Replay | Non-mutating stream event replay by sequence or timestamp range with execution summary | Completed |
| 14 | Message Replay | Message Reprocessing | Re-executing stream messages with domain business effects by sequence or timestamp range | Completed |
| 15 | Failure Handling and Reliability | Publisher Reliability | Disconnect handling, retry buffers, and connection reconnect options | Planned |
| 16 | Failure Handling and Reliability | Consumer Reliability | Graceful shutdown, crash redelivery, max delivery limits, and DLQ advisories | Planned |
| 17 | Failure Handling and Reliability | NATS Cluster Reliability | Cluster node failure, failover resilience, and Raft consensus validation | Planned |
