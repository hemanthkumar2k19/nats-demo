package natsclient

import (
	"fmt"
	"strings"
	"time"

	"services/config"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// NATSClient encapsulates a resilient Core NATS connection and an initialized JetStream context.
type NATSClient struct {
	nc *nats.Conn
	js jetstream.JetStream
}

// NewNATSClient connects to NATS and initializes both the Core NATS connection and JetStream context atomically.
func NewNATSClient(cfg *config.NATSClientConfig, clientNameOverride ...string) (*NATSClient, error) {
	nc, err := ConnectAndDiscover(cfg, clientNameOverride...)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS client: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize JetStream context: %w", err)
	}

	return &NATSClient{
		nc: nc,
		js: js,
	}, nil
}

// Conn returns the underlying Core NATS connection.
func (c *NATSClient) Conn() *nats.Conn {
	return c.nc
}

// JetStream returns the initialized JetStream context.
func (c *NATSClient) JetStream() jetstream.JetStream {
	return c.js
}

// Close closes the underlying NATS connection.
func (c *NATSClient) Close() {
	if c.nc != nil {
		c.nc.Close()
	}
}

// Drain drains and closes the underlying NATS connection gracefully.
func (c *NATSClient) Drain() error {
	if c.nc != nil {
		return c.nc.Drain()
	}
	return nil
}

// GetRTT returns the round-trip time between the client and the connected NATS server.
func (c *NATSClient) GetRTT() (time.Duration, error) {
	if c.nc == nil {
		return 0, fmt.Errorf("nats connection is nil")
	}
	return c.nc.RTT()
}

// ConnectAndDiscover initializes a resilient connection to NATS using the centralized NATSClientConfig.
// It accepts optional clientNameOverride strings to allow individual workers to identify themselves.
func ConnectAndDiscover(cfg *config.NATSClientConfig, clientNameOverride ...string) (*nats.Conn, error) {
	opts, err := cfg.BuildOptions(clientNameOverride...)
	if err != nil {
		return nil, fmt.Errorf("failed to build NATS options: %w", err)
	}

	serverList := strings.Join(cfg.Servers, ",")

	nc, err := nats.Connect(serverList, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS (%s): %w", serverList, err)
	}

	logEvent := log.Info().
		Str("url", nc.ConnectedUrl()).
		Strs("servers", cfg.Servers)

	if cfg.AuthMethod != "" {
		logEvent = logEvent.Str("auth_method", cfg.AuthMethod)
		if cfg.AuthUser != "" {
			logEvent = logEvent.Str("user", cfg.AuthUser)
		}
	}
	logEvent.Msg("Connected to NATS server")

	log.Info().
		Strs("discovered", nc.DiscoveredServers()).
		Msg("Discovered Servers")

	return nc, nil
}
