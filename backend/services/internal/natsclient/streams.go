package natsclient

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

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
