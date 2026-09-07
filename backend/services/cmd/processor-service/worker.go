package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"nats-demo/services/internal/jobs"
	"nats-demo/services/internal/messaging"
	"nats-demo/services/internal/telemetry"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// buildCoreJobHandler returns the message handler callback for Core NATS messages.
func (a *App) buildCoreJobHandler(attempts map[string]int, attemptsMu *sync.Mutex, coreWorkerCounter *uint64) messaging.JobHandler {
	return func(ctx context.Context, job jobs.Job) error {
		deliveryMode := job.DeliveryMode
		if deliveryMode == "" {
			deliveryMode = "CORE"
		}

		// If this is a JetStream message, ignore it in the Core subscription
		if deliveryMode == "JETSTREAM" {
			return nil
		}

		// Distribute across active workers (e.g. processor-1, processor-2)
		a.mu.RLock()
		workersCount := a.consumerConfig.Workers
		a.mu.RUnlock()
		if workersCount <= 0 {
			workersCount = 1
		}
		currWorkerIdx := (atomic.AddUint64(coreWorkerCounter, 1) - 1)%uint64(workersCount) + 1
		assignedWorkerName := fmt.Sprintf("processor-%d", currWorkerIdx)

		attemptsMu.Lock()
		attempts[job.JobID]++
		attemptCount := attempts[job.JobID]
		attemptsMu.Unlock()

		log.Printf("[%s] Received Core NATS job %s | Attempt: %d", assignedWorkerName, job.JobID, attemptCount)

		// Start Consumer Receive Span
		recvCtx, recvSpan := telemetry.StartSpan(ctx, "Consumer Receive",
			trace.WithSpanKind(trace.SpanKindConsumer),
			trace.WithAttributes(
				attribute.String("messaging.system", "nats"),
				attribute.String("messaging.operation", "receive"),
				attribute.String("messaging.destination.name", messaging.SubjectJobSubmitted),
				attribute.String("worker.id", assignedWorkerName),
				attribute.String("delivery.mode", deliveryMode),
				attribute.String("job.id", job.JobID),
				attribute.String("job.type", job.Type),
				attribute.Int64("delivery.count", int64(attemptCount)),
			),
		)
		defer recvSpan.End()

		telemetry.RecordMessageReceived(recvCtx, deliveryMode, assignedWorkerName, messaging.SubjectJobSubmitted)

		// 1. Publish RECEIVED lifecycle event
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobReceived,
			job.JobID,
			"RECEIVED",
			attemptCount,
			"",
			assignedWorkerName,
			deliveryMode,
			0,
		)

		// Start internal Process Job Span
		procCtx, procSpan := telemetry.StartSpan(recvCtx, "Process Job",
			trace.WithSpanKind(trace.SpanKindInternal),
			trace.WithAttributes(
				attribute.String("job.id", job.JobID),
				attribute.String("job.type", job.Type),
				attribute.String("worker.id", assignedWorkerName),
				attribute.Int64("delivery.count", int64(attemptCount)),
			),
		)

		// Publish PROCESSING lifecycle event on 3-token subject jobs.processing.started
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobProcessingStarted,
			job.JobID,
			"PROCESSING",
			attemptCount,
			"",
			assignedWorkerName,
			deliveryMode,
			0,
		)

		// 2. Simulate processing duration
		procStart := time.Now()
		time.Sleep(1 * time.Second)
		procDuration := time.Since(procStart)

		// 3. Evaluate failure simulation
		simulateFailure := false
		if val, ok := job.Payload["simulate_failure"].(bool); ok && val {
			simulateFailure = true
		}

		simulateFailureCount := 0
		if val, ok := job.Payload["simulate_failure_count"].(float64); ok {
			simulateFailureCount = int(val)
		} else if val, ok := job.Payload["simulate_failure_count"].(int); ok {
			simulateFailureCount = val
		}

		if simulateFailure && (simulateFailureCount == 0 || attemptCount <= simulateFailureCount) {
			errMsg := fmt.Sprintf("Simulated failure attempt %d of %d", attemptCount, simulateFailureCount)
			log.Printf("[%s] Job %s failed: %s", assignedWorkerName, job.JobID, errMsg)

			procSpan.RecordError(fmt.Errorf("%s", errMsg))
			procSpan.SetStatus(codes.Error, errMsg)
			procSpan.SetAttributes(attribute.String("processing.result", "failure"))
			procSpan.End()

			telemetry.RecordJobFailed(procCtx, deliveryMode, assignedWorkerName)

			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobProcessingFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				assignedWorkerName,
				deliveryMode,
				0,
			)

			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				assignedWorkerName,
				deliveryMode,
				0,
			)
			return fmt.Errorf("simulated failure: %s", errMsg)
		}

		procSpan.SetStatus(codes.Ok, "success")
		procSpan.SetAttributes(attribute.String("processing.result", "success"))
		procSpan.End()

		recvSpan.AddEvent("message_processed")

		// 4. Publish Completed lifecycle event on success
		log.Printf("[%s] Job %s processed successfully", assignedWorkerName, job.JobID)
		telemetry.RecordJobProcessed(recvCtx, deliveryMode, assignedWorkerName, "COMPLETED", procDuration)

		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobCompleted,
			job.JobID,
			"COMPLETED",
			attemptCount,
			"",
			assignedWorkerName,
			deliveryMode,
			0,
		)

		return nil
	}
}

