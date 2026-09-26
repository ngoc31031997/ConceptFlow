// Package config loads authoring-service configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every environment-derived setting for main.go's composition root.
type Config struct {
	DatabaseURL      string
	DatabaseMaxConns int32
	HTTPPort         string

	// OrchestratorURL is where project data is read (status, settings, video
	// formats, voice calibration) and where the project journal is written.
	// authoring-service has no access to the orchestrator's database.
	OrchestratorURL     string
	OrchestratorTimeout time.Duration

	// CR-039 — every language-model call goes through llm-service, which alone
	// holds HIVE_API_KEY / OLLAMA_URL and the retry/rate-limit policy.
	LLMServiceURL string
	// 0 = wait as long as llm-service does (Hive can take minutes on a reasoning model).
	LLMServiceTimeout time.Duration
	// HiveModel is the default the model picker shows; llm-service applies its
	// own HIVE_MODEL when a call names none.
	HiveModel           string
	HiveMaxInputChars   int
	HiveMaxOutputTokens int
}

// Load reads Config from the environment, failing fast if DATABASE_URL is unset.
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
	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL == "" {
		orchestratorURL = "http://orchestrator:8000"
	}
	orchestratorTimeout, err := intEnvOrDefault("ORCHESTRATOR_TIMEOUT_SECONDS", 10)
	if err != nil {
		return nil, err
	}
	llmServiceURL := os.Getenv("LLM_SERVICE_URL")
	if llmServiceURL == "" {
		llmServiceURL = "http://llm-service:8000"
	}
	llmServiceTimeoutSeconds, err := intEnvOrDefault("LLM_SERVICE_TIMEOUT_SECONDS", 0)
	if err != nil {
		return nil, err
	}
	hiveModel := os.Getenv("HIVE_MODEL")
	if hiveModel == "" {
		hiveModel = "deepseek-ai/deepseek-v4.1-flash"
	}
	// Hive's context window is 1M tokens, so this is not a context limit — it is
	// a blast radius. One broken project must not turn into one enormous billable call.
	hiveMaxInputChars, err := intEnvOrDefault("HIVE_MAX_INPUT_CHARS", 120000)
	if err != nil {
		return nil, err
	}
	// Generous on purpose: a full Manim script runs to several hundred lines,
	// and on a reasoning model part of this budget is spent before the first
	// character of the answer is written (CR-027 D13).
	hiveMaxOutputTokens, err := intEnvOrDefault("HIVE_MAX_OUTPUT_TOKENS", 128000)
	if err != nil {
		return nil, err
	}

	return &Config{
		DatabaseURL:         databaseURL,
		DatabaseMaxConns:    int32(maxConns),
		HTTPPort:            httpPort,
		OrchestratorURL:     orchestratorURL,
		OrchestratorTimeout: time.Duration(orchestratorTimeout) * time.Second,
		LLMServiceURL:       llmServiceURL,
		LLMServiceTimeout:   time.Duration(llmServiceTimeoutSeconds) * time.Second,
		HiveModel:           hiveModel,
		HiveMaxInputChars:   hiveMaxInputChars,
		HiveMaxOutputTokens: hiveMaxOutputTokens,
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
