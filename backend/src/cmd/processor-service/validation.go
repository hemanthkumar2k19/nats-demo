package main

import (
	"context"
	"fmt"
	"log"

	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// subscribeValidation registers the jobs.validate Request/Reply handler.
// The subscription stays active for the lifetime of the processor.
// Whether to respond is controlled by the processingEnabled flag checked
// inside the handler - if OFF the handler returns without replying, so the
// requester times out naturally with no race window.
func (a *App) subscribeValidation(workerName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.valSub != nil {
		return nil
	}

	validationHandler := func(ctx context.Context, job jobs.Job) (jobs.JobValidationResponse, error) {
		// Check the processing flag before doing anything.
		// If the processor is OFF, return a sentinel error that tells
		// consumer.SubscribeJobValidate to skip Respond, letting the
		// requester time out naturally.
		a.mu.RLock()
		enabled := a.processingEnabled
		a.mu.RUnlock()

		if !enabled {
			log.Printf("[%s] Validation request for job %s received but processing is disabled - not responding", workerName, job.JobID)
			// Return a special sentinel so consumer.go skips msg.Respond.
			return jobs.JobValidationResponse{}, jobs.ErrProcessorDisabled
		}

		// Start Process Validation Request Span
		reqCtx, reqSpan := telemetry.StartSpan(ctx, "Process Validation Request",
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("messaging.system", "nats"),
				attribute.String("messaging.destination.name", messaging.SubjectJobValidate),
				attribute.String("worker.id", workerName),
				attribute.String("job.id", job.JobID),
				attribute.String("job.type", job.Type),
			),
		)
		defer reqSpan.End()

		log.Printf("[%s] Received validation request for job: %s of type %s", workerName, job.JobID, job.Type)

		// Publish REQUEST_RECEIVED so job-service activity log captures it.
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobRequestReceived,
			job.JobID,
			"REQUEST_RECEIVED",
			1,
			"",
			workerName,
			"",
			0,
		)

		var resp jobs.JobValidationResponse
		if job.JobID == "" {
			resp = jobs.JobValidationResponse{Valid: false, Message: "job_id is required"}
		} else if job.Type != "image-processing" && job.Type != "data-sync" && job.Type != "email-alert" {
			resp = jobs.JobValidationResponse{Valid: false, Message: fmt.Sprintf("unsupported job type: %s", job.Type)}
		} else if len(job.Payload) == 0 {
			resp = jobs.JobValidationResponse{Valid: false, Message: "payload is required"}
		} else {
			resp = jobs.JobValidationResponse{Valid: true, Message: "Job configuration is valid."}
		}

		if resp.Valid {
			reqSpan.SetStatus(codes.Ok, "valid")
			reqSpan.SetAttributes(attribute.Bool("validation.valid", true))
		} else {
			reqSpan.SetStatus(codes.Error, resp.Message)
			reqSpan.SetAttributes(attribute.Bool("validation.valid", false))
		}

		// Create child span for NATS Reply
		_, replySpan := telemetry.StartSpan(reqCtx, "NATS Reply", trace.WithSpanKind(trace.SpanKindProducer))
		defer replySpan.End()

		// Publish REPLY_SENT before the reply is dispatched.
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobReplySent,
			job.JobID,
			"REPLY_SENT",
			1,
			"",
			workerName,
			"",
			0,
		)

		return resp, nil
	}

	sub, err := a.consumer.SubscribeJobValidate(validationHandler)
	if err != nil {
		return err
	}
	a.valSub = sub
	log.Printf("[%s] Validation subscriber activated on subject: %s", workerName, messaging.SubjectJobValidate)
	return nil
}

// unsubscribeValidation removes the jobs.validate subscriber.
// When the processor is OFF, this causes NATS requests to time out naturally.
func (a *App) unsubscribeValidation() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.valSub != nil {
		if err := a.valSub.Unsubscribe(); err != nil {
			log.Printf("[Processor] Validation unsubscribe failed: %v", err)
		} else {
			log.Println("[Processor] Validation subscriber deactivated")
		}
		a.valSub = nil
	}
}
