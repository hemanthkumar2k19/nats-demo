package messaging

import (
	"context"
	"fmt"
	"log"
	"services/model"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Publish Msg
func JSPublishMsg(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAck, error) {

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  toNatsHeader(msg.Headers),
		Data:    []byte(msg.Data),
	}

	ack, err := js.PublishMsg(ctx, natsMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %v", err)
	}

	log.Printf("Successfully published message, ack: %v", ack)

	return ack, nil
}

// Publish Msg Async
func JSPublishAsync(ctx context.Context, js jetstream.JetStream, msg *model.Message) (*jetstream.PubAckFuture, error) {

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  toNatsHeader(msg.Headers),
		Data:    []byte(msg.Data),
	}

	future, err := js.PublishMsgAsync(natsMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish message: %v", err)
	}

	select {
	case ack := <-future.Ok():
		log.Printf("Published to stream=%s sequence=%d", ack.Stream, ack.Sequence)
	case err := <-future.Err():
		return nil, fmt.Errorf("async publish failed: %w", err)
	}

	return &future, nil
}
