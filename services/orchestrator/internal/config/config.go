// Package config loads Orchestrator Service configuration from environment
// variables (deployment-architecture.md's docker-compose entry).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all environment-derived settings for main.go's composition
// root.
type Config struct {
	RabbitMQURL                   string
	DatabaseURL                   string
	OutboxPollIntervalMS          int
	DatabaseMaxConns              int32
	HTTPPort                      string
	RabbitMQReconnectInitialDelay time.Duration
	RabbitMQReconnectMaxDelay     time.Duration
}

// Load reads Config from the environment, applying the defaults documented
// in deployment-architecture.md (OUTBOX_POLL_INTERVAL_MS=500,
// DATABASE_MAX_CONNS=10) and failing fast if a required variable is unset.
func Load() (*Config, error) {
	rabbitMQURL := os.Getenv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is required")
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	pollInterval, err := intEnvOrDefault("OUTBOX_POLL_INTERVAL_MS", 500)
	if err != nil {
		return nil, err
	}
	maxConns, err := intEnvOrDefault("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	reconnectInitialMS, err := intEnvOrDefault("RABBITMQ_RECONNECT_INITIAL_DELAY_MS", 1000)
	if err != nil {
		return nil, err
	}
	reconnectMaxMS, err := intEnvOrDefault("RABBITMQ_RECONNECT_MAX_DELAY_MS", 30000)
	if err != nil {
		return nil, err
	}

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000"
	}

	return &Config{
		RabbitMQURL:                   rabbitMQURL,
		DatabaseURL:                   databaseURL,
		OutboxPollIntervalMS:          pollInterval,
		DatabaseMaxConns:              int32(maxConns),
		HTTPPort:                      httpPort,
		RabbitMQReconnectInitialDelay: time.Duration(reconnectInitialMS) * time.Millisecond,
		RabbitMQReconnectMaxDelay:     time.Duration(reconnectMaxMS) * time.Millisecond,
	}, nil
}

func intEnvOrDefault(key string, def int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", key, err)
	}
	return v, nil
}
