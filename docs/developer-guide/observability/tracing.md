# NATS Distributed Tracing Developer Guide

This guide details how to implement distributed tracing across NATS messaging patterns using OpenTelemetry and W3C Trace Context standards.

---

## 1. Overview of Distributed Tracing in Messaging Architectures

### 1.1 Distributed Tracing in Microservice Architectures
In modern microservice architectures, business transactions span multiple independent services. Distributed tracing provides visibility across these service boundaries by capturing execution spans and propagating trace context along request paths.

### 1.2 The Messaging System Tracing Challenge
When microservices communicate via messaging systems, execution boundaries become decoupled across network hops:
- Publishers transmit messages onto subjects and proceed with execution.
- Message brokers buffer, persist, and route payloads across subjects or stream partitions.
- Consumers receive or fetch messages for processing synchronously or asynchronously.

Without context propagation across the message transport, distributed traces break at the messaging layer, making it difficult to analyze end-to-end latency, isolate bottlenecks, or trace failure root causes.

### 1.3 Why Distributed Tracing is Critical for NATS
- **End-to-End Visibility**: Links publisher execution spans with subscriber processing spans into a continuous distributed trace.
- **Latency Bottleneck Identification**: Disambiguates time spent in publisher logic, broker queueing/delivery, and consumer handler processing.
- **Error Causality Tracking**: Traces consumer execution failures and redeliveries back to the originating upstream payload state.

---

## 2. W3C Trace Context Specification

OpenTelemetry and modern observability standards rely on the **W3C Trace Context** specification to propagate tracing metadata across service boundaries.

### 2.1 Core Header Fields

| Header Name | Structure / Format | Description & Example |
|---|---|---|
| `traceparent` | `version-trace_id-parent_id-trace_flags` | Uniquely identifies the trace (`128-bit hex`) and parent span (`64-bit hex`) along with sampling flags. <br>Example: `00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01` |
| `tracestate` | `vendor1=opaqueValue,vendor2=opaqueValue` | Carries vendor-specific or routing-specific metadata key-value pairs across tracing boundaries. <br>Example: `rojo=1,concho=2` |

---

## 3. Significance of NATS Headers for Trace Context Propagation

### 3.1 NATS Broker Transparency
NATS Server acts as a high-performance message broker and does not modify or inspect payload contents. NATS provides native metadata headers (`nats.Header`) attached to message payloads across Core NATS and JetStream.

### 3.2 Header-Based Trace Propagation
Because NATS preserves message headers intact during transit:
1. The publisher injects W3C `traceparent` and `tracestate` headers into NATS message headers.
2. NATS Server routes the message payload along with its headers.
3. The subscriber extracts W3C trace headers from NATS message headers to construct a child span linked to the publisher span.

This decouples trace propagation from payload encoding (e.g. JSON, Protobuf, Avro), eliminating the need to modify message body schemas.

---

## 4. Header Carrier Adapter Pattern

