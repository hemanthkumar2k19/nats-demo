package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// [TODO: USER_CHANGE]
// Current Subscribe hardcodes OrderHandler and cannot be used by generic services requiring custom callbacks.
// Sub Handler
func Subscribe(nc *nats.Conn, subject string) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(subject, OrderHandler)

	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}

	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}

// [TODO: USER_IMPLEMENT]
// SubscribeWithHandler creates a Core NATS subscription with a custom message handler.
// Logic to be implemented by user.
func SubscribeWithHandler(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	// Dummy stub: User to implement logic using nc.Subscribe(subject, handler)
	return nil, nil
}

// [TODO: USER_IMPLEMENT]
// QueueSubscribe creates a Core NATS queue subscription for load distribution across competing workers.
// Logic to be implemented by user.
func QueueSubscribe(nc *nats.Conn, subject, queueGroup string, handler nats.MsgHandler) (*nats.Subscription, error) {
	// Dummy stub: User to implement logic using nc.QueueSubscribe(subject, queueGroup, handler)
	return nil, nil
}

// Sub Handler - Reply
func SubscribeReply(nc *nats.Conn, subject string) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(subject, ReplyHandler)

	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}

	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}

// Sync Sub Handler
func SyncSubscribe(nc *nats.Conn, subject string, timeout time.Duration) (*nats.Subscription, error) {
	sub, err := nc.SubscribeSync(subject)

	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}

	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}
