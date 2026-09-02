package jobs

import (
	"context"
	"errors"
	"fmt"
)

// Publisher interface abstracts the messaging layer to decouple the domain layer from NATS.
type Publisher interface {
	PublishJobSubmitted(ctx context.Context, job Job) error
	RequestJobValidation(ctx context.Context, job Job) (*JobValidationResponse, error)
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
func (s *Service) SubmitJob(ctx context.Context, job Job) (*JobStatusResponse, error) {
	if job.JobID == "" {
		return nil, errors.New("job_id is required")
	}
	if job.Type == "" {
		return nil, errors.New("type is required")
	}

	// Publish to NATS
	if err := s.publisher.PublishJobSubmitted(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to publish job: %w", err)
	}

	s.store.AddJob(job, "SUBMITTED")

	return &JobStatusResponse{
		JobID:   job.JobID,
		Status:  "SUBMITTED",
		TraceID: job.TraceID,
	}, nil
}

// ValidateJob validates a job payload by sending a Request/Reply call over NATS.
func (s *Service) ValidateJob(ctx context.Context, job Job) (*JobValidationResponse, error) {
	if job.TraceID != "" {
		s.store.SetTraceID(job.JobID, job.TraceID)
	}

	resp, err := s.publisher.RequestJobValidation(ctx, job)
	if err != nil {
		s.store.AddJob(job, "VALIDATION_FAILED")
		return nil, err
	}

	status := "INVALID"
	if resp.Valid {
		status = "VALID"
	}
	s.store.AddJob(job, status)

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
