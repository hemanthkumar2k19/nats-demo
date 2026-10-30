package messaging

import (
	"context"
	"fmt"
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

// GetStreamSequenceBounds retrieves available boundary sequences and timestamps for a stream to guide replay requests.
func GetStreamSequenceBounds(ctx context.Context, js jetstream.JetStream, streamName string) (*model.StreamSequencesResponse, error) {
	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		return nil, err
	}

	streamInfo, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	state := streamInfo.State
	return &model.StreamSequencesResponse{
		StreamName:     streamName,
		FirstSequence:  state.FirstSeq,
		LastSequence:   state.LastSeq,
		FirstTimestamp: model.FormattedTime(state.FirstTime),
		LastTimestamp:  model.FormattedTime(state.LastTime),
		MessagesCount:  state.Msgs,
		Bytes:          state.Bytes,
		DeletedCount:   state.NumDeleted,
	}, nil
}

// [TODO: USER_IMPLEMENT]
// GetStreamMsg retrieves a raw stored message from a stream by its sequence number.
// Logic to be implemented by user.
func GetStreamMsg(ctx context.Context, js jetstream.JetStream, stream string, seq uint64) (*jetstream.RawStreamMsg, error) {
	// Dummy stub: User to implement logic using js.GetMsg(ctx, stream, seq)
	return nil, nil
}
