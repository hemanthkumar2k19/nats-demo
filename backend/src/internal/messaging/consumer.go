package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"

	"github.com/nats-io/nats.go"
)

// JobHandler defines the callback function signature for processing a job.
type JobHandler func(job jobs.Job, correlationID string) error

// ValidationHandler defines the callback function signature for validating a job request.
type ValidationHandler func(job jobs.Job) (jobs.JobValidationResponse, error)

// Consumer manages NATS subscriptions.
type Consumer struct {
	client *natsclient.Client
}

// NewConsumer initializes a new Consumer.
func NewConsumer(client *natsclient.Client) *Consumer {
	return &Consumer{client: client}
}

// SubscribeJobSubmitted registers a subscription to SubjectJobSubmitted and dispatches events to the handler.
func (c *Consumer) SubscribeJobSubmitted(handler JobHandler) (*nats.Subscription, error) {
	sub, err := c.client.Conn.Subscribe(SubjectJobSubmitted, func(msg *nats.Msg) {
		correlationID := msg.Header.Get("X-Correlation-Id")

		var job jobs.Job
		if err := json.Unmarshal(msg.Data, &job); err != nil {
			log.Printf("[Consumer] Failed to unmarshal message payload: %v", err)
			return
		}

		log.Printf("[Consumer] Received message on subject: %s | Job ID: %s | Correlation ID: %s", msg.Subject, job.JobID, correlationID)

		if err := handler(job, correlationID); err != nil {
			log.Printf("[Consumer] Error handling job %s: %v", job.JobID, err)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", SubjectJobSubmitted, err)
	}

	return sub, nil
}

// SubscribeJobValidate registers a subscription to SubjectJobValidate and replies synchronously using the handler.
func (c *Consumer) SubscribeJobValidate(handler ValidationHandler) (*nats.Subscription, error) {
	sub, err := c.client.Conn.Subscribe(SubjectJobValidate, func(msg *nats.Msg) {
		var job jobs.Job
		if err := json.Unmarshal(msg.Data, &job); err != nil {
			log.Printf("[Consumer] Failed to unmarshal validation request: %v", err)
			resp := jobs.JobValidationResponse{Valid: false, Message: "Invalid request payload"}
			respData, _ := json.Marshal(resp)
			_ = msg.Respond(respData)
			return
		}

		resp, err := handler(job)
		if err != nil {
			resp = jobs.JobValidationResponse{Valid: false, Message: err.Error()}
		}

		respData, err := json.Marshal(resp)
		if err != nil {
			log.Printf("[Consumer] Failed to marshal validation response: %v", err)
			return
		}

		if err := msg.Respond(respData); err != nil {
			log.Printf("[Consumer] Failed to send validation reply: %v", err)
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", SubjectJobValidate, err)
	}

	return sub, nil
}

