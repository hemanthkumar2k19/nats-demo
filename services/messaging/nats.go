package messaging

import (
	"fmt"
	"services/model"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// Publish on NATS
func PublishMsg(nc *nats.Conn, msg *model.Message) error {
	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  toNatsHeader(msg.Headers),
		Data:    []byte(msg.Data),
	}

	err := nc.PublishMsg(natsMsg)
	if err != nil {
		log.Error().Err(err).Str("subject", msg.Subject).Msg("Failed to publish message on NATS")
		return fmt.Errorf("error while publishing on NATS: %w", err)
	}

	log.Info().
		Str("subject", msg.Subject).
		Interface("headers", msg.Headers).
		Str("data", msg.Data).
		Msg("Successfully published message on NATS")
	return nil
}

// Publish with Reply Subject
func PublishMsgWithReply(nc *nats.Conn, msg *model.Message) error {
	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Reply:   msg.Reply,
		Header:  toNatsHeader(msg.Headers),
		Data:    []byte(msg.Data),
	}

	err := nc.PublishMsg(natsMsg)
	if err != nil {
		log.Error().
			Err(err).
			Str("subject", msg.Subject).
			Str("reply", msg.Reply).
			Msg("Failed to publish message with reply on NATS")
		return fmt.Errorf("error while publishing on NATS: %w", err)
	}

	log.Info().
		Str("subject", msg.Subject).
		Str("reply", msg.Reply).
		Interface("headers", msg.Headers).
		Str("data", msg.Data).
		Msg("Successfully published message with reply subject on NATS")
	return nil
}

// Publish Request
func PublishRequest(nc *nats.Conn, msg *model.Message) (*nats.Msg, error) {

	natsMsg := &nats.Msg{
		Subject: msg.Subject,
		Header:  toNatsHeader(msg.Headers),
		Data:    []byte(msg.Data),
	}

	resp, err := nc.RequestMsg(natsMsg, msg.Timeout)
	if err != nil {
		log.Error().Err(err).Str("subject", msg.Subject).Msg("Failed to publish request on NATS")
		return nil, fmt.Errorf("error while publishing request on NATS: %w", err)
	}

	log.Info().
		Str("subject", msg.Subject).
		Interface("headers", msg.Headers).
		Str("data", msg.Data).
		Str("response", string(resp.Data)).
		Msg("Successfully published request on NATS")
	return resp, nil
}

// Flush
func Flush(nc *nats.Conn) error {
	err := nc.Flush()
	if err != nil {
		log.Error().Err(err).Msg("Failed to flush NATS connection")
		return fmt.Errorf("error while flushing on NATS: %w", err)
	}

	log.Info().Msg("Successfully flushed NATS connection")
	return nil
}
