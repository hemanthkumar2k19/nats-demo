package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apihttp "nats-demo/api/http"
	"nats-demo/internal/config"
	"nats-demo/internal/jobs"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/gin-gonic/gin"
)

// App manages the lifecycle of the pure business job-service.
type App struct {
	cfg          *config.Config
	port         string
	natsClient   *natsclient.Client
	httpServer   *http.Server
	jobService   *jobs.Service
	otelShutdown func(context.Context) error
}

// Init loads configuration, establishes connections, and configures business routing.
func (a *App) Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg

	// Default job-service to port 8081 unless explicitly configured
	a.port = os.Getenv("JOB_SERVICE_PORT")
	if a.port == "" {
		if os.Getenv("PORT") != "" && os.Getenv("PORT") != "8080" {
			a.port = os.Getenv("PORT")
		} else {
			a.port = "8081"
		}
	}

	log.Printf("[Init] Loaded configuration: NATS_URL=%s, PORT=%s", a.cfg.NATSURL, a.port)

	// Initialize OpenTelemetry metric and trace pipeline for job-service
	otelShutdown, err := telemetry.Init(context.Background(), "job-service", a.cfg.OtelEndpoint, a.cfg.OtelInsecure)
	if err != nil {
		log.Printf("[Init] Telemetry warning: %v", err)
	}
	a.otelShutdown = otelShutdown

	client, err := natsclient.Connect(a.cfg.NATSURL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	a.natsClient = client
	log.Println("[Init] Connected to NATS client")

	// Ensure the JOBS JetStream stream exists
	if err := a.natsClient.EnsureJobsStream(); err != nil {
		log.Printf("[Init] Warning: failed to ensure JOBS stream: %v", err)
	} else {
		log.Println("[Init] Guaranteed JOBS JetStream stream exists")
	}

	publisher := messaging.NewPublisher(a.natsClient)
	a.jobService = jobs.NewService(publisher)
	jobHandler := apihttp.NewJobHandler(a.jobService)

	router := gin.Default()
	apihttp.RegisterJobRoutes(router, jobHandler)

	a.httpServer = &http.Server{
		Addr:    ":" + a.port,
		Handler: router,
	}

	return nil
}

// Run starts the HTTP server and blocks until an interrupt signal is received.
func (a *App) Run() error {
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[Run] Job Service HTTP API listening on %s", a.httpServer.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return err
	case sig := <-shutdown:
		log.Printf("[Run] Received shutdown signal: %v. Initiating graceful shutdown...", sig)
		return a.Stop()
	}
}

// Stop terminates the HTTP server and closes the NATS connection.
func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var firstErr error

	if a.httpServer != nil {
		if err := a.httpServer.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if a.natsClient != nil {
		a.natsClient.Close()
	}

	if a.otelShutdown != nil {
		if err := a.otelShutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func main() {
	log.Println("Initializing job-service...")
	app := &App{}

	if err := app.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	log.Println("Stopping job-service...")
	if err := app.Stop(); err != nil {
		log.Printf("Shutdown warning: %v", err)
	}
	log.Println("job-service stopped gracefully")
}
