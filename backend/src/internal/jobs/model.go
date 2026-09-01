package jobs

import (
	"errors"
	"time"
)

// ErrProcessorDisabled is returned by the validation handler when processing
// is disabled. consumer.SubscribeJobValidate treats this as a signal to skip
// msg.Respond(), letting the NATS requester time out naturally.
var ErrProcessorDisabled = errors.New("processor is disabled")


// Job represents a work item submitted to the platform.
type Job struct {
	JobID        string                 `json:"job_id"`
	Type         string                 `json:"type"`
	Payload      map[string]interface{} `json:"payload"`
	DeliveryMode string                 `json:"delivery_mode,omitempty"`
	TraceID      string                 `json:"trace_id,omitempty"`
}

// JobStatusResponse represents the HTTP response returned on job submission or status query.
type JobStatusResponse struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

// JobValidationResponse represents the sync response returned by validation.
type JobValidationResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
	TraceID string `json:"trace_id,omitempty"`
}

// JobHistoryItem tracks status changes with a timestamp.
type JobHistoryItem struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// JobDetailResponse contains comprehensive details about a job.
type JobDetailResponse struct {
	JobID         string           `json:"job_id"`
	Type          string           `json:"type,omitempty"`
	Status        string           `json:"status"`
	DeliveryCount int              `json:"delivery_count"`
	CorrelationID string           `json:"correlation_id"`
	TraceID       string           `json:"trace_id,omitempty"`
	History       []JobHistoryItem `json:"history"`
}

// ConsumerConfig represents the consumer configuration parameters for Consumer Lab.
type ConsumerConfig struct {
	Type     string `json:"type"`     // "durable" | "ephemeral"
	Workers  int    `json:"workers"`  // Number of worker instances: 1 or 2
	Ordering string `json:"ordering"` // "normal" | "ordered"
}

// ConsumerStatusResponse represents consumer state and metrics for GET /consumer.
type ConsumerStatusResponse struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Workers     int    `json:"workers"`
	Ordering    string `json:"ordering"`
	Delivery    string `json:"delivery"`
	Status      string `json:"status"`
	Pending     uint64 `json:"pending"`
	AckPending  int    `json:"ack_pending"`
	Redelivered int    `json:"redelivered"`
}

