package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"services/config"
	"services/logger"
	"services/messaging"
	"services/natsclient"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Worker configuration parameters.
// Modify these constants to adapt this worker for any stream and consumer.
const (
	StreamName   = "PAYMENTS"
	ConsumerName = "payments-pull"
	WorkerPrefix = "payment-worker"
)

func main() {
	// 0. Initialize Zerolog Logger (false = colorful console logs, true = raw JSON)
	logger.Init(false)

	// 1. Resolve Worker ID (appends WORKER_ID suffix to WorkerPrefix, defaulting to 1)
	workerSuffix := os.Getenv("WORKER_ID")
	if workerSuffix == "" {
		workerSuffix = "1"
	}
	workerID := fmt.Sprintf("%s-%s", WorkerPrefix, workerSuffix)

	// 2. Load Environment Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load environment configuration")
	}

	// 3. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover(&cfg.NATS, workerID)
	if err != nil {
		log.Fatal().Err(err).Str("worker_id", workerID).Msg("Failed to initialize NATS Client")
	}

	js, err := natsclient.NewJSClient(nc)
	if err != nil {
		log.Fatal().Err(err).Str("worker_id", workerID).Msg("Failed to initialize JetStream")
	}

	// 4. Setup context and WaitGroup for worker lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 5. Start JetStream consumer worker
	cc, err := StartWorker(ctx, js, &wg, workerID)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("worker_id", workerID).
			Str("stream", StreamName).
			Str("consumer", ConsumerName).
			Msg("Failed to start worker. Ensure stream and consumer exist before starting")
	}

	log.Info().
		Str("worker_id", workerID).
		Str("stream", StreamName).
		Str("consumer", ConsumerName).
		Msg("Payment worker started successfully and listening for messages")

	// 6. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Info().
		Str("worker_id", workerID).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Initiating graceful shutdown...")

	// 7. Graceful shutdown of consumer and NATS connection
	messaging.GracefulShutdown(workerID, nc, &wg, 5*time.Second, nil, cc)
}

// StartWorker binds to the configured JetStream consumer and begins asynchronous message processing.
func StartWorker(ctx context.Context, js jetstream.JetStream, wg *sync.WaitGroup, workerID string) (jetstream.ConsumeContext, error) {
	consumer, err := messaging.GetConsumer(ctx, js, StreamName, ConsumerName)
	if err != nil {
		return nil, err
	}

	cc, err := messaging.Consume(consumer, func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("worker_id", workerID).
			Str("consumer", ConsumerName).
			Str("subject", msg.Subject()).
			Str("payload", string(msg.Data())).
			Msg("Worker processing message")

		// Simulate processing duration to clearly observe concurrency across competing workers
		time.Sleep(2 * time.Second)

		// Acknowledge message processing completion via messaging utility
		messaging.AcknowledgeMessage(ctx, msg, workerID, ConsumerName, "at-least-once")
	})
	if err != nil {
		return nil, err
	}

	return cc, nil
}
