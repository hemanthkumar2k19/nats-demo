package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FormattedTime wraps time.Time to serialize and deserialize timestamps in the human-friendly
// format "YYYY-MM-DD HH:MM:SS" (in local machine time), matching the NATS CLI display standard.
// It also accepts date-only ("YYYY-MM-DD") and standard RFC3339/ISO-8601 strings during input.
type FormattedTime time.Time

// Time returns the underlying time.Time value.
func (t FormattedTime) Time() time.Time {
	return time.Time(t)
}

// IsZero reports whether t represents the zero time instant.
func (t FormattedTime) IsZero() bool {
	return time.Time(t).IsZero()
}

// Before reports whether the time instant t is before u.
func (t FormattedTime) Before(u FormattedTime) bool {
	return time.Time(t).Before(time.Time(u))
}

// After reports whether the time instant t is after u.
func (t FormattedTime) After(u FormattedTime) bool {
	return time.Time(t).After(time.Time(u))
}

// String returns the timestamp formatted as "YYYY-MM-DD HH:MM:SS" in the local timezone,
// matching the NATS CLI display format and implementing the fmt.Stringer interface.
func (t FormattedTime) String() string {
	tt := time.Time(t)
	if tt.IsZero() {
		return ""
	}
	return tt.Local().Format("2006-01-02 15:04:05")
}

// MarshalJSON formats the timestamp as "YYYY-MM-DD HH:MM:SS" in the local timezone,
// matching the NATS CLI display format.
func (t FormattedTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(tt.Local().Format("2006-01-02 15:04:05"))
}

// UnmarshalJSON parses timestamps from "YYYY-MM-DD HH:MM:SS" (local time),
// "YYYY-MM-DD" (local time midnight), "YYYY-MM-DD HH:MM", RFC3339Nano, or RFC3339.
func (t *FormattedTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(strings.TrimSpace(string(b)), "\"")
	if s == "" || s == "null" {
		return nil
	}

	// 1. NATS CLI local format: "2006-01-02 15:04:05"
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local); err == nil {
		*t = FormattedTime(parsed)
		return nil
	}

	// 2. Date-only local format: "2006-01-02"
	if parsed, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		*t = FormattedTime(parsed)
		return nil
	}

	// 3. Minute-level local format: "2006-01-02 15:04"
	if parsed, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local); err == nil {
		*t = FormattedTime(parsed)
		return nil
	}

	// 4. RFC3339Nano (standard UTC / offset with fractional seconds)
	if parsed, err := time.Parse(time.RFC3339Nano, s); err == nil {
		*t = FormattedTime(parsed)
		return nil
	}

	// 5. RFC3339
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		*t = FormattedTime(parsed)
		return nil
	}

	return fmt.Errorf("invalid time format %q: expected 'YYYY-MM-DD HH:MM:SS', 'YYYY-MM-DD', or RFC3339", s)
}

// PublishRequest defines the JSON payload for triggering a NATS publish.
type PublishRequest struct {
	Subject string `json:"subject"`
	Data    string `json:"data"`
}

// PublishResponse defines the JSON response payload returning publish status and enriched metadata.
type PublishResponse struct {
	Status    string         `json:"status"`
	Subject   string         `json:"subject"`
	MsgID     string         `json:"msgId,omitempty"`
	Stream    string         `json:"stream,omitempty"`
	Sequence  uint64         `json:"sequence,omitempty"`
	Duplicate bool           `json:"duplicate,omitempty"`
	Headers   map[string]any `json:"headers,omitempty"`
	Data      any            `json:"data,omitempty"`
}

type Message struct {
	Subject string         `json:"subject"`
	Reply   string         `json:"reply,omitempty"`
	Data    any            `json:"data"`
	Headers map[string]any `json:"headers"`
	Timeout time.Duration  `json:"timeout,omitempty"`
}

// SubscribeRequest defines the JSON payload to create a new subscription.
type SubscribeRequest struct {
	Subject     string `json:"subject"`
	HandlerType string `json:"handlerType"`
}

// SubscriptionActionRequest defines the JSON payload for unsubscribe/drain operations.
type SubscriptionActionRequest struct {
	ID string `json:"id"`
}

type CreateConsumerRequest struct {
	StreamName     string   `json:"streamName"`
	ConsumerName   string   `json:"consumerName,omitempty"`
	DurableName    string   `json:"durableName,omitempty"`
	FilterSubjects []string `json:"filterSubject"`
	AckPolicy      string   `json:"ackPolicy,omitempty"`
	DeliverPolicy  string   `json:"deliverPolicy,omitempty"`

	// Push Consumer
	DeliverSubject string `json:"DeliverSubject,omitempty"`
}

type CreateStreamRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Subjects    []string `json:"subjects,omitempty"`
	Retention   string   `json:"retention,omitempty"`
	Discard     string   `json:"discard,omitempty"`
	Replicas    int      `json:"replicas,omitempty"`
	Storage     string   `json:"storage,omitempty"`
}

type DeleteStreamRequest struct {
	Name string `json:"name"`
}

