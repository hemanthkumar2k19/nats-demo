package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
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
func (a *App) buildCoreJobHandler(coreWorkerCounter *uint64) messaging.JobHandler {
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
		currWorkerIdx := (atomic.AddUint64(coreWorkerCounter, 1)-1)%uint64(workersCount) + 1
		assignedWorkerName := fmt.Sprintf("processor-%d", currWorkerIdx)

		a.attemptsMu.Lock()
		a.attempts[job.JobID]++
		attemptCount := a.attempts[job.JobID]
		a.attemptsMu.Unlock()

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
	log.Printf("[CONFIG] Consumer: %s | Stream: JOBS | Mode: Durable Pull", a.consumerName)
	log.Printf("[CONFIG] Ack Policy: %v | AckWait: 5s | DeliverPolicy: %v", ackPolicy, deliverPolicy)

	// Scenario 6: Log consumer state retention from JetStream broker
	if info, err := consumer.Info(ctx); err == nil && info != nil {
		log.Printf("[DURABLE STATE] Consumer '%s' connected | Ack Floor: Stream Seq %d | Outstanding ACKs: %d | Stream Backlog: %d",
			a.consumerName, info.AckFloor.Stream, info.NumAckPending, info.NumPending)
	}
	return nil
}

// unsubscribeJetStream deactivates JetStream consumer
func (a *App) unsubscribeJetStream() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.jsConsumer != nil {
		a.jsConsumer = nil
		log.Println("[CONFIG] JetStream consumer deactivated")
	}
}

// ScaleWorkers dynamically resizes the JetStream pull worker pool on the fly.
// Scale UP: Spawns only the additional workers, keeping existing workers uninterrupted.
// Scale DOWN: Stops only excess workers from the tail (e.g. 4 -> 1 stops 4, 3, 2).
func (a *App) ScaleWorkers(targetCount int) {
	if targetCount <= 0 {
		targetCount = 1
	} else if targetCount > 5 {
		targetCount = 5
	}

	a.mu.Lock()
	if a.consumerConfig.Ordering == "ordered" {
		targetCount = 1
	}
	oldCount := a.consumerConfig.Workers
	currentWorkers := len(a.workerCancels)
	a.consumerConfig.Workers = targetCount

	log.Println("================================================================================")
	log.Printf("[WORKERS] Scaling worker pool request: %d -> %d worker(s)", oldCount, targetCount)
	log.Printf("[WORKERS] Competing pull consumer: '%s' | Stream: 'JOBS'", a.consumerName)
	log.Println("================================================================================")

	if targetCount > currentWorkers {
		// Incremental Scale UP: Keep existing workers running; add new workers
		addedCount := targetCount - currentWorkers
		log.Printf("[WORKERS] Scaling UP: Keeping %d existing worker(s) running uninterrupted, adding %d new worker(s)...", currentWorkers, addedCount)
		for i := currentWorkers + 1; i <= targetCount; i++ {
			workerName := fmt.Sprintf("processor-%d", i)
			wCtx, cancel := context.WithCancel(context.Background())
			a.workerCancels = append(a.workerCancels, cancel)
			go a.jsPullLoop(wCtx, workerName)
			log.Printf("[WORKERS] Added new JetStream pull worker goroutine: %s", workerName)
		}
	} else if targetCount < currentWorkers {
		// Incremental Scale DOWN: Remove excess workers from the tail (e.g. 4 down to 1 stops 4, 3, 2)
		excessCount := currentWorkers - targetCount
		log.Printf("[WORKERS] Scaling DOWN: Stopping %d excess worker goroutine(s) from tail, keeping workers 1..%d uninterrupted...", excessCount, targetCount)
		for i := currentWorkers - 1; i >= targetCount; i-- {
			workerName := fmt.Sprintf("processor-%d", i+1)
			log.Printf("[WORKERS] Stopping excess worker goroutine: %s...", workerName)
			a.workerCancels[i]()
		}
		a.workerCancels = a.workerCancels[:targetCount]
	} else {
		log.Printf("[WORKERS] Worker pool already has %d active worker(s). No adjustment needed.", targetCount)
	}

	a.activeGoroutines = len(a.workerCancels)
	a.goroutineStatus = "RUNNING"
	a.crashedWorker = ""
	a.mu.Unlock()

	log.Printf("[WORKERS] Current worker state: %d active worker goroutines", targetCount)
}

