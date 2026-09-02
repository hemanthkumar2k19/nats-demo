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
	"nats-demo/internal/activity"
	"nats-demo/internal/config"
	"nats-demo/internal/messaging"
	"nats-demo/internal/natsclient"
	"nats-demo/internal/telemetry"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// App manages the lifecycle of the demo-control-service.
type App struct {
	cfg             *config.Config
	natsClient      *natsclient.Client
	httpServer      *http.Server
	activityTracker *activity.Tracker
	observer        *messaging.Observer
	lifecycleSub    *nats.Subscription
	observerSubs    []*nats.Subscription
	jobServiceURL   string
	otelShutdown    func(context.Context) error
}

// Init loads configuration, establishes connections, and configures routing.
func (a *App) Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg

	a.jobServiceURL = os.Getenv("JOB_SERVICE_URL")
	if a.jobServiceURL == "" {
		a.jobServiceURL = "http://localhost:8081"
	}

	log.Printf("[Init] Loaded configuration: NATS_URL=%s, PORT=%s, JOB_SERVICE_URL=%s", a.cfg.NATSURL, a.cfg.Port, a.jobServiceURL)

	// Initialize OpenTelemetry metric pipeline for demo-control-service
	otelShutdown, err := telemetry.Init(context.Background(), "demo-control-service", a.cfg.OtelEndpoint, a.cfg.OtelInsecure)
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

	// Ensure the JOBS JetStream stream exists
	if err := a.natsClient.EnsureJobsStream(); err != nil {
		log.Printf("[Init] Warning: failed to ensure JOBS stream: %v", err)
	} else {
		log.Println("[Init] Guaranteed JOBS JetStream stream exists")
	}

	a.activityTracker = activity.NewTracker()
	a.observer = messaging.NewObserver()
	controlHandler := apihttp.NewControlHandler(a.activityTracker, a.natsClient, a.observer, a.jobServiceURL)

	router := gin.Default()
	apihttp.RegisterControlRoutes(router, controlHandler)

	a.httpServer = &http.Server{
		Addr:    ":" + a.cfg.Port,
		Handler: router,
	}

	return nil
}

// Run starts the HTTP server and blocks until an interrupt signal is received.
func (a *App) Run() error {
	// Subscribe to jobs.> to capture all NATS lifecycle events in-memory for the UI activity log
	sub, err := a.natsClient.Conn.Subscribe("jobs.>", func(msg *nats.Msg) {
		if msg.Reply != "" {
			// Skip Request/Reply messages
			return
		}
		correlationID := msg.Header.Get("X-Correlation-Id")
		source := msg.Header.Get("X-Source")
		msgID := msg.Header.Get("Nats-Msg-Id")
		if msgID == "" {
			msgID = msg.Header.Get("X-Message-Id")
		}

		if err := a.activityTracker.ProcessLifecycleEvent(msg.Subject, msg.Data, correlationID, source, msgID); err != nil {
			log.Printf("[App] Failed to process lifecycle event: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to lifecycle events: %w", err)
	}
	a.lifecycleSub = sub
	log.Println("[Run] Subscribed to jobs.> NATS wildcard events")

	// Subscribe to NATS observer subjects for subject addressing demo
	a.observerSubs = make([]*nats.Subscription, 0)
	subsConfig := []struct {
		name    string
		subject string
	}{
		{"exact", "jobs.submitted"},
		{"single-level", "jobs.*"},
		{"multi-level", "jobs.>"},
	}

	for _, cfg := range subsConfig {
		subName := cfg.name
		subSubject := cfg.subject
		oSub, err := a.natsClient.Conn.Subscribe(subSubject, func(msg *nats.Msg) {
			if msg.Reply != "" {
				return
			}
			msgID := msg.Header.Get("X-Message-Id")
			if msgID == "" {
				msgID = fmt.Sprintf("fallback-%s-%s", msg.Subject, string(msg.Data))
			}
			jobID := msg.Header.Get("Nats-Msg-Id")
			a.observer.RecordEvent(msgID, subName, msg.Subject, jobID)
		})
		if err != nil {
			log.Printf("[Run] Warning: failed to subscribe observer for %s: %v", subSubject, err)
			continue
		}
		a.observerSubs = append(a.observerSubs, oSub)
		log.Printf("[Run] Subscribed observer: %s (%s)", subName, subSubject)
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("[Run] Demo Control Service HTTP server listening on %s", a.httpServer.Addr)
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

// Stop terminates active subscriptions and closes the HTTP server.
func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var firstErr error

	if a.lifecycleSub != nil {
		if err := a.lifecycleSub.Unsubscribe(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for _, oSub := range a.observerSubs {
		if oSub != nil {
			_ = oSub.Unsubscribe()
		}
	}

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
	log.Println("Initializing demo-control-service...")
	app := &App{}

	if err := app.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Runtime error: %v", err)
	}

	log.Println("Stopping demo-control-service...")
	if err := app.Stop(); err != nil {
		log.Printf("Shutdown warning: %v", err)
	}
	log.Println("demo-control-service stopped gracefully")
}
