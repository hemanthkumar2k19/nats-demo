package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
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

func getStatusWeight(status string) int {
	switch status {
	case "PUBLISHED", "STORED":
		return 1
	case "RECEIVED", "DELIVERED":
		return 2
	case "PROCESSING":
		return 3
	case "COMPLETED", "FAILED", "ACKED", "NO CONSUMER":
		return 4
	default:
		return 0
	}
}

// GetActivities returns flat activity logs for the dashboard.
func (s *Service) GetActivities() []Activity {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	activitiesCopy := make([]Activity, len(s.store.activities))
	copy(activitiesCopy, s.store.activities)

	sort.SliceStable(activitiesCopy, func(i, j int) bool {
		// 1. Sort by timestamp descending
		if activitiesCopy[i].Timestamp != activitiesCopy[j].Timestamp {
			return activitiesCopy[i].Timestamp > activitiesCopy[j].Timestamp
		}
		// 2. Sort by JobID ascending to group same job events
		if activitiesCopy[i].JobID != activitiesCopy[j].JobID {
			return activitiesCopy[i].JobID < activitiesCopy[j].JobID
		}
		// 3. Sort by status weight descending
		return getStatusWeight(activitiesCopy[i].Event) > getStatusWeight(activitiesCopy[j].Event)
	})

	return activitiesCopy
}

// ProcessLifecycleEvent processes an incoming NATS lifecycle message and updates the store.
func (s *Service) ProcessLifecycleEvent(subject string, data []byte, correlationID string, source string) error {
	var payload struct {
		JobID         string `json:"job_id"`
		Status        string `json:"status"`
		DeliveryCount int    `json:"delivery_count"`
		Error         string `json:"error,omitempty"`
		DeliveryMode  string `json:"delivery_mode,omitempty"`
		Sequence      uint64 `json:"sequence,omitempty"`
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
		status = "PUBLISHED"
	case "jobs.stored":
		status = "STORED"
	case "jobs.received":
		status = "RECEIVED"
	case "jobs.delivered":
		status = "DELIVERED"
	case "jobs.completed", "jobs.processing.completed":
		status = "COMPLETED"
	case "jobs.failed", "jobs.processing.failed":
		status = "FAILED"
	case "jobs.acked":
		status = "ACKED"
	case "jobs.noconsumer":
		status = "NO CONSUMER"
	case "jobs.processing.started", "jobs.processing":
		status = "PROCESSING"
	default:
		status = payload.Status
	}

	if payload.DeliveryCount <= 0 {
		payload.DeliveryCount = 1
	}

	s.store.AddEvent(payload.JobID, status, payload.DeliveryCount, correlationID, subject, source, payload.DeliveryMode, payload.Sequence)
	return nil
}

