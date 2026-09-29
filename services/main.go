package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"services/api"
	"services/config"
	"services/consumer"
	"services/natsclient"
	"services/publisher"
	"services/telemetry"
	"syscall"
	"time"
)

func main() {
	// 1. Load Environment Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load environment configuration: %v", err)
	}

	// 2. Initialize OpenTelemetry TracerProvider
	tp, err := telemetry.InitTracer(cfg.ServiceName, cfg.OtelExporterType, cfg.OtelExporterOTLPEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry tracer: %v", err)
	}

	// 3. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover()
	if err != nil {
		log.Fatalf("Failed to initialize NATS Client: %v", err)
	}

	// 4. Initialize Services
	pubService := publisher.NewService(nc)
	consumerService := consumer.NewService(nc)

	// 5. Start background subscriber for end-to-end trace correlation
	sub, err := consumerService.StartSubscriber("orders.>")
	if err != nil {
		log.Fatalf("Failed to start consumer subscriber: %v", err)
	}
	defer sub.Unsubscribe()
	log.Println("[SERVER] NATS Consumer listening on subject [orders.>]")

	// 6. Setup HTTP REST API Layer
	apiHandler := api.NewHandler(pubService)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/publish", apiHandler.PublishHandler)

	httpServer := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: mux,
	}

	go func() {
		log.Printf("[SERVER] HTTP REST API server listening on http://localhost%s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 7. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Received shutdown signal. Initiating graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP Server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down HTTP server: %v", err)
	}

	// Drain NATS connection
	if err := nc.Drain(); err != nil {
		log.Printf("Error during NATS connection drain: %v", err)
	} else {
		log.Println("NATS connection successfully drained and closed.")
	}

	// Flush and shutdown OpenTelemetry tracer
	telemetry.ShutdownTracer(shutdownCtx, tp)
	log.Println("Tracer provider shutdown complete.")
}
