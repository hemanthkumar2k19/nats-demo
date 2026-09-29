package consumer

import (
	"context"
	"log"
	"services/messaging"

	"github.com/nats-io/nats.go"
)

// Service manages Core NATS subscriptions with inbound trace context extraction.
type Service struct {
	nc *nats.Conn
}

// NewService creates a new consumer Service instance.
func NewService(nc *nats.Conn) *Service {
	return &Service{nc: nc}
}

// StartSubscriber subscribes to a subject pattern using messaging package functions.
func (s *Service) StartSubscriber(subject string) (*nats.Subscription, error) {
	return messaging.Subscribe(context.Background(), s.nc, subject, func(msg *nats.Msg) {
		log.Printf("[CONSUMER] Received message on subject [%s] | Data: %s", msg.Subject, string(msg.Data))
	})
}
