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

// CreateConsumer creates or retrieves a JetStream consumer and returns its metadata.
func (s *Service) CreateConsumer(ctx context.Context, cr *model.CreateConsumerRequest) (*jetstream.ConsumerInfo, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	consumer, err := messaging.CreateConsumer(ctx, s.js, cr)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve consumer info: %w", err)
	}

	return info, nil
}

// DeleteConsumer deletes a JetStream consumer.
func (s *Service) DeleteConsumer(ctx context.Context, cr *model.CreateConsumerRequest) error {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	if err := messaging.DeleteConsumer(ctx, s.js, cr); err != nil {
		return fmt.Errorf("failed to delete consumer: %w", err)
	}

	return nil
}
