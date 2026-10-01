package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"services/api"
	"services/config"
	"services/logger"
	"services/natsclient"
	"services/service"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func main() {
	// 0. Initialize Zerolog Logger (false = colorful console logs, true = raw JSON)
	logger.Init(false)
	gin.SetMode(gin.ReleaseMode)

	// 1. Load Environment Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load environment configuration")
	}

	// 2. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover("order-service")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize NATS Client")
	}

	// RTT
	rtt, err := nc.RTT()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get RTT")
	}
	log.Info().Msgf("NATS RTT: %v", rtt)

	// 3. Initialize Publisher Service & API Handler
	svc := service.NewService(nc, nil)
	pubHandler := api.NewPublisherHandler(svc)

	// 4. Setup Gin HTTP Engine & Register Routes
	r := gin.New()
	r.Use(gin.Recovery())

	v1 := r.Group("/api/v1")
	pubHandler.RegisterRoutes(v1)

	httpServer := &http.Server{
		Addr:    cfg.OrderServicePort,
		Handler: r,
	}

	go func() {
		log.Info().Str("service", "order-service").Str("addr", cfg.OrderServicePort).Msg("Order Service (Publisher) listening")
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	// 5. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	cleanup(httpServer, nc)
}

// cleanup performs graceful shutdown of the HTTP server and NATS connection.
func cleanup(httpServer *http.Server, nc *nats.Conn) {
	log.Info().Msg("Received shutdown signal. Initiating graceful shutdown...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown HTTP Server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down HTTP server")
	}

	// Drain NATS connection
	if err := nc.Drain(); err != nil {
		log.Error().Err(err).Msg("Error during NATS connection drain")
	} else {
		log.Info().Msg("NATS connection successfully drained and closed.")
	}
}
