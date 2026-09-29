package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

const TracerName = "nats-demo"

// Tracer returns the unified Tracer instance for the application.
func Tracer() trace.Tracer {
	return otel.Tracer(TracerName)
}

// InitTracer configures OpenTelemetry TracerProvider with OTLP (gRPC/HTTP) or stdout exporter.
func InitTracer(serviceName, exporterType, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()
	var exporter sdktrace.SpanExporter
	var err error

	switch exporterType {
	case "otlp-http":
		opts := []otlptracehttp.Option{}
		if otlpEndpoint != "" {
			opts = append(opts, otlptracehttp.WithEndpoint(otlpEndpoint), otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
	case "otlp-grpc":
		opts := []otlptracegrpc.Option{}
		if otlpEndpoint != "" {
			opts = append(opts, otlptracegrpc.WithEndpoint(otlpEndpoint), otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		if otlpEndpoint != "" {
			exporter, err = otlptracegrpc.New(ctx,
				otlptracegrpc.WithEndpoint(otlpEndpoint),
				otlptracegrpc.WithInsecure(),
			)
		} else {
			exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	// Register W3C Trace Context and Baggage propagators globally
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// ShutdownTracer flushes pending spans and shuts down the tracer provider.
func ShutdownTracer(ctx context.Context, tp *sdktrace.TracerProvider) {
	if tp != nil {
		_ = tp.Shutdown(ctx)
	}
}
