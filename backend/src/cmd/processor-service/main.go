package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"nats-demo/internal/config"
	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// App manages the lifecycle of the processor service.
type App struct {
	mu                sync.RWMutex
	cfg               *config.Config
	natsClient        *natsclient.Client
	consumer          *messaging.Consumer
	publisher         *messaging.Publisher
	sub               *nats.Subscription
	valSub            *nats.Subscription
	statusSub         *nats.Subscription
	stateSetSub       *nats.Subscription
	consumerConfigSub *nats.Subscription
	jsSub             *nats.Subscription
	processingEnabled bool
	consumerConfig    jobs.ConsumerConfig
	consumerName      string
	workerCancels     []context.CancelFunc
	otelShutdown      func(context.Context) error

	// JetStream Consumer distribution tracking
	consumerMu           sync.Mutex
	consumerDistribution map[string]int
	consumerResetSub     *nats.Subscription

	// Core NATS Queue Group state
	queueMu           sync.Mutex
	queueWorkersCount int
	queueSubs         []*nats.Subscription
	queueDistribution map[string]int
	queueConfigSub    *nats.Subscription
	queueStatusSub    *nats.Subscription
	queueResetSub     *nats.Subscription
}

// Init loads configuration, connects to NATS, and instantiates components.
func (a *App) Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg
	log.Printf("[Init] Loaded configuration: NATS_URL=%s", a.cfg.NATSURL)

	// Initialize OpenTelemetry metric pipeline for processor-service
	otelShutdown, err := telemetry.Init(context.Background(), "processor-service", a.cfg.OtelEndpoint, a.cfg.OtelInsecure)
	if err != nil {
		log.Printf("[Init] Telemetry warning: %v", err)
	}
	a.otelShutdown = otelShutdown

	client, err := natsclient.Connect(a.cfg.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	a.natsClient = client
	log.Println("[Init] Connected to NATS wrapper client")

	// Ensure jobs stream and durable consumer exist
	if err := a.natsClient.EnsureJobsStream(); err != nil {
		log.Printf("[Init] Warning: EnsureJobsStream failed (NATS server may not be ready): %v", err)
	}

	a.consumer = messaging.NewConsumer(a.natsClient)
	a.publisher = messaging.NewPublisher(a.natsClient)
	a.processingEnabled = true
	a.consumerConfig = jobs.ConsumerConfig{
		Type:     "durable",
		Workers:  1,
		Ordering: "normal",
	}
	a.consumerName = "job-processor"

	// Initialize Core NATS Queue Group state (1 worker by default)
	a.queueWorkersCount = 1
	a.queueSubs = make([]*nats.Subscription, 0)
	a.queueDistribution = map[string]int{
		"processor-1": 0,
		"processor-2": 0,
		"processor-3": 0,
		"processor-4": 0,
		"processor-5": 0,
	}

	// Initialize JetStream Consumer distribution tracking
	a.consumerDistribution = map[string]int{
		"processor-1": 0,
		"processor-2": 0,
		"processor-3": 0,
		"processor-4": 0,
		"processor-5": 0,
	}

	return nil
}

