package service

import (
	"context"
	"fmt"
	"services/messaging"
	"services/model"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// Service manages publishing and subscription operations over NATS.
type Service struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewService creates a new Service instance.
func NewService(nc *nats.Conn) *Service {
	js, err := jetstream.New(nc)
	if err != nil {
		log.Warn().Err(err).Msg("JetStream context creation warning")
	}
	return &Service{
		nc: nc,
		js: js,
	}
}

// Publish publishes a message to Core NATS.
func (s *Service) Publish(ctx context.Context, msg *model.Message) error {
	return messaging.PublishMsg(s.nc, msg)
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

// JSPublishBatch publishes multiple messages to JetStream asynchronously using JSPublishN.
func (s *Service) JSPublishBatch(ctx context.Context, messages []model.Message) error {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	return messaging.JSPublishN(ctx, s.js, messages)
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

// CreateStream creates or retrieves a JetStream stream and returns its metadata.
func (s *Service) CreateStream(ctx context.Context, sr *model.CreateStreamRequest) (*jetstream.StreamInfo, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	stream, err := messaging.CreateStream(ctx, s.js, *sr)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream: %w", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve stream info: %w", err)
	}

	return info, nil
}

// DeleteStream deletes a JetStream stream by name.
func (s *Service) DeleteStream(ctx context.Context, streamName string) error {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	if err := messaging.DeleteStream(ctx, s.js, streamName); err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}

	return nil
}
