package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"services/messaging"
	"services/model"
	"services/natsclient"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

var shortIDCounter uint64 = uint64(time.Now().Unix()%9000) + 1000

func deriveDomainPrefix(subject string) string {
	parts := strings.Split(subject, ".")
	if len(parts) > 0 && len(parts[0]) > 0 {
		token := strings.ToUpper(parts[0])
		if strings.HasPrefix(token, "DEL") {
			return "DEL"
		}
		if strings.HasPrefix(token, "PAY") {
			return "PAY"
		}
		if strings.HasPrefix(token, "ORD") {
			return "ORD"
		}
		if len(token) >= 3 {
			return token[:3]
		}
		return token
	}
	return "MSG"
}

// GenerateShortID produces a short, human-readable unique identifier (e.g. "DEL-4821", "ORD-1092").
func GenerateShortID(subject string) string {
	prefix := deriveDomainPrefix(subject)
	num := (atomic.AddUint64(&shortIDCounter, 1) % 9000) + 1000
	return fmt.Sprintf("%s-%04d", prefix, num)
}

// PrepareMessage enriches the message headers and injects a runtime unique short ID into the payload.
func PrepareMessage(msg *model.Message) string {
	if msg.Headers == nil {
		msg.Headers = make(map[string]any)
	}

	shortID := GenerateShortID(msg.Subject)

	// Standardize source to order-service
	msg.Headers["Source"] = "order-service"

	// Correlation-ID
	if cID, ok := msg.Headers["Correlation-ID"]; !ok || cID == "" || cID == nil {
		msg.Headers["Correlation-ID"] = "corr-" + shortID
	}

	// Nats-Msg-Id for JetStream deduplication and message tracing
	if msgID, ok := msg.Headers["Nats-Msg-Id"]; !ok || msgID == "" || msgID == nil {
		msg.Headers["Nats-Msg-Id"] = "msg-" + shortID
	}

	// Published-At timestamp
	if pubAt, ok := msg.Headers["Published-At"]; !ok || pubAt == "" || pubAt == nil {
		msg.Headers["Published-At"] = time.Now().Format("2006-01-02 15:04:05")
	}

	// Enrich Data payload with runtime ID
	msg.Data = attachDataID(msg.Data, shortID, msg.Subject)

	return shortID
}

func attachDataID(data any, shortID string, subject string) any {
	if data == nil {
		return map[string]any{"id": shortID, "subject": subject}
	}
	switch v := data.(type) {
	case map[string]any:
		v["id"] = shortID
		return v
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return map[string]any{"id": shortID, "subject": subject}
		}
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			var parsed map[string]any
			if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
				parsed["id"] = shortID
				return parsed
			}
		}
		return v
	default:
		return v
	}
}

// Service manages publishing and subscription operations over NATS.
type Service struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewService creates a new Service instance.
func NewService(nc *nats.Conn) *Service {
	js, err := natsclient.NewJSClient(nc)
	if err != nil {
		log.Warn().Err(err).Msg("JetStream context creation warning")
	}
	return &Service{
		nc: nc,
		js: js,
	}
}

// Publish publishes a message to Core NATS after enriching headers and data with a runtime unique ID.
func (s *Service) Publish(ctx context.Context, msg *model.Message) (*model.PublishResponse, error) {
	PrepareMessage(msg)

	if err := messaging.PublishMsg(s.nc, msg); err != nil {
		return nil, err
	}

	msgID := ""
	if v, ok := msg.Headers["Nats-Msg-Id"]; ok && v != nil {
		msgID = fmt.Sprintf("%v", v)
	}

	return &model.PublishResponse{
		Status:  "published",
		Subject: msg.Subject,
		MsgID:   msgID,
		Headers: msg.Headers,
		Data:    msg.Data,
	}, nil
}

// JSPublish publishes a message to JetStream synchronously after enriching headers and data with a runtime unique ID.
func (s *Service) JSPublish(ctx context.Context, msg *model.Message) (*model.PublishResponse, error) {
	if s.js == nil {
		var err error
		s.js, err = natsclient.NewJSClient(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	PrepareMessage(msg)

	ack, err := messaging.JSPublishMsg(ctx, s.js, msg)
	if err != nil {
		return nil, err
	}

	msgID := ""
	if v, ok := msg.Headers["Nats-Msg-Id"]; ok && v != nil {
		msgID = fmt.Sprintf("%v", v)
	}

	return &model.PublishResponse{
		Status:    "published",
		Subject:   msg.Subject,
		MsgID:     msgID,
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Duplicate: ack.Duplicate,
		Headers:   msg.Headers,
		Data:      msg.Data,
	}, nil
}

// JSPublishBatch publishes multiple messages to JetStream asynchronously using JSPublishN.
func (s *Service) JSPublishBatch(ctx context.Context, messages []model.Message) error {
	if s.js == nil {
		var err error
		s.js, err = natsclient.NewJSClient(s.nc)
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
		s.js, err = natsclient.NewJSClient(s.nc)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
		}
	}

	cfg := jetstream.ConsumerConfig{
		Durable:        cr.ConsumerName,
		FilterSubjects: cr.FilterSubjects,
		AckPolicy:      messaging.ResolveAckPolicy(cr.AckPolicy),
		DeliverPolicy:  messaging.ResolveDeliverPolicy(cr.DeliverPolicy),
	}
	if cr.DeliverSubject != "" {
		cfg.DeliverSubject = cr.DeliverSubject
	}

	consumer, err := messaging.CreateConsumer(ctx, s.js, cr.StreamName, cfg)
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
		s.js, err = natsclient.NewJSClient(s.nc)
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
		s.js, err = natsclient.NewJSClient(s.nc)
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
		s.js, err = natsclient.NewJSClient(s.nc)
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
		s.js, err = natsclient.NewJSClient(s.nc)
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