// Run starts the subscriptions and blocks until interrupt signal.
func (a *App) Run() error {
	workerName := os.Getenv("PROCESSOR_NAME")
	if workerName == "" {
		workerName = "processor-1"
	}
	log.Printf("[Run] Starting processor instance: %s", workerName)

	var attemptsMu sync.Mutex
	attempts := make(map[string]int)
	var coreWorkerCounter uint64

	// Core NATS job processing handler
	jobHandler := func(ctx context.Context, job jobs.Job) error {
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
		currWorkerIdx := (atomic.AddUint64(&coreWorkerCounter, 1) - 1) % uint64(workersCount) + 1
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

		// Record message received metric
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

			// Record error on processing span
			procSpan.RecordError(fmt.Errorf("%s", errMsg))
			procSpan.SetStatus(codes.Error, errMsg)
			procSpan.SetAttributes(attribute.String("processing.result", "failure"))
			procSpan.End()

			// Record job failure metric
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

		// Record job processed metric
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

	// Default: Subscribe to Core NATS
	if err := a.subscribeCore(workerName, jobHandler); err != nil {
		return fmt.Errorf("failed to subscribe to Core NATS: %w", err)
	}

	// Initialize and Subscribe to JetStream Pull consumer
	if err := a.subscribeJetStream(); err != nil {
		log.Printf("[Run] Warning: JetStream Pull subscription failed (JOBS stream may not exist yet): %v", err)
	}

	// Run JetStream Pull Workers (multi-worker competing pool)
	a.startWorkers(context.Background(), attempts, &attemptsMu)

	// Subscribe to jobs.validate Request/Reply.
	// subscribeValidation manages the subscription lifecycle when the processor toggles.
	if err := a.subscribeValidation(workerName); err != nil {
		return fmt.Errorf("failed to subscribe to validation subject: %w", err)
	}
	log.Printf("[Run] Subscribed to validation subject: %s", messaging.SubjectJobValidate)

	// Subscribe to status ping responder
	statusSub, err := a.natsClient.Conn.Subscribe("status.processor", func(msg *nats.Msg) {
		a.mu.RLock()
		enabled := a.processingEnabled
		cType := a.consumerConfig.Type
		cName := a.consumerName
		cWorkers := a.consumerConfig.Workers
		cOrdering := a.consumerConfig.Ordering
		a.mu.RUnlock()

		a.consumerMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.consumerDistribution {
			distCopy[k] = v
		}
		a.consumerMu.Unlock()
		distJSON, _ := json.Marshal(distCopy)

		respBytes := []byte(fmt.Sprintf(`{"status":"ACTIVE","processing":%t,"consumer_type":"%s","consumer_name":"%s","workers":%d,"ordering":"%s","distribution":%s}`,
			enabled, cType, cName, cWorkers, cOrdering, string(distJSON)))
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Processor] Failed to send status reply: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to status responder: %w", err)
	}
	a.statusSub = statusSub
	log.Println("[Run] Subscribed to status responder subject: status.processor")

	// Subscribe to consumer configuration control subject
	consumerConfigSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectConsumerConfigSet, func(msg *nats.Msg) {
		var req jobs.ConsumerConfig
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Processor] Failed to unmarshal consumer config payload: %v", err)
			_ = msg.Respond([]byte(`{"error":"Invalid consumer config payload"}`))
			return
		}

		if req.Type == "" {
			req.Type = "durable"
		}
		if req.Workers <= 0 {
			req.Workers = 1
		} else if req.Workers > 5 {
			req.Workers = 5
		}
		if req.Ordering == "" {
			req.Ordering = "normal"
		}
		if req.Ordering == "ordered" {
			req.Workers = 1
		}

		a.mu.Lock()
		a.consumerConfig = req
		if req.Type == "ephemeral" {
			a.consumerName = fmt.Sprintf("ephemeral-%d", time.Now().UnixNano()%100000)
		} else {
			a.consumerName = "job-processor"
		}
		a.mu.Unlock()

		// Re-subscribe JetStream with new consumer configuration
		a.unsubscribeJetStream()
		if err := a.subscribeJetStream(); err != nil {
			log.Printf("[Processor] Re-subscribing JetStream consumer failed: %v", err)
		}

		// Restart workers matching the new worker count
		a.startWorkers(context.Background(), attempts, &attemptsMu)

		var pending uint64
		var ackPending, redelivered int
		js, err := a.natsClient.Conn.JetStream()
		if err == nil {
			cinfo, err := js.ConsumerInfo("JOBS", a.consumerName)
			if err == nil && cinfo != nil {
				pending = cinfo.NumPending
				ackPending = cinfo.NumAckPending
				redelivered = cinfo.NumRedelivered
			}
		}

		statusVal := "ACTIVE"
		if !a.processingEnabled {
			statusVal = "STOPPED"
		}

		a.consumerMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.consumerDistribution {
			distCopy[k] = v
		}
		a.consumerMu.Unlock()

		resp := jobs.ConsumerStatusResponse{
			Name:         a.consumerName,
			Type:         req.Type,
			Workers:      req.Workers,
			Ordering:     req.Ordering,
			Delivery:     "at-least-once",
			Status:       statusVal,
			Pending:      pending,
			AckPending:   ackPending,
			Redelivered:  redelivered,
			Distribution: distCopy,
		}
		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] Consumer reconfigured: Type=%s, Name=%s, Workers=%d, Ordering=%s",
			req.Type, a.consumerName, req.Workers, req.Ordering)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to consumer config subject: %w", err)
	}
	a.consumerConfigSub = consumerConfigSub
	log.Printf("[Run] Subscribed to consumer config responder subject: %s", messaging.SubjectConsumerConfigSet)

	// Subscribe to consumer reset responder subject
	consumerResetSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectConsumerReset, func(msg *nats.Msg) {
		a.consumerMu.Lock()
		a.consumerDistribution = make(map[string]int)
		for i := 1; i <= 5; i++ {
			a.consumerDistribution[fmt.Sprintf("processor-%d", i)] = 0
		}
		distCopy := make(map[string]int)
		for k, v := range a.consumerDistribution {
			distCopy[k] = v
		}
		a.consumerMu.Unlock()

		a.mu.RLock()
		statusVal := "ACTIVE"
		if !a.processingEnabled {
			statusVal = "STOPPED"
		}
		cName := a.consumerName
		cType := a.consumerConfig.Type
		wCount := a.consumerConfig.Workers
		ordering := a.consumerConfig.Ordering
		a.mu.RUnlock()

		var pending uint64
		var ackPending, redelivered int
		if js, err := a.natsClient.Conn.JetStream(); err == nil {
			if cinfo, err := js.ConsumerInfo("JOBS", cName); err == nil && cinfo != nil {
				pending = cinfo.NumPending
				ackPending = cinfo.NumAckPending
				redelivered = cinfo.NumRedelivered
			}
		}

		resp := jobs.ConsumerStatusResponse{
			Name:         cName,
			Type:         cType,
			Workers:      wCount,
			Ordering:     ordering,
			Delivery:     "at-least-once",
			Status:       statusVal,
			Pending:      pending,
			AckPending:   ackPending,
			Redelivered:  redelivered,
			Distribution: distCopy,
		}
		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] JetStream consumer distribution reset")
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to consumer reset subject: %w", err)
	}
	a.consumerResetSub = consumerResetSub
	log.Printf("[Run] Subscribed to consumer reset responder subject: %s", messaging.SubjectConsumerReset)

	// Subscribe to processor state control subject
	stateSetSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectProcessorStateSet, func(msg *nats.Msg) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Processor] Failed to unmarshal state set payload: %v", err)
			_ = msg.Respond([]byte(`{"error":"Invalid payload"}`))
			return
		}

		a.mu.Lock()
		a.processingEnabled = req.Enabled
		a.mu.Unlock()

		statusVal := "STOPPED"
		if req.Enabled {
			statusVal = "RUNNING"
			if err := a.subscribeCore(workerName, jobHandler); err != nil {
				log.Printf("[Processor] Core NATS subscription failed on toggle: %v", err)
			}
			// Trigger JetStream subscription try if missing.
			if err := a.subscribeJetStream(); err != nil {
				log.Printf("[Processor] JetStream subscription failed on toggle: %v", err)
			}
			a.startWorkers(context.Background(), attempts, &attemptsMu)
		} else {
			a.unsubscribeCore()
		}

		log.Printf("[Processor] Processing toggled to enabled=%t (status: %s)", req.Enabled, statusVal)
		respBytes := []byte(fmt.Sprintf(`{"enabled":%t,"status":"%s"}`, req.Enabled, statusVal))
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Processor] Failed to send state response: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to control subject: %w", err)
	}
	a.stateSetSub = stateSetSub
	log.Printf("[Run] Subscribed to control responder subject: %s", messaging.SubjectProcessorStateSet)

	// Subscribe Core NATS Queue Group workers
	if err := a.subscribeQueueGroup(); err != nil {
		return fmt.Errorf("failed to subscribe queue group workers: %w", err)
	}

	// Subscribe to queue group configuration control subject
	queueConfigSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupConfigSet, func(msg *nats.Msg) {
		var req jobs.QueueGroupConfig
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[Processor] Failed to unmarshal queue config payload: %v", err)
			_ = msg.Respond([]byte(`{"error":"Invalid payload"}`))
			return
		}

		if req.Workers < 1 {
			req.Workers = 1
		} else if req.Workers > 5 {
			req.Workers = 5
		}

		a.queueMu.Lock()
		a.queueWorkersCount = req.Workers
		a.queueMu.Unlock()

		if err := a.subscribeQueueGroup(); err != nil {
			log.Printf("[Processor] Failed to reconfigure queue group subscribers: %v", err)
			_ = msg.Respond([]byte(fmt.Sprintf(`{"error":"Failed to reconfigure: %v"}`, err)))
			return
		}

		a.queueMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] Queue Group reconfigured: Subject=%s, QueueGroup=%s, Workers=%d",
			messaging.SubjectJobQueue, messaging.QueueGroupJobWorkers, req.Workers)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue config subject: %w", err)
	}
	a.queueConfigSub = queueConfigSub
	log.Printf("[Run] Subscribed to queue config responder subject: %s", messaging.SubjectQueueGroupConfigSet)

	// Subscribe to queue group status responder subject
	queueStatusSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupStatus, func(msg *nats.Msg) {
		a.queueMu.Lock()
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue status subject: %w", err)
	}
	a.queueStatusSub = queueStatusSub
	log.Printf("[Run] Subscribed to queue status responder subject: %s", messaging.SubjectQueueGroupStatus)

	// Subscribe to queue group reset responder subject
	queueResetSub, err := a.natsClient.Conn.Subscribe(messaging.SubjectQueueGroupReset, func(msg *nats.Msg) {
		a.queueMu.Lock()
		a.queueDistribution = make(map[string]int)
		for i := 1; i <= a.queueWorkersCount; i++ {
			a.queueDistribution[fmt.Sprintf("processor-%d", i)] = 0
		}
		distCopy := make(map[string]int)
		for k, v := range a.queueDistribution {
			distCopy[k] = v
		}
		resp := jobs.QueueGroupStatusResponse{
			Subject:      messaging.SubjectJobQueue,
			QueueGroup:   messaging.QueueGroupJobWorkers,
			Workers:      a.queueWorkersCount,
			Distribution: distCopy,
		}
		a.queueMu.Unlock()

		respBytes, _ := json.Marshal(resp)
		_ = msg.Respond(respBytes)
		log.Printf("[Processor] Core NATS Queue Group distribution reset")
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to queue reset subject: %w", err)
	}
	a.queueResetSub = queueResetSub
	log.Printf("[Run] Subscribed to queue reset responder subject: %s", messaging.SubjectQueueGroupReset)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Run] Received signal: %v. Starting shutdown...", sig)

	return nil
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

