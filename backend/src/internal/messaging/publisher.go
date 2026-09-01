package messaging

import (
	"encoding/json"
	"fmt"
	"time"
	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"

	"github.com/nats-io/nats.go"
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
func (p *Publisher) PublishJobSubmitted(job jobs.Job, correlationID string) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal job payload: %w", err)
	}

	msg := nats.NewMsg(SubjectJobSubmitted)
	msg.Header.Set("Content-Type", "application/json")
	msg.Header.Set("Nats-Msg-Id", job.JobID)
	msg.Header.Set("X-Message-Id", fmt.Sprintf("msg-sub-%s-%d", job.JobID, time.Now().UnixNano()))
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	msg.Header.Set("X-Source", "demo-service")
	msg.Header.Set("X-Delivery-Mode", job.DeliveryMode)
	msg.Data = payload

	if job.DeliveryMode == "JETSTREAM" {
		js, err := p.client.Conn.JetStream()
		if err != nil {
			return fmt.Errorf("failed to get JetStream context: %w", err)
		}
		ack, err := js.PublishMsg(msg)
		if err != nil {
			return fmt.Errorf("failed to publish to JetStream: %w", err)
		}
		if ack.Duplicate {
			// Duplicate publish recognized by JetStream deduplication window
			_ = p.PublishJobLifecycle(SubjectJobDeduplicated, job.JobID, "DEDUPLICATED", 1, "Duplicate message recognized by JetStream deduplication window", correlationID, "demo-service", job.DeliveryMode, ack.Sequence)
		} else {
			// Stored successfully: publish jobs.stored event
			_ = p.PublishJobLifecycle(SubjectJobStored, job.JobID, "STORED", 1, "", correlationID, "demo-service", job.DeliveryMode, ack.Sequence)
		}
	} else {
		if err := p.client.Conn.PublishMsg(msg); err != nil {
			return fmt.Errorf("failed to publish to NATS: %w", err)
		}
	}

	return nil
}

// ErrRequestTimeout is returned when a NATS Request/Reply call times out or
// finds no active responder. The HTTP handler uses this to return 504.
var ErrRequestTimeout = fmt.Errorf("request timed out: no response from processor service")

// RequestJobValidation sends a sync validation request via NATS Request/Reply.
// correlationID is forwarded as X-Correlation-Id so the interaction can be
// traced across the demo-service and processor-service logs.
func (p *Publisher) RequestJobValidation(job jobs.Job, correlationID string) (*jobs.JobValidationResponse, error) {
	payload, err := json.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation payload: %w", err)
	}

	msg := nats.NewMsg(SubjectJobValidate)
	msg.Header.Set("Content-Type", "application/json")
	msg.Header.Set("X-Message-Id", fmt.Sprintf("msg-val-%s-%d", job.JobID, time.Now().UnixNano()))
	msg.Header.Set("X-Source", "demo-service")
	if correlationID != "" {
		msg.Header.Set("X-Correlation-Id", correlationID)
	}
	msg.Data = payload

	reply, err := p.client.Conn.RequestMsg(msg, 2*time.Second)
	if err != nil {
		// Surface timeout and no-responder as a typed sentinel so the caller
		// can distinguish them from internal errors.
		if err == nats.ErrTimeout || err == nats.ErrNoResponders {
			return nil, ErrRequestTimeout
		}
		return nil, fmt.Errorf("NATS request to %s failed: %w", SubjectJobValidate, err)
	}

	var resp jobs.JobValidationResponse
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validation reply: %w", err)
	}

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

