# NATS Platform Demo - Functional Testing Guide

This guide provides step-by-step instructions for evaluating, demonstrating, and testing each NATS capability implemented in this platform demo.

---

## Prerequisites & Starting the Platform

Open terminal windows to launch the demo components:

```bash
# Terminal 0: Start NATS Broker, Exporter, and Grafana OTEL-LGTM
docker compose -f deploy/docker-compose.yaml up -d

# Terminal 1: Demo Control Service (UI Gateway, Activity Taps, Replay & Consumer Control)
cd backend/src && go run cmd/demo-control-service/main.go

# Terminal 2: Job Service (Pure Business REST API, Publisher, Trace Context Injection)
cd backend/src && go run cmd/job-service/main.go

# Terminal 3: Processor Service (Consumers, Workers, Request/Reply Responder)
cd backend/src && go run cmd/processor-service/main.go

# Terminal 4: Demonstration UI (React SPA)
cd frontend && npm run dev
```

Open your browser to: `http://localhost:5173`

Verify that the header displays:
```text
NATS CONNECTED (Port 4222)
```

---

## Demonstration Scenarios

### Scenario 1: Core NATS Transient Messaging (Pub/Sub)

#### What NATS Feature Is Demonstrated?
* **Core NATS fire-and-forget publish/subscribe**.
* Messages are delivered directly to active connected subscribers in memory. If no subscriber is listening, messages are discarded with zero persistence.

#### Steps:
1. In the **Submit Job** panel (under `DEMO ACTIONS`):
   - Leave **Job ID** as auto-generated (e.g. `job-101`).
   - Select **Delivery Mode: Core NATS (Transient)**.
   - Click **Submit Job**.

#### What to Observe:
* **Dynamic Alert Banner**: Displays `Job job-101 submitted via Core NATS`.
* **Activity Log**:
  - `PUBLISHED` on subject `jobs.submitted` (Mode: `CORE`, Worker: `job-service`, Seq: `-`).
  - `RECEIVED` on subject `jobs.received` (Worker: `processor-1` or `processor-2`).
  - `COMPLETED` on subject `jobs.completed`.
* **Terminal Logs**:
  ```text
  [job-service] Published job job-101 to Core NATS subject jobs.submitted
  [processor-service] [worker] Received Core NATS job: job-101
  [processor-service] [worker] Job job-101 completed
  ```

---

### Scenario 2: JetStream Durable Storage & Guaranteed Delivery

#### What NATS Feature Is Demonstrated?
* **JetStream At-Least-Once Delivery & Stream Persistence**.
* Messages are written to the `JOBS` stream disk storage before acknowledgement. They remain in the stream for pull consumers and historical replays.

#### Steps:
1. In **Submit Job**:
   - Select **Delivery Mode: JetStream (Durable)**.
   - Click **Submit Job**.

#### What to Observe:
* **Current Demo Setup Topology**: The `JOBS Stream` card updates its message counter.
* **Activity Log**:
  - `PUBLISHED` with Mode: `JETSTREAM` and a global stream sequence (e.g. `#1`, `#2`).
  - `STORED`: Confirms message stored in `JOBS` stream.
  - `RECEIVED`: Pull consumer fetched message from stream.
  - `COMPLETED` and `ACKED`: Worker explicitly acknowledged the message back to JetStream.
* **Terminal Logs**:
  ```text
  [job-service] Published job job-102 to JetStream JOBS (stream seq: 2)
  [processor-service] [worker] Fetched JetStream job: job-102 (stream seq: 2)
  [processor-service] [worker] Explicitly ACKed job-102
  ```

---

### Scenario 3: Processing State Control & Offline Message Accumulation

#### What NATS Feature Is Demonstrated?
* **Store-and-Forward durability vs. Fire-and-Forget message loss**.
* Demonstrates how JetStream safely buffers messages when downstream consumers are offline, while Core NATS drops messages without consumers.

#### Steps:
1. In **Platform Status** bar at the top, find **Processing: ON**.
2. Click **[ Turn OFF ]**.
3. Observe **CURRENT DEMO SETUP** topology:
   - The connector between Consumer and Processor turns amber with badge: `[ PAUSED ]`.
   - The worker nodes transition to `Paused` status.
4. **Submit a Core NATS Job**:
   - Select **Core NATS (Transient)** and click **Submit Job**.
   - **Observation**: Activity log logs `NO CONSUMER` or message is lost. Core NATS has nowhere to buffer it.
5. **Submit a JetStream Job**:
   - Select **JetStream (Durable)** and click **Submit Job**.
   - **Observation**: Activity log logs `STORED` (e.g. `#3`).
   - In **Consumer Status** (inside Consumer Lab), notice **Pending Messages** increments to `1` (or more).
