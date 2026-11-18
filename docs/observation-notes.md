# NATS Observation Notes

## 1. Issue: Ephemeral Consumer Management Permissions for App Users

- **Symptom:**
  `Permissions Violation for Publish to "$JS.API.CONSUMER.CREATE.<STREAM>.replay-<TIMESTAMP>"`
- **Root Cause:**
  - NATS JetStream evaluates both durable and ephemeral consumers against the identical API subject:
    `$JS.API.CONSUMER.CREATE.<stream>.<consumer>`
  - `app-user` authorization policy in `auth/users.conf` places `$JS.API.CONSUMER.CREATE.>` in `publish.deny`.
  - In NATS authorization, `deny` rules have absolute precedence over `allow` rules.
  - Ephemeral consumers have no authorization bypass; attempting to create an on-demand consumer fails immediately.

---

## 2. Hindered Features

| Feature ID | Feature Name | Impact |
|---|---|---|
| #11 | Message Replay (Sequence & Time) | BLOCKED (`POST /api/v1/consumers/replay` cannot create ephemeral replay consumer). |
| #12 | Message Reprocessing | BLOCKED (`POST /api/v1/consumers/reprocess` cannot create ephemeral reprocess consumer). |
| #13 | Ephemeral Consumers & Ad-Hoc Analytics | BLOCKED (Any dynamic consumer lifecycle initiated by application runtime). |
| API | Consumer Creation Endpoint | BLOCKED (`POST /api/v1/consumers` cannot create consumers under `app-user`). |

---

## 3. Recommended Solutions

### Solution 1: Granular Whitelist Pattern (Recommended - Configuration)
- **Concept:** Remove blanket deny and leverage NATS native default-deny with subject prefix scoping.
- **Action:**
  1. Remove `$JS.API.CONSUMER.CREATE.>` and `$JS.API.CONSUMER.DELETE.>` from `publish.deny` in `auth/users.conf`.
  2. Add prefix-scoped rules to `publish.allow`:
     - `$JS.API.CONSUMER.CREATE.*.replay-*`
     - `$JS.API.CONSUMER.CREATE.*.reprocess-*`
     - `$JS.API.CONSUMER.DELETE.*.replay-*`
     - `$JS.API.CONSUMER.DELETE.*.reprocess-*`
- **Trade-off:** Minimal configuration change. App services can manage ephemeral replay consumers while durable consumer creation remains blocked by default-deny.

### Solution 2: Direct Point-Get Replay Pattern (Code Refactor)
- **Concept:** Eliminate ephemeral consumers entirely for sequence-based replay.
- **Action:**
  - `app-user` already has `$JS.API.DIRECT.GET.>` permitted.
  - Refactor sequence replay to fetch messages sequentially via direct point lookups (`js.GetMsg(stream, seq)`).
- **Trade-off:** Zero consumer creation or deletion requests, zero server-side consumer state, strictly data-plane operation. Requires updating `services/messaging/consumer.go`.

### Solution 3: Role / Persona Separation (Process & Architecture)
- **Concept:** Decouple administrative audit/replay tasks from operational application traffic.
- **Action:**
  - Replay and reprocessing APIs authenticate with `platform-user` (Control Plane) credentials.
  - Standard worker ingestion and business publishing remain on `app-user` (Data Plane).
- **Trade-off:** Preserves pure least-privilege zero-trust security on data-plane workloads. Requires multiple connection contexts or admin token routing in `order-service`.
