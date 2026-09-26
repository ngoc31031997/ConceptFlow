package config_test

import (
	"testing"
	"time"

	"orchestrator/internal/config"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("RABBITMQ_URL", "amqp://localhost:5672/")
	t.Setenv("DATABASE_URL", "postgresql://localhost:5432/orchestrator")
}

// CR-040 FR111: the orchestrator reaches authoring-service on the compose
// network by default, and both settings can be overridden.
func TestLoad_AuthoringServiceDefaultsAndOverrides(t *testing.T) {
	setRequired(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthoringServiceURL != "http://authoring-service:8000" || cfg.AuthoringServiceTimeout != 10*time.Second {
		t.Fatalf("unexpected defaults %q %v", cfg.AuthoringServiceURL, cfg.AuthoringServiceTimeout)
	}

	t.Setenv("AUTHORING_SERVICE_URL", "http://localhost:9100")
	t.Setenv("AUTHORING_SERVICE_TIMEOUT_SECONDS", "3")
	cfg, err = config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AuthoringServiceURL != "http://localhost:9100" || cfg.AuthoringServiceTimeout != 3*time.Second {
		t.Fatalf("overrides ignored: %q %v", cfg.AuthoringServiceURL, cfg.AuthoringServiceTimeout)
	}
}

func TestLoad_RejectsAnUnparseableNumber(t *testing.T) {
	setRequired(t)
	t.Setenv("AUTHORING_SERVICE_TIMEOUT_SECONDS", "soon")
	if _, err := config.Load(); err == nil {
		t.Fatal("a misspelled number must be an error, not a silent zero")
	}
}
