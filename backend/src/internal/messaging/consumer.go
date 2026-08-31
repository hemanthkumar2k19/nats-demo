package messaging

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"

	"github.com/nats-io/nats.go"
)

// JobHandler defines the callback function signature for processing a job.
type JobHandler func(job jobs.Job, correlationID string) error

// ValidationHandler defines the callback function signature for validating a job request.
// correlationID is extracted from the NATS message headers and forwarded so the
// handler can include it in any lifecycle events it publishes.
// Returning jobs.ErrProcessorDisabled signals that the handler chose not to respond;
// msg.Respond is skipped so the requester times out naturally.
type ValidationHandler func(job jobs.Job, correlationID string) (jobs.JobValidationResponse, error)

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
// The X-Correlation-Id header from the request is extracted and passed to the handler so lifecycle
// events published by the handler carry the same correlation ID.
// If the handler returns jobs.ErrProcessorDisabled, msg.Respond is skipped so the
// NATS requester times out naturally - this is the intended "processor OFF" behaviour.
func (c *Consumer) SubscribeJobValidate(handler ValidationHandler) (*nats.Subscription, error) {
	sub, err := c.client.Conn.Subscribe(SubjectJobValidate, func(msg *nats.Msg) {
		correlationID := msg.Header.Get("X-Correlation-Id")

		var job jobs.Job
		if err := json.Unmarshal(msg.Data, &job); err != nil {
			log.Printf("[Consumer] Failed to unmarshal validation request: %v", err)
			resp := jobs.JobValidationResponse{Valid: false, Message: "Invalid request payload"}
			respData, _ := json.Marshal(resp)
			_ = msg.Respond(respData)
			return
		}

		resp, err := handler(job, correlationID)
		if err != nil {
			// ErrProcessorDisabled means the handler deliberately chose not to respond.
			// Skip msg.Respond so the requester's 2-second timeout fires naturally.
			if errors.Is(err, jobs.ErrProcessorDisabled) {
				return
			}
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
