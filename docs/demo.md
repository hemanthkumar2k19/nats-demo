# NATS Platform Demo - Technical Documentation (NATS Demo View)

## 1. Objective and Scope

### 1.1 Purpose
The **NATS Demo View** (Platform Core Flow Stage 1-2-3) demonstrates Core NATS message publishing, durable at-least-once message delivery via NATS JetStream persistent streams, durable pull consumers with competing worker goroutines, dynamic scaling without service restart, and interactive runtime failure simulations.

### 1.2 Evaluation Objectives
* **NATS Server Message Publishing**: Demonstrate single and concurrent batch message ingestion with custom headers, metadata, and sliding-window deduplication (`Nats-Msg-Id`).
* **Durable Delivery via JetStream**: Demonstrate persistent, file-backed stream storage (`JOBS`) ensuring messages survive consumer detachment, service restarts, and worker crashes.
* **Durable Pull Consumers & Dynamic Scaling**: Demonstrate how worker goroutines pull messages from durable consumer `job-processor`, with on-the-fly competing worker scaling (`1W` to `5W`) with zero message loss and zero service restarts.
* **Decoupling & Offline Buffering**: Demonstrate broker-side queue buffering when worker pull loops are paused, followed by zero-loss drain upon resumption.
* **Failure Simulations & Multi-Worker Fault Recovery**: Interactively simulate worker crashes with failover to healthy peers, broker AckWait timeouts with slow workers, explicit negative acknowledgments (`NAK`), and poison pill message termination (`TERM`).

### 1.3 Scope Matrix (NATS Demo View)

| Capability Domain | In Scope (Demonstrated in this View) | Out of Scope (Capability Studio) |
| :--- | :--- | :--- |
| **Stage 1: Publisher** | Single publish (`POST /jobs`), 6-job concurrent batching (`POST /jobs/batch`), deduplication (`Nats-Msg-Id`), custom headers builder, JSON payload editor, wire envelope inspector. | Synthetic continuous load generators. |
| **Stage 2: Stream / Broker** | File-backed stream (`JOBS`), subject routing (`jobs.submitted`), limits retention, real-time message/byte metrics, copyable CLI commands, stream purge. | Multi-stream mirrors, cross-account imports. |
| **Stage 3: Consumer / Worker** | Durable pull consumer (`job-processor`), 2x2 non-editable config grid, explicit ACK policy, dynamic scaling (`1W` to `5W`), tactile pause/resume switch, Failure Lab simulations, live terminal load distribution. | Core NATS ephemeral queue groups, synchronous RPC Request/Reply. |
| **Failure Recovery** | Isolated worker crash before ACK (Scenario 7), slow worker AckWait timeout (Scenario 8), worker pool scale down (Scenario 9), worker pool scale up (Scenario 10), explicit worker NAK, poison message termination (DLQ routing), supervisor revival. | Network partition split-brain chaos tests. |

---

## 2. Architecture and Approach

### 2.1 3-Stage Pipeline Topology
The NATS Demo View aligns three distinct architectural tiers side-by-side:

```text
+---------------------------------------------------------------------------------------------------------------+
|                                            NATS DEMO VIEW CONSOLE                                             |
+-----------------------------------+-----------------------------------+---------------------------------------+
|         STAGE 1: PUBLISHER        |       STAGE 2: NATS STREAM        |          STAGE 3: PROCESSOR           |
|       Application Ingestion       |      Broker Persistence Tier      |         Worker Execution Pool         |
+-----------------------------------+-----------------------------------+---------------------------------------+
| * Microservice: job-service:8081  | * JetStream Stream: 'JOBS'        | * JetStream Object:                   |
| * Subject: jobs.submitted         | * Storage: File (Persistent)      |   - Consumer: job-processor           |
| * Single & Batch (6 Jobs)         | * Real-time Stream Counters       |   - Attached Stream: JOBS             |
| * Deduplication Window ID         | * Retention Policy: Limits        |   - Filter Subject: jobs.submitted    |
| * Custom KV Headers & Metadata    | * Live Copyable CLI Commands      |   - Ack Policy: Explicit              |
| * Live Wire Envelope Inspector    | * Instant Stream Purge Action     |   - AckWait: 5s                       |
|                                   |                                   | * Microservice:                       |
|                                   |                                   |   - processor-service:8082            |
|                                   |                                   |   - Worker Pool: [1W..5W] (Dynamic)   |
|                                   |                                   |   - Goroutine State: RUNNING/DEGRADED |
|                                   |                                   |   - Pull Loop Switch: ON/OFF          |
|                                   |                                   |   - Failure Lab: 4 Armed Scenarios    |
|                                   |                                   |   - Terminal Load Telemetry           |
+-----------------------------------+-----------------------------------+---------------------------------------+
```