6. **Resume Processing**:
   - In Platform Status, click **[ Turn ON ]**.
7. **Observation**:
   - The topology connector immediately lights up green `---> v`.
   - The pending JetStream message is instantly pulled, processed, and marked `COMPLETED` and `ACKED`!
   - **Pending Messages** drops back to `0`.

---

### Scenario 4: JetStream Message Deduplication (`Nats-Msg-Id`)

#### What NATS Feature Is Demonstrated?
* **Server-side idempotent publishing via JetStream deduplication window**.
* Prevents duplicate writes when a producer retries publishing within the 2-minute deduplication window.

#### Steps:
1. In **Submit Job**:
   - Enter a custom **Job ID**: `dedup-test-999`.
   - Select **Delivery Mode: JetStream (Durable)**.
   - Click **Submit Job**.
   - Observe: Job `dedup-test-999` publishes, is stored at sequence `#N`, and completes.
2. **Submit Duplicate Immediately**:
   - Without changing the Job ID (`dedup-test-999`), click **Submit Job** again immediately.

#### What to Observe:
* **Activity Log**:
  - Displays event badge: `DEDUPLICATED` on subject `jobs.deduplicated`.
  - The sequence number `#N` matches the original sequence.
  - The job is **NOT processed a second time** by the workers.
* **Terminal Logs**:
  ```text
  [job-service] JetStream detected duplicate publish for Msg-Id: dedup-test-999 (seq: N)
  ```

---

### Scenario 5: Competing Consumers (Multi-Worker Distribution)

#### What NATS Feature Is Demonstrated?
* **Dynamic worker pool sharing a single pull consumer**.
* Demonstrates load distribution across competing workers without message duplication.

#### Steps:
1. In **CURRENT DEMO SETUP** -> **Consumer Lab** (right side):
   - Set **Workers**: `2 Workers (Competing)`.
   - Click **Apply Configuration**.
2. Observe **Demo Topology** (left side):
   - The single shared `Consumer` branches into two distinct worker cards side-by-side: `processor-1` and `processor-2` with a `COMPETING` badge:
     ```text
                  [ Consumer: job-processor ]
                               |
                +--------------+--------------+
                |          COMPETING          |
                v                             v
      [ processor-1: ACTIVE ]       [ processor-2: ACTIVE ]
     ```
3. Submit 4 jobs in rapid succession:
   - Click **Submit Job** 4 times.
4. Filter **Activity Log**:
   - Look at the **Worker** column in the Activity Log table.
   - Notice jobs are distributed across `processor-1` and `processor-2`!
   - In the search/filter bar, select **Worker: processor-1** to see its allocated jobs, then switch to **processor-2**.

---

### Scenario 6: Durable vs. Ephemeral Consumers

#### What NATS Feature Is Demonstrated?
* **Consumer Lifecycle**: Named persistent state cursor vs temporary consumer.

#### Steps:
1. In **Consumer Lab**:
   - Select **Consumer Type: Ephemeral**.
   - Click **Apply Configuration**.
2. **Observation in Topology**:
   - The Consumer node badge switches to `EPHEMERAL`.
   - The consumer name changes to a dynamic ephemeral identifier (e.g. `ephem-...`).
3. Submit a JetStream job.
   - The ephemeral consumer pulls and acknowledges the message.
4. In **Consumer Lab**, select **Consumer Type: Durable** and click **Apply Configuration**:
   - The consumer switches back to the durable consumer `job-processor`, which resumes from the stream cursor.

---

### Scenario 7: Ordered Consumer Delivery

#### What NATS Feature Is Demonstrated?
* **Strict stream sequence preservation**.
* Guarantees messages are processed strictly one-by-one in exact stream sequence order without concurrency race conditions.

#### Steps:
1. In **Consumer Lab**:
   - Set **Ordering**: `Ordered (Strict Sequence)`.
2. **Notice**:
   - The **Workers** select box is automatically locked to `1 Worker` (ordered consumers cannot have competing concurrent workers).
3. Click **Apply Configuration**.
4. Submit multiple jobs.
5. In **Activity Log**:
   - Inspect the sequence numbers. All events occur in strictly monotonic stream order (`#1`, `#2`, `#3`...) with 1 worker (`processor-1`).

---

### Scenario 8: JetStream Replay (Time Travel)

#### What NATS Feature Is Demonstrated?
* **Stream Replay / Audit Rewind**.
* Replays historical messages from the beginning of the stream without deleting or mutating stream data.

#### Steps:
1. Ensure at least 3-5 JetStream jobs have been submitted previously.
2. In the **JetStream Replay** panel (under `DEMO ACTIONS`):
   - Verify `Stream: JOBS` is displayed.
   - Leave **Replay From** as `Sequence` (or switch to `Time`).
   - Set **Start Sequence** to `1` and **End Sequence** to `100` (or your desired historical range).
   - Leave **Replay Mode** as `Instant` (or select `Original Timing` to observe pacing).
   - Click **Start Replay**.
