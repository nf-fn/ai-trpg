package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	os.WriteFile(path, []byte("{}"), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Paths.Web != "web" {
		t.Errorf("expected web path 'web', got '%s'", cfg.Paths.Web)
	}
	if cfg.Paths.Rules != "rules" {
		t.Errorf("expected rules path 'rules', got '%s'", cfg.Paths.Rules)
	}
	if cfg.Paths.Scenarios != "scenarios" {
		t.Errorf("expected scenarios path 'scenarios', got '%s'", cfg.Paths.Scenarios)
	}
	if cfg.Ollama.URL != "http://localhost:11434" {
		t.Errorf("expected default ollama URL, got '%s'", cfg.Ollama.URL)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestLoadConfigCustomPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
paths:
  web: "/var/www/html"
  rules: "/etc/trpg/rules"
  scenarios: "/etc/trpg/scenarios"
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Paths.Web != "/var/www/html" {
		t.Errorf("expected '/var/www/html', got '%s'", cfg.Paths.Web)
	}
	if cfg.Paths.Rules != "/etc/trpg/rules" {
		t.Errorf("expected '/etc/trpg/rules', got '%s'", cfg.Paths.Rules)
	}
	if cfg.Paths.Scenarios != "/etc/trpg/scenarios" {
		t.Errorf("expected '/etc/trpg/scenarios', got '%s'", cfg.Paths.Scenarios)
	}
}

func TestLoadConfigPartialPaths(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
paths:
  web: "static"
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Paths.Web != "static" {
		t.Errorf("expected 'static', got '%s'", cfg.Paths.Web)
	}
	if cfg.Paths.Rules != "rules" {
		t.Errorf("expected default 'rules', got '%s'", cfg.Paths.Rules)
	}
	if cfg.Paths.Scenarios != "scenarios" {
		t.Errorf("expected default 'scenarios', got '%s'", cfg.Paths.Scenarios)
	}
}
