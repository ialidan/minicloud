package config

import (
	"os"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.MaxUploadSize != 100<<20 {
		t.Errorf("expected default max upload 100 MiB, got %d", cfg.Server.MaxUploadSize)
	}
	if cfg.Storage.DataDir != "./data" {
		t.Errorf("expected default data dir ./data, got %s", cfg.Storage.DataDir)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Log.Level)
	}
	if cfg.Server.ReadTimeout.Std() != 30*time.Second {
		t.Errorf("expected default read timeout 30s, got %v", cfg.Server.ReadTimeout.Std())
	}
}

func TestAddr(t *testing.T) {
	cfg := Defaults()
	if addr := cfg.Addr(); addr != "0.0.0.0:8080" {
		t.Errorf("expected 0.0.0.0:8080, got %s", addr)
	}
}

func TestDBPath_Default(t *testing.T) {
	cfg := Defaults()
	if p := cfg.DBPath(); p != "./data/minicloud.db" {
		t.Errorf("expected ./data/minicloud.db, got %s", p)
	}
}

func TestDBPath_Explicit(t *testing.T) {
	cfg := Defaults()
	cfg.Database.Path = "/custom/path.db"
	if p := cfg.DBPath(); p != "/custom/path.db" {
		t.Errorf("expected /custom/path.db, got %s", p)
	}
}

func TestApplyEnv(t *testing.T) {
	cfg := Defaults()

	t.Setenv("MINICLOUD_PORT", "9090")
	t.Setenv("MINICLOUD_DATA_DIR", "/tmp/test")
	t.Setenv("MINICLOUD_LOG_LEVEL", "DEBUG")
	t.Setenv("MINICLOUD_TLS_ENABLED", "true")

	cfg.ApplyEnv()

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Storage.DataDir != "/tmp/test" {
		t.Errorf("expected data dir /tmp/test, got %s", cfg.Storage.DataDir)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("expected log level debug, got %s", cfg.Log.Level)
	}
	if !cfg.Server.TLS.Enabled {
		t.Error("expected TLS enabled")
	}
}

func TestLoad_MissingDefault_IsOK(t *testing.T) {
	// When no explicit path is given and minicloud.yaml doesn't exist,
	// Load should return defaults without error.
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port, got %d", cfg.Server.Port)
	}
}

func TestLoad_MissingExplicit_IsError(t *testing.T) {
	_, err := Load("nonexistent-" + t.Name() + ".yaml")
	if err == nil {
		t.Fatal("expected error for explicitly missing config file")
	}
}

func TestLoad_YAML(t *testing.T) {
	content := []byte(`
server:
  port: 3000
  read_timeout: "5s"
storage:
  data_dir: "/var/minicloud"
`)
	f, err := os.CreateTemp("", "minicloud-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout.Std() != 5*time.Second {
		t.Errorf("expected read timeout 5s, got %v", cfg.Server.ReadTimeout.Std())
	}
	if cfg.Storage.DataDir != "/var/minicloud" {
		t.Errorf("expected data dir /var/minicloud, got %s", cfg.Storage.DataDir)
	}
	// Non-overridden values should keep defaults.
	if cfg.Server.WriteTimeout.Std() != 60*time.Second {
		t.Errorf("expected default write timeout 60s, got %v", cfg.Server.WriteTimeout.Std())
	}
}

func TestLoad_YAML_IntegerDuration(t *testing.T) {
	content := []byte(`
server:
  read_timeout: 10
`)
	f, err := os.CreateTemp("", "minicloud-test-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Server.ReadTimeout.Std() != 10*time.Second {
		t.Errorf("expected 10s from integer, got %v", cfg.Server.ReadTimeout.Std())
	}
}

func TestSlogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"}, // defaults to info
	}

	for _, tt := range tests {
		l := LogConfig{Level: tt.input}
		got := l.SlogLevel().String()
		if got != tt.want {
			t.Errorf("SlogLevel(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
