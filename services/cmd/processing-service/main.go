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

// Service Parameters
const (
	Client = "processing-service"
	Sub1   = "delivery.atmost-once"
	Sub2   = "delivery.atleast-once"
	Sub3   = "delivery.exactly-once"

	Stream    = "DELIVERY"
	Consumer1 = "at-least-once"
	Consumer2 = "exactly-once"
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
	nc, err := natsclient.ConnectAndDiscover(Client)
	if err != nil {
		log.Fatal().Err(err).Str("client", Client).Msg("Failed to initialize NATS Client")
	}

	js, err := natsclient.NewJSClient(nc)
	if err != nil {
		log.Fatal().Err(err).Str("client", Client).Msg("Failed to initialize JetStream")
	}

	// 3. Setup context and WaitGroup for worker lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 4. Start processors independently
	sub, err := startAtMostOnce(nc, &wg)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("client", Client).
			Str("subject", Sub1).
			Msg("Failed to start at-most-once processor")
	}

	cc1, err := startAtLeastOnce(ctx, js, &wg)
	if err != nil {
		cleanup(sub, nc, &wg)
		log.Fatal().
			Err(err).
			Str("client", Client).
			Str("stream", Stream).
			Str("consumer", Consumer1).
			Msg("Failed to start at-least-once processor")
	}

	cc2, err := startExactlyOnce(ctx, js, &wg)
	if err != nil {
		cleanup(sub, nc, &wg, cc1)
		log.Fatal().
			Err(err).
			Str("client", Client).
			Str("stream", Stream).
			Str("consumer", Consumer2).
			Msg("Failed to start exactly-once processor")
	}

	log.Info().
		Str("client", Client).
		Str("sub1", Sub1).
		Str("consumer1", Consumer1).
		Str("consumer2", Consumer2).
		Msg("All processors started successfully and listening for messages")

	// 5. Wait for OS shutdown signal (Ctrl+C / SIGINT / SIGTERM)
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Info().
		Str("client", Client).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Initiating graceful shutdown... (Press Ctrl+C again to force exit immediately)")

	// Listen for a second Ctrl+C to force exit immediately without waiting for in-flight tasks or sending ACK
	go func() {
		<-sigChan
		log.Warn().
			Str("client", Client).
			Msg("Second shutdown signal received. Forcing immediate termination without ACK...")
		os.Exit(1)
	}()

	// 6. Graceful shutdown of consumer, subscription, and NATS connection
	cleanup(sub, nc, &wg, cc1, cc2)
}

// startAtMostOnce subscribes to Sub1 using Core NATS (at-most-once delivery).
func startAtMostOnce(nc *nats.Conn, wg *sync.WaitGroup) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(Sub1, func(msg *nats.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("client", Client).
			Str("delivery", "at-most-once").
			Str("subject", msg.Subject).
			Str("message", string(msg.Data)).
			Msg("Received message")

		log.Info().
			Str("client", Client).
			Str("delivery", "at-most-once").
			Str("subject", msg.Subject).
			Str("message", string(msg.Data)).
			Msg("Processing message...")

		time.Sleep(2 * time.Second)

		log.Info().
			Str("client", Client).
			Str("delivery", "at-most-once").
			Str("subject", msg.Subject).
			Str("message", string(msg.Data)).
			Msg("Message processed successfully")
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %w", Sub1, err)
	}

	log.Info().
		Str("client", Client).
		Str("delivery", "at-most-once").
		Str("subject", Sub1).
		Msg("Subscription created")

	return sub, nil
}

