package config

import "testing"

func TestLoad_RequiresTheDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("a missing DATABASE_URL must be an error")
	}
}

func TestLoad_DefaultsAndOverrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPPort != "8000" || cfg.DatabaseMaxConns != 10 {
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
