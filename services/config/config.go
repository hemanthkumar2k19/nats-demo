package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds runtime configuration variables.
type Config struct {
	NATS                     NATSClientConfig
	NATSURL                  string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	OtelExporterType         string `env:"OTEL_EXPORTER_TYPE" envDefault:"otlp-grpc"`
	OtelExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:"localhost:4317"`
	ServiceName              string `env:"SERVICE_NAME" envDefault:"nats-tracing-demo"`
	OrderServicePort         string `env:"ORDER_SERVICE_PORT" envDefault:":8080"`
	ProcessingServicePort    string `env:"PROCESSING_SERVICE_PORT" envDefault:":8081"`
}

// Load loads configuration from a local .env file (if present) and the environment.
func Load() (*Config, error) {
	// Load environment variables from .env file if it exists.
	err := godotenv.Load()
	if err != nil {
		log.Warn().Msgf("No .env file found: %v\n", err)
	}

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks that required configuration values are present.
func (c *Config) Validate() error {
	// Sync legacy NATSURL if present and NATS.Servers is using default
	if c.NATSURL != "" && (len(c.NATS.Servers) == 0 || (len(c.NATS.Servers) == 1 && c.NATS.Servers[0] == "nats://localhost:4222")) {
		c.NATS.Servers = []string{c.NATSURL}
	}

	if err := c.NATS.Validate(); err != nil {
		return fmt.Errorf("nats config error: %w", err)
	}

	return nil
}