### 2.2 Component Roles & Specifications

| Component | Network Endpoint | Primary Responsibility in Demo |
| :--- | :--- | :--- |
| **`job-service`** (Stage 1) | `http://localhost:8081` | Ingestion REST gateway. Accepts job submissions, binds headers, injects W3C trace IDs, and publishes to NATS. |
| **NATS Server** (Stage 2) | `nats://localhost:4222`<br>`http://localhost:8222` | Core broker & JetStream engine. Manages `JOBS` stream, enforces deduplication, tracks sequence numbers, and coordinates pull consumer batches. |
| **`processor-service`** (Stage 3) | `http://localhost:8082` | Background worker daemon. Pulls messages via `job-processor` durable consumer, executes business logic, scales goroutines (`1W`..`5W`), reports status (`/processor/status`), handles state toggles (`/processor/state`), and runs Failure Lab scenarios (`/processor/scenario`). |
| **Demo Console** (UI) | `http://localhost:5173` | Unified React cockpit providing synchronized controls and live parameter inspection across all three stages. |

> [!NOTE]
> All real-time event streaming in the UI is ingested via `demo-control-service:8080`. The legacy direct endpoint `GET /processor/events` on `:8082` is deprecated; direct HTTP communication with `:8082` is used exclusively for worker pool control, status polling, and failure scenario triggers.

---

## 3. Capability Demonstration

### 3.1 Stage 1: Publisher Capabilities

| Control / Action | Type | Demo Purpose | Observable Result |
| :--- | :--- | :--- | :--- |
| **Submit Job** | Button (`POST /jobs`) | Publishes single payload to stream `JOBS` on subject `jobs.submitted`. | Message accepted; assigned unique ID and stream sequence. |
| **Batch (6 Jobs)** | Button (`POST /jobs/batch`) | Dispatches 6 concurrent jobs in one action. | Demonstrates concurrent competing pull distribution across worker goroutines. |
| **Deduplication Key** | Input (`Nats-Msg-Id`) | Tests exactly-once ingestion guarantee. | Re-submitting with unchanged ID increments duplicate counter in broker; worker does not re-process. |
| **Custom Headers Builder** | Key-Value Grid | Appends custom application metadata to the NATS envelope. | Custom headers travel untouched across the wire to Stage 3. |
| **Wire Envelope Inspector** | Collapsible JSON View | Displays raw message bytes, headers, and metadata before wire transit. | Transparent visibility into the exact payload dispatched to NATS. |

---

### 3.2 Stage 2: Broker & Stream Management

| Metric / Action | Display / Setting | Technical Significance |
| :--- | :--- | :--- |
| **Stream Name** | `JOBS` | Root storage container holding persisted job records. |
| **Storage Engine** | `File` (`file`) | Messages persist to disk, surviving server restarts and network interruptions. |
| **Retention Policy** | `Limits` (`limits`) | Enforces enterprise retention based on age, max messages, or max bytes. |
| **Real-Time Counters** | Messages, Bytes, First/Last Seq | Direct telemetry reflecting stream ingestion and consumption state. |
| **CLI Inspection Center** | Copyable Terminal Commands | Enables terminal users to cross-verify UI state with official `nats` CLI: `nats stream info JOBS`, `nats stream view JOBS`. |
| **Purge Stream** | Action Button | Instantly clears stream messages while keeping consumer configuration intact. |

---

### 3.3 Stage 3: Consumer & Worker Execution

#### 3.3.1 Consumer Configuration (JetStream Object)
The broker-side consumer parameters are displayed at the top of Stage 3 as clean, non-editable configuration fields arranged in a 2x2 grid:

