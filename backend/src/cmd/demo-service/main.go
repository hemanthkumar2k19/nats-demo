package main

import (
	"context"
	"encoding/json"
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

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

// App manages the lifecycle of the demo service.
type App struct {
	cfg          *config.Config
	natsClient   *natsclient.Client
	httpServer   *http.Server
	jobService   *jobs.Service
	lifecycleSub *nats.Subscription
	observer     *messaging.Observer
	observerSubs []*nats.Subscription
}

// Init loads configuration, establishes connections, and configures routing.
func (a *App) Init() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	a.cfg = cfg
	log.Printf("[Init] Loaded configuration: NATS_URL=%s, PORT=%s", a.cfg.NATSURL, a.cfg.Port)

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

	publisher := messaging.NewPublisher(a.natsClient)
	a.jobService = jobs.NewService(publisher)
	a.observer = messaging.NewObserver()
	handler := apihttp.NewHandler(a.jobService, a.natsClient, a.observer)

	router := gin.Default()
	apihttp.RegisterRoutes(router, handler)

	a.httpServer = &http.Server{
		Addr:    ":" + a.cfg.Port,
		Handler: router,
	}

	return nil
}

// Run starts the HTTP server and blocks until an interrupt signal is received.
func (a *App) Run() error {
	// Subscribe to jobs.> to capture all NATS lifecycle events in-memory.
	// The multi-level wildcard ensures multi-segment subjects such as
	// jobs.request.received and jobs.reply.sent are included alongside
	// the single-segment ones like jobs.submitted and jobs.completed.
	// Skip messages that have a Reply field - those are Request/Reply request
	// messages, not lifecycle events. demo-service publishes RequestMsg to
	// jobs.validate which also matches jobs.>, and we must not treat it as
	// a lifecycle event.
	sub, err := a.natsClient.Conn.Subscribe("jobs.>", func(msg *nats.Msg) {
		if msg.Reply != "" {
			// This is a NATS request message, not a lifecycle publish event.
			return
		}
		correlationID := msg.Header.Get("X-Correlation-Id")
		source := msg.Header.Get("X-Source")
		msgID := msg.Header.Get("Nats-Msg-Id")
		if msgID == "" {
			msgID = msg.Header.Get("X-Message-Id")
		}
		log.Printf("[App] Received event on subject %s (correlation: %s, source: %s, msgID: %s)", msg.Subject, correlationID, source, msgID)

		if err := a.jobService.ProcessLifecycleEvent(msg.Subject, msg.Data, correlationID, source, msgID); err != nil {
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
			// Skip request messages - we only want to observe published events,
			// not the outgoing jobs.validate RequestMsg that also matches jobs.*.
			if msg.Reply != "" {
				return
			}
			msgID := msg.Header.Get("X-Message-Id")
			if msgID == "" {
				msgID = fmt.Sprintf("fallback-%s-%s", msg.Subject, string(msg.Data))
			}
			jobID := parseJobID(msg.Data)
			a.observer.RecordEvent(msgID, subName, msg.Subject, jobID)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to observer subject %s: %w", subSubject, err)
		}
		a.observerSubs = append(a.observerSubs, oSub)
		log.Printf("[Observer] Subscribed to subject: %s as %s", subSubject, subName)
	}

	go func() {
		log.Printf("[Run] HTTP server listening on :%s", a.cfg.Port)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[Run] HTTP server ListenAndServe error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("[Run] Received signal: %v. Starting shutdown...", sig)

	return nil
}

// Stop gracefully tears down the HTTP server and NATS connection.
func (a *App) Stop() {
	log.Println("[Stop] Unsubscribing addressing observer subscriptions...")
	for _, sub := range a.observerSubs {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				log.Printf("[Stop] Observer unsubscribe failed: %v", err)
			}
		}
	}

	log.Println("[Stop] Unsubscribing lifecycle events...")
	if a.lifecycleSub != nil {
		if err := a.lifecycleSub.Unsubscribe(); err != nil {
			log.Printf("[Stop] Lifecycle unsubscribe failed: %v", err)
		} else {
			log.Println("[Stop] Lifecycle unsubscribed successfully")
		}
	}

	log.Println("[Stop] Shutting down HTTP server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		log.Printf("[Stop] HTTP server shutdown failed: %v", err)
	} else {
		log.Println("[Stop] HTTP server stopped successfully")
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

	log.Println("Initializing demo-service...")
	if err := app.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	log.Println("Starting execution...")
	if err := app.Run(); err != nil {
		log.Printf("Run phase encountered error: %v", err)
	}

	log.Println("Stopping demo-service...")
	app.Stop()
	log.Println("demo-service stopped gracefully")
}

func parseJobID(data []byte) string {
	var payload struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return ""
	}
	return payload.JobID
}
