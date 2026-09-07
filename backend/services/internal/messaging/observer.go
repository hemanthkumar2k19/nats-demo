package messaging

import (
	"sync"
	"time"
)

// ObservedEvent represents a single NATS message delivery to one or more subscriptions.
type ObservedEvent struct {
	MessageID  string    `json:"-"`
	JobID      string    `json:"job_id"`
	Subject    string    `json:"subject"`
	ReceivedBy []string  `json:"received_by"`
	Timestamp  time.Time `json:"timestamp"`
}

// Observer manages the observation of subject addressing demo.
type Observer struct {
	mu     sync.RWMutex
	events []*ObservedEvent
}

// NewObserver initializes a new Observer.
func NewObserver() *Observer {
	return &Observer{
		events: make([]*ObservedEvent, 0),
	}
}

// RecordEvent records a message delivery to a subscription.
func (o *Observer) RecordEvent(messageID string, subscriptionName string, subject string, jobID string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Check if this message has already been received by another subscription
	for _, e := range o.events {
		if e.MessageID == messageID && e.Subject == subject {
			if e.JobID == "" {
				e.JobID = jobID
			}
			// Add subscription name if not already present
			for _, sub := range e.ReceivedBy {
				if sub == subscriptionName {
					return
				}
			}
			e.ReceivedBy = append(e.ReceivedBy, subscriptionName)
			return
		}
	}

	// If not found, create a new event
	o.events = append(o.events, &ObservedEvent{
		MessageID:  messageID,
		JobID:      jobID,
		Subject:    subject,
		ReceivedBy: []string{subscriptionName},
		Timestamp:  time.Now(),
	})

	// Cap at 100 events to prevent memory leak
	if len(o.events) > 100 {
		o.events = o.events[1:]
	}
}

// GetEvents returns the list of observed events.
func (o *Observer) GetEvents() []*ObservedEvent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	// Create a copy to prevent race conditions
	eventsCopy := make([]*ObservedEvent, len(o.events))
	for i, e := range o.events {
		eventsCopy[i] = &ObservedEvent{
			JobID:      e.JobID,
			Subject:    e.Subject,
			ReceivedBy: append([]string(nil), e.ReceivedBy...),
			Timestamp:  e.Timestamp,
		}
	}
	return eventsCopy
}
