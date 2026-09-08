# NATS Platform Demo - Technical Documentation (NATS Demo View)

## 1. Objective and Scope

### 1.1 Purpose
The **NATS Demo View** demonstrates Core NATS message publishing, durable at-least-once message delivery via NATS JetStream persistent streams, durable pull consumers with competing consumer workers, and interactive runtime failure simulations.

### 1.2 Evaluation Objectives
* **NATS Server Message Publishing**: Demonstrate single and concurrent batch message ingestion with custom headers, metadata, and sliding-window deduplication (`Nats-Msg-Id`).
* **Durable Delivery via JetStream**: Demonstrate persistent, file-backed stream storage (`JOBS`) ensuring messages survive consumer detachment and service restarts.
* **Durable Pull Consumers & Dynamic Scaling**: Demonstrate how worker goroutines pull messages from durable consumer `job-processor`, with on-the-fly competing worker scaling (`1W` to `5W`) without restart.
* **Decoupling & Offline Buffering**: Demonstrate broker-side queue buffering when worker pull loops are paused, followed by zero-loss drain upon resumption.
* **Failure Simulations & Fault Recovery**: Interactively simulate worker crashes, broker AckWait timeouts, explicit negative acknowledgments (`NAK`), and poison pill message termination (`TERM`).

### 1.3 Scope Matrix (NATS Demo View)

| Capability Domain | In Scope (Demonstrated in this View) | Out of Scope (Other Views) |
| :--- | :--- | :--- |
| **Stage 1: Publisher** | Single publish, 6-job concurrent batching, deduplication (`Nats-Msg-Id`), custom headers builder, wire envelope inspector. | Auxiliary background traffic generators. |
| **Stage 2: Stream / Broker** | File-backed stream (`JOBS`), subject routing (`jobs.submitted`), limits retention, real-time message/byte metrics, CLI commands, stream purge. | Multi-stream mirror configurations, cross-account imports. |
| **Stage 3: Consumer / Worker** | Durable pull consumer (`job-processor`), explicit ACK policy, dynamic scaling (`1W` to `5W`), tactile pause/resume switch, Failure Lab simulations. | Core NATS ephemeral queue groups, synchronous RPC Request/Reply. |
| **Failure Recovery** | Worker crash before ACK, AckWait expiration (5s), explicit worker NAK, poison message termination (DLQ routing). | Network partition split-brain chaos tests. |

---

## 2. Architecture and Approach

### 2.1 3-Stage Pipeline Topology
The NATS Demo View aligns three distinct architectural tiers side-by-side:

```text
+-------------------------------------------------------------------------------------------------------+
|                                        NATS DEMO VIEW CONSOLE                                         |
+-----------------------------------+-----------------------------------+-------------------------------+
|         STAGE 1: PUBLISHER        |       STAGE 2: NATS STREAM        |      STAGE 3: PROCESSOR       |
|       Application Ingestion       |      Broker Persistence Tier      |     Worker Execution Pool     |
+-----------------------------------+-----------------------------------+-------------------------------+
| * Microservice: job-service:8081  | * JetStream Stream: 'JOBS'        | * JetStream Object:           |
| * Subject: jobs.submitted         | * Storage: File (Persistent)      |   - Consumer: job-processor   |
| * Single & Batch (6 Jobs)         | * Real-time Stream Counters       |   - Ack Policy: Explicit      |
| * Deduplication Window ID         | * Retention Policy: Limits        |   - AckWait: 5s               |
| * Custom KV Headers & Metadata    | * Live Copyable CLI Commands      | * Microservice:               |
| * Live Wire Envelope Inspector    | * Instant Stream Purge Action     |   - processor-service:8082    |
|                                   |                                   |   - Worker Pool: [1W..5W]     |
|                                   |                                   |   - Pull Loop Switch: ON/OFF  |
|                                   |                                   |   - Failure Lab: 4 Scenarios  |
+-----------------------------------+-----------------------------------+-------------------------------+
```

### 2.2 Component Roles & Specifications

| Component | Network Endpoint | Primary Responsibility in Demo |
| :--- | :--- | :--- |
| **`job-service`** (Stage 1) | `http://localhost:8081` | Ingestion REST gateway. Accepts job payloads, binds headers, injects W3C trace IDs, and publishes to NATS. |
| **NATS Server** (Stage 2) | `nats://localhost:4222`<br>`http://localhost:8222` | Core broker & JetStream engine. Manages `JOBS` stream, enforces deduplication, tracks sequence numbers, and coordinates pull consumer batches. |
| **`processor-service`** (Stage 3) | `http://localhost:8082` | Background worker daemon. Pulls messages via `job-processor` durable consumer, runs business logic, scales goroutines, and executes Failure Lab scenarios. |
| **Demo Console** (UI) | `http://localhost:5173` | Unified React cockpit providing synchronized controls and live parameter inspection across all three stages. |

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
The broker-side consumer parameters are displayed as clean non-editable configuration fields:

| Field Name | Configured Value | Operational Purpose |
| :--- | :--- | :--- |
| **Attached Stream** | `JOBS` | Binds this consumer specifically to the `JOBS` storage stream. |
| **Filter Subject** | `jobs.submitted` | Ensures workers only receive messages routed to `jobs.submitted`. |
| **Ack Policy** | `Explicit (msg.Ack())` | Broker requires explicit acknowledgment; unacknowledged messages are safely held. |
| **AckWait Threshold** | `5s (Redelivery Timer)` | Server countdown timer for unacknowledged messages before triggering redelivery. |

