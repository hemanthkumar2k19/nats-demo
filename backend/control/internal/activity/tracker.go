package activity

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
	Category      string `json:"category,omitempty"` // "BUSINESS" vs "LIFECYCLE"
	Action        string `json:"action,omitempty"`   // Human-readable action name
}

// Tracker maintains an in-memory capped ring-buffer of observed lifecycle activities.
type Tracker struct {
	mu         sync.RWMutex
	activities []Activity
	jobTypes   map[string]string
}

// NewTracker initializes a new Activity Tracker.
func NewTracker() *Tracker {
	return &Tracker{
		activities: make([]Activity, 0),
		jobTypes:   make(map[string]string),
	}
}

// DetermineCategory classifies an activity into BUSINESS (actual payloads published to topics)
// or LIFECYCLE (broker ingestion, worker execution, ack/nak telemetry).
func DetermineCategory(subject string, status string) string {
	switch status {
	case "PUBLISHED", "SCHEDULED", "REPROCESSED", "REQUEST_SENT", "REPLY_RECEIVED",
		"SAGA_TRIGGERED", "SAGA_STARTED", "OP1_RESERVE", "OP2_PAYMENT",
		"OP1_COMPLETED", "OP1_FAILED", "OP2_COMPLETED", "OP2_FAILED",
		"OP1_COMPENSATING", "OP1_COMPENSATED", "SAGA_COMPLETED", "SAGA_FAILED":
		return "BUSINESS"
	case "STORED", "DEDUPLICATED", "RECEIVED", "DELIVERED", "PROCESSING",
		"COMPLETED", "FAILED", "ACKED", "NO CONSUMER", "NAK_WITH_DELAY",
		"ACK_TIMEOUT_SIMULATED", "DLQ_PUBLISHED", "REPLAYED", "REQUEST_RECEIVED",
		"REPLY_SENT", "SAGA_STEP", "SAGA_COMPENSATING":
		return "LIFECYCLE"
	default:
		if subject == "jobs.submitted" || subject == "jobs.queue" || subject == "jobs.validate" {
			return "BUSINESS"
		}
		return "LIFECYCLE"
	}
}

// DetermineAction assigns a descriptive operational action label.
func DetermineAction(subject string, status string) string {
	switch status {
	case "PUBLISHED":
		if subject == "jobs.queue" {
			return "Published to Queue Group"
		}
		return "Published by Client"
	case "SCHEDULED":
		return "Scheduled for Publishing"
	case "REPROCESSED":
		return "Reprocessed from DLQ"
	case "STORED":
		return "JetStream Stream Ingestion"
	case "DEDUPLICATED":
		return "JetStream Deduplication Discard"
	case "DELIVERED":
		return "Worker Pull / Delivery"
	case "RECEIVED":
		if subject == "jobs.queue" || subject == "jobs.queue.received" {
			return "Queue Group Worker Received"
		}
		return "Message Received by Worker"
	case "PROCESSING":
		return "Worker Task Processing"
	case "COMPLETED":
		if subject == "jobs.queue" || subject == "jobs.queue.completed" {
			return "Queue Group Task Completed"
		}
		return "Task Completed"
	case "FAILED":
		return "Task Failed"
	case "ACKED":
		return "Message Acknowledged (msg.Ack())"
	case "NAK_WITH_DELAY":
		return "Retry Delay Requested (msg.NakWithDelay())"
	case "ACK_TIMEOUT_SIMULATED":
		return "AckWait Missing ACK Timeout"
	case "DLQ_PUBLISHED":
		return "Poison Pill Routed to DLQ"
	case "REPLAYED":
		return "Ephemeral Stream Replay"
	case "NO CONSUMER":
		return "No Active Consumer"
	case "REQUEST_SENT":
		return "Sync Request Sent"
	case "REQUEST_RECEIVED":
		return "Request Received by Worker"
	case "REPLY_SENT":
		return "Sync Reply Dispatched"
	case "REPLY_RECEIVED":
		return "Sync Reply Received"
	case "SAGA_TRIGGERED", "SAGA_STARTED":
		return "Saga Workflow Triggered"
	case "OP1_RESERVE":
		return "Op1: Inventory Reserved"
	case "OP1_COMPLETED":
		return "Op1: Reservation Confirmed"
	case "OP1_FAILED":
		return "Op1: Reservation Failed"
	case "OP2_PAYMENT":
		return "Op2: Payment Charged"
	case "OP2_COMPLETED":
		return "Op2: Payment Settled"
	case "OP2_FAILED":
		return "Op2: Payment Declined"
	case "OP1_COMPENSATING":
		return "Op1: Compensating Release"
	case "OP1_COMPENSATED":
		return "Op1: Compensation Finished"
	case "SAGA_COMPLETED":
		return "Saga Transaction Succeeded"
	case "SAGA_FAILED":
		return "Saga Transaction Aborted"
	default:
		return status
	}
}

