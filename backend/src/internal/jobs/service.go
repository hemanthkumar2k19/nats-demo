package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Publisher interface abstracts the messaging layer to decouple the domain layer from NATS.
type Publisher interface {
	PublishJobSubmitted(job Job, correlationID string) error
	RequestJobValidation(job Job) (*JobValidationResponse, error)
}

// Service manages the business workflows for jobs.
type Service struct {
	publisher Publisher
	store     *JobStore
}

// NewService instantiates a new jobs Service.
func NewService(publisher Publisher) *Service {
	return &Service{
		publisher: publisher,
		store:     NewJobStore(),
	}
}

// SubmitJob processes and submits a job to the publisher.
func (s *Service) SubmitJob(job Job, correlationID string) (*JobStatusResponse, error) {
	if job.JobID == "" {
		return nil, errors.New("job_id is required")
	}
	if job.Type == "" {
		return nil, errors.New("type is required")
	}

	// Publish to NATS
	if err := s.publisher.PublishJobSubmitted(job, correlationID); err != nil {
		return nil, fmt.Errorf("failed to publish job: %w", err)
	}

	return &JobStatusResponse{
		JobID:         job.JobID,
		Status:        "SUBMITTED",
		CorrelationID: correlationID,
	}, nil
}

// ValidateJob validates a job payload by sending a Request/Reply call over NATS.
func (s *Service) ValidateJob(job Job) (*JobValidationResponse, error) {
	return s.publisher.RequestJobValidation(job)
}

// ListJobs returns the list of all tracked jobs.
func (s *Service) ListJobs() []*JobDetailResponse {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	list := make([]*JobDetailResponse, 0, len(s.store.jobs))
	for _, job := range s.store.jobs {
		list = append(list, job)
	}
	return list
}

// GetJob returns detailed status of a specific job.
func (s *Service) GetJob(jobID string) (*JobDetailResponse, bool) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	job, exists := s.store.jobs[jobID]
	return job, exists
}

// GetActivities returns flat activity logs for the dashboard.
func (s *Service) GetActivities() []Activity {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	activitiesCopy := make([]Activity, len(s.store.activities))
	copy(activitiesCopy, s.store.activities)
	return activitiesCopy
}

// ProcessLifecycleEvent processes an incoming NATS lifecycle message and updates the store.
func (s *Service) ProcessLifecycleEvent(subject string, data []byte, correlationID string, source string) error {
	var payload struct {
		JobID         string `json:"job_id"`
		Status        string `json:"status"`
		DeliveryCount int    `json:"delivery_count"`
		Error         string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal lifecycle event payload: %w", err)
	}

	if payload.JobID == "" {
		return fmt.Errorf("lifecycle event payload missing job_id")
	}

	var status string
	switch subject {
	case "jobs.submitted":
		status = "SUBMITTED"
	case "jobs.processing":
		status = "PROCESSING"
	case "jobs.completed":
		status = "COMPLETED"
	case "jobs.failed":
		status = "FAILED"
	default:
		status = payload.Status
	}

	if payload.DeliveryCount <= 0 {
		payload.DeliveryCount = 1
	}

	s.store.AddEvent(payload.JobID, status, payload.DeliveryCount, correlationID, subject, source)
	return nil
}