// startWorkers spawns the configured number of JetStream pull workers (e.g. processor-1, processor-2)
func (a *App) startWorkers(ctx context.Context) {
	a.mu.Lock()
	oldWorkers := len(a.workerCancels)
	if oldWorkers > 0 {
		log.Printf("[WORKERS] Stopping %d active worker goroutine(s)...", oldWorkers)
		for _, cancel := range a.workerCancels {
			cancel()
		}
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
		go a.jsPullLoop(wCtx, workerName)
	}
	a.workerCancels = cancels
	a.activeGoroutines = workersCount
	a.goroutineStatus = "RUNNING"
	a.crashedWorker = ""
	a.mu.Unlock()

	log.Printf("[WORKERS] Initializing %d JetStream pull worker goroutine(s) for consumer '%s'...", workersCount, a.consumerName)
}

// jsPullLoop performs pull operations from JetStream stream when enabled
func (a *App) jsPullLoop(ctx context.Context, workerName string) {
	log.Printf("[%s] Worker goroutine READY and polling consumer '%s'", workerName, a.consumerName)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[%s] Worker goroutine stopped (context cancelled)", workerName)
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

			a.scenarioMu.RLock()
			currentScenario := a.activeScenario
			scenarioOnce := a.scenarioOnce
			a.scenarioMu.RUnlock()

			if currentScenario == "out_of_order_ack" {
				// Step 5 Demo: Pull a batch of 3 messages to demonstrate Out-of-Order execution & Hole Safety
				batch, err := jsConsumer.Fetch(3, jetstream.FetchMaxWait(2*time.Second))
				if err != nil {
					if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
						continue
					}
					log.Printf("[%s] Fetch notice: %v", workerName, err)
					time.Sleep(200 * time.Millisecond)
					continue
				}

				var msgs []jetstream.Msg
				for msg := range batch.Messages() {
					msgs = append(msgs, msg)
				}

				if len(msgs) == 3 {
					a.handleOutOfOrderBatch(msgs, workerName)
					if scenarioOnce {
						a.scenarioMu.Lock()
						a.activeScenario = "normal"
						a.scenarioMu.Unlock()
					}
					continue
				} else {
					for _, msg := range msgs {
						shouldTerminate := a.handleJetStreamMsg(msg, workerName)
						if shouldTerminate {
							log.Printf("[%s] Pull loop TERMINATING due to simulated crash.", workerName)
							return
						}
					}
					continue
				}
			}

			// Try to fetch 1 message with a short timeout
			batch, err := jsConsumer.Fetch(1, jetstream.FetchMaxWait(500*time.Millisecond))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("[%s] Fetch notice: %v", workerName, err)
				a.mu.Lock()
				a.jsConsumer = nil
				a.mu.Unlock()
				_ = a.subscribeJetStream()
				time.Sleep(1 * time.Second)
				continue
			}

			for msg := range batch.Messages() {
				shouldTerminate := a.handleJetStreamMsg(msg, workerName)
				if shouldTerminate {
					log.Printf("[%s] Pull loop TERMINATING due to simulated crash.", workerName)
					return
				}
			}
		}
	}
}

