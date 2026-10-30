package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// NATSClientConfig maps externalized enterprise connection variables.
type NATSClientConfig struct {
	Servers              []string      `env:"NATS_SERVERS" envSeparator:"," envDefault:"nats://localhost:4222"`
	ClientName           string        `env:"NATS_CLIENT_NAME" envDefault:"order-service"`
	ConnectTimeout       time.Duration `env:"NATS_CONNECT_TIMEOUT" envDefault:"2s"`
	AllowReconnect       bool          `env:"NATS_ALLOW_RECONNECT" envDefault:"true"`
	MaxReconnectAttempts int           `env:"NATS_MAX_RECONNECT_ATTEMPTS" envDefault:"10"`
	ReconnectWait        time.Duration `env:"NATS_RECONNECT_WAIT" envDefault:"2s"`
	RetryOnFailedConnect bool          `env:"NATS_RETRY_ON_FAILED_CONNECT" envDefault:"true"`
	PingInterval         time.Duration `env:"NATS_PING_INTERVAL" envDefault:"5s"`
	MaxPingsOut          int           `env:"NATS_MAX_PINGS_OUT" envDefault:"3"`
	DrainTimeout         time.Duration `env:"NATS_DRAIN_TIMEOUT" envDefault:"30s"`
	TLSEnabled           bool          `env:"NATS_TLS_ENABLED" envDefault:"false"`
	TLSRootCASource      string        `env:"NATS_TLS_ROOT_CA_SOURCE"`
	TLSClientCertSource  string        `env:"NATS_TLS_CLIENT_CERT_SOURCE"`
	TLSClientKeySource   string        `env:"NATS_TLS_CLIENT_KEY_SOURCE"`
	AuthMethod           string        `env:"NATS_AUTH_METHOD"`
	AuthToken            string        `env:"NATS_AUTH_TOKEN"`
	AuthUser             string        `env:"NATS_AUTH_USER"`
	AuthPassword         string        `env:"NATS_AUTH_PASSWORD"`
	AuthCredentialsFile  string        `env:"NATS_AUTH_CREDENTIALS_FILE"`
}

// Validate verifies that required connection and security configurations are consistent.
func (c *NATSClientConfig) Validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("NATS_SERVERS cannot be empty")
	}
	for i, s := range c.Servers {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("NATS_SERVERS contains empty server URL at index %d", i)
		}
	}

	if c.TLSEnabled {
		if (c.TLSClientCertSource != "" && c.TLSClientKeySource == "") ||
			(c.TLSClientCertSource == "" && c.TLSClientKeySource != "") {
			return fmt.Errorf("both NATS_TLS_CLIENT_CERT_SOURCE and NATS_TLS_CLIENT_KEY_SOURCE must be set when configuring client certificates")
		}
	}

	switch strings.ToLower(strings.TrimSpace(c.AuthMethod)) {
	case "":
		// No auth method specified, standard unauthenticated connection.
	case "token":
		if strings.TrimSpace(c.AuthToken) == "" {
			return fmt.Errorf("NATS_AUTH_TOKEN is required when NATS_AUTH_METHOD is 'token'")
		}
	case "user_password":
		if strings.TrimSpace(c.AuthUser) == "" {
			return fmt.Errorf("NATS_AUTH_USER is required when NATS_AUTH_METHOD is 'user_password'")
		}
	case "jwt", "credentials":
		if strings.TrimSpace(c.AuthCredentialsFile) == "" {
			return fmt.Errorf("NATS_AUTH_CREDENTIALS_FILE is required when NATS_AUTH_METHOD is 'jwt' or 'credentials'")
		}
	default:
		return fmt.Errorf("unsupported NATS_AUTH_METHOD %q: expected 'token', 'user_password', 'jwt', or 'credentials'", c.AuthMethod)
	}

	return nil
}

// BuildOptions translates the configuration entity into a nats.Option slice.
// If clientNameOverride is supplied, it takes precedence over c.ClientName.
func (c *NATSClientConfig) BuildOptions(clientNameOverride ...string) ([]nats.Option, error) {
	effectiveClientName := c.ClientName
	if len(clientNameOverride) > 0 && strings.TrimSpace(clientNameOverride[0]) != "" {
		effectiveClientName = strings.TrimSpace(clientNameOverride[0])
	}

	opts := []nats.Option{
		nats.Name(effectiveClientName),
		nats.Timeout(c.ConnectTimeout),
		nats.ReconnectWait(c.ReconnectWait),
		nats.PingInterval(c.PingInterval),
		nats.MaxPingsOutstanding(c.MaxPingsOut),
		nats.DrainTimeout(c.DrainTimeout),
		nats.ReconnectBufSize(8 * 1024 * 1024), // 8MB buffer for outbound messages while disconnected
		nats.CustomReconnectDelay(func(attempts int) time.Duration {
			// Exponential backoff up to 10 seconds to prevent thundering herd
			delay := time.Duration(attempts) * time.Second
			if delay > 10*time.Second {
				return 10 * time.Second
			}
			return delay
		}),
		nats.ErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
			if sub != nil {
				log.Error().Err(err).Str("subject", sub.Subject).Msg("NATS async error on subject")
			} else {
				log.Error().Err(err).Msg("NATS async error")
			}
		}),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			log.Warn().Err(err).Msg("Disconnected from NATS. Swapping to next cluster node...")
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			log.Info().
				Str("url", conn.ConnectedUrl()).
				Strs("discovered", conn.DiscoveredServers()).
				Msg("Reconnected to NATS server")
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			log.Info().Str("url", conn.ConnectedUrl()).Msg("NATS connection closed")
		}),
	}

	if !c.AllowReconnect {
		opts = append(opts, nats.NoReconnect())
	} else {
		opts = append(opts, nats.MaxReconnects(c.MaxReconnectAttempts))
	}

	if c.RetryOnFailedConnect {
		opts = append(opts, nats.RetryOnFailedConnect(true))
	}

	// TLS Transport Security
	if c.TLSEnabled {
		if c.TLSRootCASource != "" {
			opts = append(opts, nats.RootCAs(c.TLSRootCASource))
		}
		if c.TLSClientCertSource != "" && c.TLSClientKeySource != "" {
			opts = append(opts, nats.ClientCert(c.TLSClientCertSource, c.TLSClientKeySource))
		}
	}

	// Authentication Methods
	switch strings.ToLower(strings.TrimSpace(c.AuthMethod)) {
	case "token":
		if c.AuthToken != "" {
			opts = append(opts, nats.Token(c.AuthToken))
		}
	case "user_password":
		if c.AuthUser != "" {
			opts = append(opts, nats.UserInfo(c.AuthUser, c.AuthPassword))
		}
	case "jwt", "credentials":
		if c.AuthCredentialsFile != "" {
			opts = append(opts, nats.UserCredentials(c.AuthCredentialsFile))
		}
	}

	return opts, nil
}