func getStatusWeight(status string) int {
	switch status {
	case "SCHEDULED", "PUBLISHED", "STORED", "REQUEST_SENT", "DEDUPLICATED", "REPROCESSED", "SAGA_STARTED", "SAGA_TRIGGERED", "OP1_RESERVE", "OP2_PAYMENT":
		return 1
	case "RECEIVED", "DELIVERED", "REQUEST_RECEIVED", "REDELIVERED", "SAGA_STEP", "SAGA_COMPENSATING", "OP1_COMPLETED", "OP1_FAILED", "OP2_COMPLETED", "OP2_FAILED", "OP1_COMPENSATING", "OP1_COMPENSATED":
		return 2
	case "PROCESSING", "REPLY_SENT", "NAK_WITH_DELAY", "ACK_TIMEOUT_SIMULATED":
		return 3
	case "COMPLETED", "FAILED", "ACKED", "NO CONSUMER", "REPLY_RECEIVED", "REQUEST_TIMEOUT", "REPLAYED", "DLQ_PUBLISHED", "SAGA_COMPLETED", "SAGA_COMPENSATED", "SAGA_FAILED":
		return 4
	default:
		return 0
	}
}

// AddEvent records a lifecycle event in the in-memory buffer.
func (t *Tracker) AddEvent(jobID string, status string, deliveryCount int, subject string, worker string, deliveryMode string, sequence uint64, msgID string, jobType string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if jobType != "" {
		t.jobTypes[jobID] = jobType
	} else if cachedType, ok := t.jobTypes[jobID]; ok {
		jobType = cachedType
	}

	category := DetermineCategory(subject, status)
	action := DetermineAction(subject, status)

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
		Category:      category,
		Action:        action,
	}}, t.activities...)

	// Cap activities at 200 items to prevent unbounded memory growth
	if len(t.activities) > 200 {
		t.activities = t.activities[:200]
	}
}

// ClearActivities resets the activity buffer to empty.
// Useful for clearing the log between demo scenarios.
func (t *Tracker) ClearActivities() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activities = make([]Activity, 0)
	t.jobTypes = make(map[string]string)
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
	case "jobs.scheduled":
		status = "SCHEDULED"
	case "jobs.nak.delayed":
		status = "NAK_WITH_DELAY"
	case "jobs.ack.timeout":
		status = "ACK_TIMEOUT_SIMULATED"
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
	case "jobs.dlq":
		// jobs.dlq is the raw message payload routed to the JOBS_DLQ stream.
		// The lifecycle transition event is emitted separately on jobs.dlq.published.
		return nil
	case "jobs.dlq.published":
		status = "DLQ_PUBLISHED"
	case "jobs.reprocessed", "jobs.dlq.reprocessed":
		status = "REPROCESSED"
		subject = "jobs.submitted"
	case "jobs.queue":
		status = "PUBLISHED"
	case "jobs.queue.received":
		status = "RECEIVED"
		subject = "jobs.queue"
	case "jobs.queue.completed":
		status = "COMPLETED"
		subject = "jobs.queue"
	case "saga.job.started":
		status = "SAGA_STARTED"
		payload.DeliveryMode = "SAGA"
	case "saga.job.step.completed":
		status = "SAGA_STEP"
		payload.DeliveryMode = "SAGA"
	case "saga.job.compensation.started":
		status = "SAGA_COMPENSATING"
		payload.DeliveryMode = "SAGA"
	case "saga.job.compensation.completed":
		status = "SAGA_COMPENSATED"
		payload.DeliveryMode = "SAGA"
	case "saga.job.completed":
		status = "SAGA_COMPLETED"
		payload.DeliveryMode = "SAGA"
	case "saga.job.failed":
		status = "SAGA_FAILED"
		payload.DeliveryMode = "SAGA"
	case "saga.start":
		status = "SAGA_TRIGGERED"
		payload.DeliveryMode = "SAGA"
	case "saga.op1.reserve":
		status = "OP1_RESERVE"
		payload.DeliveryMode = "SAGA"
	case "saga.op1.completed":
		status = "OP1_COMPLETED"
		payload.DeliveryMode = "SAGA"
	case "saga.op1.failed":
		status = "OP1_FAILED"
		payload.DeliveryMode = "SAGA"
	case "saga.op2.payment":
		status = "OP2_PAYMENT"
		payload.DeliveryMode = "SAGA"
	case "saga.op2.completed":
		status = "OP2_COMPLETED"
		payload.DeliveryMode = "SAGA"
	case "saga.op2.failed":
		status = "OP2_FAILED"
		payload.DeliveryMode = "SAGA"
	case "saga.op1.compensate":
		status = "OP1_COMPENSATING"
		payload.DeliveryMode = "SAGA"
	case "saga.op1.compensated":
		status = "OP1_COMPENSATED"
		payload.DeliveryMode = "SAGA"
	case "saga.completed":
		status = "SAGA_COMPLETED"
		payload.DeliveryMode = "SAGA"
	case "saga.failed":
		status = "SAGA_FAILED"
		payload.DeliveryMode = "SAGA"
	default:
		status = payload.Status
	}

	if source == "" {
		if subject == "jobs.submitted" {
			source = "job-service"
		} else if strings.HasPrefix(subject, "saga.") {
			source = "saga-orchestrator"
		}
	}

	if payload.DeliveryMode == "" {
		if subject == "jobs.queue" || subject == "jobs.queue.received" || subject == "jobs.queue.completed" {
			payload.DeliveryMode = "CORE"
		} else if subject == "jobs.submitted" {
			payload.DeliveryMode = "JETSTREAM"
		}
	}

	if payload.DeliveryCount <= 0 {
		payload.DeliveryCount = 1
	}

	t.AddEvent(payload.JobID, status, payload.DeliveryCount, subject, source, payload.DeliveryMode, payload.Sequence, msgID, payload.Type)
	return nil
}
