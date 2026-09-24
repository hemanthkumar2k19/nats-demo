package natsclient

import (
	"log"

	"github.com/nats-io/nats.go"
)

func PrintConnectionDetails(nc *nats.Conn) {
	if nc == nil {
		log.Println("[NATS] Connection is nil")
		return
	}

	log.Println("=== NATS Connection Health & Status ===")
	log.Printf("  Status:               %v", nc.Status())
	log.Printf("  Is Connected:         %t", nc.IsConnected())
	log.Printf("  Is Reconnecting:      %t", nc.IsReconnecting())
	log.Printf("  Is Draining:          %t", nc.IsDraining())
	log.Printf("  Is Closed:            %t", nc.IsClosed())
	if err := nc.LastError(); err != nil {
		log.Printf("  Last Error:           %v", err)
	} else {
		log.Printf("  Last Error:           None")
	}

	log.Println("=== NATS Server & Topology Metadata ===")
	log.Printf("  Connected URL:        %s", nc.ConnectedUrl())
	log.Printf("  Connected IP/Addr:    %s", nc.ConnectedAddr())
	log.Printf("  Server ID:            %s", nc.ConnectedServerId())
	log.Printf("  Server Name:          %s", nc.ConnectedServerName())
	log.Printf("  Server Version:       %s", nc.ConnectedServerVersion())
	log.Printf("  Cluster Name:         %s", nc.ConnectedClusterName())
	jsEnabled, _ := nc.ConnectedServerJetStream()
	log.Printf("  JetStream Enabled:    %t", jsEnabled)
	log.Printf("  Max Payload (Bytes):  %d", nc.MaxPayload())
	log.Printf("  Discovered Servers:   %v", nc.DiscoveredServers())
	log.Printf("  Known Servers Pool:   %v", nc.Servers())

	stats := nc.Stats()
	log.Println("=== NATS Connection Statistics ===")
	log.Printf("  In Messages:          %d", stats.InMsgs)
	log.Printf("  Out Messages:         %d", stats.OutMsgs)
	log.Printf("  In Bytes:             %d", stats.InBytes)
	log.Printf("  Out Bytes:            %d", stats.OutBytes)
	log.Printf("  Reconnect Count:      %d", stats.Reconnects)
}
