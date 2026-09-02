package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Publisher manages NATS message publishing operations.
type Publisher struct {
	client *natsclient.Client
}

// NewPublisher initializes a new Publisher.
func NewPublisher(client *natsclient.Client) *Publisher {
	return &Publisher{client: client}
}

// PublishJobSubmitted marshals the job to JSON and publishes it to the jobs.submitted subject using the selected delivery mode.
func (p *Publisher) PublishJobSubmitted(ctx context.Context, job jobs.Job, correlationID string) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	pubCtx, pubSpan := telemetry.StartSpan(ctx, "NATS Publish "+SubjectJobSubmitted,
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation", "publish"),
			attribute.String("messaging.destination.name", SubjectJobSubmitted),
			attribute.String("delivery.mode", job.DeliveryMode),
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
		),
	)
	defer pubSpan.End()

	msg := nats.NewMsg(SubjectJobSubmitted)
	msg.Header.Set("Content-Type", "application/json")
	msg.Header.Set("Nats-Msg-Id", job.JobID)
	msg.Header.Set("X-Message-Id", fmt.Sprintf("msg-sub-%s-%d", job.JobID, time.Now().UnixNano()))
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	msg.Header.Set("X-Source", "job-service")
	msg.Header.Set("X-Delivery-Mode", job.DeliveryMode)
	msg.Data = payload

	// Inject W3C traceparent into NATS message headers
	telemetry.InjectTraceContext(pubCtx, msg.Header)

	if job.DeliveryMode == "JETSTREAM" {
		pubSpan.SetAttributes(attribute.String("jetstream.stream", "JOBS"))
		js, err := p.client.Conn.JetStream()
		if err != nil {
			pubSpan.RecordError(err)
			pubSpan.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to get JetStream context: %w", err)
		}
		ack, err := js.PublishMsg(msg)
		if err != nil {
			pubSpan.RecordError(err)
			pubSpan.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to publish to JetStream: %w", err)
		}
		pubSpan.SetAttributes(attribute.Int64("jetstream.sequence", int64(ack.Sequence)))
		if ack.Duplicate {
			pubSpan.SetAttributes(attribute.Bool("jetstream.duplicate", true))
			// Duplicate publish recognized by JetStream deduplication window
			_ = p.PublishJobLifecycle(SubjectJobDeduplicated, job.JobID, "DEDUPLICATED", 1, "Duplicate message recognized by JetStream deduplication window", correlationID, "job-service", job.DeliveryMode, ack.Sequence)
		} else {
			// Stored successfully: publish jobs.stored event
			_ = p.PublishJobLifecycle(SubjectJobStored, job.JobID, "STORED", 1, "", correlationID, "job-service", job.DeliveryMode, ack.Sequence)
		}
	} else {
		if err := p.client.Conn.PublishMsg(msg); err != nil {
			pubSpan.RecordError(err)
			pubSpan.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to publish to NATS: %w", err)
		}
	}

	pubSpan.SetStatus(codes.Ok, "published")
	return nil
}

// ErrRequestTimeout is returned when a NATS Request/Reply call times out or
// finds no active responder. The HTTP handler uses this to return 504.
var ErrRequestTimeout = fmt.Errorf("request timed out: no response from processor service")