// subscribeQueueGroup configures Core NATS Queue Group subscriptions based on a.queueWorkersCount.
func (a *App) subscribeQueueGroup() error {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()

	for _, sub := range a.queueSubs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	a.queueSubs = make([]*nats.Subscription, 0, a.queueWorkersCount)

	for i := 1; i <= a.queueWorkersCount; i++ {
		workerName := fmt.Sprintf("processor-%d", i)
		wName := workerName
		sub, err := a.natsClient.Conn.QueueSubscribe(messaging.SubjectJobQueue, messaging.QueueGroupJobWorkers, func(msg *nats.Msg) {
			a.handleQueueMessage(wName, msg)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe %s to queue group %s: %w", wName, messaging.QueueGroupJobWorkers, err)
		}
		a.queueSubs = append(a.queueSubs, sub)
		log.Printf("[%s] Subscribed to Core NATS queue group '%s' on subject '%s'",
			wName, messaging.QueueGroupJobWorkers, messaging.SubjectJobQueue)
	}

	return nil
}

// unsubscribeQueueGroup cleans up active Core NATS queue group subscriptions.
func (a *App) unsubscribeQueueGroup() {
	a.queueMu.Lock()
	defer a.queueMu.Unlock()

	for _, sub := range a.queueSubs {
		if sub != nil {
			_ = sub.Unsubscribe()
		}
	}
	a.queueSubs = nil
	log.Println("[Processor] Core NATS queue group subscriptions deactivated")
}

// handleQueueMessage processes a message delivered by Core NATS to a queue group member.
func (a *App) handleQueueMessage(workerName string, msg *nats.Msg) {
	var job jobs.Job
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("[%s] [queue-group: %s] Failed to unmarshal message: %v",
			workerName, messaging.QueueGroupJobWorkers, err)
		return
	}

	a.queueMu.Lock()
	a.queueDistribution[workerName]++
	currCount := a.queueDistribution[workerName]
	a.queueMu.Unlock()

	log.Printf("[%s] [queue-group: %s] Received job %s (worker total: %d)",
		workerName, messaging.QueueGroupJobWorkers, job.JobID, currCount)

	// Publish RECEIVED lifecycle event to jobs.queue.received with worker name
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobQueueReceived,
		job.JobID,
		"RECEIVED",
		1,
		"",
		workerName,
		"CORE",
		0,
	)

	// Simulate processing time
	time.Sleep(150 * time.Millisecond)

	// Publish COMPLETED lifecycle event to jobs.queue.completed with worker name
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobQueueCompleted,
		job.JobID,
		"COMPLETED",
		1,
		"",
		workerName,
		"CORE",
		0,
	)
}

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

