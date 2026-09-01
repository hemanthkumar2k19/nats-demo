package natsclient

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Client wraps the NATS connection for future custom extension and centralization.
type Client struct {
	Conn *nats.Conn
}

// Connect initializes a NATS connection and wraps it in our Client struct.
func Connect(url string) (*Client, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &Client{Conn: nc}, nil
}

// Close closes the underlying NATS connection.
func (c *Client) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}

// EnsureJobsStream guarantees that the JOBS stream (subjects: jobs.submitted) exists in NATS.
func (c *Client) EnsureJobsStream() error {
	js, err := c.Conn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	_, err = js.StreamInfo("JOBS")
	if err != nil {
		// If stream does not exist, create it.
		// Set a 2-minute deduplication window for JetStream message deduplication.
		_, err = js.AddStream(&nats.StreamConfig{
			Name:       "JOBS",
			Subjects:   []string{"jobs.submitted"},
			Duplicates: 2 * time.Minute,
		})
		if err != nil {
			return fmt.Errorf("failed to add JOBS stream: %w", err)
		}
	} else {
		// Update stream config to ensure deduplication window is set
		_, _ = js.UpdateStream(&nats.StreamConfig{
			Name:       "JOBS",
			Subjects:   []string{"jobs.submitted"},
			Duplicates: 2 * time.Minute,
		})
	}

	// Explicitly create durable consumer "job-processor" on the JOBS stream
	_, err = js.ConsumerInfo("JOBS", "job-processor")
	if err != nil {
		_, err = js.AddConsumer("JOBS", &nats.ConsumerConfig{
			Durable:       "job-processor",
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			FilterSubject: "jobs.submitted",
		})
		if err != nil {
			return fmt.Errorf("failed to create durable consumer job-processor: %w", err)
		}
	}

	// Also ensure processor-durable for backward compatibility
	_, err = js.ConsumerInfo("JOBS", "processor-durable")
	if err != nil {
		_, _ = js.AddConsumer("JOBS", &nats.ConsumerConfig{
			Durable:       "processor-durable",
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			FilterSubject: "jobs.submitted",
		})
	}

	return nil
}
