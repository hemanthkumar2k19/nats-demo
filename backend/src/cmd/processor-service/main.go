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
	"syscall"
	"time"

	"nats-demo/internal/config"
	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"

	"github.com/nats-io/nats.go"
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
	jsSub             *nats.Subscription
	processingEnabled bool
	jsPullCancel      context.CancelFunc
}

// Init loads configuration, connects to NATS, and instantiates components.
func (a *App) Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg
	log.Printf("[Init] Loaded configuration: NATS_URL=%s", a.cfg.NATSURL)

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

	// Core NATS job processing handler
	jobHandler := func(job jobs.Job, correlationID string) error {
		deliveryMode := job.DeliveryMode
		if deliveryMode == "" {
			deliveryMode = "CORE"
		}

		// If this is a JetStream message, ignore it in the Core subscription
		if deliveryMode == "JETSTREAM" {
			return nil
		}

		attemptsMu.Lock()
		attempts[job.JobID]++
		attemptCount := attempts[job.JobID]
		attemptsMu.Unlock()

		log.Printf("[%s] Received Core NATS job %s | Attempt: %d | Correlation: %s", workerName, job.JobID, attemptCount, correlationID)

		// 1. Publish RECEIVED lifecycle event
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobReceived,
			job.JobID,
			"RECEIVED",
			attemptCount,
			"",
			correlationID,
			workerName,
			deliveryMode,
			0,
		)

		// 2. Simulate processing duration
		time.Sleep(1 * time.Second)

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
			log.Printf("[%s] Job %s failed: %s", workerName, job.JobID, errMsg)

			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobProcessingFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				correlationID,
				workerName,
				deliveryMode,
				0,
			)

			_ = a.publisher.PublishJobLifecycle(
				messaging.SubjectJobFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				correlationID,
				workerName,
				deliveryMode,
				0,
			)
			return fmt.Errorf("simulated failure: %s", errMsg)
		}

		// 4. Publish Completed lifecycle event on success
		log.Printf("[%s] Job %s processed successfully", workerName, job.JobID)
		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobCompleted,
			job.JobID,
			"COMPLETED",
			attemptCount,
			"",
			correlationID,
			workerName,
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

	// Run JetStream Pull Loop
	pullCtx, pullCancel := context.WithCancel(context.Background())
	a.jsPullCancel = pullCancel
	go a.jsPullLoop(pullCtx, workerName, attempts, &attemptsMu)

	// Subscribe to jobs.validate Request/Reply
	validationHandler := func(job jobs.Job) (jobs.JobValidationResponse, error) {
		log.Printf("[%s] Received validation request for job: %s of type %s", workerName, job.JobID, job.Type)
		if job.JobID == "" {
			return jobs.JobValidationResponse{Valid: false, Message: "job_id is required"}, nil
		}
		if job.Type != "image-processing" && job.Type != "data-sync" && job.Type != "email-alert" {
			return jobs.JobValidationResponse{Valid: false, Message: fmt.Sprintf("unsupported job type: %s", job.Type)}, nil
		}
		if len(job.Payload) == 0 {
			return jobs.JobValidationResponse{Valid: false, Message: "payload is required"}, nil
		}
		return jobs.JobValidationResponse{Valid: true, Message: "Job configuration is valid."}, nil
	}

	valSub, err := a.consumer.SubscribeJobValidate(validationHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to validation subject: %w", err)
	}
	a.valSub = valSub
	log.Printf("[Run] Subscribed to validation subject: %s", messaging.SubjectJobValidate)

	// Subscribe to status ping responder
	statusSub, err := a.natsClient.Conn.Subscribe("status.processor", func(msg *nats.Msg) {
		a.mu.RLock()
		enabled := a.processingEnabled
		a.mu.RUnlock()

		respBytes := []byte(fmt.Sprintf(`{"status":"ACTIVE","processing":%t}`, enabled))
		if err := msg.Respond(respBytes); err != nil {
			log.Printf("[Processor] Failed to send status reply: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to status responder: %w", err)
	}
	a.statusSub = statusSub
	log.Println("[Run] Subscribed to status responder subject: status.processor")

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
			// Trigger JetStream subscription try if missing
			if err := a.subscribeJetStream(); err != nil {
				log.Printf("[Processor] JetStream subscription failed on toggle: %v", err)
			}
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

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Run] Received signal: %v. Starting shutdown...", sig)

	return nil
}