// handleJetStreamMsg processes a pulled JetStream message.
// Returns true if the worker goroutine must terminate immediately (crash simulation).
func (a *App) handleJetStreamMsg(msg jetstream.Msg, workerName string) bool {
	deliveryMode := msg.Headers().Get("X-Delivery-Mode")
	if deliveryMode == "" {
		deliveryMode = "CORE"
	}

	var job jobs.Job
	if err := json.Unmarshal(msg.Data(), &job); err != nil {
		log.Printf("[%s] Failed to unmarshal message: %v", workerName, err)
		_ = msg.Ack()
		return false
	}

	// If this is a CORE message, discard from JetStream
	if deliveryMode == "CORE" || job.DeliveryMode == "CORE" {
		log.Printf("[%s] Core NATS message %s fetched. Discarding from JetStream.", workerName, job.JobID)
		_ = msg.Ack()
		return false
	}

	meta, err := msg.Metadata()
	var sequence uint64
	var numDelivered uint64 = 1
	if err == nil && meta != nil {
		sequence = meta.Sequence.Stream
		numDelivered = meta.NumDelivered
	}

	attemptCount := int(numDelivered)

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
		wLoad, poolSummary := a.recordWorkerPull(workerName)
		log.Println("================================================================================")
		log.Printf("[%s] [REDELIVERED] JetStream job %s REDELIVERED by server (Delivery attempt #%d, Stream Seq #%d) | Worker load: %d (Pool: [%s])", workerName, job.JobID, attemptCount, sequence, wLoad, poolSummary)
		a.mu.RLock()
		crashed := a.crashedWorker
		a.mu.RUnlock()
		if crashed != "" && crashed != workerName {
			log.Printf("[%s] [FAILOVER ACTIVE] Surviving healthy worker %s picked up redelivered job from crashed worker %s!", workerName, workerName, crashed)
		} else if crashed == "" {
			log.Printf("[%s] [REDELIVERY TAKEOVER] Competing worker %s picked up redelivered job (peer worker exceeded AckWait)!", workerName, workerName)
		}
		log.Printf("[%s] [RETRY EXECUTION] Worker processing redelivered message...", workerName)
		log.Println("================================================================================")
		a.recordWorkerEvent(job.JobID, job.Type, "[REDELIVERED]", "#FBBF24", fmt.Sprintf("Redelivery #%d from JOBS stream (Seq #%d)", attemptCount, sequence), attemptCount, deliveryMode)
	} else {
		wLoad, poolSummary := a.recordWorkerPull(workerName)
		log.Printf("[%s] [PULLED] %s | Stream Seq: %d | Delivery: #%d | Worker load: %d (Pool: [%s])", workerName, job.JobID, sequence, attemptCount, wLoad, poolSummary)
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
	log.Printf("[%s] [PROCESSING] %s | Executing business transaction...", workerName, job.JobID)
	a.recordWorkerEvent(job.JobID, job.Type, "[PROCESSING]", "#FBBF24", "Executing business logic", attemptCount, deliveryMode)

	// 2. Simulate processing duration
	procStart := time.Now()
	time.Sleep(1 * time.Second)
	procDuration := time.Since(procStart)

	// 3. Evaluate failure scenarios (Scenario 1: Goroutine Crash Before ACK, Scenario 2: Exceed AckWait)
	a.scenarioMu.RLock()
	currentScenario := a.activeScenario
	scenarioOnce := a.scenarioOnce
	a.scenarioMu.RUnlock()

	// Check if scenario is triggered via payload override as well
	if val, ok := job.Payload["scenario"].(string); ok && val != "" {
		currentScenario = val
	}
	if val, ok := job.Payload["simulate_crash"].(bool); ok && val {
		currentScenario = "crash_before_ack"
	}
	if val, ok := job.Payload["simulate_no_ack"].(bool); ok && val {
		currentScenario = "crash_before_ack"
	}
	if val, ok := job.Payload["simulate_nak"].(bool); ok && val {
		currentScenario = "nak_message"
	}
	if val, ok := job.Payload["simulate_term"].(bool); ok && val {
		currentScenario = "term_message"
	}
	if val, ok := job.Payload["simulate_out_of_order"].(bool); ok && val {
		currentScenario = "out_of_order_ack"
	}

	// Scenario 1: Worker Goroutine Crash Before ACK
	if currentScenario == "crash_before_ack" && attemptCount == 1 {
		if scenarioOnce {
			a.scenarioMu.Lock()
			a.activeScenario = "normal"
			a.scenarioMu.Unlock()
		}

		a.mu.Lock()
		if a.activeGoroutines > 0 {
			a.activeGoroutines--
		}
		totalWorkers := len(a.workerCancels)
		if totalWorkers == 0 {
			totalWorkers = 1
		}
		a.crashedWorker = workerName
		if a.activeGoroutines > 0 {
			a.goroutineStatus = "DEGRADED"
		} else {
			a.goroutineStatus = "CRASHED"
		}
		activeRemaining := a.activeGoroutines
		a.mu.Unlock()

		log.Println("================================================================================")
		log.Printf("[FATAL GOROUTINE CRASH] Simulating unhandled panic in worker goroutine %s!", workerName)
		log.Printf("panic: runtime error: worker %s crashed abruptly before sending ACK (job: %s)", workerName, job.JobID)
		log.Println("")
		log.Println("goroutine [running]:")
		log.Printf("main.(*App).handleJetStreamMsg(...)")
		log.Printf("    backend/services/cmd/processor-service/worker.go")
		log.Printf("main.(*App).jsPullLoop(...)")
		log.Printf("    backend/services/cmd/processor-service/worker.go")
		log.Printf("created by main.(*App).startWorkers")
		log.Printf("    backend/services/cmd/processor-service/worker.go")
		log.Println("================================================================================")
		log.Printf("[CRASH DETECTED] Worker goroutine %s DIED. Pull loop TERMINATED.", workerName)
		log.Printf("[POOL STATUS] %d/%d active worker(s) remaining alive.", activeRemaining, totalWorkers)
		if activeRemaining > 0 {
			log.Printf("[POOL STATUS] Surviving healthy workers continue polling durable consumer '%s' uninterrupted.", a.consumerName)
		}
		log.Printf("[BROKER STATE] No ACK transmitted. JetStream AckWait (5s) countdown active on server.")
		log.Printf("[SUPERVISOR] Scheduled automatic goroutine respawn for %s in 5.5s (post-AckWait)...", workerName)

		crashReason := fmt.Sprintf("Worker goroutine %s panicked before sending ACK. JetStream AckWait (5s) ticking...", workerName)
		procSpan.SetStatus(codes.Error, crashReason)
		procSpan.SetAttributes(attribute.String("processing.result", "goroutine_crashed"))
		procSpan.End()

		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)
		a.recordWorkerEvent(job.JobID, job.Type, "[GOROUTINE CRASH]", "#EF4444", crashReason, attemptCount, deliveryMode)

		// Spawn supervisor timer to revive goroutine after AckWait expires
		go func(wName string, jID, jType, dMode string, att int) {
			time.Sleep(5500 * time.Millisecond)
			log.Println("================================================================================")
			log.Printf("[SUPERVISOR] Respawning worker goroutine %s for consumer '%s'...", wName, a.consumerName)
			log.Println("================================================================================")
			a.mu.Lock()
			a.activeGoroutines++
			total := len(a.workerCancels)
			if a.activeGoroutines >= total {
				a.goroutineStatus = "RUNNING"
				a.crashedWorker = ""
			}
			wCtx, cancel := context.WithCancel(context.Background())
			var idx int
			if _, err := fmt.Sscanf(wName, "processor-%d", &idx); err == nil && idx >= 1 && idx <= len(a.workerCancels) {
				a.workerCancels[idx-1] = cancel
			} else {
				a.workerCancels = append(a.workerCancels, cancel)
			}
			a.mu.Unlock()
			a.recordWorkerEvent(jID, jType, "[SUPERVISOR RESPAWN]", "#8B5CF6", fmt.Sprintf("Supervisor revived %s after AckWait expiry", wName), att, dMode)

			// Launch brand new worker goroutine!
			go a.jsPullLoop(wCtx, wName)
		}(workerName, job.JobID, job.Type, deliveryMode, attemptCount)

		// Intentionally exit without msg.Ack() or msg.Nak() AND signal jsPullLoop to terminate
		return true
	}

	// Scenario 2: Processing Exceeds AckWait (Slow processing, 7s duration > 5s AckWait)
	if currentScenario == "exceed_ack_wait" && attemptCount == 1 {
		if scenarioOnce {
			a.scenarioMu.Lock()
			a.activeScenario = "normal"
			a.scenarioMu.Unlock()
		}

		slowReason := fmt.Sprintf("Worker %s executing slow transaction (7s duration, exceeds 5s AckWait threshold)...", workerName)
		log.Println("================================================================================")
		log.Printf("[%s] [SLOW WORKER] %s | %s", workerName, job.JobID, slowReason)
		log.Printf("[BROKER NOTICE] JetStream server AckWait timer (5s) will expire while %s is still processing!", workerName)
		a.mu.RLock()
		totalWorkers := len(a.workerCancels)
		a.mu.RUnlock()
		if totalWorkers > 1 {
			log.Printf("[POOL STATUS] Surviving healthy workers continue polling consumer '%s' concurrently.", a.consumerName)
		}
		log.Println("================================================================================")

		a.recordWorkerEvent(job.JobID, job.Type, "[SLOW PROCESSING]", "#F59E0B", slowReason, attemptCount, deliveryMode)

		// Intentionally delay for 7 seconds
		time.Sleep(7 * time.Second)

		// Send late ACK after delay
		if a.consumerConfig.AckPolicy != "none" {
			if err := msg.Ack(); err != nil {
				log.Printf("[%s] [LATE ACK] Notice (message likely acknowledged by peer worker): %v", workerName, err)
			} else {
				log.Printf("[%s] [LATE ACK SENT] %s | Late explicit msg.Ack() transmitted to JetStream. | Pool: [%s]", workerName, job.JobID, a.getLoadDistributionSummary())
			}
		}

		procSpan.SetStatus(codes.Ok, "slow_execution_completed")
		procSpan.SetAttributes(attribute.String("processing.result", "slow_execution_success"))
		procSpan.End()

		log.Printf("[%s] [COMPLETED] %s | Slow transaction finished after 7.0s.", workerName, job.JobID)
		a.recordWorkerEvent(job.JobID, job.Type, "[COMPLETED]", "#34D399", "Slow execution finished after 7s", attemptCount, deliveryMode)
		a.recordWorkerEvent(job.JobID, job.Type, "[LATE ACK SENT]", "#10B981", "Sent late msg.Ack() after 7s delay", attemptCount, deliveryMode)
		return false
	}

	// Scenario 4: Worker NAKs Message (msg.Nak())
	if currentScenario == "nak_message" && attemptCount == 1 {
		if scenarioOnce {
			a.scenarioMu.Lock()
			a.activeScenario = "normal"
			a.scenarioMu.Unlock()
		}

		nakReason := fmt.Sprintf("Simulating worker processing failure on attempt #%d -> sending explicit msg.Nak()", attemptCount)
		log.Println("================================================================================")
		log.Printf("[%s] [NAK TRIGGERED] %s | %s", workerName, job.JobID, nakReason)
		log.Printf("[%s] [NAK SENT] Explicit msg.Nak() transmitted to JetStream broker", workerName)
		log.Printf("[BROKER STATE] NATS server marks message unacknowledged. Redelivery scheduled immediately.")
		log.Println("================================================================================")

		procSpan.SetStatus(codes.Error, "simulated_nak_failure")
		procSpan.SetAttributes(attribute.String("processing.result", "nak_sent"))
		procSpan.End()

		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)
		a.recordWorkerEvent(job.JobID, job.Type, "[NAK SENT]", "#F87171", "Explicit msg.Nak() sent. Broker redelivery incoming.", attemptCount, deliveryMode)

		if err := msg.Nak(); err != nil {
			log.Printf("[%s] Error transmitting NAK: %v", workerName, err)
		}
		return false
	}

	// Scenario 5: Worker Terminates Message (msg.Term())
	if currentScenario == "term_message" {
		if scenarioOnce {
			a.scenarioMu.Lock()
			a.activeScenario = "normal"
			a.scenarioMu.Unlock()
		}

		termReason := fmt.Sprintf("Poison message condition detected on job %s -> sending terminal ACK (msg.Term())", job.JobID)
		log.Println("================================================================================")
		log.Printf("[%s] [TERM TRIGGERED] %s | %s", workerName, job.JobID, termReason)
		log.Printf("[%s] [TERM SENT] Explicit msg.Term() sent. Message permanently terminated by broker.", workerName)
		log.Printf("[BROKER STATE] NATS server terminates message. Message will NEVER be redelivered. ACK floor advances.")
		log.Println("================================================================================")

		procSpan.SetStatus(codes.Error, "poison_message_terminated")
		procSpan.SetAttributes(attribute.String("processing.result", "term_sent"))
		procSpan.End()

		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)
		a.recordWorkerEvent(job.JobID, job.Type, "[TERM SENT]", "#EF4444", "Explicit msg.Term() sent. Message permanently terminated without redelivery.", attemptCount, deliveryMode)

		if err := msg.Term(); err != nil {
			log.Printf("[%s] Error transmitting Term: %v", workerName, err)
		}
		return false
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
			return false
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
			return false
		}

		// Standard Nak for immediate redelivery
		_ = msg.Nak()
		a.recordWorkerEvent(job.JobID, job.Type, "[FAILED]", "#EF4444", errMsg, attemptCount, deliveryMode)
		return false
	}

	// 4. Processing completed successfully
	procSpan.SetStatus(codes.Ok, "success")
	procSpan.SetAttributes(attribute.String("processing.result", "success"))
	procSpan.End()

	log.Printf("[%s] [COMPLETED] %s | Transaction finished successfully (took %v)", workerName, job.JobID, procDuration)
	a.recordWorkerEvent(job.JobID, job.Type, "[COMPLETED]", "#34D399", "Execution completed successfully", attemptCount, deliveryMode)

	// 5. Send explicit ACK to JetStream
	if a.consumerConfig.AckPolicy != "none" {
		if err := msg.Ack(); err != nil {
			log.Printf("[%s] Failed to ACK message %s: %v", workerName, job.JobID, err)
		} else {
			log.Printf("[%s] [ACK SENT] %s | Explicit msg.Ack() confirmed to JetStream | Pool: [%s]", workerName, job.JobID, a.getLoadDistributionSummary())
		}
	}

	recvSpan.AddEvent("message_acknowledged")
	telemetry.RecordMessageAcked(recvCtx, deliveryMode, workerName)
	telemetry.RecordJobProcessed(recvCtx, deliveryMode, workerName, "COMPLETED", procDuration)
	a.recordWorkerEvent(job.JobID, job.Type, "[ACK SENT]", "#10B981", "Explicit msg.Ack() sent to JetStream", attemptCount, deliveryMode)
	return false
}