3. **What to Observe**:
   - The status banner confirms replay started with the ephemeral consumer ID (e.g. `replay-xxxxxx`).
   - The **Activity Log** receives historical stream messages with the `REPLAYED` status badge.
   - The original messages in the stream remain untouched.

---

### Scenario 9: NATS Request / Reply (Synchronous RPC & Timeout)

#### What NATS Feature Is Demonstrated?
* **Point-to-Point Request/Reply over NATS ephemeral inboxes (`_INBOX.>`)**.
* Demonstrates synchronous RPC over NATS messaging, including natural timeout handling when no responder is active.

#### Steps - Normal Flow:
1. In the **Request / Reply** panel (under `DEMO ACTIONS`):
   - Enter a test Job ID: `req-001`.
   - Enter a test payload (e.g. `{"valid": true}`).
   - Click **Send Validation Request**.
2. **Observation**:
   - Result card displays `VALID` with message: `Job payload is valid`.
   - Activity Log logs:
     - `REQUEST_SENT` on `jobs.validate`.
     - `REQUEST_RECEIVED` by `processor-1`.
     - `REPLY_SENT` back to the dynamic `_INBOX.*` address.
     - `REPLY_RECEIVED` by `job-service`.

#### Steps - Timeout Flow:
1. In Platform Status, toggle **Processing: OFF**.
2. In the **Request / Reply** panel, click **Send Validation Request** again.
3. **Observation**:
   - The request waits for 2 seconds (the NATS Request timeout).
   - The result returns `TIMEOUT` / `504 Gateway Timeout`.
   - Activity Log logs: `REQUEST_TIMEOUT`.
   - Demonstrates that NATS Request/Reply fails gracefully with natural timeout semantics when services are down or paused.
4. Toggle **Processing: ON** to resume normal operation.

---

### Scenario 10: NATS Subject Addressing & Wildcards

#### What NATS Feature Is Demonstrated?
* **NATS Subject Hierarchies & Pattern Matching**:
  - Exact token match: `jobs.submitted`
  - Single-level wildcard (`*`): `jobs.*` matches subjects with exactly 2 tokens (`a.b`).
  - Multi-level wildcard (`>`): `jobs.>` matches subjects with any number of tokens >= 2 (`a.b`, `a.b.c`, etc.).

#### Steps:
1. In the **Submit Job** panel, submit a job (either Core NATS or JetStream).
2. Scroll to the **NATS Subject Addressing** panel (bottom right) and inspect the **Subject Routing Activity** table.
3. **What to Observe**:
   - **`jobs.submitted` (2 tokens `a.b`)**:
     - Exact: **Yes**
     - Single-Level (*): **Yes**
     - Multi-Level (>): **Yes**
   - **`jobs.processing.started` (3 tokens `a.b.c`)**:
     - Exact: **No**
     - Single-Level (*): **No** (single-level wildcard rejects 3-segment subjects)
     - Multi-Level (>): **Yes (> only)** (multi-level wildcard accepts any number of trailing segments)
   - **`jobs.completed` (2 tokens `a.b`)**:
     - Exact: **No**
     - Single-Level (*): **Yes**
     - Multi-Level (>): **Yes**
4. Demonstrates the critical distinction between single-level (`*`) and multi-level (`>`) wildcards: `*` matches strictly one token, while `>` matches arbitrarily deep hierarchies.

---

### Scenario 11: Activity Log Search, Filters & Identifier Tooltips

#### What NATS Feature Is Demonstrated?
* **Full-text search, multi-criteria filtering, and message deduplication identifier tooltips**.

#### Steps:
1. **Search Bar**:
   - In the Activity Log toolbar, type a Job ID (e.g. `job-101`) in the search box.
   - Notice the table instantaneously filters down to only events for that job.
   - Click the **[ x ]** button to clear.
2. **Event Filter Dropdown**:
   - In the toolbar, select **Event: COMPLETED**.
   - Notice only completion events appear.
   - The result counter displays: `Showing X of Y events`.
3. **Hover Legend Tooltips**:
   - Look at the **NATS Msg ID** column.
   - Notice long message IDs are formatted as clean compact badges.
   - Hover your mouse cursor over the badge:
     - A floating dark tooltip legend appears showing the full `Nats-Msg-Id` header string used by JetStream deduplication.
4. Click **Clear** to reset all filters.

---

### Scenario 12: Observing NATS & Application Metrics in Grafana

