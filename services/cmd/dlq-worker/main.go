package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"
	"time"

	"services/config"
	"services/logger"
	"services/messaging"
	"services/model"
	"services/natsclient"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// DLQ parameters
const (
	DLQWorkerID    = "dlq-worker"
	DLQStreamName  = "dlq_stream"
	DLQConsumer    = "dlq-consumer"
	DLQAdvisorySub = "$JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.>"
)

func main() {
	// 0. Initialize Zerolog Logger
	logger.Init(false)

	// 1. Load Environment Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load environment configuration")
	}

	// 2. Connect to NATS Server
	nc, err := natsclient.ConnectAndDiscover(&cfg.NATS, DLQWorkerID)
	if err != nil {
		log.Fatal().Err(err).Str("worker", DLQWorkerID).Msg("Failed to connect to NATS")
	}
	defer nc.Close()

	// 3. Initialize JetStream Context
	js, err := natsclient.NewJSClient(nc)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize JetStream context")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 4. Ensure dlq_stream exists
	_, err = messaging.CreateStream(ctx, js, model.CreateStreamRequest{
		Name:        DLQStreamName,
		Description: "Dead Letter Queue stream capturing MaxDeliveries advisories",
		Subjects:    []string{DLQAdvisorySub},
		Storage:     "file",
		Retention:   "limits",
	})
	if err != nil {
		log.Fatal().Err(err).Str("stream", DLQStreamName).Msg("Failed to ensure dlq_stream exists")
	}

	// 5. Create Durable Pull Consumer on dlq_stream
	consumer, err := messaging.CreateConsumer(ctx, js, DLQStreamName, jetstream.ConsumerConfig{
		Durable:       DLQConsumer,
		Name:          DLQConsumer,
		Description:   "Dedicated worker consumer for dead-lettered job advisories",
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		DeliverPolicy: jetstream.DeliverAllPolicy,
	})
	if err != nil {
		log.Fatal().Err(err).Str("consumer", DLQConsumer).Msg("Failed to configure DLQ consumer")
	}

	log.Info().
		Str("worker", DLQWorkerID).
		Str("stream", DLQStreamName).
		Str("consumer", DLQConsumer).
		Msg("DLQ Worker ready and consuming dead-lettered advisories")

	// 6. Start Consuming Advisory Events
	consumeCtx, err := messaging.Consume(consumer, func(msg jetstream.Msg) {
		handleDLQAdvisory(ctx, js, msg)
	}, jetstream.PullMaxMessages(10))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start DLQ consumer pull loop")
	}
	if consumeCtx != nil {
		defer consumeCtx.Stop()
	}

	// 7. Wait for Graceful Shutdown Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan

	log.Info().
		Str("worker", DLQWorkerID).
		Str("signal", sig.String()).
		Msg("Received shutdown signal. Stopping DLQ Worker gracefully...")

	messaging.GracefulShutdown(DLQWorkerID, nc, nil, 5*time.Second, nil, consumeCtx)
}

// handleDLQAdvisory parses the NATS server advisory, fetches the original message payload from the source stream,
// logs comprehensive forensic audit information, and acknowledges the advisory event.
func handleDLQAdvisory(ctx context.Context, js jetstream.JetStream, msg jetstream.Msg) {
	var adv model.MaxDeliverAdvisory
	if err := json.Unmarshal(msg.Data(), &adv); err != nil {
		log.Error().
			Err(err).
			Str("raw_advisory", string(msg.Data())).
			Msg("Failed to parse MaxDeliverAdvisory JSON payload")
		messaging.AcknowledgeMessage(ctx, msg, DLQWorkerID, DLQConsumer, "at-least-once")
		return
	}

	log.Warn().
		Str("event", "POISON_MESSAGE_MAX_DELIVERIES_EXHAUSTED").
		Str("origin_stream", adv.Stream).
		Str("origin_consumer", adv.Consumer).
		Uint64("origin_sequence", adv.StreamSeq).
		Uint64("delivery_attempts", adv.Deliveries).
		Time("advisory_time", adv.Timestamp).
		Msg("Dead-letter event detected: Consumer retries exhausted")

	// Fetch original message content via messaging utility using stream_seq
	rawMsg, err := messaging.GetStreamMsg(ctx, js, adv.Stream, adv.StreamSeq)
	if err != nil {
		log.Error().
			Err(err).
			Str("stream", adv.Stream).
			Uint64("sequence", adv.StreamSeq).
			Msg("Failed to fetch original message payload from origin stream")
	} else if rawMsg != nil {
		headersMap := make(map[string][]string)
		if rawMsg.Header != nil {
			for k, v := range rawMsg.Header {
				headersMap[k] = v
			}
		}

		log.Info().
			Str("event", "DLQ_PAYLOAD_CAPTURED").
			Str("origin_stream", adv.Stream).
			Uint64("origin_sequence", adv.StreamSeq).
			Str("origin_subject", rawMsg.Subject).
			Interface("headers", headersMap).
			Str("original_payload", string(rawMsg.Data)).
			Time("original_timestamp", rawMsg.Time).
			Msg("Original dead-letter payload successfully retrieved for forensic audit")
	}

	// Settle the DLQ advisory message so it is not redelivered
	messaging.AcknowledgeMessage(ctx, msg, DLQWorkerID, DLQConsumer, "at-least-once")
}