// subscribeJetStream registers JetStream pull subscriber based on a.consumerConfig
func (a *App) subscribeJetStream() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.jsSub != nil {
		return nil
	}

	js, err := a.natsClient.Conn.JetStream()
	if err != nil {
		return err
	}

	cType := a.consumerConfig.Type
	if cType == "" {
		cType = "durable"
	}

	if cType == "ephemeral" {
		if a.consumerName == "" || a.consumerName == "job-processor" || a.consumerName == "processor-durable" {
			a.consumerName = fmt.Sprintf("ephemeral-%d", time.Now().UnixNano()%100000)
		}
		cinfo, err := js.AddConsumer("JOBS", &nats.ConsumerConfig{
			Name:          a.consumerName,
			DeliverPolicy: nats.DeliverNewPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       5 * time.Second,
			FilterSubject: messaging.SubjectJobSubmitted,
		})
		if err != nil {
			return fmt.Errorf("failed to add ephemeral consumer: %w", err)
		}
		sub, err := js.PullSubscribe(messaging.SubjectJobSubmitted, cinfo.Name, nats.Bind("JOBS", cinfo.Name))
		if err != nil {
			return fmt.Errorf("failed to bind ephemeral consumer %s: %w", cinfo.Name, err)
		}
		a.jsSub = sub
		log.Printf("[Processor] Bound to ephemeral consumer: %s", a.consumerName)
		return nil
	}

	// Durable consumer
	a.consumerName = "job-processor"
	_, err = js.ConsumerInfo("JOBS", a.consumerName)
	if err != nil {
		_, err = js.AddConsumer("JOBS", &nats.ConsumerConfig{
			Durable:       a.consumerName,
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       5 * time.Second,
			FilterSubject: messaging.SubjectJobSubmitted,
		})
		if err != nil {
			return fmt.Errorf("failed to create durable consumer %s: %w", a.consumerName, err)
		}
	} else {
		_, _ = js.UpdateConsumer("JOBS", &nats.ConsumerConfig{
			Durable:       a.consumerName,
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			AckWait:       5 * time.Second,
			FilterSubject: messaging.SubjectJobSubmitted,
		})
	}

	sub, err := js.PullSubscribe(messaging.SubjectJobSubmitted, a.consumerName, nats.Bind("JOBS", a.consumerName))
	if err != nil {
		return fmt.Errorf("failed to bind durable consumer %s: %w", a.consumerName, err)
	}
	a.jsSub = sub
	log.Printf("[Processor] JetStream consumer bound to %s Pull subscriber", a.consumerName)
	return nil
}

