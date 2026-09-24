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
