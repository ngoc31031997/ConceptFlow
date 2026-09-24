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
	OllamaURL                     string
	OllamaModel                   string
	OllamaTimeout                 time.Duration
	// CR-027 — Hive is the primary LLM provider; Ollama stays as the
	// fallback for the light tasks (see llm.OllamaProvider).
	LLMProvider         string
	HiveAPIKey          string
	HiveBaseURL         string
	HiveModel           string
	HiveTimeout         time.Duration
	HiveMaxRetries      int
	HiveMaxInputChars   int
	HiveMaxOutputTokens int
	// CR-023 D2: base URL of the video-assembly service, whose own database
	// owns channel_assets — Orchestrator reads it synchronously to attach the
	// channel intro/outro to assemble_video.
	VideoAssemblyURL     string
	VideoAssemblyTimeout time.Duration

	// QCEnforce turns CR-021's publish gate from indicate-only into a real
	// block. Default FALSE on purpose (D5 / CR-021 Decision #3): the rules ship
	// unproven against real footage, and a gate that cries wolf on its first
	// week is a gate Creators learn to click past — which costs more than
	// having no gate at all. Findings are recorded with their true severity
	// either way; this only decides whether a blocking one stops the Saga.
	QCEnforce bool
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

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://ollama:11434"
	}
	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = "llama3.2"
	}
	ollamaTimeoutSeconds, err := intEnvOrDefault("OLLAMA_TIMEOUT_SECONDS", 120)
	if err != nil {
		return nil, err
	}

	// CR-027 FR83.2 — no key is a supported way to run: the app falls back to
	// Ollama for the light tasks and every prompt stays copy-out-to-an-AI, as
	// it was before CR-027. Refusing to start would turn an optional paid
	// service into a hard dependency of the whole orchestrator.
	hiveAPIKey := os.Getenv("HIVE_API_KEY")
	llmProvider := os.Getenv("LLM_PROVIDER")
	if llmProvider == "" {
		if hiveAPIKey != "" {
			llmProvider = "hive"
		} else {
			llmProvider = "ollama"
		}
	}
	hiveBaseURL := os.Getenv("HIVE_BASE_URL")
	if hiveBaseURL == "" {
		// api-cdn, not api-va1: measured 2026-09-21, api-va1 answered 500
		// even for a key carrying va1:* permissions.
		hiveBaseURL = "https://api-cdn.thehive.ai/api/v3"
	}
	hiveModel := os.Getenv("HIVE_MODEL")
	if hiveModel == "" {
		hiveModel = "deepseek-ai/deepseek-v4.1-flash"
	}
	hiveTimeoutSeconds, err := intEnvOrDefault("HIVE_TIMEOUT_SECONDS", 0) // 0 = no timeout
	if err != nil {
		return nil, err
	}
	hiveMaxRetries, err := intEnvOrDefault("HIVE_MAX_RETRIES", 3)
	if err != nil {
		return nil, err
	}
	// Hive's context window is 1M tokens, so this is not a context limit —
	// it is a blast radius. One broken project must not turn into one
	// enormous billable call.
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

	videoAssemblyURL := os.Getenv("VIDEO_ASSEMBLY_URL")
	if videoAssemblyURL == "" {
		videoAssemblyURL = "http://video-assembly:8000"
	}
	videoAssemblyTimeoutSeconds, err := intEnvOrDefault("VIDEO_ASSEMBLY_TIMEOUT_SECONDS", 5)
	if err != nil {
		return nil, err
	}

	qcEnforce, err := boolEnvOrDefault("QC_ENFORCE", false)
	if err != nil {
		return nil, err
	}

	return &Config{
		QCEnforce:                     qcEnforce,
		RabbitMQURL:                   rabbitMQURL,
		DatabaseURL:                   databaseURL,
		OutboxPollIntervalMS:          pollInterval,
		DatabaseMaxConns:              int32(maxConns),
		HTTPPort:                      httpPort,
		RabbitMQReconnectInitialDelay: time.Duration(reconnectInitialMS) * time.Millisecond,
		RabbitMQReconnectMaxDelay:     time.Duration(reconnectMaxMS) * time.Millisecond,
		OllamaURL:                     ollamaURL,
		OllamaModel:                   ollamaModel,
		OllamaTimeout:                 time.Duration(ollamaTimeoutSeconds) * time.Second,
		LLMProvider:                   llmProvider,
		HiveAPIKey:                    hiveAPIKey,
		HiveBaseURL:                   hiveBaseURL,
		HiveModel:                     hiveModel,
		HiveTimeout:                   time.Duration(hiveTimeoutSeconds) * time.Second,
		HiveMaxRetries:                hiveMaxRetries,
		HiveMaxInputChars:             hiveMaxInputChars,
		HiveMaxOutputTokens:           hiveMaxOutputTokens,
		VideoAssemblyURL:              videoAssemblyURL,
		VideoAssemblyTimeout:          time.Duration(videoAssemblyTimeoutSeconds) * time.Second,
	}, nil
}

// boolEnvOrDefault reads a boolean env var, rejecting anything strconv does
// not recognise rather than silently reading a typo as false — a misspelled
// QC_ENFORCE that quietly disables the gate is the failure mode worth refusing
// to start over.
func boolEnvOrDefault(key string, def bool) (bool, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean (true/false): %w", key, err)
	}
	return v, nil
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
