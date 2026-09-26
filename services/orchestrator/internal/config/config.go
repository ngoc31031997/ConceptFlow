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
	// CR-039 — every language-model call goes through llm-service, which alone
	// holds HIVE_API_KEY / OLLAMA_URL and the retry/rate-limit policy. The
	// orchestrator keeps only where to reach it and the cost ceilings it
	// enforces on what it sends.
	LLMServiceURL string
	// 0 = wait as long as llm-service does (Hive can take minutes on a
	// reasoning model).
	LLMServiceTimeout time.Duration
	// HiveModel is the default the model picker shows; llm-service applies its
	// own HIVE_MODEL when a call names none, and both read the same .env.
	HiveModel           string
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

	llmServiceURL := os.Getenv("LLM_SERVICE_URL")
	if llmServiceURL == "" {
		llmServiceURL = "http://llm-service:8000"
	}
	llmServiceTimeoutSeconds, err := intEnvOrDefault("LLM_SERVICE_TIMEOUT_SECONDS", 0) // 0 = no timeout
	if err != nil {
		return nil, err
	}

	hiveModel := os.Getenv("HIVE_MODEL")
	if hiveModel == "" {
		hiveModel = "deepseek-ai/deepseek-v4.1-flash"
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
		LLMServiceURL:                 llmServiceURL,
		LLMServiceTimeout:             time.Duration(llmServiceTimeoutSeconds) * time.Second,
		HiveModel:                     hiveModel,
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
