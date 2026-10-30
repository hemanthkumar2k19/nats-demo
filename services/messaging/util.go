package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

func ResolveStorageType(s string) jetstream.StorageType {
	switch s {
	case "memory":
		return jetstream.MemoryStorage
	case "file":
		return jetstream.FileStorage
	default:
		return jetstream.FileStorage
	}
}

func ResolveDiscardPolicy(s string) jetstream.DiscardPolicy {
	switch s {
	case "old":
		return jetstream.DiscardOld
	case "new":
		return jetstream.DiscardNew
	default:
		return jetstream.DiscardOld
	}
}

func ResolveRetentionPolicy(s string) jetstream.RetentionPolicy {
	switch s {
	case "interest":
		return jetstream.InterestPolicy
	case "workqueue":
		return jetstream.WorkQueuePolicy
	case "limits":
		return jetstream.LimitsPolicy
	default:
		return jetstream.LimitsPolicy
	}
}

func ResolveAckPolicy(policy string) jetstream.AckPolicy {
	switch policy {
	case "explicit":
		return jetstream.AckExplicitPolicy
	case "none":
		return jetstream.AckNonePolicy
	default:
		return jetstream.AckExplicitPolicy
	}
}

func ResolveDeliverPolicy(policy string) jetstream.DeliverPolicy {
	switch policy {
	case "all":
		return jetstream.DeliverAllPolicy
	case "last":
		return jetstream.DeliverLastPolicy
	case "new":
		return jetstream.DeliverNewPolicy
	default:
		return jetstream.DeliverAllPolicy
	}
}