### 4.1 Concept & Pattern Overview
OpenTelemetry SDKs across all programming languages (Go, Java, Python, Node.js, C#) use a `TextMapCarrier` abstraction to inject and extract W3C trace headers into key-value map structures. Because NATS message headers store key-value string metadata (`nats.Header`), an adapter bridges native NATS message headers with the OpenTelemetry Context Propagator.

### Go SDK Adapter Implementation

```go
package tracing

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HeaderCarrier adapts nats.Header to satisfy propagation.TextMapCarrier.
type HeaderCarrier nats.Header

func (hc HeaderCarrier) Get(key string) string {
	return nats.Header(hc).Get(key)
}

func (hc HeaderCarrier) Set(key, value string) {
	nats.Header(hc).Set(key, value)
}

func (hc HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(hc))
	for k := range hc {
		keys = append(keys, k)
	}
	return keys
}

// Ensure HeaderCarrier implements propagation.TextMapCarrier
var _ propagation.TextMapCarrier = HeaderCarrier{}
```

---

## 5. The Two Universal Tracing Operations

Distributed tracing in NATS reduces to **two universal operations** regardless of transport (Core NATS vs JetStream, Sync vs Async, Push vs Pull):

### 5.1 Operation 1: Outbound Context Injection (Publisher Side)

Extract the active trace context from `context.Context`, start a producer span (`SpanKindProducer`), and inject W3C trace headers into the outgoing `nats.Msg` header before executing any publish or request operation.

#### Go SDK Implementation

```go
// InstrumentOutboundMessage creates a producer span and injects W3C trace context into nats.Msg
func InstrumentOutboundMessage(ctx context.Context, spanName string, msg *nats.Msg) (context.Context, trace.Span) {
	tr := otel.Tracer("nats-publisher")
	ctx, span := tr.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))

	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, HeaderCarrier(msg.Header))

	return ctx, span
}
```

---

### 5.2 Operation 2: Inbound Context Extraction (Subscriber / Consumer Side)

Extract W3C trace headers from the incoming message header (`nats.Msg.Header` or `jetstream.Msg.Headers()`), reconstruct the parent `context.Context`, and start a consumer span (`SpanKindConsumer`) before executing processing logic.

#### Go SDK Implementation

```go
// InstrumentInboundMessage extracts W3C trace context from incoming headers and starts a consumer span
func InstrumentInboundMessage(parentCtx context.Context, spanName string, header nats.Header) (context.Context, trace.Span) {
	tr := otel.Tracer("nats-consumer")

	if header != nil {
		parentCtx = otel.GetTextMapPropagator().Extract(parentCtx, HeaderCarrier(header))
	}

	return tr.Start(parentCtx, spanName, trace.WithSpanKind(trace.SpanKindConsumer))
}
```

---

## 6. Practical Application Across NATS Workflows

By applying `InstrumentOutboundMessage` on the publisher side and `InstrumentInboundMessage` on the subscriber side, any NATS workflow can be instrumented cleanly.

### 6.1 Publishing Workflows (Core NATS & JetStream)

Whether performing a simple publish, structured publish, async publish, or JetStream stream publish, call `InstrumentOutboundMessage` before executing the publish call:

```go
// Example: Core NATS or JetStream Sync Publish
func PublishTraced(ctx context.Context, nc *nats.Conn, subject string, payload []byte) error {
	msg := &nats.Msg{Subject: subject, Data: payload}

	// 1. Apply Operation 1: Outbound Injection
	_, span := InstrumentOutboundMessage(ctx, "nats.publish", msg)
	defer span.End()

	// 2. Execute publish
	if err := nc.PublishMsg(msg); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}
```

---

### 6.2 Subscribing & Consuming Workflows (Core NATS & JetStream)

Whether handling asynchronous callbacks, synchronous pull loops, queue group workers, or JetStream consumers, call `InstrumentInboundMessage` upon receiving each message:

```go
// Example: Core NATS Callback or JetStream Consumer Handler
func SubscribeTraced(nc *nats.Conn, subject string) (*nats.Subscription, error) {
	return nc.Subscribe(subject, func(msg *nats.Msg) {
		// 1. Apply Operation 2: Inbound Extraction
		_, span := InstrumentInboundMessage(context.Background(), "nats.process", msg.Header)
		defer span.End()

		// 2. Business processing logic
		log.Printf("Processed message on [%s]: %s", msg.Subject, string(msg.Data))
	})
}
```

---

### 6.3 Request-Reply & Redelivery Mechanics

- **Requestor**: Calls `InstrumentOutboundMessage` on `reqMsg` with `SpanKindClient`, executes `nc.RequestMsg`, and extracts returned response headers using `InstrumentInboundMessage` when `replyMsg` arrives.
- **Responder**: Calls `InstrumentInboundMessage` on `msg.Header` with `SpanKindServer`, processes the request, and calls `InstrumentOutboundMessage` on `respMsg` before sending `msg.Respond`.
- **JetStream Nak / Term**: Call `span.SetAttributes(attribute.Int64("nats.delivery_count", ...))` and record errors via `span.RecordError(err)` prior to calling `msg.NakWithDelay()` or `msg.Term()`.

---

## 7. Pattern Trace Mapping Matrix

| Messaging Pattern | NATS Transport Category | Universal Operation | Trace Propagation Strategy & Span Kind |
|---|---|---|---|
| **Simple / Structured Publish** | Core NATS | Operation 1: Outbound Injection | Inject W3C trace context into `nats.Msg.Header`; create producer span (`SpanKindProducer`). |
| **Request-Reply** | Core NATS | Both Operations | Requestor uses Operation 1 (`SpanKindClient`); responder uses Operation 2 (`SpanKindServer`) and returns reply trace headers. |
| **Async Callback Subscription** | Core NATS | Operation 2: Inbound Extraction | Extract parent trace context from incoming headers inside callback; create consumer span (`SpanKindConsumer`). |
| **Sync Subscription Pull** | Core NATS | Operation 2: Inbound Extraction | Extract parent trace context from message headers after synchronous retrieval; create consumer span (`SpanKindConsumer`). |
| **Queue Group Workers** | Core NATS | Operation 2: Inbound Extraction | Extract parent trace context inside worker callback across load-balanced worker instances (`SpanKindConsumer`). |
| **JetStream Sync Publish** | JetStream | Operation 1: Outbound Injection | Inject W3C trace context into headers; create producer span (`SpanKindProducer`) and close upon receiving `PubAck`. |
| **JetStream Async Publish** | JetStream | Operation 1: Outbound Injection | Inject headers into message; resolve producer span (`SpanKindProducer`) upon async `PubAckFuture` resolution. |
| **JetStream Push / Pull Handler** | JetStream | Operation 2: Inbound Extraction | Extract headers in consumer callback; create consumer span (`SpanKindConsumer`) and end span after `msg.Ack()`. |
| **JetStream Batch Fetch** | JetStream | Operation 2: Inbound Extraction | Iterate fetched message batch; extract trace context and manage child consumer span (`SpanKindConsumer`) per message item. |
| **JetStream Single-Message Iterator** | JetStream | Operation 2: Inbound Extraction | Extract trace context from retrieved message; create consumer span (`SpanKindConsumer`) and acknowledge message. |
| **Ack / Nak / Term Redelivery** | JetStream | Consumer Span Enrichment | Record delivery count, redelivery backoff delays (`NakWithDelay`), and poison message terminations (`Term`) as span attributes and errors. |
