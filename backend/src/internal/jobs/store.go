package jobs

import (
	"sync"
	"time"
)

// JobStore tracks job details and domain status history.
type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*JobDetailResponse
}

// NewJobStore initializes a new JobStore.
func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*JobDetailResponse),
	}
}

// AddJob registers or updates a job record in the domain store.
func (s *JobStore) AddJob(job Job, status string, correlationID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	detail, exists := s.jobs[job.JobID]
	if !exists {
		detail = &JobDetailResponse{
			JobID:         job.JobID,
			Type:          job.Type,
			CorrelationID: correlationID,
			TraceID:       job.TraceID,
			History:       make([]JobHistoryItem, 0),
		}
		s.jobs[job.JobID] = detail
	}

	detail.Status = status
	if correlationID != "" {
		detail.CorrelationID = correlationID
	}
	if job.TraceID != "" {
		detail.TraceID = job.TraceID
	}

	detail.History = append(detail.History, JobHistoryItem{
		Status:    status,
		Timestamp: time.Now(),
	})
}

// SetTraceID associates a trace ID with an existing job.
func (s *JobStore) SetTraceID(jobID, traceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.TraceID = traceID
	}
}
