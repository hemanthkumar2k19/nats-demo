# Authorization

- Core Principle - Authorization answers "What can an authenticated client access?"
- Subject-Based Access Control (SBAC) - Every permission is strictly evaluated against NATS subjects.
- Independent Permissions - Publish and Subscribe permissions are completely independent of each other.

---

## Publish Permissions

Controls which subjects an authenticated user can send messages to.

```conf
permissions {
  publish: {
    allow: ["orders.>"]
    deny: ["orders.system.>"]
  }
}
```

### Allow
- Empty List / Omitted - User can publish to any subject (unrestricted).
- Single/Multiple Subjects - User can publish only to the subjects in the list; all other subjects are implicitly denied.
- Wildcards (*, >) - Expands publish scope across matching tokens or hierarchies.

### Deny
- Omitted - User can publish to any subject (unrestricted).
- Empty List([]) - User can publish to no subjects.
- Single/Multiple Subjects - User is strictly prohibited from publishing to the specified subjects.
- Wildcards (*, >) - Blocks entire branches (e.g. deny `$JS.API.STREAM.>` prevents stream admin operations).

---

## Subscribe Permissions

Controls which subjects an authenticated user can listen or consume from.

```conf
permissions {
  subscribe: {
    allow: ["orders.>", "_INBOX.>"]
    deny: ["orders.secret.>"]
  }
}
```

### Allow
- Omitted - User can subscribe to any subject (unrestricted).
- Empty List([]) - User can subscribe to no subjects.
- Single/Multiple Subjects - User can subscribe only to the subjects in the list; all other subjects are implicitly denied.
- Wildcards (*, >) - Expands subscription scope across matching tokens or hierarchies.

### Deny
- Empty List / Omitted - No subjects explicitly blocked.
- Single/Multiple Subjects - User is strictly prohibited from subscribing to the specified subjects.
- Wildcards (*, >) - Blocks listening on sensitive namespaces (e.g. deny `$SYS.>` blocks server telemetry).

> Note:
- Any client performing JetStream publishing must be granted subscribe: ["_INBOX.>"]. Because under hood its request reply.
- 
---

## Precedence & Rule Evaluation

Defines how overlapping Allow and Deny rules are resolved.

- Deny Always Wins - If a subject matches both an Allow rule and a Deny rule, the action is DENIED.
- Implicit Deny - If an Allow list exists, any subject not explicitly matched is DENIED.
- Evaluation Order:
  1. Check Deny list - If subject matches, DENY immediately.
  2. Check Allow list - If subject matches, ALLOW.
  3. Default Fallthrough - If an Allow list is defined, DENY (implicit deny). If no Allow list is defined, ALLOW.
- Specificity Does Not Override - A specific Allow rule (e.g. allow `orders.us.create`) does NOT override a broad Deny (e.g. deny `orders.>`).

---

## Wildcards Syntax

Pattern matching syntax supported across both Publish and Subscribe rules.

- `*` (Single Token Wildcard) - Matches exactly one token at that position (e.g. `orders.*.created` matches `orders.eu.created`, but not `orders.eu.b2b.created`).
- `>` (Multi-Token Wildcard) - Matches one or more trailing tokens; must be the last token in the subject (e.g. `orders.>` matches `orders.created` and `orders.us.b2b.created`).
- Exact Match - Subjects without wildcards match only the identical subject string.

---

## Request-Reply & Response Permissions

Governs reply behavior for request-reply messaging patterns.

```conf
permissions {
  # Responder subscribes to incoming service requests
  subscribe: ["orders.lookup"]

  # Dynamically permits replying without broad publish rights to all inboxes
  allow_responses: {
    max: 1
    expires: "30s"
  }
}
```

- Inboxes (`_INBOX.>`) - Requesters generate dynamic, temporary reply subjects under `_INBOX.>`.
- Requester Requirement - Requesting clients must have `subscribe: ["_INBOX.>"]` to receive responses.
- Replier Requirement - Service responders need permission to publish to the requester's reply inbox.
- `allow_responses` - Grants dynamic, temporary permission to reply to requests without granting broad permanent publish access to all inboxes.
- `max` (Response Count) - Limits the maximum number of responses permitted per request (e.g. `max: 1` prevents duplicate reply flooding).
- `expires` (TTL) - Duration window in which the reply permission remains valid (e.g. `expires: "30s"`; default has no expiration).

> Note:
- Incase of multi-part responses or streaming progress updates, the messages are silently dropped if we configure `max:1`
- `expires` should cover worst-case downstream latency plus network jitter
---

## JetStream Whitelisting

- JS APIs for Clients - JetStream SDK data-plane operations run over system subjects ($JS.API, $JS.ACK, _INBOX) and must be separated from control-plane administration.

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