// subscribeCore registers Core NATS subscriber if missing
func (a *App) subscribeCore(workerName string, jobHandler messaging.JobHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sub != nil {
		return nil
	}

	sub, err := a.consumer.SubscribeJobSubmitted(jobHandler)
	if err != nil {
		return err
	}
	a.sub = sub
	log.Printf("[%s] Subscribed to Core NATS subject: %s", workerName, messaging.SubjectJobSubmitted)
	return nil
}

// unsubscribeCore removes Core NATS subscriber
func (a *App) unsubscribeCore() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.sub != nil {
		_ = a.sub.Unsubscribe()
		a.sub = nil
		log.Println("[Processor] Core NATS subscriber deactivated")
	}
}

// subscribeJetStream registers JetStream pull consumer based on a.consumerConfig
func (a *App) subscribeJetStream() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.jsConsumer != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := a.natsClient.JS.Stream(ctx, "JOBS")
	if err != nil {
		return fmt.Errorf("failed to get JOBS stream: %w", err)
	}

	cType := a.consumerConfig.Type
	if cType == "" {
		cType = "durable"
	}

	deliverPolicy := jetstream.DeliverAllPolicy
	switch a.consumerConfig.DeliverPolicy {
	case "new":
		deliverPolicy = jetstream.DeliverNewPolicy
	case "last":
		deliverPolicy = jetstream.DeliverLastPolicy
	case "last_per_subject":
		deliverPolicy = jetstream.DeliverLastPerSubjectPolicy
	case "all":
		deliverPolicy = jetstream.DeliverAllPolicy
	default:
		if cType == "ephemeral" {
			deliverPolicy = jetstream.DeliverNewPolicy
		} else {
			deliverPolicy = jetstream.DeliverAllPolicy
		}
	}

	ackPolicy := jetstream.AckExplicitPolicy
	switch a.consumerConfig.AckPolicy {
	case "none":
		ackPolicy = jetstream.AckNonePolicy
	case "all":
		ackPolicy = jetstream.AckAllPolicy
	case "explicit":
		ackPolicy = jetstream.AckExplicitPolicy
	default:
		ackPolicy = jetstream.AckExplicitPolicy
	}

	if cType == "ephemeral" {
		if a.consumerName == "" || a.consumerName == "job-processor" || a.consumerName == "processor-durable" {
			a.consumerName = fmt.Sprintf("ephemeral-%d", time.Now().UnixNano()%100000)
		}
		consumer, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			Name:          a.consumerName,
			DeliverPolicy: deliverPolicy,
			AckPolicy:     ackPolicy,
			AckWait:       5 * time.Second,
			FilterSubject: messaging.SubjectJobSubmitted,
		})
		if err != nil {
			return fmt.Errorf("failed to add ephemeral consumer: %w", err)
		}
		a.jsConsumer = consumer
		log.Printf("[Processor] Bound to ephemeral consumer: %s (Deliver=%v, Ack=%v)", a.consumerName, deliverPolicy, ackPolicy)
		return nil
	}

	// Durable consumer
	a.consumerName = "job-processor"

	// If the durable consumer already exists with a different DeliverPolicy or AckPolicy,
	// recreate it because DeliverPolicy and AckPolicy are immutable on existing NATS consumers.
	if existing, err := stream.Consumer(ctx, a.consumerName); err == nil && existing != nil {
		if info, err := existing.Info(ctx); err == nil && info != nil {
			if info.Config.DeliverPolicy != deliverPolicy || info.Config.AckPolicy != ackPolicy {
				log.Printf("[Processor] Deliver/Ack policy changed on durable consumer; recreating %s", a.consumerName)
				_ = stream.DeleteConsumer(ctx, a.consumerName)
			}
		}
	}

	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       a.consumerName,
		DeliverPolicy: deliverPolicy,
		AckPolicy:     ackPolicy,
		AckWait:       5 * time.Second,
		FilterSubject: messaging.SubjectJobSubmitted,
	})
	if err != nil {
		return fmt.Errorf("failed to create or update durable consumer %s: %w", a.consumerName, err)
	}
	a.jsConsumer = consumer
	log.Printf("[Processor] JetStream consumer bound to %s Pull subscriber (Deliver=%v, Ack=%v)", a.consumerName, deliverPolicy, ackPolicy)
	return nil
}