| Field Name | Configured Value | Operational Purpose |
| :--- | :--- | :--- |
| **ATTACHED STREAM** | `JOBS` | Binds this consumer specifically to the `JOBS` storage stream. |
| **FILTER SUBJECT** | `jobs.submitted` | Ensures workers only receive messages routed to `jobs.submitted`. |
| **ACK POLICY** | `EXPLICIT` (`msg.Ack()`) | Broker requires explicit acknowledgment; unacknowledged messages are safely held. |
| **ACKWAIT THRESHOLD** | `5s (Redelivery Timer)` | Server countdown timer for unacknowledged messages before triggering redelivery. |

#### 3.3.2 Microservice Worker Pool (`processor-service:8082`)
Directly below the JetStream consumer card, the microservice parameters and interactive controls are displayed:

| Control / Parameter | Setting / Range | Operational Behavior |
| :--- | :--- | :--- |
| **ROLE** | `Go Worker Daemon` | Non-editable identifier for the backend processing daemon. |
| **CONSUMING FROM** | `job-processor (Pull)` | Non-editable indicator of the bound JetStream consumer. |
| **GOROUTINE STATE** | `RUNNING` / `DEGRADED` / `CRASHED` | Real-time health status. When a worker crashes in Failure Lab, displays `DEGRADED: X/Y active` with the crashed worker name and a `Revive Goroutine Now` action. |
| **WORKER POOL SIZE** | `[ 1W ] [ 2W ] [ 3W ] [ 5W ]` | **Zero-Disruption Dynamic Scaling**:<br>* **Scale UP**: Spawns additional worker goroutines; existing workers continue running without interruption.<br>* **Scale DOWN**: Gracefully stops excess worker goroutines from the tail without dropping in-flight messages. |
| **Worker Pull Loop Switch** | `ON` / `OFF` | **Offline Buffering Demonstration**:<br>* When **OFF**: Workers pause pulling. Inbound messages buffer safely in the JetStream stream.<br>* When **ON**: Pull loop resumes immediately, draining the buffered backlog. |

#### 3.3.3 Real-Time Terminal Load Telemetry
When workers process messages, `processor-service` emits real-time load distribution summaries to the terminal log:

```text
[Processor] [PULLED] Msg Seq=14 -> processor-1 | Worker load: 1 (Pool: [processor-1=1, processor-2=0])
[Processor] [PULLED] Msg Seq=15 -> processor-2 | Worker load: 1 (Pool: [processor-1=1, processor-2=1])
[Processor] [ACK SENT] Msg Seq=14 from processor-1 | Worker load: 1 (Pool: [processor-1=1, processor-2=1])
```

---

### 3.4 Failure Lab (Simulated Edge Cases & Multi-Worker Failures)

The **Failure Lab** is an arrow-based collapsible accordion embedded in Stage 3 (collapsed by default). It allows interactive simulation of real-world worker failures, broker timeouts, and competing consumer recovery directly mapping to `docs/feature.md`:

| Scenario | Mode / Parameter | Simulated Failure | Broker Behavior | Worker Behavior | Outcome / Recovery |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1. One Worker Crashes (Scenario 7)** | `crash_before_ack` | Simulates an unhandled Go runtime panic in one worker midway through transaction before `msg.Ack()`. | Starts 5s `AckWait` timer; holds message unacknowledged. | Panicked worker dies abruptly; state transitions to `DEGRADED: X/Y active` (identifying crashed worker). Surviving healthy workers continue pulling other jobs normally. | At `T = 5.0s`, server redelivers message (`Delivery: #2`) to an active healthy peer worker, which ACKs. Supervisor revives crashed worker automatically after 5.5s (or operator clicks `Revive Goroutine Now`). |
| **2. One Worker Is Slow (Scenario 8)** | `exceed_ack_wait` | One worker takes 7.0s to process a job, exceeding the 5.0s broker `AckWait` threshold. | At `T = 5.0s`, server timer expires and marks message for redelivery. | Slow worker keeps running. Surviving peer workers continue pulling other messages concurrently. | At `T = 5.0s`, healthy peer pulls redelivered message, finishes, and ACKs at `T = 6.0s`. Slow worker sends late ACK notice at `T = 7.0s`. |
| **3. Worker NAKs Message** | `nak_message` | Worker detects a transient downstream error and transmits explicit negative acknowledgment (`msg.Nak()`). | Broker does not wait for 5s timeout; immediately re-queues message for redelivery. | Worker drops attempt #1 and immediately re-pulls message. | Message is redelivered instantly (`Attempt #2`) without waiting for timeout dead time. |
| **4. Worker Terminates Message** | `term_message` | Worker detects an unrecoverable poison message and issues terminal acknowledgment (`msg.Term()`). | Broker halts redelivery loop immediately; stops scheduling retries. | Worker drops the payload and records poison message event. | Protects worker pool from infinite retry poison pills; routes message to Dead Letter handling. |

