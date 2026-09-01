package config

import (
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

// Config holds runtime configuration configuration variables.
type Config struct {
	NATSURL string `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	Port    string `env:"PORT" envDefault:"8080"`
}

// Load loads configuration from a local .env file (if present) and the environment.
func Load() (*Config, error) {
	// Load environment variables from .env file if it exists.
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("No .env file found: %v\n", err)
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

// Validate checks that the configuration values are present.
func (c *Config) Validate() error {
	if c.NATSURL == "" {
		return fmt.Errorf("NATS_URL cannot be empty")
	}
	if c.Port == "" {
		return fmt.Errorf("PORT cannot be empty")
	}
	return nil
}