// subscribeCore registers Core NATS subscriber if missing
func (a *App) subscribeCore(workerName string, jobHandler func(jobs.Job, string) error) error {
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
	log.Printf("[%s] Core NATS subscriber activated on subject: %s", workerName, messaging.SubjectJobSubmitted)
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

// subscribeJetStream registers JetStream pull subscriber if missing
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

	// Create pull durable subscription on stream JOBS, binding to explicitly created consumer
	sub, err := js.PullSubscribe(messaging.SubjectJobSubmitted, "processor-durable", nats.Bind("JOBS", "processor-durable"))
	if err != nil {
		return err
	}
	a.jsSub = sub
	log.Println("[Processor] JetStream consumer bound to processor-durable Pull subscriber")
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
				log.Printf("[Processor JS] Fetch error: %v", err)
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
	correlationID := msg.Header.Get("X-Correlation-Id")
	deliveryMode := msg.Header.Get("X-Delivery-Mode")
	if deliveryMode == "" {
		deliveryMode = "CORE"
	}

	var job jobs.Job
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Printf("[Processor JS] Failed to unmarshal message: %v", err)
		_ = msg.Ack()
		return
	}

	// Crucial rule: if this is a CORE message, we immediately ACK and skip.
	// This ensures that transient messages published when processor was OFF are discarded from the stream.
	if deliveryMode == "CORE" || job.DeliveryMode == "CORE" {
		log.Printf("[Processor JS] Core NATS message %s fetched. Discarding from JetStream.", job.JobID)
		_ = msg.Ack()
		return
	}

	attemptsMu.Lock()
	attempts[job.JobID]++
	attemptCount := attempts[job.JobID]
	attemptsMu.Unlock()

	log.Printf("[%s] Received JetStream job %s | Attempt: %d | Correlation: %s", workerName, job.JobID, attemptCount, correlationID)

	meta, err := msg.Metadata()
	var sequence uint64
	if err == nil {
		sequence = meta.Sequence.Stream
	}

	// 1. Publish DELIVERED lifecycle event
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobDelivered,
		job.JobID,
		"DELIVERED",
		attemptCount,
		"",
		correlationID,
		workerName,
		deliveryMode,
		sequence,
	)

	// 2. Simulate processing duration
	time.Sleep(1 * time.Second)

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
		log.Printf("[%s] JetStream Job %s failed: %s", workerName, job.JobID, errMsg)

		// Nak for redelivery
		_ = msg.Nak()

		_ = a.publisher.PublishJobLifecycle(
			messaging.SubjectJobProcessingFailed,
			job.JobID,
			"FAILED",
			attemptCount,
			errMsg,
			correlationID,
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
			correlationID,
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

	log.Printf("[%s] JetStream Job %s processed successfully", workerName, job.JobID)
	_ = a.publisher.PublishJobLifecycle(
		messaging.SubjectJobAcked,
		job.JobID,
		"ACKED",
		attemptCount,
		"",
		correlationID,
		workerName,
		deliveryMode,
		sequence,
	)
}

// Stop deactivates subscriptions and closes NATS connection.
func (a *App) Stop() {
	log.Println("[Stop] Cancelling JetStream pull loop...")
	if a.jsPullCancel != nil {
		a.jsPullCancel()
	}

	a.unsubscribeCore()
	a.unsubscribeJetStream()

	log.Println("[Stop] Unsubscribing validation consumer...")
	if a.valSub != nil {
		if err := a.valSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Validation unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Validation unsubscribed successfully")
		}
	}

	log.Println("[Stop] Unsubscribing status responder...")
	if a.statusSub != nil {
		if err := a.statusSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Status responder unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Status responder unsubscribed successfully")
		}
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