---

### 3.5 Live Demonstration Runbook

| Step # | Demo Narrative | Operator Action | Observable Confirmation |
| :--- | :--- | :--- | :--- |
| **1. Baseline Ingestion** | Show normal message publishing and processing. | Click **Submit Job** in Stage 1. | Message ingested in Stage 2, processed by Stage 3, and acknowledged. |
| **2. Server Deduplication** | Prove server prevents duplicate ingestion. | Keep same `Nats-Msg-Id` and click **Submit Job** again. | Stream message count does not increase; duplicate rejected by server. |
| **3. Offline Buffering** | Show broker buffers jobs when consumers are offline. | Toggle Worker Pull Loop to **OFF** in Stage 3, then click **Submit Job** 3 times in Stage 1. | Stream messages increment to `3`. Zero jobs processed. Jobs buffered safely in broker. |
| **4. Backlog Recovery** | Show zero message loss upon worker recovery. | Toggle Worker Pull Loop to **ON** in Stage 3. | All 3 buffered jobs immediately pulled and processed in sequence. |
| **5. Worker Pool Scales Up (Scenario 10)** | Show backlog distribution across expanding workers without restart. | Set workers to **1W**, toggle pull loop **OFF**, click **Batch (6 Jobs)** to build backlog, switch to **3W**, and toggle pull loop **ON**. | New workers (`processor-2`, `processor-3`) join immediately; backlog drains 3x faster; no duplicate deliveries. |
| **6. Worker Pool Scales Down (Scenario 9)** | Show worker pool reduction with zero dropped messages. | Set workers to **3W**, click **Batch (6 Jobs)**, then immediately click **1W**. | Excess workers stop gracefully from tail; remaining worker (`processor-1`) processes all remaining jobs without loss. |
| **7. One Worker Crashes (Scenario 7)** | Prove failover to surviving healthy workers upon crash. | Set workers to **3W**, expand **Failure Lab**, click **Arm Crash**, click **Submit Job**. | `processor-1` panics (`DEGRADED: 2/3 active`). Surviving workers remain alive. At `T = 5s`, healthy peer pulls redelivery and ACKs. Supervisor revives worker at `T = 5.5s`. |
| **8. One Worker Is Slow (Scenario 8)** | Prove timeout failover while other workers continue. | Set workers to **2W**, click **Arm Exceed (7s)**, click **Submit Job**, then click **Submit Job** again. | `processor-1` is slow (7s). `processor-2` processes job 2 immediately. At `T = 5s`, `processor-2` takes over job 1 redelivery and ACKs at `T = 6s`. |
| **9. Fast Retry (NAK)** | Show immediate retry without timeout delay. | In Failure Lab, click **Arm NAK**, click **Submit Job**. | Worker sends `msg.Nak()`. Message redelivered instantly on Attempt #2 without waiting 5s. |
| **10. Poison Pill (TERM)** | Show poison message isolation. | In Failure Lab, click **Arm Term**, click **Submit Job**. | Worker issues `msg.Term()`. Redeliveries halt immediately; stream remains stable. |

---

## 4. Verification & Inspection CLI Commands

Operators can verify the state of the broker and consumers at any point using the official `nats` CLI:

```bash
# Check JOBS stream status, message count, and bytes
nats stream info JOBS

# Inspect the last messages stored in the JOBS stream
nats stream view JOBS

# Check durable consumer job-processor status, pending acks, and redelivery counts
nats consumer info JOBS job-processor

# View real-time consumer message delivery
nats consumer next JOBS job-processor --count 1
```
