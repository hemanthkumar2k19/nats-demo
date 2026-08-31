package natsclient

import (
	"fmt"

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

// EnsureJobsStream guarantees that the JOBS stream (subjects: jobs.>) exists in NATS.
func (c *Client) EnsureJobsStream() error {
	js, err := c.Conn.JetStream()
	if err != nil {
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	_, err = js.StreamInfo("JOBS")
	if err != nil {
		// If stream does not exist, create it
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     "JOBS",
			Subjects: []string{"jobs.>"},
		})
		if err != nil {
			return fmt.Errorf("failed to add JOBS stream: %w", err)
		}
	}

	// Explicitly create the durable consumer "processor-durable" on the JOBS stream
	// so that it persists across application restarts and doesn't get automatically
	// deleted on unsubscribe.
	_, err = js.ConsumerInfo("JOBS", "processor-durable")
	if err != nil {
		// Consumer doesn't exist, create it
		_, err = js.AddConsumer("JOBS", &nats.ConsumerConfig{
			Durable:       "processor-durable",
			DeliverPolicy: nats.DeliverAllPolicy,
			AckPolicy:     nats.AckExplicitPolicy,
			FilterSubject: "jobs.submitted",
		})
		if err != nil {
			return fmt.Errorf("failed to create durable consumer: %w", err)
		}
	}

	return nil
}
