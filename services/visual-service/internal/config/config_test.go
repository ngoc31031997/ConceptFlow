package config

import "testing"

func TestLoad_RequiresTheDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be an error")
	}
}

func TestLoad_RequiresTheServicesItReads(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("AUTHORING_SERVICE_URL", "")
	t.Setenv("LLM_SERVICE_URL", "http://llm")
	if _, err := Load(); err == nil {
		t.Fatal("a missing AUTHORING_SERVICE_URL must be an error")
	}
	t.Setenv("AUTHORING_SERVICE_URL", "http://authoring")
	t.Setenv("LLM_SERVICE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("a missing LLM_SERVICE_URL must be an error")
	}
}

func TestLoad_DefaultsAndOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("AUTHORING_SERVICE_URL", "http://authoring")
	t.Setenv("LLM_SERVICE_URL", "http://llm")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPPort != "8000" || cfg.DatabaseMaxConns != 10 || cfg.LLMTimeoutSeconds != 0 || cfg.MaxOutputTokens != 128000 || cfg.RenderingURL != "http://rendering:8000" {
		t.Fatalf("defaults %+v", cfg)
	}
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("DATABASE_MAX_CONNS", "3")
	if cfg, err = Load(); err != nil || cfg.HTTPPort != "9000" || cfg.DatabaseMaxConns != 3 {
		t.Fatalf("overrides %+v err %v", cfg, err)
	}
	t.Setenv("DATABASE_MAX_CONNS", "many")
	if _, err := Load(); err == nil {
		t.Fatal("a non-number DATABASE_MAX_CONNS must be an error")
	}
}