// getLoadDistributionSummary formats the active worker pool load distribution for JetStream workers.
func (a *App) getLoadDistributionSummary() string {
	a.mu.RLock()
	totalWorkers := len(a.workerCancels)
	a.mu.RUnlock()
	if totalWorkers == 0 {
		totalWorkers = 1
	}

	a.consumerDistMu.Lock()
	defer a.consumerDistMu.Unlock()

	maxIdx := totalWorkers
	for i := 1; i <= 5; i++ {
		w := fmt.Sprintf("processor-%d", i)
		if a.consumerDistribution[w] > 0 && i > maxIdx {
			maxIdx = i
		}
	}

	var parts []string
	for i := 1; i <= maxIdx; i++ {
		w := fmt.Sprintf("processor-%d", i)
		parts = append(parts, fmt.Sprintf("%s=%d", w, a.consumerDistribution[w]))
	}
	return strings.Join(parts, ", ")
}

// recordWorkerPull increments the worker's pull count and returns its count along with the pool summary.
func (a *App) recordWorkerPull(workerName string) (int, string) {
	a.consumerDistMu.Lock()
	if a.consumerDistribution == nil {
		a.consumerDistribution = make(map[string]int)
	}
	a.consumerDistribution[workerName]++
	count := a.consumerDistribution[workerName]
	a.consumerDistMu.Unlock()

	return count, a.getLoadDistributionSummary()
}

