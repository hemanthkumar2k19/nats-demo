package main

import (
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
	cfg        *config.Config
	natsClient *natsclient.Client
	consumer   *messaging.Consumer
	publisher  *messaging.Publisher
	sub        *nats.Subscription
	valSub     *nats.Subscription
	statusSub  *nats.Subscription
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

	a.consumer = messaging.NewConsumer(a.natsClient)
	a.publisher = messaging.NewPublisher(a.natsClient)

	return nil
}

// Run starts the subscription consumer and blocks until an interrupt signal is received.
func (a *App) Run() error {
	workerName := os.Getenv("PROCESSOR_NAME")
	if workerName == "" {
		workerName = "processor-1"
	}
	log.Printf("[Run] Starting processor instance: %s", workerName)

	// Keep track of attempt counts locally per job ID
	var attemptsMu sync.Mutex
	attempts := make(map[string]int)

	jobHandler := func(job jobs.Job, correlationID string) error {
		attemptsMu.Lock()
		attempts[job.JobID]++
		attemptCount := attempts[job.JobID]
		attemptsMu.Unlock()

		log.Printf("[%s] Received job %s | Attempt: %d | Correlation: %s", workerName, job.JobID, attemptCount, correlationID)

		// 1. Publish Processing lifecycle event
		if err := a.publisher.PublishJobLifecycle(
			messaging.SubjectJobProcessing,
			job.JobID,
			"PROCESSING",
			attemptCount,
			"",
			correlationID,
			workerName,
		); err != nil {
			log.Printf("[%s] Failed to publish processing event: %v", workerName, err)
		}

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

			// Publish Failed lifecycle event
			if err := a.publisher.PublishJobLifecycle(
				messaging.SubjectJobFailed,
				job.JobID,
				"FAILED",
				attemptCount,
				errMsg,
				correlationID,
				workerName,
			); err != nil {
				log.Printf("[%s] Failed to publish failed event: %v", workerName, err)
			}
			return fmt.Errorf("simulated failure: %s", errMsg)
		}

		// 4. Publish Completed lifecycle event on success
		log.Printf("[%s] Job %s processed successfully", workerName, job.JobID)
		if err := a.publisher.PublishJobLifecycle(
			messaging.SubjectJobCompleted,
			job.JobID,
			"COMPLETED",
			attemptCount,
			"",
			correlationID,
			workerName,
		); err != nil {
			log.Printf("[%s] Failed to publish completed event: %v", workerName, err)
		}

		return nil
	}

	sub, err := a.consumer.SubscribeJobSubmitted(jobHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to job submission: %w", err)
	}
	a.sub = sub
	log.Printf("[Run] Subscribed to subject: %s", messaging.SubjectJobSubmitted)

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
		log.Printf("[Processor] Received status ping on reply subject: %s", msg.Reply)
		if err := msg.Respond([]byte(`{"status":"ACTIVE"}`)); err != nil {
			log.Printf("[Processor] Failed to send status reply: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to status responder: %w", err)
	}
	a.statusSub = statusSub
	log.Println("[Run] Subscribed to status responder subject: status.processor")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Run] Received signal: %v. Starting shutdown...", sig)

	return nil
}

// Stop unsubscribes the consumer and closes NATS connection wrapper.
func (a *App) Stop() {
	log.Println("[Stop] Unsubscribing consumer...")
	if a.sub != nil {
		if err := a.sub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Unsubscribed successfully")
		}
	}

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
