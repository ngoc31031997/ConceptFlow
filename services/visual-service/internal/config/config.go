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
	// AuthoringServiceURL is where the locked script is read.
	AuthoringServiceURL string
	// LLMServiceURL is where the model calls, prompts and models are.
	LLMServiceURL string
	// LLMTimeoutSeconds bounds a model call; 0 waits as long as llm-service.
	LLMTimeoutSeconds int
	// MaxOutputTokens caps the answer of a model call.
	MaxOutputTokens int
	// RenderingURL is where the fonts rendering draws text in are listed.
	RenderingURL string
}

// Load reads the configuration; DATABASE_URL, AUTHORING_SERVICE_URL and
// LLM_SERVICE_URL are required.
func Load() (*Config, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	authoringURL := os.Getenv("AUTHORING_SERVICE_URL")
	if authoringURL == "" {
		return nil, fmt.Errorf("AUTHORING_SERVICE_URL is required")
	}
	llmURL := os.Getenv("LLM_SERVICE_URL")
	if llmURL == "" {
		return nil, fmt.Errorf("LLM_SERVICE_URL is required")
	}
	renderingURL := os.Getenv("RENDERING_URL")
	if renderingURL == "" {
		renderingURL = "http://rendering:8000"
	}
	maxTokens, err := intEnvOrDefault("MAX_OUTPUT_TOKENS", 128000)
	if err != nil {
		return nil, err
	}
	llmTimeout := 0
	if v := os.Getenv("LLM_TIMEOUT_SECONDS"); v != "" {
		if llmTimeout, err = strconv.Atoi(v); err != nil || llmTimeout < 0 {
			return nil, fmt.Errorf("LLM_TIMEOUT_SECONDS must be a non-negative integer, got %q", v)
		}
	}
	maxConns, err := intEnvOrDefault("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8000"
	}
	return &Config{
		DatabaseURL: databaseURL, DatabaseMaxConns: int32(maxConns), HTTPPort: httpPort,
		AuthoringServiceURL: authoringURL, LLMServiceURL: llmURL, LLMTimeoutSeconds: llmTimeout,
		MaxOutputTokens: maxTokens, RenderingURL: renderingURL,
	}, nil
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
