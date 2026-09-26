package config_test

import (
	"testing"
	"time"

	"orchestrator/internal/config"
)

// setRequired fills the two variables Load refuses to start without, so each
// test below can speak only about the LLM settings.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("DATABASE_URL", "postgresql://localhost:5432/orchestrator")
}

// The orchestrator no longer knows any provider credential: with nothing set
// it still starts (CR-027 FR83.2 — the AI path is optional) and points at
// llm-service on the compose network.
func TestLoad_LLMServiceDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("no LLM configuration must not stop startup: %v", err)
	}
	if cfg.LLMServiceURL != "http://llm-service:8000" {
		t.Fatalf("unexpected llm-service URL %q", cfg.LLMServiceURL)
	}
	if cfg.LLMServiceTimeout != 0 {
		t.Fatalf("unexpected timeout %v", cfg.LLMServiceTimeout)
	}
	if cfg.HiveModel != "deepseek-ai/deepseek-v4.1-flash" {
		t.Fatalf("unexpected model %q", cfg.HiveModel)
	}
	if cfg.HiveMaxInputChars != 120000 || cfg.HiveMaxOutputTokens != 128000 {
		t.Fatalf("unexpected ceilings %d / %d", cfg.HiveMaxInputChars, cfg.HiveMaxOutputTokens)
	}
}

func TestLoad_LLMServiceOverrides(t *testing.T) {
	setRequired(t)
	t.Setenv("LLM_SERVICE_URL", "http://localhost:9000")
	t.Setenv("LLM_SERVICE_TIMEOUT_SECONDS", "90")

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMServiceURL != "http://localhost:9000" || cfg.LLMServiceTimeout != 90*time.Second {
		t.Fatalf("overrides ignored: %q %v", cfg.LLMServiceURL, cfg.LLMServiceTimeout)
	}
}

func TestLoad_RejectsAnUnparseableNumber(t *testing.T) {
	setRequired(t)
	t.Setenv("HIVE_MAX_OUTPUT_TOKENS", "lots")

	if _, err := config.Load(); err == nil {
		t.Fatal("a misspelled number must be an error, not a silent zero")
	}
}
