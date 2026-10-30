package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"services/model"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// CreateConsumer idempotently retrieves an existing consumer or creates a new one with cfg.
func CreateConsumer(ctx context.Context, js jetstream.JetStream, stream string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	consumerName := cfg.Name
	if consumerName == "" {
		consumerName = cfg.Durable
	}

	consumer, err := js.Consumer(ctx, stream, consumerName)
	if err == nil {
		info, _ := consumer.Info(ctx)
		log.Info().Str("stream", stream).Str("consumer", info.Name).Msg("Consumer already exists")
		return consumer, nil
	}

	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		log.Info().Str("stream", stream).Str("consumer", consumerName).Msg("Creating new consumer")
		consumer, err = js.CreateConsumer(ctx, stream, cfg)
		if err != nil {
			return nil, err
		}
		info, _ := consumer.Info(ctx)
		log.Info().Str("stream", stream).Str("consumer", info.Name).Msg("Consumer created successfully")
		return consumer, nil
	}

	return nil, err
}

func DeleteConsumer(ctx context.Context, js jetstream.JetStream, cr *model.CreateConsumerRequest) error {
	err := js.DeleteConsumer(ctx, cr.StreamName, cr.ConsumerName)
	if err == nil {
		log.Info().Msg("Consumer deleted: " + cr.ConsumerName)
	}

	if err == jetstream.ErrConsumerNotFound {
		log.Info().Msg("Consumer not found: " + cr.ConsumerName)
		return nil
	}

	return err
}

// [TODO: USER_IMPLEMENT]
// GetConsumer retrieves a reference to an existing consumer on the given stream.
// Logic to be implemented by user.
func GetConsumer(ctx context.Context, js jetstream.JetStream, stream, consumer string) (jetstream.Consumer, error) {
	// Dummy stub: User to implement logic using js.Consumer(ctx, stream, consumer)
	return nil, nil
}

// [TODO: USER_IMPLEMENT]
// Consume binds a message handler loop to a JetStream consumer and returns its ConsumeContext.
// Logic to be implemented by user.
func Consume(consumer jetstream.Consumer, handler func(msg jetstream.Msg), opts ...jetstream.PullConsumeOpt) (jetstream.ConsumeContext, error) {
	// Dummy stub: User to implement logic using consumer.Consume(handler, opts...)
	return nil, nil
}

// ProcessBusinessMessage simulates and applies domain business effects for a reprocessed message.
func ProcessBusinessMessage(ctx context.Context, seq uint64, subject string, data []byte, headers map[string][]string) error {
	log.Info().
		Uint64("sequence", seq).
		Str("subject", subject).
		Str("data", string(data)).
		Msg("Executing business effect for reprocessed message")
	// Reference implementation: In production, this invokes domain order/payment reconciliation,
	// state mutations, ledgers, or downstream dispatching.
	return nil
}

