package jobs

import "time"

// Job represents a work item submitted to the platform.
type Job struct {
	JobID   string                 `json:"job_id"`
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

// JobStatusResponse represents the HTTP response returned on job submission or status query.
type JobStatusResponse struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// JobValidationResponse represents the sync response returned by validation.
type JobValidationResponse struct {
	Valid   bool   `json:"valid"`
	Message string `json:"message"`
}

// JobHistoryItem tracks status changes with a timestamp.
type JobHistoryItem struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// JobDetailResponse contains comprehensive details about a job.
type JobDetailResponse struct {
	JobID         string           `json:"job_id"`
	Status        string           `json:"status"`
	DeliveryCount int              `json:"delivery_count"`
	CorrelationID string           `json:"correlation_id"`
	History       []JobHistoryItem `json:"history"`
}

