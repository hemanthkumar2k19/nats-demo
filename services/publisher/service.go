package publisher

import (
	"context"
	"fmt"
	"services/messaging"

	"github.com/nats-io/nats.go"
)

// Service manages message publishing over Core NATS with trace propagation.
type Service struct {
	nc *nats.Conn
}

// NewService creates a new publisher Service instance.
func NewService(nc *nats.Conn) *Service {
	return &Service{nc: nc}
}

// PublishEvent delegates message publishing to the messaging layer.
func (s *Service) PublishEvent(ctx context.Context, subject string, payload []byte) error {
	msg := &nats.Msg{
		Subject: subject,
		Data:    payload,
		Header:  make(nats.Header),
	}

	if err := messaging.PublishMsg(ctx, s.nc, msg); err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", subject, err)
	}

	return nil
}
