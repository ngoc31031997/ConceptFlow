package config_test

import (
	"testing"

	"orchestrator/internal/config"
)

// setRequired fills the two variables Load refuses to start without, so each
// test below can speak only about the CR-027 Hive settings.
func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("DATABASE_URL", "postgresql://localhost:5432/orchestrator")
}

// TestLoad_NoHiveKeyStillStarts is CR-027 FR83.2. Running without a paid key
// is a supported state — the copy-the-prompt-out flow keeps working and the
// light tasks stay on Ollama — so a missing key must never stop the
// orchestrator from booting.
func TestLoad_NoHiveKeyStillStarts(t *testing.T) {
	setRequired(t)
	t.Setenv("HIVE_API_KEY", "")
	t.Setenv("LLM_PROVIDER", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("a missing Hive key must not stop startup: %v", err)
	}
	if cfg.LLMProvider != "ollama" {
		t.Fatalf("want the ollama fallback, got %q", cfg.LLMProvider)
	}
}

func TestLoad_AKeySelectsHiveByDefault(t *testing.T) {
	setRequired(t)
	t.Setenv("HIVE_API_KEY", "some-key")
	t.Setenv("LLM_PROVIDER", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMProvider != "hive" {
		t.Fatalf("want hive, got %q", cfg.LLMProvider)
	}
}

// TestLoad_ExplicitProviderWinsOverTheKey — having a key must not force Hive
// on. Pinning LLM_PROVIDER=ollama is how a Creator with a funded account
// runs offline without deleting their key.
func TestLoad_ExplicitProviderWinsOverTheKey(t *testing.T) {
	setRequired(t)
	t.Setenv("HIVE_API_KEY", "some-key")
	t.Setenv("LLM_PROVIDER", "ollama")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LLMProvider != "ollama" {
		t.Fatalf("an explicit LLM_PROVIDER must win, got %q", cfg.LLMProvider)
	}
}

func TestLoad_HiveDefaults(t *testing.T) {
	setRequired(t)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// api-cdn, not api-va1: measured 2026-09-21, api-va1 answered 500 even
	// for a key carrying va1:* permissions.
	if cfg.HiveBaseURL != "https://api-cdn.thehive.ai/api/v3" {
		t.Fatalf("unexpected base URL %q", cfg.HiveBaseURL)
	}
	if cfg.HiveModel != "deepseek-ai/deepseek-v4.1-flash" {
		t.Fatalf("unexpected model %q", cfg.HiveModel)
	}
	// Generous on purpose — a reasoning model spends part of this budget
	// before writing the first character of the answer (CR-027 D13).
	if cfg.HiveMaxOutputTokens != 16000 {
		t.Fatalf("unexpected output budget %d", cfg.HiveMaxOutputTokens)
	}
	if cfg.HiveTimeout.Seconds() != 180 {
		t.Fatalf("unexpected timeout %v", cfg.HiveTimeout)
	}
}

func TestLoad_RejectsAnUnparseableHiveNumber(t *testing.T) {
	setRequired(t)
	t.Setenv("HIVE_MAX_RETRIES", "three")

	if _, err := config.Load(); err == nil {
		t.Fatal("a misspelled number must be an error, not a silent zero")
	}
}
