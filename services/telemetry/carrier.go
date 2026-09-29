package telemetry

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// HeaderCarrier adapts nats.Header to satisfy propagation.TextMapCarrier interface.
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

var _ propagation.TextMapCarrier = HeaderCarrier{}

// InstrumentOutboundMessage creates a producer span and injects W3C trace context into nats.Msg headers.
func InstrumentOutboundMessage(ctx context.Context, spanName string, msg *nats.Msg) (context.Context, trace.Span) {
	tr := Tracer()
	ctx, span := tr.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindProducer))

	if msg.Header == nil {
		msg.Header = make(nats.Header)
	}
	otel.GetTextMapPropagator().Inject(ctx, HeaderCarrier(msg.Header))

	return ctx, span
}

// InstrumentInboundMessage extracts W3C trace context from incoming headers and starts a consumer span.
func InstrumentInboundMessage(parentCtx context.Context, spanName string, header nats.Header) (context.Context, trace.Span) {
	tr := Tracer()

	if header != nil {
		parentCtx = otel.GetTextMapPropagator().Extract(parentCtx, HeaderCarrier(header))
	}

	return tr.Start(parentCtx, spanName, trace.WithSpanKind(trace.SpanKindConsumer))
}