#### 3.3.2 Microservice Worker Pool (`processor-service:8082`)

| Control / Parameter | Setting / Range | Operational Behavior |
| :--- | :--- | :--- |
| **Role** | `Go Worker Daemon` | Non-editable identifier for the backend processing daemon. |
| **Consuming From** | `job-processor (Pull)` | Non-editable indicator of the bound JetStream consumer. |
| **Worker Pool Size** | `[ 1W ] [ 2W ] [ 3W ] [ 5W ]` | **Zero-Disruption Dynamic Scaling**:<br>* **Scale UP**: Spawns only additional worker goroutines; existing workers continue running without interruption.<br>* **Scale DOWN**: Gracefully stops excess worker goroutines from the tail. |
| **Goroutine State** | `RUNNING` / `CRASHED` | Real-time health status. If a worker panics in the Failure Lab, displays `CRASHED` with a `Revive Goroutine Now` action. |
| **Worker Pull Loop Switch** | `ON` / `OFF` | **Offline Buffering Demonstration**:<br>* When **OFF**: Workers pause pulling. Inbound messages buffer safely in the JetStream stream.<br>* When **ON**: Pull loop resumes immediately, draining the buffered backlog. |

---

### 3.4 Failure Lab (Simulated Edge Cases)

The **Failure Lab** is an arrow-based collapsible accordion embedded in Stage 3 (default collapsed). It allows interactive simulation of real-world worker and communication failures.

| Scenario | Mode / Parameter | Simulated Failure | Broker Behavior | Worker Behavior | Outcome / Recovery |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **1. Worker Crash Before ACK** | `crash_before_ack` | Simulates an unhandled Go runtime panic midway through transaction before `msg.Ack()` is sent. | Starts 5s `AckWait` timer; holds message unacknowledged. | Goroutine dies abruptly (`goroutineStatus = CRASHED`). | After 5s, server redelivers message (`Delivery: #2`) to healthy worker. Supervisor revives crashed worker. |
| **2. Exceed AckWait Threshold** | `exceed_ack_wait` | Injects an artificial 7.0s delay, exceeding the 5.0s broker `AckWait` threshold. | At `T = 5.0s`, server timer expires and marks message for redelivery. | Worker finishes transaction at `T = 7.0s` and transmits late `msg.Ack()`. | Demonstrates timeout race condition and importance of sending `msg.InProgress()` or tuning AckWait. |
| **3. Worker NAKs Message** | `nak_message` | Worker detects a transient downstream error and transmits explicit negative acknowledgment (`msg.Nak()`). | Broker does not wait for 5s timeout; immediately re-queues message. | Worker drops attempt #1 and immediately re-pulls message. | Message is redelivered instantly (`Attempt #2`) without waiting for timeout dead time. |
| **4. Worker Terminates Message** | `term_message` | Worker detects an unrecoverable poison message and issues terminal acknowledgment (`msg.Term()`). | Broker halts redelivery loop immediately; stops scheduling retries. | Worker drops the payload and records poison message event. | Protects worker pool from infinite retry poison pills; routes to Dead Letter handling. |

---

### 3.5 Live Demonstration

| Step # | Demo Narrative | Operator Action | Observable Confirmation |
| :--- | :--- | :--- | :--- |
| **1. Baseline Flow** | Show normal message publishing and processing. | Click **Submit Job** in Stage 1. | Message ingested in Stage 2, processed by Stage 3, and acknowledged. |
| **2. Deduplication** | Prove server prevents duplicate ingestion. | Keep same `Nats-Msg-Id` and click **Submit Job** again. | Stream message count does not increase; duplicate rejected by server. |
| **3. Offline Buffering** | Show broker buffers jobs when consumers are offline. | Toggle Worker Pull Loop to **OFF** in Stage 3, then click **Submit Job** 3 times in Stage 1. | Stream messages increment to `3`. Zero jobs processed. Jobs buffered safely in broker. |
| **4. Backlog Recovery** | Show zero message loss upon worker recovery. | Toggle Worker Pull Loop to **ON** in Stage 3. | All 3 buffered jobs immediately pulled and processed in sequence. |
| **5. Dynamic Scaling** | Show horizontal scaling without service restart. | Click **3W** in Stage 3, then click **Batch (6 Jobs)** in Stage 1. | 3 concurrent worker goroutines pull and execute 6 jobs simultaneously. |
| **6. Crash Recovery** | Prove automated recovery from fatal worker panics. | Open **Failure Lab**, click **Arm Crash**, click **Submit Job**. | Worker panics (`CRASHED`). Broker waits 5s AckWait, then redelivers to healthy worker. |
| **7. Fast Retry (NAK)** | Show immediate retry without timeout delay. | In Failure Lab, click **Arm NAK**, click **Submit Job**. | Worker sends `msg.Nak()`. Message redelivered instantly on Attempt #2. |
| **8. Poison Pill (TERM)** | Show poison message isolation. | In Failure Lab, click **Arm Term**, click **Submit Job**. | Worker issues `msg.Term()`. Redeliveries halt immediately. |
