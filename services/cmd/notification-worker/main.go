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
	"services/natsclient"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Worker configuration parameters.
// Modify these constants to adapt this worker for any stream, push consumer, and deliver subject.
const (
	StreamName     = "PAYMENTS"
	ConsumerName   = "payments-push"
	DeliverSubject = "notifications.payments"
	WorkerPrefix   = "notification-worker"
	QueueGroup     = "notification-workers"
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

	// 5. Start JetStream push consumer worker
	sub, err := StartWorker(ctx, nc, js, &wg, workerID)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("worker_id", workerID).
			Str("stream", StreamName).
			Str("consumer", ConsumerName).
			Str("deliverSubject", DeliverSubject).
			Msg("Failed to start worker. Ensure stream and push consumer exist before starting")
	}

	log.Info().
		Str("worker_id", workerID).
		Str("stream", StreamName).
		Str("consumer", ConsumerName).
		Str("deliverSubject", DeliverSubject).
		Str("queueGroup", QueueGroup).
		Msg("Push Worker started successfully and listening for messages")

	// 6. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Info().
		Str("worker_id", workerID).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Initiating graceful shutdown...")

	// 7. Graceful shutdown of push subscription and NATS connection
	cleanup(sub, nc, &wg, workerID)
}

// StartWorker verifies the push consumer in JetStream and subscribes to its DeliverSubject with QueueGroup.
func StartWorker(ctx context.Context, nc *nats.Conn, js jetstream.JetStream, wg *sync.WaitGroup, workerID string) (*nats.Subscription, error) {
	// Verify consumer exists in JetStream
	consumer, err := js.Consumer(ctx, StreamName, ConsumerName)
	if err != nil {
		return nil, err
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, err
	}

	targetSubject := DeliverSubject
	if info.Config.DeliverSubject != "" {
		targetSubject = info.Config.DeliverSubject
	}

	// Core NATS QueueSubscribe distributes pushed messages across competing workers in QueueGroup
	sub, err := nc.QueueSubscribe(targetSubject, QueueGroup, func(msg *nats.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("worker_id", workerID).
			Str("consumer", ConsumerName).
			Str("subject", msg.Subject).
			Str("payload", string(msg.Data)).
			Msg("Worker processing pushed message")

		// Simulate processing duration to observe concurrency across competing workers
		time.Sleep(2 * time.Second)

		// Acknowledge message processing completion back to JetStream
		if err := msg.Ack(); err != nil {
			log.Error().
				Err(err).
				Str("worker_id", workerID).
				Str("consumer", ConsumerName).
				Msg("Failed to ACK message")
			return
		}

		log.Info().
			Str("worker_id", workerID).
			Str("consumer", ConsumerName).
			Str("subject", msg.Subject).
			Msg("Pushed message processed and ACKed successfully")
	})
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// cleanup gracefully drains the push subscription, waits for in-flight tasks, and drains NATS.
func cleanup(sub *nats.Subscription, nc *nats.Conn, wg *sync.WaitGroup, workerID string) {
	// 1. Drain push subscription to stop receiving new pushed messages
	if sub != nil {
		if err := sub.Drain(); err != nil {
			log.Error().Err(err).Str("worker_id", workerID).Msg("Error during subscription drain")
		} else {
			log.Info().Str("worker_id", workerID).Msg("Push subscription drained successfully")
		}
	}

	// 2. Wait for in-flight message processing with a bounded timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Str("worker_id", workerID).Msg("All in-flight messages completed and ACKed cleanly")
	case <-shutdownCtx.Done():
		log.Warn().Str("worker_id", workerID).Msg("Shutdown timeout reached before all in-flight messages finished")
	}

	// 3. Drain and close NATS connection
	if nc != nil {
		if err := nc.Drain(); err != nil {
			log.Error().Err(err).Str("worker_id", workerID).Msg("Error during NATS connection drain")
		} else {
			log.Info().Str("worker_id", workerID).Msg("NATS connection successfully drained and closed")
		}
	}
}
