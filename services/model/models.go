package model

import "time"

// PublishRequest defines the JSON payload for triggering a NATS publish.
type PublishRequest struct {
	Subject string `json:"subject"`
	Data    string `json:"data"`
}

// PublishResponse defines the JSON response payload returning publish status.
type PublishResponse struct {
	Status  string `json:"status"`
	Subject string `json:"subject"`
}

type Message struct {
	Subject string         `json:"subject"`
	Reply   string         `json:"reply,omitempty"`
	Data    string         `json:"data"`
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
