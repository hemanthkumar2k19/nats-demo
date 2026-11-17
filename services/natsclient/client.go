package natsclient

import (
	"fmt"
	"strings"

	"services/config"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

type NATSClient struct {
	nc *nats.Conn
	js jetstream.JetStream
}

func NewNATSClient(cfg *config.NATSClientConfig, clientNameOverride string) (*NATSClient, error)
{	nc, err := ConnectAndDiscover(cfg, clientNameOverride)
	if err !=nil{
		return nil, fmt.Errorf("failed to create NATS Client")
	}
	return &NATSClient{}
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

	log.Info().
		Str("url", nc.ConnectedUrl()).
		Strs("servers", cfg.Servers).
		Msg("Connected to NATS server")

	log.Info().
		Strs("discovered", nc.DiscoveredServers()).
		Msg("Discovered Servers")

	return nc, nil
}