// RequestJobValidation sends a sync validation request via NATS Request/Reply.
// correlationID is forwarded as X-Correlation-Id so the interaction can be
// traced across the job-service and processor-service logs.
func (p *Publisher) RequestJobValidation(ctx context.Context, job jobs.Job, correlationID string) (*jobs.JobValidationResponse, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation payload: %w", err)
	}

	reqCtx, reqSpan := telemetry.StartSpan(ctx, "NATS Request "+SubjectJobValidate,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation", "request"),
			attribute.String("messaging.destination.name", SubjectJobValidate),
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
		),
	)
	defer reqSpan.End()

	msg := nats.NewMsg(SubjectJobValidate)
	msg.Header.Set("Content-Type", "application/json")
	msg.Header.Set("X-Message-Id", fmt.Sprintf("msg-val-%s-%d", job.JobID, time.Now().UnixNano()))
	msg.Header.Set("X-Source", "job-service")
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	msg.Data = payload

	// Inject W3C traceparent into NATS message headers
	telemetry.InjectTraceContext(reqCtx, msg.Header)

	// Publish jobs.request.sent so observers can record the outgoing request
	reqEvtData, _ := json.Marshal(map[string]interface{}{
		"job_id":         job.JobID,
		"type":           job.Type,
		"status":         "REQUEST_SENT",
		"delivery_count": 1,
		"correlation_id": correlationID,
	})
	reqEvtMsg := nats.NewMsg(SubjectJobRequestSent)
	reqEvtMsg.Data = reqEvtData
	reqEvtMsg.Header.Set("X-Source", "job-service")
	reqEvtMsg.Header.Set("X-Correlation-Id", correlationID)
	_ = p.client.Conn.PublishMsg(reqEvtMsg)

	reply, err := p.client.Conn.RequestMsg(msg, 2*time.Second)
	if err != nil {
		reqSpan.RecordError(err)
		reqSpan.SetStatus(codes.Error, err.Error())

		timeoutEvtData, _ := json.Marshal(map[string]interface{}{
			"job_id":         job.JobID,
			"type":           job.Type,
			"status":         "REQUEST_TIMEOUT",
			"delivery_count": 1,
			"correlation_id": correlationID,
		})
		timeoutEvtMsg := nats.NewMsg(SubjectJobRequestTimeout)
		timeoutEvtMsg.Data = timeoutEvtData
		timeoutEvtMsg.Header.Set("X-Source", "job-service")
		timeoutEvtMsg.Header.Set("X-Correlation-Id", correlationID)
		_ = p.client.Conn.PublishMsg(timeoutEvtMsg)

		// Surface timeout and no-responder as a typed sentinel so the caller
		// can distinguish them from internal errors.
		if err == nats.ErrTimeout || err == nats.ErrNoResponders {
			return nil, ErrRequestTimeout
		}
		return nil, fmt.Errorf("NATS request to %s failed: %w", SubjectJobValidate, err)
	}

	var resp jobs.JobValidationResponse
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		reqSpan.RecordError(err)
		reqSpan.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("failed to unmarshal validation reply: %w", err)
	}

	replyEvtData, _ := json.Marshal(map[string]interface{}{
		"job_id":         job.JobID,
		"type":           job.Type,
		"status":         "REPLY_RECEIVED",
		"delivery_count": 1,
		"correlation_id": correlationID,
	})
	replyEvtMsg := nats.NewMsg(SubjectJobReplyReceived)
	replyEvtMsg.Data = replyEvtData
	replyEvtMsg.Header.Set("X-Source", "job-service")
	replyEvtMsg.Header.Set("X-Correlation-Id", correlationID)
	_ = p.client.Conn.PublishMsg(replyEvtMsg)

	reqSpan.SetStatus(codes.Ok, "validated")
	return &resp, nil
}

// JobLifecycleEvent represents the payload for tracking job status changes.
type JobLifecycleEvent struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	DeliveryCount int    `json:"delivery_count"`
	Error         string `json:"error,omitempty"`
	DeliveryMode  string `json:"delivery_mode,omitempty"`
	Sequence      uint64 `json:"sequence,omitempty"`
}

// PublishJobLifecycle publishes a job lifecycle transition event.
func (p *Publisher) PublishJobLifecycle(subject string, jobID string, status string, deliveryCount int, errMsg string, correlationID string, workerName string, deliveryMode string, sequence uint64) error {
	event := JobLifecycleEvent{
		JobID:         jobID,
		Status:        status,
		DeliveryCount: deliveryCount,
		Error:         errMsg,
		DeliveryMode:  deliveryMode,
		Sequence:      sequence,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal lifecycle event: %w", err)
	}

	msg := nats.NewMsg(subject)
	msg.Header.Set("Content-Type", "application/json")
	msg.Header.Set("Nats-Msg-Id", jobID)
	msg.Header.Set("X-Message-Id", fmt.Sprintf("msg-lf-%s-%s-%d", jobID, status, time.Now().UnixNano()))
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	msg.Header.Set("X-Source", workerName)
	msg.Data = payload

	if err := p.client.Conn.PublishMsg(msg); err != nil {
		return fmt.Errorf("failed to publish lifecycle event to NATS: %w", err)
	}

	return nil
}

