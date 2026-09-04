package natsclient

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Client wraps the NATS connection and JetStream instance for centralized access.
type Client struct {
	Conn *nats.Conn
	JS   jetstream.JetStream
}

// Connect initializes a NATS connection and JetStream instance, wrapping them in Client.
func Connect(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	return &Client{
		Conn: nc,
		JS:   js,
	}, nil
}

// Close closes the underlying NATS connection.
func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

// EnsureJobsStream guarantees that the JOBS stream (subjects: jobs.submitted) exists in NATS.
func (c *Client) EnsureJobsStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Idempotently create or update JOBS stream with 2-minute deduplication window
	stream, err := c.JS.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       "JOBS",
		Subjects:   []string{"jobs.submitted"},
		Duplicates: 2 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("failed to add or update JOBS stream: %w", err)
	}

	// Explicitly create durable consumer "job-processor" on JOBS stream with AckWait: 5s
	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "job-processor",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       5 * time.Second,
		FilterSubject: "jobs.submitted",
	})
	if err != nil {
		return fmt.Errorf("failed to create durable consumer job-processor: %w", err)
	}

	// Also ensure processor-durable for backward compatibility
	_, _ = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "processor-durable",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       5 * time.Second,
		FilterSubject: "jobs.submitted",
	})

	// Guarantee JOBS_DLQ stream and dlq-inspector consumer also exist
	_ = c.EnsureDLQStream()

	return nil
}

// EnsureDLQStream guarantees that the JOBS_DLQ stream and dlq-inspector consumer exist in NATS.
func (c *Client) EnsureDLQStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := c.JS.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     "JOBS_DLQ",
		Subjects: []string{"jobs.dlq"},
	})
	if err != nil {
		return fmt.Errorf("failed to add or update JOBS_DLQ stream: %w", err)
	}

	// Explicitly create durable consumer "dlq-inspector" on JOBS_DLQ stream
	_, err = stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "dlq-inspector",
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		FilterSubject: "jobs.dlq",
	})
	if err != nil {
		return fmt.Errorf("failed to create durable consumer dlq-inspector: %w", err)
	}

	return nil
}
