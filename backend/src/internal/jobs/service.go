package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// Publisher interface abstracts the messaging layer to decouple the domain layer from NATS.
type Publisher interface {
	PublishJobSubmitted(ctx context.Context, job Job) error
	RequestJobValidation(ctx context.Context, job Job) (*JobValidationResponse, error)
	PublishJobQueue(ctx context.Context, job Job) error
	PublishJobLifecycle(subject string, jobID string, status string, deliveryCount int, errMsg string, workerName string, deliveryMode string, sequence uint64) error
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

// ScheduleJob registers a job with a scheduled delay before publishing it to NATS.
func (s *Service) ScheduleJob(ctx context.Context, req ScheduleJobRequest) (*ScheduleJobResponse, error) {
	if req.JobID == "" {
		return nil, errors.New("job_id is required")
	}
	if req.Type == "" {
		req.Type = "image-processing"
	}
	if req.DeliveryMode == "" {
		req.DeliveryMode = "JETSTREAM"
	}

	delay := time.Duration(req.DeliverAfterSec) * time.Second
	if req.DeliverAt != "" {
		if targetTime, err := time.Parse(time.RFC3339, req.DeliverAt); err == nil {
			computed := time.Until(targetTime)
			if computed > 0 {
				delay = computed
			}
		}
	}
	if delay <= 0 {
		delay = 5 * time.Second
	}
	delaySec := int(delay.Seconds())
	scheduledFor := time.Now().Add(delay).Format(time.RFC3339)

	job := Job{
		JobID:        req.JobID,
		Type:         req.Type,
		Payload:      req.Payload,
		DeliveryMode: req.DeliveryMode,
	}

	// 1. Record in store as SCHEDULED
	s.store.AddJob(job, "SCHEDULED")

	// 2. Publish SCHEDULED lifecycle event immediately so observability and activity log see it
	_ = s.publisher.PublishJobLifecycle(
		"jobs.scheduled",
		job.JobID,
		"SCHEDULED",
		1,
		fmt.Sprintf("Application scheduler will publish in %d seconds (at %s)", delaySec, scheduledFor),
		"job-service",
		req.DeliveryMode,
		0,
	)

	// 3. Launch background timer goroutine
	go func(j Job, waitDuration time.Duration) {
		time.Sleep(waitDuration)
		_, _ = s.SubmitJob(context.Background(), j)
	}(job, delay)

	return &ScheduleJobResponse{
		JobID:        job.JobID,
		Status:       "SCHEDULED",
		ScheduledFor: scheduledFor,
		DelaySeconds: delaySec,
		TraceID:      job.TraceID,
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

var queueJobSeq uint64

// SubmitQueueJobs publishes a batch of test jobs to jobs.queue for Core NATS Queue Group processing.
func (s *Service) SubmitQueueJobs(ctx context.Context, req QueuePublishRequest) (*QueuePublishResponse, error) {
	count := req.Count
	if count <= 0 {
		count = 10
	}
	jobType := req.Type
	if jobType == "" {
		jobType = "queue-task"
	}

	publishedIDs := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		seq := atomic.AddUint64(&queueJobSeq, 1)
		jobID := fmt.Sprintf("job-q-%d", 100+seq)
		job := Job{
			JobID:        jobID,
			Type:         jobType,
			DeliveryMode: "CORE",
			Payload: map[string]interface{}{
				"sequence":    seq,
				"batch_index": i,
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		}

		if err := s.publisher.PublishJobQueue(ctx, job); err != nil {
			return nil, fmt.Errorf("failed publishing queue job %s: %w", jobID, err)
		}

		s.store.AddJob(job, "PUBLISHED")
		publishedIDs = append(publishedIDs, jobID)
	}

	return &QueuePublishResponse{
		Published: len(publishedIDs),
		Subject:   "jobs.queue",
		Jobs:      publishedIDs,
	}, nil
}

// SubmitStreamJobs publishes a batch of test jobs directly to JetStream stream JOBS.
func (s *Service) SubmitStreamJobs(ctx context.Context, req QueuePublishRequest) (*QueuePublishResponse, error) {
	count := req.Count
	if count <= 0 {
		count = 10
	}
	jobType := req.Type
	if jobType == "" {
		jobType = "image-processing"
	}

	publishedIDs := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		seq := atomic.AddUint64(&queueJobSeq, 1)
		jobID := fmt.Sprintf("job-js-%d", 100+seq)
		job := Job{
			JobID:        jobID,
			Type:         jobType,
			DeliveryMode: "JETSTREAM",
			Payload: map[string]interface{}{
				"file":        fmt.Sprintf("img-%d.jpg", seq),
				"sequence":    seq,
				"batch_index": i,
				"timestamp":   time.Now().Format(time.RFC3339),
			},
		}

		if err := s.publisher.PublishJobSubmitted(ctx, job); err != nil {
			return nil, fmt.Errorf("failed publishing stream job %s: %w", jobID, err)
		}

		s.store.AddJob(job, "SUBMITTED")
		publishedIDs = append(publishedIDs, jobID)
	}

	return &QueuePublishResponse{
		Published: len(publishedIDs),
		Subject:   "jobs.submitted",
		Jobs:      publishedIDs,
	}, nil
}

