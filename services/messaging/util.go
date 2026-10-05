package messaging

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// toNatsHeader converts map[string]any headers to nats.Header.
func toNatsHeader(headers map[string]any) nats.Header {
	if len(headers) == 0 {
		return nil
	}
	h := make(nats.Header)
	for k, v := range headers {
		h.Set(k, fmt.Sprintf("%v", v))
	}
	return h
}

// toMap converts nats.Header to map[string]interface{}.
func toMap(header nats.Header) map[string]interface{} {
	if header == nil {
		return nil
	}
	result := make(map[string]interface{})
	for k, v := range header {
		result[k] = v
	}
	return result
}

func OrderHandler(msg *nats.Msg) {
	log.Info().
		Str("subject", msg.Subject).
		Any("headers", toMap(msg.Header)).
		Str("payload", string(msg.Data)).
		Msg("Received order!")

	log.Info().
		Str("subject", msg.Subject).
		Any("headers", toMap(msg.Header)).
		Str("payload", string(msg.Data)).
		Msg("Processing order...")

	time.Sleep(time.Second * 3)

	log.Info().
		Str("subject", msg.Subject).
		Any("headers", toMap(msg.Header)).
		Str("payload", string(msg.Data)).
		Msg("Order processed successfully")
}

func ReplyHandler(msg *nats.Msg) {

	// Verify requester provided a reply subject
	if len(msg.Reply) == 0 {
		log.Error().Msg("Error: received message without reply subject")
		return
	}

	// Structured response with headers using msg.RespondMsg
	replyMsg := nats.NewMsg(msg.Reply)
	replyMsg.Data = []byte(`{"valid": true, "reason": "Order approved"}`)
	replyMsg.Header.Set("Status-Code", "200")
	replyMsg.Header.Set("Content-Type", "application/json")

	if err := msg.RespondMsg(replyMsg); err != nil {
		log.Error().Msg("Failed to send structured response")
	}
}
