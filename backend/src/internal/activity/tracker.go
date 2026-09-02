package activity

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Activity represents a NATS lifecycle event log item for the demo UI.
type Activity struct {
	Timestamp     string `json:"timestamp"`
	JobID         string `json:"job_id"`
	Event         string `json:"event"`
	Subject       string `json:"subject"`
	Worker        string `json:"worker"`
	DeliveryCount int    `json:"delivery_count"`
	DeliveryMode  string `json:"delivery_mode,omitempty"`
	Sequence      uint64 `json:"sequence,omitempty"`
	MsgID         string `json:"msg_id,omitempty"`
	JobType       string `json:"job_type,omitempty"`
	TraceID       string `json:"trace_id,omitempty"`
}

// Tracker maintains an in-memory capped ring-buffer of observed lifecycle activities.
type Tracker struct {
	mu         sync.RWMutex
	activities []Activity
}

// NewTracker initializes a new Activity Tracker.
func NewTracker() *Tracker {
	return &Tracker{
		activities: make([]Activity, 0),
	}
}

func getStatusWeight(status string) int {
	switch status {
	case "PUBLISHED", "STORED", "REQUEST_SENT", "DEDUPLICATED":
		return 1
	case "RECEIVED", "DELIVERED", "REQUEST_RECEIVED", "REDELIVERED":
		return 2
	case "PROCESSING", "REPLY_SENT":
		return 3
	case "COMPLETED", "FAILED", "ACKED", "NO CONSUMER", "REPLY_RECEIVED", "REQUEST_TIMEOUT", "REPLAYED", "DLQ_PUBLISHED":
		return 4
	default:
		return 0
	}
}

// AddEvent records a lifecycle event in the in-memory buffer.
func (t *Tracker) AddEvent(jobID string, status string, deliveryCount int, subject string, worker string, deliveryMode string, sequence uint64, msgID string, jobType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	timestamp := time.Now().Format("15:04:05")
	t.activities = append([]Activity{{
		Timestamp:     timestamp,
		JobID:         jobID,
		Event:         status,
		Subject:       subject,
		Worker:        worker,
		DeliveryCount: deliveryCount,
		DeliveryMode:  deliveryMode,
		Sequence:      sequence,
		MsgID:         msgID,
		JobType:       jobType,
	}}, t.activities...)

	// Cap activities at 200 items to prevent unbounded memory growth
	if len(t.activities) > 200 {
		t.activities = t.activities[:200]
	}
}

// GetActivities returns a copy of the activity buffer sorted by timestamp and status weight.
func (t *Tracker) GetActivities() []Activity {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activitiesCopy := make([]Activity, len(t.activities))
	copy(activitiesCopy, t.activities)

	sort.SliceStable(activitiesCopy, func(i, j int) bool {
		if activitiesCopy[i].Timestamp != activitiesCopy[j].Timestamp {
			return activitiesCopy[i].Timestamp > activitiesCopy[j].Timestamp
		}
		if activitiesCopy[i].JobID != activitiesCopy[j].JobID {
			return activitiesCopy[i].JobID < activitiesCopy[j].JobID
		}
		return getStatusWeight(activitiesCopy[i].Event) > getStatusWeight(activitiesCopy[j].Event)
	})

	return activitiesCopy
}

// ProcessLifecycleEvent parses an incoming NATS lifecycle message and appends it to the activity log.
func (t *Tracker) ProcessLifecycleEvent(subject string, data []byte, source string, msgID string) error {
	var payload struct {
		JobID         string `json:"job_id"`
		Type          string `json:"type,omitempty"`
		Status        string `json:"status"`
		DeliveryCount int    `json:"delivery_count"`
		Error         string `json:"error,omitempty"`
		DeliveryMode  string `json:"delivery_mode,omitempty"`
		Sequence      uint64 `json:"sequence,omitempty"`
		MsgID         string `json:"msg_id,omitempty"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal lifecycle event payload: %w", err)
	}

	if payload.JobID == "" {
		return fmt.Errorf("lifecycle event payload missing job_id")
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
		// Outgoing request message to jobs.validate is a Job payload, not a lifecycle event.
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
	case "jobs.replayed":
		status = "REPLAYED"
	case "jobs.dlq", "jobs.dlq.published":
		status = "DLQ_PUBLISHED"
	default:
		status = payload.Status
	}

	if payload.DeliveryCount <= 0 {
		payload.DeliveryCount = 1
	}

	t.AddEvent(payload.JobID, status, payload.DeliveryCount, subject, source, payload.DeliveryMode, payload.Sequence, msgID, payload.Type)
	return nil
}
