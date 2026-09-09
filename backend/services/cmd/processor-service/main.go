package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"nats-demo/services/internal/config"
	"nats-demo/services/internal/jobs"
	"nats-demo/services/internal/messaging"
	"nats-demo/services/internal/natsclient"
	"nats-demo/services/internal/telemetry"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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
	jsConsumer        jetstream.Consumer
	processingEnabled bool
	consumerConfig    jobs.ConsumerConfig
	consumerName      string
	workerCancels     []context.CancelFunc
	otelShutdown      func(context.Context) error
	eventTracker      *EventTracker
	httpServer        *http.Server

	// Failure Scenarios & Goroutine Tracking
	scenarioMu       sync.RWMutex
	activeScenario   string // "normal", "crash_before_ack", "exceed_ack_wait", "out_of_order_ack"
	scenarioOnce     bool
	activeGoroutines int
	goroutineStatus  string // "RUNNING", "DEGRADED", "CRASHED", "PAUSED"
	crashedWorker    string // "processor-X" if a worker crashed

	// JetStream Consumer state
	attemptsMu            sync.Mutex
	attempts              map[string]int
	consumerDistMu        sync.Mutex
	consumerDistribution  map[string]int

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
	log.Printf("[Init] Loaded configuration: NATS_URL=%s, USER=%s, PORT=%s, MODE=%s", a.cfg.NATSURL, a.cfg.NATSUser, a.cfg.ProcessorPort, a.cfg.GetProcessorMode())

	// Initialize thread-safe in-memory event tracker for direct UI observation
	a.eventTracker = NewEventTracker(50)

	// Initialize OpenTelemetry metric pipeline for processor-service
	otelShutdown, err := telemetry.Init(context.Background(), "processor-service", a.cfg.OtelEndpoint, a.cfg.OtelInsecure, a.cfg.EnableOtelMetrics, a.cfg.EnableOtelTraces)
	if err != nil {
		log.Printf("[Init] Telemetry warning: %v", err)
	}
	a.otelShutdown = otelShutdown

	client, err := natsclient.ConnectWithAuth(a.cfg.NATSURL, a.cfg.NATSUser, a.cfg.NATSPassword)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	a.natsClient = client
	log.Println("[Init] Connected to NATS wrapper client")

	// Ensure primary demo stream (JOBS) and durable consumer (job-processor) exist
	if err := a.natsClient.EnsureJobsStream(); err != nil {
		log.Printf("[Init] Warning: EnsureJobsStream failed (NATS server may not be ready): %v", err)
	}

	a.consumer = messaging.NewConsumer(a.natsClient)
	a.publisher = messaging.NewPublisher(a.natsClient)
	a.processingEnabled = true
	a.consumerConfig = jobs.ConsumerConfig{
		Type:          "durable",
		Workers:       1,
		Ordering:      "normal",
		DeliverPolicy: "all",
		AckPolicy:     "explicit",
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
	a.attempts = make(map[string]int)
	a.consumerDistribution = map[string]int{
		"processor-1": 0,
		"processor-2": 0,
		"processor-3": 0,
		"processor-4": 0,
		"processor-5": 0,
	}

	// Initialize default failure scenario and goroutine status
	a.activeScenario = "normal"
	a.scenarioOnce = true
	a.activeGoroutines = 1
	a.goroutineStatus = "RUNNING"

	return nil
}

// Run starts the subscriptions and blocks until interrupt signal.
func (a *App) Run() error {
	workerName := os.Getenv("PROCESSOR_NAME")
	if workerName == "" {
		workerName = "processor-1"
	}

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("MODE")))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(os.Getenv("PROCESSOR_MODE")))
	}
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(a.cfg.GetProcessorMode()))
	}
	if mode == "" {
		mode = "all"
	}

	log.Printf("[Run] Starting processor instance: %s (mode: %s)", workerName, mode)

	// Initialize and Subscribe to JetStream Pull consumer (active in all modes)
	if err := a.subscribeJetStream(); err != nil {
		log.Printf("[Run] Warning: JetStream Pull subscription failed (JOBS stream may not exist yet): %v", err)
	}

	// Run JetStream Pull Workers (multi-worker competing pool)
	a.startWorkers(context.Background())

	// Start direct processor HTTP API server (:8082)
	a.httpServer = a.startHTTPServer(a.cfg.ProcessorPort)

	var coreWorkerCounter uint64
	jobHandler := a.buildCoreJobHandler(&coreWorkerCounter)

	// Mode handling:
	// "model" (or "demo") runs JetStream pull consumer, workers, direct HTTP server (:8082), and control responders.
	// "all" runs all components, including Core NATS transient pub/sub, RPC validation, and queue groups.
	if mode == "model" || mode == "demo" {
		log.Printf("[Run] Mode '%s' active: Running JetStream 'job-processor' pull workers, direct HTTP server (:8082), and control responders.", mode)
		if err := a.subscribeControlResponders(workerName, jobHandler); err != nil {
			log.Printf("[Run] Warning: failed to subscribe control responders: %v", err)
		}
	} else {
		// Subscribe to Core NATS
		if err := a.subscribeCore(workerName, jobHandler); err != nil {
			return fmt.Errorf("failed to subscribe to Core NATS: %w", err)
		}

		// Subscribe to jobs.validate Request/Reply
		if err := a.subscribeValidation(workerName); err != nil {
			return fmt.Errorf("failed to subscribe to validation subject: %w", err)
		}
		log.Printf("[Run] Subscribed to validation subject: %s", messaging.SubjectJobValidate)

		// Subscribe Core NATS Queue Group workers
		if err := a.subscribeQueueGroup(); err != nil {
			return fmt.Errorf("failed to subscribe queue group workers: %w", err)
		}

		// Subscribe runtime demo control responders
		if err := a.subscribeControlResponders(workerName, jobHandler); err != nil {
			return fmt.Errorf("failed to subscribe control responders: %w", err)
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Run] Received signal: %v. Starting shutdown...", sig)

	return nil
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

	if len(a.queueSubs) > 0 {
		log.Println("[Stop] Unsubscribing queue group...")
		a.unsubscribeQueueGroup()
	}

	if a.valSub != nil {
		log.Println("[Stop] Unsubscribing validation consumer...")
		a.unsubscribeValidation()
	}

	if a.statusSub != nil || a.stateSetSub != nil {
		log.Println("[Stop] Unsubscribing control responders...")
		a.unsubscribeControlResponders()
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

	if a.httpServer != nil {
		log.Println("[Stop] Shutting down processor HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = a.httpServer.Shutdown(shutdownCtx)
		cancel()
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
