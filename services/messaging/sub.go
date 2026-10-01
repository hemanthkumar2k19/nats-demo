package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// Sub Handler
func Subscribe(nc *nats.Conn, subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := nc.Subscribe(subject, handler)

	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}

	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}

// Sync Sub Handler
func SyncSubscribe(nc *nats.Conn, subject string, timeout time.Duration, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := nc.SubscribeSync(subject)

	if err != nil {
		log.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe on NATS")
		return nil, fmt.Errorf("error while subscribing on NATS: %w", err)
	}

	log.Info().Str("subject", subject).Msg("Successfully subscribed on NATS")
	return sub, nil
}
