// Package config reads visual-service's settings from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config is visual-service's runtime configuration.
type Config struct {
	DatabaseURL      string
	DatabaseMaxConns int32
	HTTPPort         string
}

// Load reads the configuration; DATABASE_URL is required.
func Load() (*Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	maxConns, err := intEnvOrDefault("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000"
	}
	return &Config{DatabaseURL: databaseURL, DatabaseMaxConns: int32(maxConns), HTTPPort: httpPort}, nil
}

func intEnvOrDefault(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", key, v)
	}
	return n, nil
}
