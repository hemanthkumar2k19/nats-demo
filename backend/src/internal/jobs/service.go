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
	RequestJobValidation(job Job, correlationID string) (*JobValidationResponse, error)
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
// correlationID is propagated into the NATS request so the full interaction can
// be traced in the activity log.
func (s *Service) ValidateJob(job Job, correlationID string) (*JobValidationResponse, error) {
	// Record that demo-service is about to send the request.
	s.store.AddEvent(job.JobID, "REQUEST_SENT", 1, correlationID, "jobs.validate", "demo-service", "", 0, job.JobID, job.Type)

	resp, err := s.publisher.RequestJobValidation(job, correlationID)
	if err != nil {
		// Record timeout so it appears in the activity log.
		s.store.AddEvent(job.JobID, "REQUEST_TIMEOUT", 1, correlationID, "jobs.validate", "demo-service", "", 0, job.JobID, job.Type)
		return nil, err
	}

	// Record that demo-service received the reply.
	s.store.AddEvent(job.JobID, "REPLY_RECEIVED", 1, correlationID, "jobs.validate", "demo-service", "", 0, job.JobID, job.Type)
	return resp, nil
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
	case "PUBLISHED", "STORED", "REQUEST_SENT", "DEDUPLICATED":
		return 1
	case "RECEIVED", "DELIVERED", "REQUEST_RECEIVED", "REDELIVERED":
		return 2
	case "PROCESSING", "REPLY_SENT":
		return 3
	case "COMPLETED", "FAILED", "ACKED", "NO CONSUMER", "REPLY_RECEIVED", "REQUEST_TIMEOUT":
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
func (s *Service) ProcessLifecycleEvent(subject string, data []byte, correlationID string, source string, msgID string) error {
	var payload struct {
		JobID         string `json:"job_id"`
		Type          string `json:"type,omitempty"`
		Status        string `json:"status"`
		DeliveryCount int    `json:"delivery_count"`
		Error         string `json:"error,omitempty"`
		DeliveryMode  string `json:"delivery_mode,omitempty"`
		Sequence      uint64 `json:"sequence,omitempty"`
		CorrelationID string `json:"correlation_id,omitempty"`
		MsgID         string `json:"msg_id,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal lifecycle event payload: %w", err)
	}

	if payload.JobID == "" {
		return fmt.Errorf("lifecycle event payload missing job_id")
	}

	if correlationID == "" && payload.CorrelationID != "" {
		correlationID = payload.CorrelationID
	}
	if msgID == "" && payload.MsgID != "" {
		msgID = payload.MsgID
	}
	if msgID == "" {
		msgID = payload.JobID
	}

	var status string
	switch subject {
	case "jobs.validate":
		// demo-service's jobs.> wildcard subscription catches its own outgoing
		// RequestMsg to jobs.validate. That message is a Job payload, not a
		// lifecycle event. Silently discard it to avoid a blank activity row.
		return nil
	case "jobs.submitted":
		status = "PUBLISHED"
	case "jobs.stored":
		status = "STORED"
	case "jobs.deduplicated":
		status = "DEDUPLICATED"
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
	case "jobs.request.received":
		status = "REQUEST_RECEIVED"
	case "jobs.reply.sent":
		status = "REPLY_SENT"
	default:
		status = payload.Status
	}

	if payload.DeliveryCount <= 0 {
		payload.DeliveryCount = 1
	}

	s.store.AddEvent(payload.JobID, status, payload.DeliveryCount, correlationID, subject, source, payload.DeliveryMode, payload.Sequence, msgID, payload.Type)
	return nil
}

