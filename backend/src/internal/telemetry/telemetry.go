package telemetry

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	meter  metric.Meter
	tracer trace.Tracer

	// Demo Service Instruments
	jobsSubmittedCounter          metric.Int64Counter
	jobValidationRequestsCounter  metric.Int64Counter
	natsPublishCounter            metric.Int64Counter
	natsRequestCounter            metric.Int64Counter
	natsRequestErrorsCounter      metric.Int64Counter
	jobSubmissionDurationHist     metric.Float64Histogram
	natsRequestDurationHist       metric.Float64Histogram

	// Processor Service Instruments
	jobsProcessedCounter          metric.Int64Counter
	jobsFailedCounter             metric.Int64Counter
	natsMessagesReceivedCounter   metric.Int64Counter
	natsMessagesAckedCounter      metric.Int64Counter
	natsMessagesRedeliveredCounter metric.Int64Counter
	jobProcessingDurationHist     metric.Float64Histogram
)

// Init initializes OpenTelemetry MeterProvider and TracerProvider with OTLP gRPC export.
func Init(ctx context.Context, serviceName, endpoint string, isInsecure bool) (func(context.Context) error, error) {
	var metricOpts []otlpmetricgrpc.Option
	metricOpts = append(metricOpts, otlpmetricgrpc.WithEndpoint(endpoint))

	var traceOpts []otlptracegrpc.Option
	traceOpts = append(traceOpts, otlptracegrpc.WithEndpoint(endpoint))

	if isInsecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))

		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		traceOpts = append(traceOpts, otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			attribute.String("deployment.environment", "demo"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	// 1. Initialize Tracing Provider
	var tp *sdktrace.TracerProvider
	traceExporter, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		log.Printf("[telemetry] Warning: failed to create OTLP trace exporter: %v. Continuing with no-op tracer.", err)
		tracer = otel.GetTracerProvider().Tracer(serviceName)
	} else {
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(traceExporter),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)
		otel.SetTracerProvider(tp)
		tracer = tp.Tracer(serviceName)
	}

	// Register global W3C text map propagator for context injection across NATS
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 2. Initialize Metrics Provider
	var mp *sdkmetric.MeterProvider
	metricExporter, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		log.Printf("[telemetry] Warning: failed to create OTLP metric exporter: %v. Continuing with no-op metrics.", err)
		initNoopInstruments()
	} else {
		reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(2*time.Second))
		mp = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(reader),
		)
		otel.SetMeterProvider(mp)
		meter = mp.Meter(serviceName)

		if err := registerInstruments(); err != nil {
			return nil, fmt.Errorf("failed to register metric instruments: %w", err)
		}
	}

	log.Printf("[telemetry] OpenTelemetry (Metrics & Traces) initialized for %s -> %s", serviceName, endpoint)

	shutdown := func(shutdownCtx context.Context) error {
		var firstErr error
		if tp != nil {
			if err := tp.Shutdown(shutdownCtx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if mp != nil {
			if err := mp.Shutdown(shutdownCtx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	return shutdown, nil
}

// Tracer returns the global tracer.
func Tracer() trace.Tracer {
	if tracer == nil {
		return otel.GetTracerProvider().Tracer("nats-demo")
	}
	return tracer
}

// StartSpan starts a new span using the global tracer.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// NatsHeaderCarrier implements propagation.TextMapCarrier for nats.Header.
type NatsHeaderCarrier nats.Header

func (c NatsHeaderCarrier) Get(key string) string {
	for k, v := range c {
		if strings.EqualFold(k, key) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

func (c NatsHeaderCarrier) Set(key, val string) {
	c[key] = []string{val}
}

func (c NatsHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceContext injects W3C traceparent headers into a nats.Header.
func InjectTraceContext(ctx context.Context, header nats.Header) {
	if header == nil {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, NatsHeaderCarrier(header))
}

// ExtractTraceContext extracts W3C trace context from a nats.Header.
func ExtractTraceContext(ctx context.Context, header nats.Header) context.Context {
	if header == nil {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, NatsHeaderCarrier(header))
}

func initNoopInstruments() {
	meter = otel.GetMeterProvider().Meter("noop")
	_ = registerInstruments()
}

func registerInstruments() error {
	var err error

	// Demo Service Instruments
	jobsSubmittedCounter, err = meter.Int64Counter("jobs_submitted_total",
		metric.WithDescription("Total number of jobs submitted"),
		metric.WithUnit("{job}"))
	if err != nil {
		return err
	}

	jobValidationRequestsCounter, err = meter.Int64Counter("job_validation_requests_total",
		metric.WithDescription("Total validation requests sent"),
		metric.WithUnit("{request}"))
	if err != nil {
		return err
	}

	natsPublishCounter, err = meter.Int64Counter("nats_publish_total",
		metric.WithDescription("Total messages published to NATS"),
		metric.WithUnit("{message}"))
	if err != nil {
		return err
	}

	natsRequestCounter, err = meter.Int64Counter("nats_request_total",
		metric.WithDescription("Total request/reply messages sent"),
		metric.WithUnit("{request}"))
	if err != nil {
		return err
	}

	natsRequestErrorsCounter, err = meter.Int64Counter("nats_request_errors_total",
		metric.WithDescription("Total request/reply errors or timeouts"),
		metric.WithUnit("{error}"))
	if err != nil {
		return err
	}

	jobSubmissionDurationHist, err = meter.Float64Histogram("job_submission_duration",
		metric.WithDescription("Duration of job submission processing in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}

	natsRequestDurationHist, err = meter.Float64Histogram("nats_request_duration",
		metric.WithDescription("Round-trip duration of NATS request/reply in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}

	// Processor Service Instruments
	jobsProcessedCounter, err = meter.Int64Counter("jobs_processed_total",
		metric.WithDescription("Total jobs successfully processed by workers"),
		metric.WithUnit("{job}"))
	if err != nil {
		return err
	}

	jobsFailedCounter, err = meter.Int64Counter("jobs_failed_total",
		metric.WithDescription("Total jobs that failed during processing"),
		metric.WithUnit("{job}"))
	if err != nil {
		return err
	}

	natsMessagesReceivedCounter, err = meter.Int64Counter("nats_messages_received_total",
		metric.WithDescription("Total messages received by processor workers"),
		metric.WithUnit("{message}"))
	if err != nil {
		return err
	}

	natsMessagesAckedCounter, err = meter.Int64Counter("nats_messages_acked_total",
		metric.WithDescription("Total JetStream messages explicitly acknowledged"),
		metric.WithUnit("{message}"))
	if err != nil {
		return err
	}

	natsMessagesRedeliveredCounter, err = meter.Int64Counter("nats_messages_redelivered_total",
		metric.WithDescription("Total messages redelivered to processor workers"),
		metric.WithUnit("{message}"))
	if err != nil {
		return err
	}

	jobProcessingDurationHist, err = meter.Float64Histogram("job_processing_duration",
		metric.WithDescription("Execution duration of worker job processing in seconds"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}

	return nil
}

// Demo Service Recorders

func RecordJobSubmitted(ctx context.Context, deliveryMode, jobType string, duration time.Duration) {
	if jobsSubmittedCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("type", jobType),
		}
		jobsSubmittedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		if jobSubmissionDurationHist != nil {
			jobSubmissionDurationHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
	}
}

func RecordValidationRequest(ctx context.Context, result string, duration time.Duration) {
	if jobValidationRequestsCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("result", result),
		}
		jobValidationRequestsCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		if natsRequestDurationHist != nil {
			natsRequestDurationHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
	}
}

func RecordNatsPublish(ctx context.Context, deliveryMode, subject string) {
	if natsPublishCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("subject", subject),
		}
		natsPublishCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func RecordNatsRequest(ctx context.Context, subject string, duration time.Duration, err error) {
	if natsRequestCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("subject", subject),
		}
		natsRequestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		if natsRequestDurationHist != nil {
			natsRequestDurationHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
		if err != nil && natsRequestErrorsCounter != nil {
			errAttrs := append(attrs, attribute.String("error", err.Error()))
			natsRequestErrorsCounter.Add(ctx, 1, metric.WithAttributes(errAttrs...))
		}
	}
}

// Processor Service Recorders

func RecordMessageReceived(ctx context.Context, deliveryMode, worker, subject string) {
	if natsMessagesReceivedCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("worker", worker),
			attribute.String("subject", subject),
		}
		natsMessagesReceivedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func RecordJobProcessed(ctx context.Context, deliveryMode, worker, status string, duration time.Duration) {
	if jobsProcessedCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("worker", worker),
			attribute.String("status", status),
		}
		jobsProcessedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		if jobProcessingDurationHist != nil {
			jobProcessingDurationHist.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
		}
	}
}

func RecordJobFailed(ctx context.Context, deliveryMode, worker string) {
	if jobsFailedCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("worker", worker),
		}
		jobsFailedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func RecordMessageAcked(ctx context.Context, deliveryMode, worker string) {
	if natsMessagesAckedCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("worker", worker),
		}
		natsMessagesAckedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func RecordMessageRedelivered(ctx context.Context, deliveryMode, worker string) {
	if natsMessagesRedeliveredCounter != nil {
		attrs := []attribute.KeyValue{
			attribute.String("delivery_mode", deliveryMode),
			attribute.String("worker", worker),
		}
		natsMessagesRedeliveredCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}
