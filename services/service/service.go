package service

import (
	"context"
	"fmt"
	"time"

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

// GetStreamSequenceBounds retrieves available boundary sequences and timestamps for a stream to guide replay requests.
func (s *Service) GetStreamSequenceBounds(ctx context.Context, streamName string) (*model.StreamSequencesResponse, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	return messaging.GetStreamSequenceBounds(ctx, s.js, streamName)
}

// ReplayMessages replays stream messages by sequence range or time range for historical audit without applying business effects.
func (s *Service) ReplayMessages(ctx context.Context, req *model.ReplayRequest) (*model.ReplayResponse, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	rangeParams := messaging.RangeParams{
		Type:          messaging.RangeBySequence,
		StartSequence: req.StartSequence,
		EndSequence:   req.EndSequence,
	}
	mode := "sequence"

	if req.StartTime != nil && !req.StartTime.IsZero() {
		rangeParams.Type = messaging.RangeByTime
		rangeParams.StartTime = req.StartTime.Time()
		if req.EndTime != nil && !req.EndTime.IsZero() {
			t := req.EndTime.Time()
			rangeParams.EndTime = &t
		}
		mode = "time"
	}

	summary, err := messaging.ReplayMessages(
		ctx,
		s.js,
		req.StreamName,
		req.FilterSubjects,
		rangeParams,
		req.BatchSize,
	)
	if err != nil {
		return nil, err
	}

	return &model.ReplayResponse{
		Status:             "completed",
		StreamName:         req.StreamName,
		Mode:               mode,
		RequestedStartSeq:  req.StartSequence,
		RequestedEndSeq:    req.EndSequence,
		RequestedStartTime: req.StartTime,
		RequestedEndTime:   req.EndTime,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		StartSequence:      req.StartSequence,
		EndSequence:        req.EndSequence,
		FirstSequence:      summary.FirstSequence,
		LastSequence:       summary.LastSequence,
		FirstTimestamp:     summary.FirstTimestamp,
		LastTimestamp:      summary.LastTimestamp,
		FilterSubjects:     req.FilterSubjects,
		ReplayedCount:      summary.Count,
		MessagesCount:      summary.Count,
	}, nil
}

// ReprocessMessages fetches messages within a sequence or time range and executes domain business processing logic for each message.
func (s *Service) ReprocessMessages(ctx context.Context, req *model.ReprocessRequest) (*model.ReprocessResponse, error) {
	if s.js == nil {
		var err error
		s.js, err = jetstream.New(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	rangeParams := messaging.RangeParams{
		Type:          messaging.RangeBySequence,
		StartSequence: req.StartSequence,
		EndSequence:   req.EndSequence,
	}
	mode := "sequence"

	if req.StartTime != nil && !req.StartTime.IsZero() {
		rangeParams.Type = messaging.RangeByTime
		rangeParams.StartTime = req.StartTime.Time()
		if req.EndTime != nil && !req.EndTime.IsZero() {
			t := req.EndTime.Time()
			rangeParams.EndTime = &t
		}
		mode = "time"
	}

	startTime := time.Now()
	summary, err := messaging.ReprocessMessages(
		ctx,
		s.js,
		req.StreamName,
		req.FilterSubjects,
		rangeParams,
		req.BatchSize,
	)
	if err != nil {
		return nil, err
	}
	duration := time.Since(startTime).Milliseconds()

	return &model.ReprocessResponse{
		Status:             "completed",
		StreamName:         req.StreamName,
		Mode:               mode,
		RequestedStartSeq:  req.StartSequence,
		RequestedEndSeq:    req.EndSequence,
		RequestedStartTime: req.StartTime,
		RequestedEndTime:   req.EndTime,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		StartSequence:      req.StartSequence,
		EndSequence:        req.EndSequence,
		FirstSequence:      summary.FirstSequence,
		LastSequence:       summary.LastSequence,
		FirstTimestamp:     summary.FirstTimestamp,
		LastTimestamp:      summary.LastTimestamp,
		FilterSubjects:     req.FilterSubjects,
		ReprocessedCount:   summary.Count,
		DurationMs:         duration,
		Message:            "Reprocessing completed with business effects applied",
	}, nil
}
