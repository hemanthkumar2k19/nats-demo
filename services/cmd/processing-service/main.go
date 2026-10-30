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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load environment configuration")
	}

	// 2. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover(&cfg.NATS, Client)
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
		messaging.GracefulShutdown(Client, nc, &wg, 5*time.Second, []*nats.Subscription{sub})
		log.Fatal().
			Err(err).
			Str("client", Client).
			Str("stream", Stream).
			Str("consumer", Consumer1).
			Msg("Failed to start at-least-once processor")
	}

	cc2, err := startExactlyOnce(ctx, js, &wg)
	if err != nil {
		messaging.GracefulShutdown(Client, nc, &wg, 5*time.Second, []*nats.Subscription{sub}, cc1)
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
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	log.Info().
		Str("client", Client).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Initiating graceful shutdown...")

	// 6. Graceful shutdown of consumer, subscription, and NATS connection
	messaging.GracefulShutdown(Client, nc, &wg, 5*time.Second, []*nats.Subscription{sub}, cc1, cc2)
}

// startAtMostOnce subscribes to Sub1 using Core NATS (at-most-once delivery).
func startAtMostOnce(nc *nats.Conn, wg *sync.WaitGroup) (*nats.Subscription, error) {
	sub, err := messaging.SubscribeWithHandler(nc, Sub1, func(msg *nats.Msg) {
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

		time.Sleep(500 * time.Millisecond)

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
	consumer, err := messaging.CreateConsumer(ctx, js, Stream, jetstream.ConsumerConfig{
		Durable:       Consumer1,
		FilterSubject: Sub2,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       10 * time.Second,
		MaxDeliver:    3,
		BackOff: []time.Duration{
			2 * time.Second,
			5 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer %s on stream %s: %w", Consumer1, Stream, err)
	}

	cc, err := messaging.Consume(consumer, func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		meta, _ := msg.Metadata()
		var deliveryAttempt uint64 = 1
		if meta != nil {
			deliveryAttempt = meta.NumDelivered
		}

		log.Info().
			Str("client", Client).
			Str("delivery", "at-least-once").
			Str("consumer", Consumer1).
			Str("subject", msg.Subject()).
			Uint64("attempt", deliveryAttempt).
			Str("payload", string(msg.Data())).
			Msg("Received message")

		messaging.ExecuteSimulationAction(ctx, msg, Client, Consumer1, "at-least-once", deliveryAttempt)
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
	consumer, err := messaging.CreateConsumer(ctx, js, Stream, jetstream.ConsumerConfig{
		Durable:       Consumer2,
		FilterSubject: Sub3,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       10 * time.Second,
		MaxDeliver:    3,
		BackOff: []time.Duration{
			2 * time.Second,
			5 * time.Second,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create or update consumer %s on stream %s: %w", Consumer2, Stream, err)
	}

	cc, err := messaging.Consume(consumer, func(msg jetstream.Msg) {
		wg.Add(1)
		defer wg.Done()

		meta, _ := msg.Metadata()
		var deliveryAttempt uint64 = 1
		if meta != nil {
			deliveryAttempt = meta.NumDelivered
		}

		log.Info().
			Str("client", Client).
			Str("delivery", "exactly-once").
			Str("consumer", Consumer2).
			Str("subject", msg.Subject()).
			Uint64("attempt", deliveryAttempt).
			Str("payload", string(msg.Data())).
			Msg("Received message")

		messaging.ExecuteSimulationAction(ctx, msg, Client, Consumer2, "exactly-once", deliveryAttempt)
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