// toNatsHeader converts map[string]any headers to nats.Header.
func toNatsHeader(headers map[string]any) nats.Header {
	if len(headers) == 0 {
		return nil
	}
	h := make(nats.Header)
	for k, v := range headers {
		if strings.EqualFold(k, nats.MsgIdHdr) {
			h.Set(nats.MsgIdHdr, fmt.Sprintf("%v", v))
		} else {
			h.Set(k, fmt.Sprintf("%v", v))
		}
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

// serializeData converts string, []byte, or any JSON-serializable object to []byte.
func serializeData(data any) ([]byte, error) {
	if data == nil {
		return []byte{}, nil
	}
	switch v := data.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return json.Marshal(v)
	}
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

// ExtractSimulationAction inspects the X-Simulation-Action header (or JSON payload field "simulateAction")
// returning normalized action strings: "ack", "nak", "nak-delay", "term", "in-progress", or "timeout".
func ExtractSimulationAction(msg jetstream.Msg) string {
	if headers := msg.Headers(); headers != nil {
		if val := headers.Get("X-Simulation-Action"); val != "" {
			return strings.ToLower(strings.TrimSpace(val))
		}
	}

	// Fallback to checking payload JSON field: {"simulateAction": "..."}
	var body map[string]any
	if err := json.Unmarshal(msg.Data(), &body); err == nil {
		if val, ok := body["simulateAction"]; ok && val != nil {
			return strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", val)))
		}
	}

	return "ack"
}

// ExecuteSimulationAction standardizes message processing simulation across all NATS acknowledgment verbs.
func ExecuteSimulationAction(ctx context.Context, msg jetstream.Msg, clientName, consumerName, deliveryMode string, attempt uint64) {
	action := ExtractSimulationAction(msg)

	switch action {
	case "nak":
		log.Warn().
			Str("client", clientName).
			Str("consumer", consumerName).
			Str("delivery", deliveryMode).
			Uint64("attempt", attempt).
			Msg("SIMULATION [nak]: Simulating transient error -> Sending msg.Nak() (message will be redelivered per BackOff)")
		_ = msg.Nak()

	case "nak-delay":
		delay := 5 * time.Second
		log.Warn().
			Str("client", clientName).
			Str("consumer", consumerName).
			Str("delivery", deliveryMode).
			Uint64("attempt", attempt).
			Dur("delay", delay).
			Msg("SIMULATION [nak-delay]: Simulating rate limit -> Sending msg.NakWithDelay(5s)")
		_ = msg.NakWithDelay(delay)

	case "term":
		log.Error().
			Str("client", clientName).
			Str("consumer", consumerName).
			Str("delivery", deliveryMode).
			Uint64("attempt", attempt).
			Msg("SIMULATION [term]: Simulating poison pill -> Sending msg.Term() (permanently terminated without redelivery)")
		_ = msg.Term()

	case "in-progress":
		log.Info().
			Str("client", clientName).
			Str("consumer", consumerName).
			Str("delivery", deliveryMode).
			Uint64("attempt", attempt).
			Msg("SIMULATION [in-progress]: Simulating long-running job -> Sending msg.InProgress() heartbeat to reset AckWait")
		_ = msg.InProgress()

		time.Sleep(1 * time.Second) // Process additional work after extending timeout
		AcknowledgeMessage(ctx, msg, clientName, consumerName, deliveryMode)

	case "timeout":
		log.Warn().
			Str("client", clientName).
			Str("consumer", consumerName).
			Str("delivery", deliveryMode).
			Uint64("attempt", attempt).
			Msg("SIMULATION [timeout]: Simulating worker hang -> Letting AckWait expire without ACK (server will auto-redeliver)")
		// Do nothing; allow server-side AckWait timer to expire

	default:
		// Standard Happy Path
		time.Sleep(500 * time.Millisecond) // Clean simulated processing time
		AcknowledgeMessage(ctx, msg, clientName, consumerName, deliveryMode)
	}
}

// AcknowledgeMessage acknowledges completion back to JetStream (DoubleAck for exactly-once, Ack for at-least-once).
func AcknowledgeMessage(ctx context.Context, msg jetstream.Msg, clientName, consumerName, deliveryMode string) {
	if deliveryMode == "exactly-once" {
		if err := msg.DoubleAck(ctx); err != nil {
			log.Error().
				Err(err).
				Str("client", clientName).
				Str("consumer", consumerName).
				Msg("Failed to DoubleACK message")
			return
		}
		log.Info().
			Str("client", clientName).
			Str("consumer", consumerName).
			Msg("DoubleACK sent and confirmed by server successfully")
	} else {
		if err := msg.Ack(); err != nil {
			log.Error().
				Err(err).
				Str("client", clientName).
				Str("consumer", consumerName).
				Msg("Failed to ACK message")
			return
		}
		log.Info().
			Str("client", clientName).
			Str("consumer", consumerName).
			Msg("ACK sent successfully")
	}
}

// GracefulShutdown performs a unified graceful shutdown across JetStream consumers, Core NATS subscriptions,
// waits for active goroutines, and drains the connection.
func GracefulShutdown(clientName string, nc *nats.Conn, wg *sync.WaitGroup, timeout time.Duration, subs []*nats.Subscription, ccs ...jetstream.ConsumeContext) {
	// 1. Stop JetStream consume context fetch loops
	for _, cc := range ccs {
		if cc != nil {
			cc.Stop()
		}
	}
	if len(ccs) > 0 {
		log.Info().Str("client", clientName).Msg("Stopped JetStream consumer message fetch loop(s)")
	}

	// 2. Drain Core NATS subscriptions
	for _, sub := range subs {
		if sub != nil {
			if err := sub.Drain(); err != nil {
				log.Error().Err(err).Str("client", clientName).Msg("Error during subscription drain")
			} else {
				log.Info().Str("client", clientName).Msg("Subscription drained successfully")
			}
		}
	}

	// 3. Wait for in-flight tasks with a bounded timeout
	if wg != nil {
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			log.Info().Str("client", clientName).Msg("All in-flight messages completed cleanly")
		case <-shutdownCtx.Done():
			log.Warn().Str("client", clientName).Msg("Shutdown timeout reached before all in-flight messages finished")
		}
	}

	// 4. Drain and close NATS connection
	if nc != nil {
		if err := nc.Drain(); err != nil {
			log.Error().Err(err).Str("client", clientName).Msg("Error during NATS connection drain")
		} else {
			log.Info().Str("client", clientName).Msg("NATS connection successfully drained and closed")
		}
	}
}

// validateRange verifies that replay and reprocess sequence/time ranges are logically valid.
func validateRange(rng RangeParams) error {
	switch rng.Type {
	case RangeBySequence:
		if rng.StartSequence == 0 || rng.EndSequence < rng.StartSequence {
			return fmt.Errorf("invalid sequence range: start=%d, end=%d", rng.StartSequence, rng.EndSequence)
		}
	case RangeByTime:
		if rng.StartTime.IsZero() {
			return fmt.Errorf("invalid start time: startTime cannot be zero")
		}
		if rng.EndTime != nil && !rng.EndTime.IsZero() && rng.EndTime.Before(rng.StartTime) {
			return fmt.Errorf("invalid time range: endTime %v cannot be before startTime %v", rng.EndTime, rng.StartTime)
		}
	default:
		return fmt.Errorf("unsupported range type: %q", rng.Type)
	}
	return nil
}
