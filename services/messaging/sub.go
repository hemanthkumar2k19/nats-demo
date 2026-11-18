package messaging

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// SubscribeWithHandler creates a Core NATS subscription with a custom message handler.
func SubscribeWithHandler(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}
	sub, err := nc.Subscribe(subject, handler)
	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}
	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}

// QueueSubscribe creates a Core NATS queue subscription for load distribution across competing workers.
func QueueSubscribe(nc *nats.Conn, subject, queueGroup string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}
	sub, err := nc.QueueSubscribe(subject, queueGroup, handler)
	if err != nil {
		log.Error().Err(err).Str("subject", subject).Str("queue_group", queueGroup).Msg("Failed to queue subscribe on NATS")
		return nil, fmt.Errorf("failed to queue subscribe to %s (%s): %w", subject, queueGroup, err)
	}
	log.Info().Str("subject", subject).Str("queue_group", queueGroup).Msg("Successfully queue subscribed on NATS")
	return sub, nil
}

// SyncSubscribe creates a synchronous Core NATS subscription. Messages are pulled synchronously via sub.NextMsg(timeout).
func SyncSubscribe(nc *nats.Conn, subject string) (*nats.Subscription, error) {
	if nc == nil {
		return nil, fmt.Errorf("nats connection is nil")
	}
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to sync subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}
	log.Info().Str("subject", subject).Msg("Successfully sync subscribed on NATS")
	return sub, nil
}
