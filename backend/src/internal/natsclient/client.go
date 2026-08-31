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
