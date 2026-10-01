package service

import (
	"context"
	"fmt"
	"services/messaging"
	"services/model"
	"services/subscription"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Service manages publishing and subscription operations over NATS.
type Service struct {
	nc         *nats.Conn
	js         jetstream.JetStream
	subManager *subscription.Manager
}

// NewService creates a new Service instance.
func NewService(nc *nats.Conn, subManager *subscription.Manager) *Service {
	js, err := jetstream.New(nc)
	if err != nil {
		log.Warn().Err(err).Msg("JetStream context creation warning")
	}
	return &Service{
		nc:         nc,
		js:         js,
		subManager: subManager,
	}
}

// JSPublish publishes a message to JetStream synchronously.
func (s *Service) JSPublish(ctx context.Context, msg *model.Message) (*jetstream.PubAck, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	info, err := s.js.AccountInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	log.Info().Any("Account Info: ", info).Msg("")

	return messaging.JSPublishMsg(ctx, s.js, msg)
}

// JSPublishAsync publishes a message to JetStream asynchronously.
func (s *Service) JSPublishAsync(ctx context.Context, msg *model.Message) (*jetstream.PubAck, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}
	return messaging.JSPublishAsync(ctx, s.js, msg)
}

// JSPublishAsyncPending returns the pending async messages count.
func (s *Service) JSPublishAsyncPending() map[string]any {
	if s.js == nil {
		return map[string]any{"pending": 0}
	}
	return messaging.JSPublishAsyncPending(s.js)
}

// Publish delegates message publishing to the messaging layer.
func (s *Service) Publish(msg *model.Message) error {
	if err := messaging.PublishMsg(s.nc, msg); err != nil {
		return fmt.Errorf("failed to publish message to subject %s: %w", msg.Subject, err)
	}
	return nil
}

// PublishWithReply delegates message publishing with a reply subject to the messaging layer.
func (s *Service) PublishWithReply(msg *model.Message) error {

	sub, err := messaging.Subscribe(s.nc, msg.Reply, messaging.ReplyHandler)
	if err != nil {
		return fmt.Errorf("failed to subscribe to reply subject %s: %w", msg.Reply, err)
	}
	log.Info().Str("reply_subject", msg.Reply).Msg("Subscribed to reply subject")

	defer sub.Unsubscribe()

	if err := messaging.PublishMsgWithReply(s.nc, msg); err != nil {
		return fmt.Errorf("failed to publish message with reply to subject %s: %w", msg.Subject, err)
	}
	return nil
}

// PublishRequest delegates request message publishing to the messaging layer.
func (s *Service) PublishRequest(msg *model.Message) (*nats.Msg, error) {
	resp, err := messaging.PublishRequest(s.nc, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to publish request to subject %s: %w", msg.Subject, err)
	}
	return resp, nil
}

// CreateSubscription creates a NATS subscription and registers it in the subscription manager.
func (s *Service) CreateSubscription(subject string, handlerType string) (*subscription.Item, error) {

	var sub *nats.Subscription
	var err error

	switch handlerType {
	case "order":
		sub, err = messaging.Subscribe(s.nc, subject, messaging.OrderHandler)
	case "reply":
		sub, err = messaging.Subscribe(s.nc, subject, messaging.ReplyHandler)
	default:
		return nil, fmt.Errorf("invalid handler type: %s", handlerType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create NATS subscription for subject %s: %w", subject, err)
	}

	item, err := s.subManager.Subscribe(sub)
	if err != nil {
		return nil, fmt.Errorf("failed to register subscription for subject %s: %w", subject, err)
	}

	return item, nil
}

// ListSubscriptions lists all active subscriptions from the manager.
func (s *Service) ListSubscriptions() []*subscription.Item {
	if s.subManager == nil {
		return nil
	}
	return s.subManager.List()
}

// Unsubscribe unsubscribes a subscription by ID.
func (s *Service) Unsubscribe(id string) error {
	if s.subManager == nil {
		return fmt.Errorf("subscription manager is not initialized")
	}
	return s.subManager.Unsubscribe(id)
}

// DrainSubscription gracefully drains a subscription by ID.
func (s *Service) DrainSubscription(id string) error {
	if s.subManager == nil {
		return fmt.Errorf("subscription manager is not initialized")
	}
	return s.subManager.Drain(id)
}

// DrainAllSubscriptions drains all active subscriptions.
func (s *Service) DrainAllSubscriptions() {
	if s.subManager != nil {
		s.subManager.DrainAll()
	}
}
