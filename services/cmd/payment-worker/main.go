package main

import (
	"context"
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
// Modify these constants to adapt this worker for any stream and consumer.
const (
	StreamName   = "PAYMENTS"
	ConsumerName = "payments-pull"
	ClientName   = "payment-worker"
)

func main() {
	// 0. Initialize Zerolog Logger (false = colorful console logs, true = raw JSON)
	logger.Init(false)

	// 1. Load Environment Configuration
	_, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load environment configuration")
	}

	// 2. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover(ClientName)
	if err != nil {
		log.Fatal().Err(err).Str("client", ClientName).Msg("Failed to initialize NATS Client")
	}

	js, err := natsclient.NewJSClient(nc)
	if err != nil {
		log.Fatal().Err(err).Str("client", ClientName).Msg("Failed to initialize JetStream")
	}

	// 3. Setup context and WaitGroup for worker lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 4. Start JetStream consumer worker
	cc, err := StartWorker(ctx, js, &wg)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("client", ClientName).
			Str("stream", StreamName).
			Str("consumer", ConsumerName).
			Msg("Failed to start worker. Ensure stream and consumer exist before starting")
	}

	log.Info().
		Str("client", ClientName).
		Str("stream", StreamName).
		Str("consumer", ConsumerName).
		Msg("Worker started successfully and listening for messages")

	// 5. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Info().
		Str("client", ClientName).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Initiating graceful shutdown...")

	// 6. Graceful shutdown of consumer and NATS connection
	cleanup(cc, nc, &wg)
}

// StartWorker binds to the configured JetStream consumer and begins asynchronous message processing.
func StartWorker(ctx context.Context, js jetstream.JetStream, wg *sync.WaitGroup) (jetstream.ConsumeContext, error) {
	consumer, err := js.Consumer(ctx, StreamName, ConsumerName)
	if err != nil {
		return nil, err
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("client", ClientName).
			Str("consumer", ConsumerName).
			Str("subject", msg.Subject()).
			Str("payload", string(msg.Data())).
			Msg("Worker processing message")

		// Acknowledge message processing completion
		if err := msg.Ack(); err != nil {
			log.Error().
				Err(err).
				Str("client", ClientName).
				Str("consumer", ConsumerName).
				Msg("Failed to ACK message")
		}
	})
	if err != nil {
		return nil, err
	}

	return cc, nil
}

// cleanup performs graceful shutdown of the consume context, waits for in-flight tasks, and drains NATS.
func cleanup(cc jetstream.ConsumeContext, nc *nats.Conn, wg *sync.WaitGroup) {
	// 1. Stop consumer fetch loop
	if cc != nil {
		cc.Stop()
		log.Info().Str("client", ClientName).Msg("Stopped consumer message fetch loop")
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
		log.Info().Str("client", ClientName).Msg("All in-flight messages completed and ACKed cleanly")
	case <-shutdownCtx.Done():
		log.Warn().Str("client", ClientName).Msg("Shutdown timeout reached before all in-flight messages finished")
	}

	// 3. Drain and close NATS connection
	if nc != nil {
		if err := nc.Drain(); err != nil {
			log.Error().Err(err).Str("client", ClientName).Msg("Error during NATS connection drain")
		} else {
			log.Info().Str("client", ClientName).Msg("NATS connection successfully drained and closed")
		}
	}
}
