package messaging

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"services/model"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Publish Msg
func JSPublishMsg(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAck, error) {
	header := toNatsHeader(msg.Headers)
	dataBytes, err := serializeData(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message data: %w", err)
	}

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  header,
		Data:    dataBytes,
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
		Str("data", string(dataBytes)).
		Msg("Successfully published message to JetStream")

	return ack, nil
}

// Publish Msg Async
func JSPublishAsync(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAck, error) {
	header := toNatsHeader(msg.Headers)
	dataBytes, err := serializeData(msg.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize message data: %w", err)
	}

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  header,
		Data:    dataBytes,
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
	case <-ctx.Done():
		return nil, ctx.Err()
	case ack := <-future.Ok():
		log.Info().
			Str("subject", msg.Subject).
			Str("stream", ack.Stream).
			Uint64("sequence", ack.Sequence).
			Bool("duplicate", ack.Duplicate).
			Str("msgId", msgID).
			Str("data", string(dataBytes)).
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

// Publish N Messages concurrently with distinct, consistent order numbers for concurrency testing
func JSPublishN(ctx context.Context, js jetstream.JetStream, messages []model.Message) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	orderBatch := r.Intn(9000) + 1000

	futures := make([]jetstream.PubAckFuture, 0, len(messages))
	for i := range messages {
		msg := &messages[i]
		orderID := fmt.Sprintf("ORD-%d-%02d", orderBatch, i+1)

		if msg.Data == nil || msg.Data == "" {
			msg.Data = fmt.Sprintf("Order %s is received for processing", orderID)
		} else {
			msg.Data = fmt.Sprintf("Order %s: %v", orderID, msg.Data)
		}

		header := toNatsHeader(msg.Headers)
		dataBytes, err := serializeData(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to serialize message data: %w", err)
		}

		natsMsg := &nats.Msg{
			Subject: msg.Subject,
			Header:  header,
			Data:    dataBytes,
		}

		var opts []jetstream.PublishOpt
		if header != nil {
			if msgID := header.Get(nats.MsgIdHdr); msgID != "" {
				opts = append(opts, jetstream.WithMsgID(msgID))
			}
		}

		future, err := js.PublishMsgAsync(natsMsg, opts...)
		if err != nil {
			return fmt.Errorf("failed to publish async message: %w", err)
		}
		futures = append(futures, future)
	}

	for _, f := range futures {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-f.Ok():
		case err := <-f.Err():
			return fmt.Errorf("batch async publish failed: %w", err)
		}
	}

	log.Info().Int("count", len(messages)).Msg("Successfully published message batch concurrently to JetStream")
	return nil
}