// handleOutOfOrderBatch demonstrates JetStream Out-of-Order ACK and Hole Safety (Step 5 / Scenario 3).
// 1. Message 1 is ACKed immediately -> AckFloor advances to Msg 1 sequence
// 2. Message 2 is held pending for 10 seconds -> creates a sequence hole in JetStream
// 3. Message 3 is ACKed out-of-order -> AckFloor stays pinned at Msg 1 sequence; JetStream records Msg 3 in pending bitset
// 4. After 10s, Message 2 ACKs -> Gap is healed! AckFloor cascades forward to Msg 3 sequence
func (a *App) handleOutOfOrderBatch(msgs []jetstream.Msg, workerName string) {
	if len(msgs) != 3 {
		return
	}

	type msgInfo struct {
		msg      jetstream.Msg
		job      jobs.Job
		seq      uint64
		delMode  string
		attempts int
	}

	var parsed [3]msgInfo
	for i := 0; i < 3; i++ {
		parsed[i].msg = msgs[i]
		parsed[i].delMode = msgs[i].Headers().Get("X-Delivery-Mode")
		if parsed[i].delMode == "" {
			parsed[i].delMode = "JETSTREAM"
		}
		_ = json.Unmarshal(msgs[i].Data(), &parsed[i].job)
		if meta, err := msgs[i].Metadata(); err == nil && meta != nil {
			parsed[i].seq = meta.Sequence.Stream
			parsed[i].attempts = int(meta.NumDelivered)
		}
		if parsed[i].attempts == 0 {
			parsed[i].attempts = 1
		}
	}

	seq1, seq2, seq3 := parsed[0].seq, parsed[1].seq, parsed[2].seq
	job1, job2, job3 := parsed[0].job, parsed[1].job, parsed[2].job

	log.Println("================================================================================")
	log.Printf("[%s] [OUT-OF-ORDER BATCH] Fetched batch of 3 messages: Msg 1 (Seq %d, %s), Msg 2 (Seq %d, %s), Msg 3 (Seq %d, %s)",
		workerName, seq1, job1.JobID, seq2, job2.JobID, seq3, job3.JobID)
	log.Printf("[DEMO RULE] NATS Contiguous AckFloor: AckFloor cannot skip unacknowledged gaps!")
	log.Println("================================================================================")

	// Msg 1: Process and ACK immediately
	a.recordWorkerEvent(job1.JobID, job1.Type, "[PROCESSING]", "#FBBF24", "Msg 1/3 in out-of-order batch", parsed[0].attempts, parsed[0].delMode)
	time.Sleep(500 * time.Millisecond)
	if err := parsed[0].msg.Ack(); err != nil {
		log.Printf("[%s] Error ACKing Msg 1: %v", workerName, err)
	} else {
		log.Printf("[%s] [ACK SENT] Msg 1/3 (Stream Seq %d, Job %s) ACKed. AckFloor advances to %d.", workerName, seq1, job1.JobID, seq1)
		a.recordWorkerEvent(job1.JobID, job1.Type, "[ACK SENT]", "#10B981", fmt.Sprintf("Msg 1 ACKed -> AckFloor advances to Seq %d", seq1), parsed[0].attempts, parsed[0].delMode)
	}

	// Verify broker state after Msg 1 ACK
	a.mu.RLock()
	consumer := a.jsConsumer
	a.mu.RUnlock()
	if consumer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if cinfo, err := consumer.Info(ctx); err == nil && cinfo != nil {
			log.Printf("[BROKER STATE] AckFloor: Stream Seq %d | Outstanding ACKs: %d", cinfo.AckFloor.Stream, cinfo.NumAckPending)
		}
		cancel()
	}

	// Msg 2: Spawn background goroutine to hold execution for 10 seconds (hole creation)
	a.recordWorkerEvent(job2.JobID, job2.Type, "[HOLE CREATED]", "#F59E0B", fmt.Sprintf("Msg 2 held pending (10s delay) -> sequence hole at Seq %d", seq2), parsed[1].attempts, parsed[1].delMode)
	log.Printf("[%s] [HOLE CREATED] Msg 2/3 (Stream Seq %d, Job %s) DELAYED 10s! Unacknowledged hole active in broker.", workerName, seq2, job2.JobID)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Sleep for 10 seconds with msg.InProgress() heartbeats so the 5s AckWait timer does not expire
		time.Sleep(4 * time.Second)
		_ = parsed[1].msg.InProgress()
		time.Sleep(4 * time.Second)
		_ = parsed[1].msg.InProgress()
		time.Sleep(2 * time.Second)

		log.Println("--------------------------------------------------------------------------------")
		log.Printf("[%s] [GAP HEALING] 10s delay expired for Msg 2/3 (Stream Seq %d, Job %s). Sending explicit msg.Ack()...", workerName, seq2, job2.JobID)
		if err := parsed[1].msg.Ack(); err != nil {
			log.Printf("[%s] Error ACKing Msg 2: %v", workerName, err)
		} else {
			log.Printf("[%s] [GAP HEALED] Msg 2/3 ACKed! Gap at Seq %d resolved. AckFloor cascades up to %d!", workerName, seq2, seq3)
			a.recordWorkerEvent(job2.JobID, job2.Type, "[GAP HEALED]", "#34D399", fmt.Sprintf("Msg 2 ACKed -> Gap healed! AckFloor cascades to Seq %d", seq3), parsed[1].attempts, parsed[1].delMode)
		}

		if consumer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			if cinfo, err := consumer.Info(ctx); err == nil && cinfo != nil {
				log.Printf("[BROKER STATE] AckFloor: Stream Seq %d | Outstanding ACKs: %d (Gap fully resolved)", cinfo.AckFloor.Stream, cinfo.NumAckPending)
			}
			cancel()
		}
		log.Println("--------------------------------------------------------------------------------")
	}()

	// Msg 3: Process and ACK after a short delay (while Msg 2 is still pending!)
	time.Sleep(1 * time.Second)
	log.Printf("[%s] [OUT-OF-ORDER ACK] Transmitting explicit msg.Ack() for Msg 3/3 (Stream Seq %d, Job %s) while Msg 2 (Seq %d) is still pending!",
		workerName, seq3, job3.JobID, seq2)
	if err := parsed[2].msg.Ack(); err != nil {
		log.Printf("[%s] Error ACKing Msg 3: %v", workerName, err)
	} else {
		log.Printf("[%s] [ACKFLOOR PINNED] Msg 3/3 ACKed! Broker keeps AckFloor pinned at %d. JetStream records Seq %d in pending bitset.",
			workerName, seq1, seq3)
		a.recordWorkerEvent(job3.JobID, job3.Type, "[OUT-OF-ORDER ACK]", "#60A5FA", fmt.Sprintf("Msg 3 ACKed out-of-order! AckFloor stays pinned at Seq %d (hole at %d)", seq1, seq2), parsed[2].attempts, parsed[2].delMode)
	}

	if consumer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		if cinfo, err := consumer.Info(ctx); err == nil && cinfo != nil {
			log.Printf("[BROKER STATE] AckFloor: Stream Seq %d | Outstanding ACKs: %d (Hole at Seq %d prevents AckFloor advancement!)",
				cinfo.AckFloor.Stream, cinfo.NumAckPending, seq2)
		}
		cancel()
	}

	// Wait for Msg 2 gap healing goroutine to finish before exiting batch handler
	wg.Wait()
}
