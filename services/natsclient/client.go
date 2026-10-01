package natsclient

import (
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func ConnectAndDiscover(cn string) (*nats.Conn, error) {
	server := "nats://localhost:4222"

	nc, err := nats.Connect(server,

		// Client Identification
		nats.Name(cn),

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
		nats.PingInterval(5*time.Second), // Send PING every 5s to verify server health
		nats.MaxPingsOutstanding(3),      // Miss 3 PONG responses before declaring connection dead

		nats.ErrorHandler(
			func(c *nats.Conn, s *nats.Subscription, err error) {
				if s != nil {
					log.Error().Err(err).Str("subject", s.Subject).Msg("NATS async error on subject")
				} else {
					log.Error().Err(err).Msg("NATS async error")
				}
			}),

		nats.DisconnectErrHandler(
			func(nc *nats.Conn, err error) {
				log.Warn().Err(err).Msg("Disconnected from NATS. Swapping to next cluster node...")
			}),

		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Info().
				Str("url", c.ConnectedUrl()).
				Strs("discovered", c.DiscoveredServers()).
				Msg("Reconnected to NATS server")
		}),

		nats.ClosedHandler(
			func(c *nats.Conn) {
				log.Info().Str("url", c.ConnectedUrl()).Msg("NATS connection closed")
			}),
	)

	if err != nil {
		return nil, err
	}

	log.Info().
		Str("url", nc.ConnectedUrl()).
		Msg("Connected to NATS server")

	log.Info().
		Strs("discovered", nc.DiscoveredServers()).
		Msg("Discovered Servers")

	return nc, nil
}