// startAtLeastOnce configures and starts a JetStream pull consumer on Sub2 (at-least-once delivery).
func startAtLeastOnce(ctx context.Context, js jetstream.JetStream, wg *sync.WaitGroup) (jetstream.ConsumeContext, error) {
	consumer, err := js.CreateOrUpdateConsumer(ctx, Stream, jetstream.ConsumerConfig{
		Durable:       Consumer1,
		FilterSubject: Sub2,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer %s on stream %s: %w", Consumer1, Stream, err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("client", Client).
			Str("delivery", "at-least-once").
			Str("consumer", Consumer1).
			Str("subject", msg.Subject()).
			Str("payload", string(msg.Data())).
			Msg("Received message")

		logger.Countdown(10, "Processing Message")

		// Acknowledge message processing completion back to JetStream
		if err := msg.Ack(); err != nil {
			log.Error().
				Err(err).
				Str("client", Client).
				Str("consumer", Consumer1).
				Msg("Failed to ACK message")
			return
		}
		log.Info().
			Str("client", Client).
			Str("consumer", Consumer1).
			Msg("ACK sent successfully")

	},
		jetstream.PullMaxMessages(1), // Limits the number of messages fetched at once
	)

	if err != nil {
		return nil, fmt.Errorf("failed to start consume on %s: %w", Consumer1, err)
	}

	log.Info().
		Str("client", Client).
		Str("delivery", "at-least-once").
		Str("stream", Stream).
		Str("consumer", Consumer1).
		Msg("Consumer created and consuming")

	return cc, nil
}

func startExactlyOnce(ctx context.Context, js jetstream.JetStream, wg *sync.WaitGroup) (jetstream.ConsumeContext, error) {
	consumer, err := js.CreateOrUpdateConsumer(ctx, Stream, jetstream.ConsumerConfig{
		Durable:       Consumer2,
		FilterSubject: Sub3,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer %s on stream %s: %w", Consumer2, Stream, err)
	}

	cc, err := consumer.Consume(func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		log.Info().
			Str("client", Client).
			Str("delivery", "exactly-once").
			Str("consumer", Consumer2).
			Str("subject", msg.Subject()).
			Str("payload", string(msg.Data())).
			Msg("Received message")

		logger.Countdown(10, "Processing Message")

		// Acknowledge message processing completion back to JetStream using DoubleAck (synchronous server confirmation)
		if err := msg.DoubleAck(ctx); err != nil {
			log.Error().
				Err(err).
				Str("client", Client).
				Str("consumer", Consumer2).
				Msg("Failed to DoubleACK message")
			return
		}
		log.Info().
			Str("client", Client).
			Str("consumer", Consumer2).
			Msg("DoubleACK sent and confirmed by server successfully")

	},
		jetstream.PullMaxMessages(1), // Limits the number of messages fetched at once
	)

	if err != nil {
		return nil, fmt.Errorf("failed to start consume on %s: %w", Consumer2, err)
	}

	log.Info().
		Str("client", Client).
		Str("delivery", "exactly-once").
		Str("stream", Stream).
		Str("consumer", Consumer2).
		Msg("Consumer created and consuming")

	return cc, nil
}

// cleanup performs graceful shutdown of consume contexts, subscription, waits for in-flight tasks, and drains NATS.
func cleanup(sub *nats.Subscription, nc *nats.Conn, wg *sync.WaitGroup, ccs ...jetstream.ConsumeContext) {
	// 1. Stop JetStream consume context fetch loops
	for _, cc := range ccs {
		if cc != nil {
			cc.Stop()
		}
	}
	log.Info().Str("client", Client).Msg("Stopped JetStream consumer message fetch loop(s)")

	// 2. Drain Core NATS subscription to stop receiving new messages while completing in-flight processing
	if sub != nil {
		if err := sub.Drain(); err != nil {
			log.Error().Err(err).Str("client", Client).Msg("Error during subscription drain")
		} else {
			log.Info().Str("client", Client).Msg("Subscription drained successfully")
		}
	}

	// 3. Wait for in-flight message processing with a bounded timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Str("client", Client).Msg("All in-flight messages completed cleanly")
	case <-shutdownCtx.Done():
		log.Warn().Str("client", Client).Msg("Shutdown timeout reached before all in-flight messages finished")
	}

	// 4. Drain and close NATS connection
	if nc != nil {
		if err := nc.Drain(); err != nil {
			log.Error().Err(err).Str("client", Client).Msg("Error during NATS connection drain")
		} else {
			log.Info().Str("client", Client).Msg("NATS connection successfully drained and closed")
		}
	}
}
