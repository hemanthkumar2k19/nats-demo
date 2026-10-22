package messaging

import (
	"context"
	"fmt"
	"services/model"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Publish Msg
func JSPublishMsg(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAck, error) {
	header := toNatsHeader(msg.Headers)
	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  header,
		Data:    []byte(msg.Data),
	}

	var opts []jetstream.PublishOpt
	var msgID string
	if header != nil {
		msgID = header.Get(nats.MsgIdHdr)
		if msgID != "" {
			opts = append(opts, jetstream.WithMsgID(msgID))
		}
	}

	ack, err := js.PublishMsg(ctx, natsMsg, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %v", err)
	}

	log.Info().
		Str("subject", msg.Subject).
		Str("stream", ack.Stream).
		Uint64("sequence", ack.Sequence).
		Bool("duplicate", ack.Duplicate).
		Str("msgId", msgID).
		Msg("Successfully published message to JetStream")

	return ack, nil
}

// Publish Msg Async
func JSPublishAsync(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAck, error) {
	header := toNatsHeader(msg.Headers)
	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  header,
		Data:    []byte(msg.Data),
	}

	var opts []jetstream.PublishOpt
	var msgID string
	if header != nil {
		msgID = header.Get(nats.MsgIdHdr)
		if msgID != "" {
			opts = append(opts, jetstream.WithMsgID(msgID))
		}
	}

	future, err := js.PublishMsgAsync(natsMsg, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %v", err)
	}

	select {
	case ack := <-future.Ok():
		log.Info().
			Str("subject", msg.Subject).
			Str("stream", ack.Stream).
			Uint64("sequence", ack.Sequence).
			Bool("duplicate", ack.Duplicate).
			Str("msgId", msgID).
			Msg("Successfully published async message to JetStream")
		return ack, nil
	case err := <-future.Err():
		return nil, fmt.Errorf("async publish failed: %w", err)
	}
}

// Publish Async Pending
func JSPublishAsyncPending(js jetstream.JetStream) map[string]any {
	pending := js.PublishAsyncPending()
	log.Info().Msgf("Pending messages: %d", pending)

	return map[string]any{
		"pending": pending,
	}
}