#### What NATS Feature Is Demonstrated?
* **Full-stack Metrics Observability**:
  - NATS Server connections, subscriptions, and message rates via `nats-io/prometheus-nats-exporter`.
  - JetStream stream storage, pending message queue build-up, and consumer lag.
  - Application job submissions, validation RPCs, and execution latencies via OpenTelemetry.

#### Steps:
1. Ensure the observability stack is running:
   ```bash
   docker compose -f deploy/docker-compose.yaml up -d
   ```
2. Open Grafana at: `http://localhost:3000` (or click **[ Open Grafana -> ]** in the **OBSERVABILITY SETUP** section of the React UI).
3. Log in with credentials: `admin` / `admin`.
4. Open the dashboard: **NATS Platform Demo - Metrics**.
5. **Observe NATS Server Metrics**:
   - Verify **Connections** shows active clients (e.g. 2 or 3).
   - Verify **Subscriptions** shows active topic registrations.
6. **Observe JetStream Metrics Under Load**:
   - In the React UI, toggle **Processing: OFF**.
   - Submit 5 JetStream jobs.
   - In Grafana, observe **Pending Messages** in the JetStream section rise to `5`.
   - In the React UI, toggle **Processing: ON**.
   - In Grafana, observe **Pending Messages** drop back down to `0` and **Jobs Processed** increment by 5.
7. **Observe Failure & Redelivery Metrics**:
   - In Submit Job, tick "Simulate Failure" and submit a JetStream job.
   - In Grafana, observe **Redelivered Messages** and **Jobs Failed** increase.
8. **Observe Request/Reply RPC Latency**:
   - In Request / Reply, click **Send Validation Request**.
   - In Grafana, observe **NATS Requests** count increment and latency registered in **Job Processing Duration (Latency)**.

---

### Scenario 13: End-to-End Distributed Tracing with OpenTelemetry & Tempo

#### What NATS Feature Is Demonstrated?
* **W3C Trace Context Propagation over NATS Headers**:
  - OpenTelemetry `traceparent` header injection into `nats.Msg.Header` upon publish.
  - Context extraction by consumers to link consumer receive and processing spans to the original publisher trace.
  - Distributed tracing across asynchronous pub/sub, JetStream persistent delivery, and synchronous request/reply patterns.
  - Trace visualization in Grafana Tempo.

#### Steps:
1. Ensure the stack is running:
   ```bash
   docker compose -f deploy/docker-compose.yaml up -d
   ```
2. In the React UI, submit a JetStream job.
3. In the **ACTIVITY LOG**, click on the newly submitted job ID to open the **Job Details Inspector**.
4. Observe the **Trace ID** field displayed in purple/indigo monospace, alongside a **[ View in Tempo -> ]** button.
5. Click **[ View in Tempo -> ]** (or open `http://localhost:3000/explore` and paste the Trace ID in the Tempo query input).
6. Observe the complete distributed trace waterfall diagram in Tempo:
   - `POST /jobs` (Server span, job-service)
     - `NATS Publish jobs.submitted` (Producer span, messaging.system: nats, jetstream.stream: JOBS)
       - `Consumer Receive` (Consumer span, processor-service, messaging.destination: jobs.submitted)
         - `Process Job` (Internal span, worker: processor-1, delivery.count: 1)
7. **Trace Validation RPC (Request/Reply)**:
   - In the **REQUEST / REPLY VALIDATION** panel, click **Send Validation Request**.
   - Check the response card or the Activity Log for the Trace ID.
   - In Tempo, view the synchronous trace waterfall:
     - `POST /jobs/validate` (Server span, job-service)
       - `NATS Request jobs.validate` (Client span, job-service)
         - `Process Validation Request` (Server span, processor-service)
           - `NATS Reply` (Producer span, processor-service)
8. **Trace Error / Redelivery Scenarios**:
   - Submit a job with "Simulate Failure" checked.
   - In Tempo, note that the `Process Job` span records the error with `status: Error` and the `Consumer Receive` span registers the `redelivery_scheduled` event.

---

## Troubleshooting & Verification Checklist

| Symptom | Likely Cause | Solution |
| :--- | :--- | :--- |
| NATS Disconnected in UI | NATS Server not running | Ensure NATS is running on `localhost:4222` (`nats-server -js`). |
| Processor Offline in UI | `processor-service` not started | Run `go run ./src/cmd/processor-service/main.go`. |
| JetStream Jobs Not Processing | Processing is toggled OFF | Click `[ Turn ON ]` in Platform Status. |
| Replay yields no messages | No JetStream messages in stream | Submit at least 1 JetStream job before triggering Replay. |
| Duplicate job processed anyway | Outside 2-minute dedup window | JetStream deduplication window is 2 minutes; submit duplicate within 120 seconds. |
| Tempo shows no traces | OTLP exporter not connected | Ensure `otel-lgtm` is running and port 4317 is accessible. |