// processEphemeralConsumer creates an ephemeral pull consumer, iterates available messages using FetchNoWait,
// captures boundary statistics, and executes business or audit handlers per message.
func processEphemeralConsumer(
	ctx context.Context,
	js jetstream.JetStream,
	streamName string,
	filterSubjects []string,
	rng RangeParams,
	batchSize int,
	effect ConsumerEffect,
) (*ReplaySummary, error) {
	if err := validateRange(rng); err != nil {
		return nil, err
	}

	if batchSize <= 0 {
		batchSize = defaultReplayBatchSize
	} else if batchSize > maxReplayBatchSize {
		batchSize = maxReplayBatchSize
	}

	prefix := "replay"
	consumerType := "historical-replay"
	if effect == EffectBusiness {
		prefix = "reprocess"
		consumerType = "message-reprocessing"
	}
	consumerName := fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())

	cc := jetstream.ConsumerConfig{
		Name:              consumerName,
		FilterSubjects:    filterSubjects,
		ReplayPolicy:      jetstream.ReplayInstantPolicy,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: 5 * time.Minute,
		Metadata: map[string]string{
			"type":   consumerType,
			"effect": string(effect),
		},
	}

	if rng.Type == RangeBySequence {
		cc.DeliverPolicy = jetstream.DeliverByStartSequencePolicy
		cc.OptStartSeq = rng.StartSequence
	} else {
		cc.DeliverPolicy = jetstream.DeliverByStartTimePolicy
		cc.OptStartTime = &rng.StartTime
	}

	cons, err := js.CreateConsumer(ctx, streamName, cc)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s consumer: %w", prefix, err)
	}
	defer cleanup(ctx, js, streamName, consumerName)

	log.Info().
		Str("consumer", consumerName).
		Str("rangeType", string(rng.Type)).
		Str("effect", string(effect)).
		Int("batchSize", batchSize).
		Msgf("Message %s started", prefix)

	var firstSeq, lastSeq uint64
	var firstTime, lastTime *model.FormattedTime
	count := 0
	nextSeq := rng.StartSequence
	reachedEnd := false

	for {
		if ctx.Err() != nil {
			return &ReplaySummary{
				FirstSequence:  firstSeq,
				LastSequence:   lastSeq,
				FirstTimestamp: firstTime,
				LastTimestamp:  lastTime,
				Count:          count,
			}, ctx.Err()
		}

		if rng.Type == RangeBySequence && nextSeq > rng.EndSequence {
			break
		}

		currentBatch := batchSize
		if rng.Type == RangeBySequence {
			remaining := rng.EndSequence - nextSeq + 1
			if remaining < uint64(currentBatch) {
				currentBatch = int(remaining)
			}
		}

		batch, err := cons.FetchNoWait(currentBatch)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s messages: %w", prefix, err)
		}

		msgCount := 0

		for msg := range batch.Messages() {
			metadata, err := msg.Metadata()
			if err != nil {
				return nil, fmt.Errorf("get message metadata: %w", err)
			}

			seq := metadata.Sequence.Stream

			if rng.Type == RangeBySequence && seq > rng.EndSequence {
				reachedEnd = true
				break
			}

			if rng.Type == RangeByTime && rng.EndTime != nil && !rng.EndTime.IsZero() && metadata.Timestamp.After(*rng.EndTime) {
				reachedEnd = true
				break
			}

			nextSeq = seq + 1
			t := model.FormattedTime(metadata.Timestamp)
			if count == 0 {
				firstSeq = seq
				firstTime = &t
			}
			lastSeq = seq
			lastTime = &t
			count++

			dataStr := string(msg.Data())

			if effect == EffectBusiness {
				if err := ProcessBusinessMessage(ctx, seq, msg.Subject(), msg.Data(), msg.Headers()); err != nil {
					log.Error().Err(err).Uint64("sequence", seq).Msg("Business reprocessing failed for message")
					return nil, fmt.Errorf("reprocess message sequence %d: %w", seq, err)
				}
				evt := log.Info().
					Uint64("sequence", seq).
					Str("subject", msg.Subject()).
					Str("data", dataStr).
					Str("timestamp", t.String())
				if len(msg.Headers()) > 0 {
					evt = evt.Any("headers", msg.Headers())
				}
				evt.Msg("Reprocessing message completed with business effect")
			} else {
				evt := log.Info().
					Uint64("sequence", seq).
					Str("subject", msg.Subject()).
					Str("data", dataStr).
					Str("timestamp", t.String())
				if len(msg.Headers()) > 0 {
					evt = evt.Any("headers", msg.Headers())
				}
				evt.Msg("Replaying message (audit-only, no business effect)")
			}
		}

		if err := batch.Error(); err != nil {
			return nil, fmt.Errorf("read %s batch: %w", prefix, err)
		}

		if reachedEnd || msgCount == 0 || (rng.Type == RangeBySequence && nextSeq > rng.EndSequence) {
			break
		}
	}

	log.Info().
		Str("consumer", consumerName).
		Int("count", count).
		Msgf("Message %s completed", prefix)

	return &ReplaySummary{
		FirstSequence:  firstSeq,
		LastSequence:   lastSeq,
		FirstTimestamp: firstTime,
		LastTimestamp:  lastTime,
		Count:          count,
	}, nil
}

// ReplayMessages replays stream messages for historical audit without business side-effects.
func ReplayMessages(
	ctx context.Context,
	js jetstream.JetStream,
	streamName string,
	filterSubjects []string,
	rng RangeParams,
	batchSize int,
) (*ReplaySummary, error) {
	return processEphemeralConsumer(ctx, js, streamName, filterSubjects, rng, batchSize, EffectAudit)
}

// ReprocessMessages fetches stream messages and applies domain business processing logic for each message.
func ReprocessMessages(
	ctx context.Context,
	js jetstream.JetStream,
	streamName string,
	filterSubjects []string,
	rng RangeParams,
	batchSize int,
) (*ReplaySummary, error) {
	return processEphemeralConsumer(ctx, js, streamName, filterSubjects, rng, batchSize, EffectBusiness)
}

func cleanup(ctx context.Context, js jetstream.JetStream, streamName string, consumerName string) {
	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		5*time.Second,
	)
	defer cancel()

	if err := js.DeleteConsumer(
		cleanupCtx,
		streamName,
		consumerName,
	); err != nil && err != jetstream.ErrConsumerNotFound {
		log.Error().
			Err(err).
			Str("consumer", consumerName).
			Msg("Failed to delete replay consumer")
	}
}
