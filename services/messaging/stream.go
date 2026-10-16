package messaging

import (
	"context"
	"services/model"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

func CreateStream(ctx context.Context, js jetstream.JetStream, sr model.CreateStreamRequest) (jetstream.Stream, error) {
	stream, err := js.Stream(ctx, sr.Name)
	if err == nil {
		return stream, nil
	}

	if err == jetstream.ErrStreamNotFound {

		cfg := jetstream.StreamConfig{
			Name:        sr.Name,
			Description: sr.Description,
			Subjects:    sr.Subjects,
		}

		if sr.Retention != "" {
			cfg.Retention = ResolveRetentionPolicy(sr.Retention)
		}

		if sr.Discard != "" {
			cfg.Discard = ResolveDiscardPolicy(sr.Discard)
		}

		if sr.Replicas != 0 {
			cfg.Replicas = sr.Replicas
		}

		if sr.Storage != "" {
			cfg.Storage = ResolveStorageType(sr.Storage)
		}

		stream, err = js.CreateStream(ctx, cfg)
		if err != nil {
			return nil, err
		}
		log.Info().Msg("Stream created: " + sr.Name)
		return stream, nil
	}

	return nil, err
}

func DeleteStream(ctx context.Context, js jetstream.JetStream, streamName string) error {
	err := js.DeleteStream(ctx, streamName)
	if err == nil {
		log.Info().Msg("Stream deleted: " + streamName)
	}

	if err == jetstream.ErrStreamNotFound {
		log.Info().Msg("Stream not found: " + streamName)
		return nil
	}

	return err
}
