package telemetry

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	meter metric.Meter

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

// Init initializes the OpenTelemetry MeterProvider with an OTLP gRPC exporter.
// It exports metrics periodically (every 2 seconds) for a responsive demo experience.
func Init(ctx context.Context, serviceName, endpoint string, isInsecure bool) (func(context.Context) error, error) {
	var opts []otlpmetricgrpc.Option
	opts = append(opts, otlpmetricgrpc.WithEndpoint(endpoint))

	if isInsecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
		opts = append(opts, otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())))
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		log.Printf("[telemetry] Warning: failed to create OTLP metric exporter: %v. Continuing without OTLP.", err)
		initNoopInstruments()
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", "demo"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create otel resource: %w", err)
	}

	// 2-second reader interval for responsive live Grafana demo updates
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(2*time.Second))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	otel.SetMeterProvider(mp)
	meter = mp.Meter(serviceName)

	if err := registerInstruments(); err != nil {
		return nil, fmt.Errorf("failed to register metric instruments: %w", err)
	}

	log.Printf("[telemetry] OpenTelemetry metrics initialized for %s -> %s", serviceName, endpoint)

	return mp.Shutdown, nil
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