// unsubscribeJetStream deactivates JetStream consumer
func (a *App) unsubscribeJetStream() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.jsConsumer != nil {
		a.jsConsumer = nil
		log.Println("[Processor] JetStream consumer deactivated")
	}
}

// startWorkers spawns the configured number of JetStream pull workers (e.g. processor-1, processor-2)
func (a *App) startWorkers(ctx context.Context, attempts map[string]int, attemptsMu *sync.Mutex) {
	a.mu.Lock()
	for _, cancel := range a.workerCancels {
		cancel()
	}
	a.workerCancels = nil

	workersCount := a.consumerConfig.Workers
	if workersCount <= 0 {
		workersCount = 1
	}
	if a.consumerConfig.Ordering == "ordered" {
		workersCount = 1
	}

	cancels := make([]context.CancelFunc, 0, workersCount)
	for i := 1; i <= workersCount; i++ {
		workerName := fmt.Sprintf("processor-%d", i)
		wCtx, cancel := context.WithCancel(ctx)
		cancels = append(cancels, cancel)
		go a.jsPullLoop(wCtx, workerName, attempts, attemptsMu)
	}
	a.workerCancels = cancels
	a.mu.Unlock()

	log.Printf("[Processor] Started %d JetStream pull worker(s)", workersCount)
}

// jsPullLoop performs pull operations from JetStream stream when enabled
func (a *App) jsPullLoop(ctx context.Context, workerName string, attempts map[string]int, attemptsMu *sync.Mutex) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			a.mu.RLock()
			enabled := a.processingEnabled
			jsConsumer := a.jsConsumer
			a.mu.RUnlock()

			if !enabled || jsConsumer == nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			// Try to fetch 1 message with a short timeout
			batch, err := jsConsumer.Fetch(1, jetstream.FetchMaxWait(500*time.Millisecond))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("[%s JS] Fetch error: %v", workerName, err)
				time.Sleep(1 * time.Second)
				continue
			}

			for msg := range batch.Messages() {
				a.handleJetStreamMsg(msg, workerName, attempts, attemptsMu)
			}
		}
	}
}