// unsubscribeJetStream deactivates JetStream subscription
func (a *App) unsubscribeJetStream() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.jsSub != nil {
		_ = a.jsSub.Unsubscribe()
		a.jsSub = nil
		log.Println("[Processor] JetStream pull subscriber deactivated")
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
			jsSub := a.jsSub
			a.mu.RUnlock()

			if !enabled || jsSub == nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}

			// Try to fetch 1 message with a short timeout
			msgs, err := jsSub.Fetch(1, nats.MaxWait(500*time.Millisecond))
			if err != nil {
				if errors.Is(err, nats.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
					continue
				}
				log.Printf("[%s JS] Fetch error: %v", workerName, err)
				time.Sleep(1 * time.Second)
				continue
			}

			for _, msg := range msgs {
				a.handleJetStreamMsg(msg, workerName, attempts, attemptsMu)
			}
		}
	}
}

// handleJetStreamMsg processes a pulled JetStream message
func (a *App) handleJetStreamMsg(msg *nats.Msg, workerName string, attempts map[string]int, attemptsMu *sync.Mutex) {
	deliveryMode := msg.Header.Get("X-Delivery-Mode")
	if deliveryMode == "" {
		deliveryMode = "CORE"
	}

	var job jobs.Job
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("[%s JS] Failed to unmarshal message: %v", workerName, err)
		_ = msg.Ack()
		return
	}

	// Crucial rule: if this is a CORE message, we immediately ACK and skip.
	// This ensures that transient messages published when processor was OFF are discarded from the stream.
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
	parentCtx := telemetry.ExtractTraceContext(context.Background(), msg.Header)
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
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobDelivered,
			job.JobID,
			"REDELIVERED",
			attemptCount,
			fmt.Sprintf("Message redelivered by JetStream (delivery #%d)", attemptCount),
			workerName,
			deliveryMode,
			sequence,
		)
	} else {
		log.Printf("[%s] Received JetStream job %s | Attempt: %d", workerName, job.JobID, attemptCount)
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobDelivered,
			job.JobID,
			"DELIVERED",
			attemptCount,
			"",
			workerName,
			deliveryMode,
			sequence,
		)
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

	// Publish PROCESSING lifecycle event on 3-token subject jobs.processing.started
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobProcessingStarted,
		job.JobID,
		"PROCESSING",
		attemptCount,
		"",
		workerName,
		deliveryMode,
		sequence,
	)

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

		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobAckTimeout,
			job.JobID,
			"ACK_TIMEOUT_SIMULATED",
			attemptCount,
			noAckReason,
			workerName,
			deliveryMode,
			sequence,
		)

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
			js, jsErr := a.natsClient.Conn.JetStream()
			if jsErr == nil {
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
				if _, pubErr := js.PublishMsg(dlqMsg); pubErr != nil {
					log.Printf("[%s] Failed to publish message %s to JOBS_DLQ: %v", workerName, job.JobID, pubErr)
				}
			}

			// 2. Publish DLQ_PUBLISHED lifecycle event
			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobDLQPublished,
				job.JobID,
				"DLQ_PUBLISHED",
				attemptCount,
				dlqReason,
				workerName,
				deliveryMode,
				sequence,
			)

			// 3. Acknowledge original message from JOBS stream so it does not redeliver
			if err := msg.Ack(); err != nil {
				log.Printf("[%s] Failed to ACK original message %s after DLQ routing: %v", workerName, job.JobID, err)
			}
			return
		}

		// Record error on processing span
		procSpan.RecordError(fmt.Errorf("%s", errMsg))
		procSpan.SetStatus(codes.Error, errMsg)
		procSpan.SetAttributes(attribute.String("processing.result", "failure"))
		procSpan.End()

		recvSpan.AddEvent("redelivery_scheduled")

		// Record failure metric
		telemetry.RecordJobFailed(procCtx, deliveryMode, workerName)

		// Check for NAK with Delay vs standard NAK
		if nakDelaySec > 0 {
			log.Printf("[%s] JetStream Job %s NAK with delay: %ds", workerName, job.JobID, nakDelaySec)
			_ = msg.NakWithDelay(time.Duration(nakDelaySec) * time.Second)

			nakReason := fmt.Sprintf("Worker requested explicit retry delay of %d seconds", nakDelaySec)
			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobNakDelayed,
				job.JobID,
				"NAK_WITH_DELAY",
				attemptCount,
				nakReason,
				workerName,
				deliveryMode,
				sequence,
			)

			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobProcessingFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				workerName,
				deliveryMode,
				sequence,
			)
			return
		}

		// Standard Nak for immediate redelivery
		_ = msg.Nak()

		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobProcessingFailed,
			job.JobID,
			"FAILED",
			attemptCount,
			errMsg,
			workerName,
			deliveryMode,
			sequence,
		)

		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobFailed,
			job.JobID,
			"FAILED",
			attemptCount,
			errMsg,
			workerName,
			deliveryMode,
			sequence,
		)
		return
	}

	// ACK on success
	if err := msg.Ack(); err != nil {
		log.Printf("[%s] Failed to ACK message %s: %v", workerName, job.JobID, err)
	}

	procSpan.SetStatus(codes.Ok, "success")
	procSpan.SetAttributes(attribute.String("processing.result", "success"))
	procSpan.End()

	recvSpan.AddEvent("message_acknowledged")

	// Record ACK and completion metrics
	telemetry.RecordMessageAcked(recvCtx, deliveryMode, workerName)
	telemetry.RecordJobProcessed(recvCtx, deliveryMode, workerName, "COMPLETED", procDuration)

	log.Printf("[%s] JetStream Job %s processed successfully", workerName, job.JobID)
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobCompleted,
		job.JobID,
		"COMPLETED",
		attemptCount,
		"",
		workerName,
		deliveryMode,
		sequence,
	)
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobAcked,
		job.JobID,
		"ACKED",
		attemptCount,
		"",
		workerName,
		deliveryMode,
		sequence,
	)
}

