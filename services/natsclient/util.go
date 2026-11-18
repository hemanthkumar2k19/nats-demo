package natsclient

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// GetRTT returns the round-trip time between the client and the connected NATS server.
// For callers with an initialized *NATSClient, prefer calling client.GetRTT() directly.
func GetRTT(nc *nats.Conn) (time.Duration, error) {
	if nc == nil {
		return 0, fmt.Errorf("nats connection is nil")
	}
	return nc.RTT()
}

// PrintConnectionDetails logs diagnostic connection, topology, and statistics metadata using Zerolog.
func PrintConnectionDetails(nc *nats.Conn) {
	if nc == nil {
		log.Warn().Msg("NATS connection is nil")
		return
	}

	lastErr := "none"
	if err := nc.LastError(); err != nil {
		lastErr = err.Error()
	}

	jsEnabled, _ := nc.ConnectedServerJetStream()
	stats := nc.Stats()

	log.Info().
		Str("status", nc.Status().String()).
		Bool("connected", nc.IsConnected()).
		Bool("reconnecting", nc.IsReconnecting()).
		Bool("draining", nc.IsDraining()).
		Bool("closed", nc.IsClosed()).
		Str("last_error", lastErr).
		Str("connected_url", nc.ConnectedUrl()).
		Str("connected_addr", nc.ConnectedAddr()).
		Str("server_id", nc.ConnectedServerId()).
		Str("server_name", nc.ConnectedServerName()).
		Str("server_version", nc.ConnectedServerVersion()).
		Str("cluster_name", nc.ConnectedClusterName()).
		Bool("jetstream_enabled", jsEnabled).
		Int64("max_payload_bytes", nc.MaxPayload()).
		Strs("discovered_servers", nc.DiscoveredServers()).
		Strs("servers_pool", nc.Servers()).
		Uint64("in_msgs", stats.InMsgs).
		Uint64("out_msgs", stats.OutMsgs).
		Uint64("in_bytes", stats.InBytes).
		Uint64("out_bytes", stats.OutBytes).
		Uint64("reconnect_count", stats.Reconnects).
		Msg("NATS Connection Status & Topology Metadata")
}
