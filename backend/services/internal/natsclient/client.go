package natsclient

import (
	"context"
	"fmt"
	"log"
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
func Connect(url string, opts ...nats.Option) (*Client, error) {
	nc, err := nats.Connect(url, opts...)
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

// ConnectWithAuth initializes a NATS connection with user credentials and JetStream.
func ConnectWithAuth(url, user, password string, opts ...nats.Option) (*Client, error) {
	if user != "" {
		opts = append(opts, nats.UserInfo(user, password))
	}
	return Connect(url, opts...)
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

	return nil
}

// ResetJobsStream deletes the JOBS stream (if it exists) and recreates it fresh with sequence starting at 1.
func (c *Client) ResetJobsStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Delete existing JOBS stream (which also removes attached consumers and resets sequence counter)
	if err := c.JS.DeleteStream(ctx, "JOBS"); err != nil {
		log.Printf("[NATS] Notice during stream deletion (may not exist): %v", err)
	} else {
		log.Println("[NATS] Successfully deleted existing JOBS stream")
	}

	// Recreate fresh JOBS stream and durable consumer
	return c.EnsureJobsStream()
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

// DeleteDLQStream deletes the JOBS_DLQ stream and its consumers from NATS.
func (c *Client) DeleteDLQStream() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.JS.DeleteStream(ctx, "JOBS_DLQ")
}

// IsDLQStreamActive checks whether the JOBS_DLQ stream currently exists in NATS.
func (c *Client) IsDLQStreamActive() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.JS.Stream(ctx, "JOBS_DLQ")
	return err == nil
}

