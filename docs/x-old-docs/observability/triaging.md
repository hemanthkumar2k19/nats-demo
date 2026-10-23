# NATS Triaging Guide

This guide provides a compact, practical playbook for triaging application and messaging failures in a development environment using a **MELT (Metrics, Events, Logs, Traces)** observability approach across application code, NATS Server protocol streams, and `nats-cli` diagnostic tools.

---

## 1. Problem Statement & MELT Triaging Hierarchy

In asynchronous NATS messaging architectures, execution is decoupled across microservices:
- **Publishers** send messages to NATS subjects without blocking on consumer processing.
- **NATS Broker** routes, filters, or persists messages into JetStream streams.
- **Consumers** receive or fetch messages asynchronously.

When a message is missing, delayed, or repeatedly redelivered, developers face a distributed systems challenge: **Publisher** -> **NATS Broker** -> **Consumer**.

To efficiently isolate root causes, developers follow the **MELT Triaging Hierarchy**:

1. **Traces (Primary Entry Point)**: Start with the distributed trace waterfall using `trace_id`. The trace immediately reveals which span failed (`SpanKindProducer` vs `SpanKindConsumer`), where latency occurred (broker queuing vs consumer execution), and whether trace context propagation broke.
2. **Logs (Deep Execution Context)**: Use the `trace_id` from the trace to query correlated application JSON logs and NATS Server protocol logs (`HPUB`, `HMSG`) for exact exception messages, stack traces, and header attributes.
3. **Events & Metrics (Broker & State Inspection)**: Inspect NATS system advisories (`$SYS.>`, `$JS.EVENT.ADVISORY.>`) and runtime metrics (`nats consumer info`, consumer lag, slow consumer warnings) to diagnose broker-level or consumer-state root causes (such as max redeliveries exceeded or stream quota limits).

---

## 2. End-to-End MELT Correlation Architecture

The following diagram illustrates how telemetry pillars correlate across the message processing lifecycle:

```bash
TRACES (W3C Trace Context)
Trace ID: 4bf92f3577b34da6a3ce929d0e0e4736
|
+-- Producer Span (Publish)
|     |
|     v W3C Header (traceparent)
|
+-- Consumer Span (Worker Handler)

LOGS (Correlated by Trace ID)
+-- Publisher Log  -> trace_id=4bf92f3577b34da6a3ce929d0e0e4736
+-- NATS Server    -> HPUB orders.created traceparent:00-4bf92f3577b34da6a3ce929d0e0e4736...
+-- Consumer Log   -> trace_id=4bf92f3577b34da6a3ce929d0e0e4736

EVENTS & METRICS (Broker & State Inspection)
+-- JetStream Advisories -> $JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES...
+-- Consumer Metrics     -> Pending: 12, Unacknowledged: 1, Redelivered: 5
```

---

## 3. MELT Triaging Playbook

### 3.1 Scenario: Message Published but Not Received by Consumer

1. **Trace Inspection**:
   Search distributed traces by `trace_id`.
   - **Producer span exists, Consumer child span missing**: Context was injected, but the consumer never initiated processing.
2. **Log Correlation**:
   Search NATS Server protocol logs for `trace_id` or `HPUB`.
   - **No `HPUB` log**: Message never reached NATS Server. Check publisher client connection or publish error.
   - **`HPUB` log present**: Note the exact subject string in the `HPUB` log line (e.g. `HPUB orders.created`).
3. **Event & Metric Verification**:
   Inspect broker routing and subscriptions using `nats-cli`:
   ```bash
   # Check active Core NATS subscriptions on the subject
   nats sub "orders.created" --dump

   # Inspect JetStream Stream subject bindings
   nats stream info ORDERS_STREAM
   ```

---

### 3.2 Scenario: Handler Errors & JetStream Redelivery Loops

1. **Trace Inspection**:
   Locate the trace by `trace_id`. Identify the consumer span marked with `status=Error` and inspect error span attributes.
2. **Log Correlation**:
   Search consumer container logs using `trace_id` to inspect exception messages, stack traces, and acknowledgement calls:
   - **Explicit NAK**: Log indicates handler called `msg.Nak()`. JetStream reschedules delivery based on backoff config.
   - **AckWait Timeout**: Log shows handler execution exceeded `AckWait` without calling `msg.Ack()`.
3. **Event & Metric Verification**:
   - Check if JetStream emitted a Max Deliveries Exceeded advisory (`$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>`):
     ```bash
     nats sub "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>" --dump
     ```
   - Inspect consumer redelivery stats:
     ```bash
     nats consumer info ORDERS_STREAM fulfillment-consumer
     ```

---

### 3.3 Scenario: Request-Reply Timeout (`nats: timeout`)

1. **Trace Inspection**:
   Inspect the client request span. Check if a child responder span was created.
2. **Log Correlation**:
   Grep responder service logs for the inbox reply subject (`_INBOX...`) to check if the request was received and if `msg.Respond()` was called.
3. **Event & Metric Verification**:
   Test responder availability using `nats-cli`:
   ```bash
   nats req "orders.get" "" --timeout=1s
   ```
   If `No Responders Available` is returned, no application worker is listening on the request subject.

---

### 3.4 Scenario: Slow Consumer & Connection Drops

1. **Trace Inspection**:
   Observe a high latency gap between message publish time and consumer span start time.
2. **Log Correlation**:
   Grep NATS Server logs for `Slow Consumer Detected` warnings and note the client connection ID (`cid`).
3. **Event & Metric Verification**:
   Inspect connection details and buffer limits:
   ```bash
   nats connection info <cid>
   ```
   *Resolution*: Ensure consumer callbacks do not execute long-running blocking operations directly on the NATS client event loop.

---

## 4. CLI Diagnostic Commands Cheat-Sheet

```bash
# Monitor live messages and headers on a subject
nats sub "orders.>" --dump

# Check JetStream Stream status and bound subjects
nats stream info ORDERS_STREAM

# Check consumer processing lag, unacknowledged counts, and redeliveries
nats consumer info ORDERS_STREAM fulfillment-consumer

# Monitor JetStream advisory events (max deliveries, quota limits)
nats sub "$JS.EVENT.ADVISORY.>" --dump
```

---

## 5. Quick MELT Triaging Matrix

| Symptom | Trace Finding | Log Finding | Event / Metric Finding | Root Cause & Resolution |
|---|---|---|---|---|
| **Missing Delivery** | Producer span only; missing Consumer child span. | NATS log has `HPUB <subject>`; missing Consumer log. | `nats sub` shows no active subscriber on subject. | Subject mismatch or missing Stream subject binding. Update subject hierarchy or stream config. |
| **Repeated Redeliveries** | Consumer span marked `status=Error` repeatedly. | Consumer log shows exception or `Nak()` call. | `nats consumer info` shows high `Redelivered` count. | Handler error or `AckWait` timeout. Fix handler exception or adjust `AckWait`. |
| **Request-Reply Timeout** | Client request span times out; missing responder span. | NATS log has `HPUB _INBOX...`; missing responder log. | `nats req` returns `No Responders Available`. | No active service listening on request subject, or handler execution exceeded client timeout. |
| **Slow Consumer** | Large latency gap before Consumer span start. | NATS log shows `Slow Consumer` warning for `cid`. | `nats connection info` shows pending buffer overflow. | Worker callback executing blocking operations on NATS event loop. Offload to async worker pool. |
