package jobs

import (
	"sync"
	"time"
)

// Activity represents a NATS lifecycle event log item.
type Activity struct {
	Timestamp     string `json:"timestamp"`
	JobID         string `json:"job_id"`
	Event         string `json:"event"`
	Subject       string `json:"subject"`
	Worker        string `json:"worker"`
	DeliveryCount int    `json:"delivery_count"`
	DeliveryMode  string `json:"delivery_mode,omitempty"`
	Sequence      uint64 `json:"sequence,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
	MsgID         string `json:"msg_id,omitempty"`
	JobType       string `json:"job_type,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

// JobStore tracks job details and a capped list of activities.
type JobStore struct {
	mu         sync.RWMutex
	jobs       map[string]*JobDetailResponse
	activities []Activity
}

// NewJobStore initializes a new JobStore.
func NewJobStore() *JobStore {
	return &JobStore{
		jobs:       make(map[string]*JobDetailResponse),
		activities: make([]Activity, 0),
	}
}

// AddEvent records a lifecycle event and updates the corresponding job detail.
func (s *JobStore) AddEvent(jobID string, status string, deliveryCount int, correlationID string, subject string, worker string, deliveryMode string, sequence uint64, msgID string, jobType string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Get or create job detail
	job, exists := s.jobs[jobID]
	if !exists {
		job = &JobDetailResponse{
			JobID:         jobID,
			Type:          jobType,
			CorrelationID: correlationID,
			History:       make([]JobHistoryItem, 0),
		}
		s.jobs[jobID] = job
	}

	job.Status = status
	if deliveryCount > job.DeliveryCount {
		job.DeliveryCount = deliveryCount
	}
	if correlationID != "" {
		job.CorrelationID = correlationID
	}
	if jobType != "" {
		job.Type = jobType
	} else if job.Type != "" {
		jobType = job.Type
	}
	if correlationID == "" && job.CorrelationID != "" {
		correlationID = job.CorrelationID
	}
	if msgID == "" {
		msgID = jobID
	}

	// Append to history
	job.History = append(job.History, JobHistoryItem{
		Status:    status,
		Timestamp: time.Now(),
	})

	// 2. Append to activities list (prepend newest)
	timestamp := time.Now().Format("15:04:05")
	s.activities = append([]Activity{{
		Timestamp:     timestamp,
		JobID:         jobID,
		Event:         status,
		Subject:       subject,
		Worker:        worker,
		DeliveryCount: deliveryCount,
		DeliveryMode:  deliveryMode,
		Sequence:      sequence,
		CorrelationID: correlationID,
		MsgID:         msgID,
		JobType:       jobType,
	}}, s.activities...)

	// Cap activities at 200 to prevent unbounded memory growth
	if len(s.activities) > 200 {
		s.activities = s.activities[:200]
	}
}

// SetTraceID associates a trace ID with an existing job and its activities.
func (s *JobStore) SetTraceID(jobID, traceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.jobs[jobID]; exists {
		job.TraceID = traceID
	}

	for i := range s.activities {
		if s.activities[i].JobID == jobID && s.activities[i].TraceID == "" {
			s.activities[i].TraceID = traceID
		}
	}
}