// StreamSequencesResponse provides stream sequence and timestamp boundaries to guide replay requests.
type StreamSequencesResponse struct {
	StreamName     string        `json:"streamName"`
	FirstSequence  uint64        `json:"firstSequence"`
	LastSequence   uint64        `json:"lastSequence"`
	FirstTimestamp FormattedTime `json:"firstTimestamp"`
	LastTimestamp  FormattedTime `json:"lastTimestamp"`
	MessagesCount  uint64        `json:"messagesCount"`
	Bytes          uint64        `json:"bytes"`
	DeletedCount   int           `json:"deletedCount"`
}

// ReplayRequest defines the JSON payload for replaying messages from a stream within a sequence range or time window.
type ReplayRequest struct {
	StreamName     string         `json:"streamName"`
	FilterSubjects []string       `json:"filterSubjects,omitempty"`
	StartSequence  uint64         `json:"startSequence,omitempty"`
	EndSequence    uint64         `json:"endSequence,omitempty"`
	StartTime      *FormattedTime `json:"startTime,omitempty"`
	EndTime        *FormattedTime `json:"endTime,omitempty"`
	BatchSize      int            `json:"batchSize,omitempty"`
}

// ReplayedMessage captures the stream metadata and payload of an individual replayed message.
type ReplayedMessage struct {
	Sequence  uint64            `json:"sequence"`
	Subject   string            `json:"subject"`
	Data      string            `json:"data"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timestamp FormattedTime     `json:"timestamp"`
}

// ReplayResponse defines the JSON response payload returning replay execution status and summary metrics.
type ReplayResponse struct {
	Status             string         `json:"status"`
	StreamName         string         `json:"streamName"`
	Mode               string         `json:"mode"`
	RequestedStartSeq  uint64         `json:"requestedStartSequence,omitempty"`
	RequestedEndSeq    uint64         `json:"requestedEndSequence,omitempty"`
	RequestedStartTime *FormattedTime `json:"requestedStartTime,omitempty"`
	RequestedEndTime   *FormattedTime `json:"requestedEndTime,omitempty"`
	FirstSequence      uint64         `json:"firstSequence"`
	LastSequence       uint64         `json:"lastSequence"`
	FirstTimestamp     *FormattedTime `json:"firstTimestamp,omitempty"`
	LastTimestamp      *FormattedTime `json:"lastTimestamp,omitempty"`
	FilterSubjects     []string       `json:"filterSubjects,omitempty"`
	ReplayedCount      int            `json:"replayedCount"`
	MessagesCount      int            `json:"messagesCount"`

	// Backward-compatible field aliases
	StartSequence uint64         `json:"startSequence,omitempty"`
	EndSequence   uint64         `json:"endSequence,omitempty"`
	StartTime     *FormattedTime `json:"startTime,omitempty"`
	EndTime       *FormattedTime `json:"endTime,omitempty"`
}

// ReprocessRequest defines the JSON payload for reprocessing messages from a stream with business effects.
type ReprocessRequest struct {
	StreamName     string         `json:"streamName"`
	FilterSubjects []string       `json:"filterSubjects,omitempty"`
	StartSequence  uint64         `json:"startSequence,omitempty"`
	EndSequence    uint64         `json:"endSequence,omitempty"`
	StartTime      *FormattedTime `json:"startTime,omitempty"`
	EndTime        *FormattedTime `json:"endTime,omitempty"`
	BatchSize      int            `json:"batchSize,omitempty"`
}

// ReprocessResponse defines the JSON response payload returning reprocessing execution status and summary metrics.
type ReprocessResponse struct {
	Status             string         `json:"status"`
	StreamName         string         `json:"streamName"`
	Mode               string         `json:"mode"`
	RequestedStartSeq  uint64         `json:"requestedStartSequence,omitempty"`
	RequestedEndSeq    uint64         `json:"requestedEndSequence,omitempty"`
	RequestedStartTime *FormattedTime `json:"requestedStartTime,omitempty"`
	RequestedEndTime   *FormattedTime `json:"requestedEndTime,omitempty"`
	FirstSequence      uint64         `json:"firstSequence"`
	LastSequence       uint64         `json:"lastSequence"`
	FirstTimestamp     *FormattedTime `json:"firstTimestamp,omitempty"`
	LastTimestamp      *FormattedTime `json:"lastTimestamp,omitempty"`
	FilterSubjects     []string       `json:"filterSubjects,omitempty"`
	ReprocessedCount   int            `json:"reprocessedCount"`
	DurationMs         int64          `json:"durationMs"`
	Message            string         `json:"message"`

	// Backward-compatible field aliases
	StartSequence uint64         `json:"startSequence,omitempty"`
	EndSequence   uint64         `json:"endSequence,omitempty"`
	StartTime     *FormattedTime `json:"startTime,omitempty"`
	EndTime       *FormattedTime `json:"endTime,omitempty"`
}

// MaxDeliverAdvisory represents the NATS JetStream advisory event emitted to
// $JS.EVENT.ADVISORY.CONSUMER.MAX_DELIVERIES.<stream>.<consumer> when delivery attempts reach MaxDeliver.
type MaxDeliverAdvisory struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	Stream     string    `json:"stream"`
	Consumer   string    `json:"consumer"`
	StreamSeq  uint64    `json:"stream_seq"`
	Deliveries uint64    `json:"deliveries"`
	Domain     string    `json:"domain,omitempty"`
}