// handleJetStreamMsg processes a pulled JetStream message
func (a *App) handleJetStreamMsg(msg jetstream.Msg, workerName string, attempts map[string]int, attemptsMu *sync.Mutex) {
	deliveryMode := msg.Headers().Get("X-Delivery-Mode")
	if deliveryMode == "" {
		deliveryMode = "CORE"
	}

	var job jobs.Job
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		log.Printf("[%s JS] Failed to unmarshal message: %v", workerName, err)
		_ = msg.Ack()
		return
	}

	// If this is a CORE message, discard from JetStream
	if deliveryMode == "CORE" || job.DeliveryMode == "CORE" {
		log.Printf("[%s JS] Core NATS message %s fetched. Discarding from JetStream.", workerName, job.JobID)
		_ = msg.Ack()
		return
	}

	meta, err := msg.Metadata()
	var sequence uint64
	var numDelivered uint64 = 1
	if err == nil && meta != nil {
		sequence = meta.Sequence.Stream
		numDelivered = meta.NumDelivered
	}

	attemptCount := int(numDelivered)
	if attemptCount <= 0 {
		attemptsMu.Lock()
		attempts[job.JobID]++
		attemptCount = attempts[job.JobID]
		attemptsMu.Unlock()
	}

	a.consumerMu.Lock()
	a.consumerDistribution[workerName]++
	a.consumerMu.Unlock()

	// 1. Extract Trace Context and start Consumer Receive span
	parentCtx := telemetry.ExtractTraceContext(context.Background(), msg.Headers())
	recvCtx, recvSpan := telemetry.StartSpan(parentCtx, "Consumer Receive",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation", "receive"),
			attribute.String("messaging.destination.name", messaging.SubjectJobSubmitted),
			attribute.String("messaging.consumer.name", a.consumerName),
			attribute.String("worker.id", workerName),
			attribute.String("delivery.mode", deliveryMode),
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
			attribute.Int64("delivery.count", int64(attemptCount)),
			attribute.Int64("jetstream.sequence", int64(sequence)),
			attribute.String("jetstream.stream", "JOBS"),
		),
	)
	defer recvSpan.End()

	telemetry.RecordMessageReceived(recvCtx, deliveryMode, workerName, messaging.SubjectJobSubmitted)

	if numDelivered > 1 {
		telemetry.RecordMessageRedelivered(recvCtx, deliveryMode, workerName)
		log.Printf("[%s] JetStream job %s REDELIVERED (delivery #%d)", workerName, job.JobID, attemptCount)
		a.recordWorkerEvent(job.JobID, job.Type, "[REDELIVERED]", "#FBBF24", fmt.Sprintf("Redelivery #%d from JOBS stream (Seq #%d)", attemptCount, sequence), attemptCount, deliveryMode)
	} else {
		log.Printf("[%s] Received JetStream job %s | Attempt: %d", workerName, job.JobID, attemptCount)
		a.recordWorkerEvent(job.JobID, job.Type, "[PULLED]", "#60A5FA", fmt.Sprintf("Fetched from JOBS stream (Seq #%d)", sequence), attemptCount, deliveryMode)
	}

	// Start internal Process Job Span
	procCtx, procSpan := telemetry.StartSpan(recvCtx, "Process Job",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("job.id", job.JobID),
			attribute.String("job.type", job.Type),
			attribute.String("worker.id", workerName),
			attribute.Int64("delivery.count", int64(attemptCount)),
		),
	)

	// Record PROCESSING event locally
	a.recordWorkerEvent(job.JobID, job.Type, "[PROCESSING]", "#FBBF24", "Executing business logic", attemptCount, deliveryMode)

	// 2. Simulate processing duration
	procStart := time.Now()
	time.Sleep(1 * time.Second)
	procDuration := time.Since(procStart)

	// 3. Evaluate simulated worker crash / missing ACK (AckWait timeout)
	simulateNoAck := false
	if val, ok := job.Payload["simulate_no_ack"].(bool); ok && val {
		simulateNoAck = true
	}

	if simulateNoAck && attemptCount == 1 {
		noAckReason := "Simulating worker hang / missing ACK. JetStream AckWait (5s) will trigger redelivery."
		log.Printf("[%s] JetStream Job %s: %s", workerName, job.JobID, noAckReason)

		procSpan.SetStatus(codes.Error, noAckReason)
		procSpan.SetAttributes(attribute.String("processing.result", "no_ack_simulated"))
		procSpan.End()

		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)

		a.recordWorkerEvent(job.JobID, job.Type, "[NO ACK / TIMEOUT]", "#FB923C", noAckReason, attemptCount, deliveryMode)

		// Intentionally skip msg.Ack() and msg.Nak() so JetStream AckWait timer expires and triggers redelivery
		return
	}

	// 4. Evaluate failure simulation
	simulateFailure := false
	if val, ok := job.Payload["simulate_failure"].(bool); ok && val {
		simulateFailure = true
	}

	simulateFailureCount := 0
	if val, ok := job.Payload["simulate_failure_count"].(float64); ok {
		simulateFailureCount = int(val)
	} else if val, ok := job.Payload["simulate_failure_count"].(int); ok {
		simulateFailureCount = val
	}

	maxDeliveryAttempts := 3
	if val, ok := job.Payload["max_delivery_attempts"].(float64); ok && int(val) > 0 {
		maxDeliveryAttempts = int(val)
	} else if val, ok := job.Payload["max_delivery_attempts"].(int); ok && val > 0 {
		maxDeliveryAttempts = val
	}

	nakDelaySec := 0
	if val, ok := job.Payload["nak_delay_seconds"].(float64); ok && int(val) > 0 {
		nakDelaySec = int(val)
	} else if val, ok := job.Payload["nak_delay_seconds"].(int); ok && val > 0 {
		nakDelaySec = val
	}

	if simulateFailure && (simulateFailureCount == 0 || attemptCount <= simulateFailureCount) {
		errMsg := fmt.Sprintf("Simulated failure attempt %d of %d", attemptCount, maxDeliveryAttempts)
		log.Printf("[%s] JetStream Job %s failed: %s", workerName, job.JobID, errMsg)

		// Check if message exhausted maximum delivery attempts -> route to Dead Letter Queue (JOBS_DLQ)
		if attemptCount >= maxDeliveryAttempts {
			dlqReason := fmt.Sprintf("Exhausted maximum delivery attempts (%d of %d)", attemptCount, maxDeliveryAttempts)
			log.Printf("[%s] Job %s reached max delivery attempts (%d). Routing to JOBS_DLQ.", workerName, job.JobID, maxDeliveryAttempts)

			procSpan.RecordError(fmt.Errorf("%s", dlqReason))
			procSpan.SetStatus(codes.Error, dlqReason)
			procSpan.SetAttributes(attribute.String("processing.result", "dlq_routed"))
			procSpan.End()

			telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)

			// 1. Publish failed message to JOBS_DLQ stream on subject jobs.dlq
			dlqPayload, _ := json.Marshal(map[string]any{
				"job_id":            job.JobID,
				"type":              job.Type,
				"original_subject":  messaging.SubjectJobSubmitted,
				"delivery_attempts": attemptCount,
				"failure_reason":    dlqReason,
				"timestamp":         time.Now().UTC().Format(time.RFC3339),
				"worker":            workerName,
				"payload":           job.Payload,
			})
			dlqMsg := nats.NewMsg(messaging.SubjectJobDLQ)
			dlqMsg.Data = dlqPayload
			dlqMsg.Header.Set("X-Delivery-Attempts", fmt.Sprintf("%d", attemptCount))
			dlqMsg.Header.Set("X-Original-Subject", messaging.SubjectJobSubmitted)
			dlqMsg.Header.Set("Nats-Msg-Id", fmt.Sprintf("dlq-%s-%d", job.JobID, attemptCount))
			if _, pubErr := a.natsClient.JS.PublishMsg(context.Background(), dlqMsg); pubErr != nil {
				log.Printf("[%s] Failed to publish message %s to JOBS_DLQ: %v", workerName, job.JobID, pubErr)
			}

			// 2. Record DLQ_ROUTED event locally
			a.recordWorkerEvent(job.JobID, job.Type, "[DLQ ROUTED]", "#EC4899", dlqReason, attemptCount, deliveryMode)

			// 3. Acknowledge original message from JOBS stream so it does not redeliver
			if a.consumerConfig.AckPolicy != "none" {
				if err := msg.Ack(); err != nil {
					log.Printf("[%s] Failed to ACK original message %s after DLQ routing: %v", workerName, job.JobID, err)
				}
			}
			return
		}

		// Record error on processing span
		procSpan.RecordError(fmt.Errorf("%s", errMsg))
		procSpan.SetStatus(codes.Error, errMsg)
		procSpan.SetAttributes(attribute.String("processing.result", "failure"))
		procSpan.End()

		recvSpan.AddEvent("redelivery_scheduled")
		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)

		// Check for NAK with Delay vs standard NAK
		if nakDelaySec > 0 {
			log.Printf("[%s] JetStream Job %s NAK with delay: %ds", workerName, job.JobID, nakDelaySec)
			_ = msg.NakWithDelay(time.Duration(nakDelaySec) * time.Second)

			// NakWithDelay for throttled retry
			nakReason := fmt.Sprintf("Explicit NAK sent with %ds delay (%s)", nakDelaySec, errMsg)
			a.recordWorkerEvent(job.JobID, job.Type, "[NAK SENT]", "#F87171", nakReason, attemptCount, deliveryMode)
			return
		}

		// Standard Nak for immediate redelivery
		_ = msg.Nak()
		a.recordWorkerEvent(job.JobID, job.Type, "[FAILED]", "#EF4444", errMsg, attemptCount, deliveryMode)
		return
	}

	// ACK on success
	if a.consumerConfig.AckPolicy != "none" {
		if err := msg.Ack(); err != nil {
			log.Printf("[%s] Failed to ACK message %s: %v", workerName, job.JobID, err)
		}
	}

	procSpan.SetStatus(codes.Ok, "success")
	procSpan.SetAttributes(attribute.String("processing.result", "success"))
	procSpan.End()

	recvSpan.AddEvent("message_acknowledged")

	telemetry.RecordMessageAcked(recvCtx, deliveryMode, workerName)
	telemetry.RecordJobProcessed(recvCtx, deliveryMode, workerName, "COMPLETED", procDuration)

	log.Printf("[%s] JetStream Job %s processed successfully", workerName, job.JobID)
	a.recordWorkerEvent(job.JobID, job.Type, "[COMPLETED]", "#34D399", "Execution completed successfully", attemptCount, deliveryMode)
	a.recordWorkerEvent(job.JobID, job.Type, "[ACK SENT]", "#10B981", "Explicit msg.Ack() sent to JetStream", attemptCount, deliveryMode)
}
