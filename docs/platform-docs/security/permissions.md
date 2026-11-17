# Permissions

Defines operational Subject-Based Access Control (SBAC) permission profiles across three user levels: System Operator Users, Platform Users, and Application Users for NATS and JetStream operations.

---

## 1. System Operator User Permissions

The System Operator user manages NATS cluster nodes, server processes, and server diagnostics. By default, this user holds standard, untweaked system privileges with unrestricted access to:

- `$SYS.REQ.SERVER.PING` - Server ping and latency monitoring.
- `$SYS.REQ.SERVER.STATZ` - Server metrics and JetStream cluster reports.
- `$SYS.REQ.SERVER.CONNZ` - Client connection listings and telemetry.
- `$SYS.REQ.ACCOUNT.STATZ` - Account resource reports.
- `$SYS.>` - Server event advisories, cluster consensus state, and system logging.

---

## 2. Platform User Permissions

The Platform user manages messaging infrastructure, resource provisioning, capacity limits, and operational health for JetStream. They have full control over the JetStream administrative control plane with enterprise guardrails protecting audit trails and stability.

### Allow

| Subject | API | Reason |
|---|---|---|
| `$JS.API.STREAM.>` | Stream Management (Publish) | Provision, update, purge, snapshot, restore, and decommission streams. |
| `$JS.API.CONSUMER.>` | Consumer Management (Publish) | Provision, tune, pause, inspect, and decommission consumers. |
| `$JS.API.ACCOUNT.>` | Account Administration (Publish) | Configure JetStream storage limits, stream quotas, and tiers. |
| `$JS.API.INFO` | Account Info (Publish) | Query JetStream capacity utilization and resource counts. |
| `$JS.API.*.LEADER.STEPDOWN.>` | Cluster Rebalance (Publish) | Manage leader election and stepdown during cluster maintenance. |
| `$JS.API.*.PEER.>` / `.EVACUATE.>` | Replica Placement (Publish) | Evacuate, remove, or rebalance Raft peers across cluster nodes. |
| `$JS.API.META.>` / `$JS.API.SERVER.>` | Cluster Meta (Publish) | Execute cluster meta-controller placement and consensus recovery. |
| `_INBOX.>` | Reply Inbox (Publish/Subscribe) | Receive administrative CLI responses and health probe outputs. |
| `orders.>`, `payments.>`, `delivery.>` | Domain Topic Validation (Publish) | Validate stream ingestion and consumer routing. |

### Deny

| Subject | API | Reason |
|---|---|---|
| `$JS.API.ACCOUNT.PURGE` | Indiscriminate Account Purge (Publish) | Block single-command account wiping; enforce stream-by-stream decommissioning. |
| `$JS.API.STREAM.MSG.DELETE.>` | Message Tampering (Publish) | Protect audit log immutability; prevent point deletion of records. |
| `*.secret.>` | Sensitive Business Payloads (Publish/Subscribe) | Enforce data-plane zero trust; operators manage pipes, not confidential payloads. |
| `$SYS.>` | System Diagnostics & Telemetry (Publish/Subscribe) | Server ping, varz, and connz are reserved strictly for the System Operator role. |

---

## 3. Application User Permissions

Application users operate strictly on the data plane using least privilege. They are granted access only to the operational JetStream system subjects required for SDK functionality, business topics, and temporary inboxes, while all administrative and cluster management subjects are blocked.

### Allow

| Subject | API | Reason |
|---|---|---|
| `$JS.API.CONSUMER.MSG.NEXT.<stream>.<consumer>` | Consumer Next (Publish) | Request next message batch for pull consumers (`Fetch` / `Consume`). |
| `$JS.ACK.<stream>.<consumer>.>` (or `$JS.ACK.>`) | Consumer Ack (Publish) | Acknowledge processed messages (`Ack`, `Nak`, `Term`, `InProgress`). |
| `$JS.API.CONSUMER.INFO.<stream>.<consumer>` | Consumer Info (Publish) | Inspect consumer state during SDK initialization and health checks. |
| `$JS.API.STREAM.INFO.<stream>` | Stream Info (Publish) | Inspect stream status and bounds during SDK initialization. |
| `$JS.API.STREAM.MSG.GET.<stream>` | Direct Msg Get (Publish) | Retrieve individual message by sequence or last-by-subject. |
| `$JS.API.DIRECT.GET.<stream>.>` | Direct Scale Get (Publish) | High-performance read directly from stream replicas or KV buckets. |
| `_INBOX.>` | Reply Inbox (Subscribe) | Receive `PubAck`, pull message batches, and JetStream API responses. |

### Deny

| Subject | API | Reason |
|---|---|---|
| `$JS.API.STREAM.CREATE.>` | Stream Create (Publish) | Block unauthorized stream creation outside SRE lifecycle. |
| `$JS.API.STREAM.UPDATE.>` | Stream Update (Publish) | Block altering retention, storage, max bytes, or subject filters. |
| `$JS.API.STREAM.DELETE.>` | Stream Delete (Publish) | Block accidental or unauthorized stream deletion. |
| `$JS.API.STREAM.PURGE.>` | Stream Purge (Publish) | Block clearing persisted stream message data. |
| `$JS.API.STREAM.MSG.DELETE.>` | Message Delete (Publish) | Block deleting individual persisted messages. |
| `$JS.API.STREAM.SNAPSHOT.>` | Stream Snapshot (Publish) | Block triggering resource-heavy backup snapshots. |
| `$JS.API.STREAM.RESTORE.>` | Stream Restore (Publish) | Block restoring or overwriting existing streams. |
| `$JS.API.STREAM.LIST` / `.NAMES` | Stream Discovery (Publish) | Block enumerating all account streams. |
| `$JS.API.CONSUMER.CREATE.>` | Consumer Create (Publish) | Block creating unmanaged durable consumers outside SRE intake. |
| `$JS.API.CONSUMER.DELETE.>` | Consumer Delete (Publish) | Block deleting provisioned consumers. |
| `$JS.API.CONSUMER.PAUSE.>` | Consumer Pause (Publish) | Block pausing message deliveries. |
| `$JS.API.CONSUMER.LIST.>` / `.NAMES.>` | Consumer Discovery (Publish) | Block enumerating all stream consumers. |
| `$JS.API.*.LEADER.STEPDOWN.>` | Leader Stepdown (Publish) | Block disrupting stream or consumer Raft cluster leaders. |
| `$JS.API.*.PEER.> / .UNPIN.>` | Peer Operations (Publish) | Block evacuating or modifying Raft replica topology. |
| `$JS.API.ACCOUNT.PURGE` | Account Purge (Publish) | Block wiping all stream data across the entire account. |
| `$JS.API.ACCOUNT.>` | Account Admin (Publish) | Block modifying account JetStream tier limits and allocations. |
| `$JS.API.META.>` / `$JS.API.SERVER.>` | Cluster Meta (Publish) | Block cluster consensus and server placement controls. |
| `$SYS.>` | System Telemetry (Publish/Subscribe) | Block access to server internals, cluster events, and metrics. |