// Stop deactivates subscriptions and closes NATS connection.
func (a *App) Stop() {
	log.Println("[Stop] Cancelling JetStream pull workers...")
	a.mu.Lock()
	for _, cancel := range a.workerCancels {
		cancel()
	}
	a.workerCancels = nil
	a.mu.Unlock()

	a.unsubscribeCore()
	a.unsubscribeJetStream()

	log.Println("[Stop] Unsubscribing queue group...")
	a.unsubscribeQueueGroup()
	if a.queueConfigSub != nil {
		_ = a.queueConfigSub.Unsubscribe()
	}
	if a.queueStatusSub != nil {
		_ = a.queueStatusSub.Unsubscribe()
	}
	if a.queueResetSub != nil {
		_ = a.queueResetSub.Unsubscribe()
	}

	log.Println("[Stop] Unsubscribing validation consumer...")
	a.unsubscribeValidation()

	log.Println("[Stop] Unsubscribing status responder...")
	if a.statusSub != nil {
		if err := a.statusSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Status responder unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Status responder unsubscribed successfully")
		}
	}

	log.Println("[Stop] Unsubscribing consumer config responder...")
	if a.consumerConfigSub != nil {
		if err := a.consumerConfigSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Consumer config responder unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Consumer config responder unsubscribed successfully")
		}
	}
	if a.consumerResetSub != nil {
		_ = a.consumerResetSub.Unsubscribe()
	}

	log.Println("[Stop] Unsubscribing control responder...")
	if a.stateSetSub != nil {
		if err := a.stateSetSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Control responder unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Control responder unsubscribed successfully")
		}
	}

	log.Println("[Stop] Closing NATS connection...")
	if a.natsClient != nil {
		a.natsClient.Close()
		log.Println("[Stop] NATS connection closed successfully")
	}

	if a.otelShutdown != nil {
		log.Println("[Stop] Shutting down OpenTelemetry metrics...")
		_ = a.otelShutdown(context.Background())
	}

	log.Println("[Stop] Teardown completed")
}

func main() {
	app := &App{}

	log.Println("Initializing processor-service...")
	if err := app.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	log.Println("Starting execution...")
	if err := app.Run(); err != nil {
		log.Printf("Run phase encountered error: %v", err)
	}

	log.Println("Stopping processor-service...")
	app.Stop()
	log.Println("processor-service stopped gracefully")
}
