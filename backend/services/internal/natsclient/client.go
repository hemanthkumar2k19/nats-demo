package natsclient

import (
	"fmt"

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

