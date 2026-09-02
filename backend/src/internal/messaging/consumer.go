package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"nats-demo/internal/jobs"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/nats-io/nats.go"
)

// JobHandler defines the callback function signature for processing a job.
type JobHandler func(ctx context.Context, job jobs.Job) error

// ValidationHandler defines the callback function signature for validating a job request.
type ValidationHandler func(ctx context.Context, job jobs.Job) (jobs.JobValidationResponse, error)

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
		var job jobs.Job
		if err := json.Unmarshal(msg.Data, &job); err != nil {
			log.Printf("[Consumer] Failed to unmarshal message payload: %v", err)
			return
		}

		log.Printf("[Consumer] Received message on subject: %s | Job ID: %s", msg.Subject, job.JobID)

		parentCtx := telemetry.ExtractTraceContext(context.Background(), msg.Header)
		if err := handler(parentCtx, job); err != nil {
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

		parentCtx := telemetry.ExtractTraceContext(context.Background(), msg.Header)
		resp, err := handler(parentCtx, job)
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
