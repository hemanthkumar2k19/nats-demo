package natsclient

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func ConnectAndDiscover() (*nats.Conn, error) {
	server := "nats://localhost:4222"

	nc, err := nats.Connect(server,

		nats.DisconnectErrHandler(
			func(nc *nats.Conn, err error) {
				log.Printf("Disconnected: %v. Swapping to next cluster node...", err)
			}),

		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("Reconnected to server: %s", c.ConnectedUrl())
			log.Printf("Updated servers list: %v", c.DiscoveredServers())
		}),

		nats.ClosedHandler(
			func(c *nats.Conn) {
				log.Printf("Connection closed: %s", c.ConnectedUrl())
			}),
	)

	if err != nil {
		return nil, err
	}

	log.Printf("Connected to server: %s", nc.ConnectedUrl())
	log.Printf("Discovered servers: %v", nc.DiscoveredServers())

	return nc, nil
}

func ConnectWithOptions() (*nats.Conn, error) {
	servers := "nats://localhost:4222,nats://localhost:4223,nats://localhost:4224"

	nc, err := nats.Connect(servers,
		// Client Identification
		nats.Name("demo-service"),

		// Reconnection & Retry Semantics
		nats.MaxReconnects(10), // Set to -1 for infinite reconnect attempts
		nats.ReconnectWait(2*time.Second),
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			// Exponential backoff up to 10 seconds to prevent thundering herd
			delay := time.Duration(attempts) * time.Second
			if delay > 10*time.Second {
				return 10 * time.Second
			}
			return delay
		}),
		nats.ReconnectBufSize(8*1024*1024), // 8MB buffer for outbound messages while disconnected

		// Heartbeat & Connection Liveness Monitoring
		nats.PingInterval(15*time.Second), // Send PING every 15s to verify server health
		nats.MaxPingsOutstanding(3),       // Miss 3 PONG responses before declaring connection dead

		// Event Handlers for Lifecycle and Failover Observability
		nats.ConnectHandler(func(c *nats.Conn) {
			log.Printf("[NATS] Connected to server: %s", c.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			log.Printf("[NATS] Disconnected: %v. Attempting reconnect...", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("[NATS] Reconnected to server: %s", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			log.Printf("[NATS] Connection closed permanently: %s", c.ConnectedUrl())
		}),
		nats.DiscoveredServersHandler(func(c *nats.Conn) {
			log.Printf("[NATS] Cluster servers updated: %v", c.DiscoveredServers())
		}),
		nats.ErrorHandler(func(c *nats.Conn, sub *nats.Subscription, err error) {
			if sub != nil {
				log.Printf("[NATS] Async error on subject %s: %v", sub.Subject, err)
			} else {
				log.Printf("[NATS] Async error: %v", err)
			}
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS cluster: %w", err)
	}

	return nc, nil
}

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
