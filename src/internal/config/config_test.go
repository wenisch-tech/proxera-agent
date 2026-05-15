package config

import (
	"testing"
	"time"
)

func TestLoadUsesEnvValues(t *testing.T) {
	t.Setenv(envServerURL, "wss://example.com/tunnel")
	t.Setenv(envAPIKey, "secret")
	t.Setenv(envHeartbeatInterval, "5s")
	t.Setenv(envHeartbeatTimeout, "2s")

	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.ServerURL != "wss://example.com/tunnel" {
		t.Fatalf("unexpected server URL: %s", cfg.ServerURL)
	}
	if cfg.APIKey != "secret" {
		t.Fatalf("unexpected API key")
	}
	if cfg.HeartbeatInterval != 5*time.Second {
		t.Fatalf("unexpected heartbeat interval: %v", cfg.HeartbeatInterval)
	}
}

func TestLoadFlagOverridesEnv(t *testing.T) {
	t.Setenv(envServerURL, "wss://env.example.com/tunnel")
	t.Setenv(envAPIKey, "env-secret")

	cfg, err := Load([]string{"--server-url", "wss://flag.example.com/tunnel", "--api-key", "flag-secret"})
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}

	if cfg.ServerURL != "wss://flag.example.com/tunnel" {
		t.Fatalf("expected flag server URL to win, got: %s", cfg.ServerURL)
	}
	if cfg.APIKey != "flag-secret" {
		t.Fatalf("expected flag API key to win")
	}
}

func TestValidateRejectsMissingAPIKey(t *testing.T) {
	t.Setenv(envServerURL, "wss://example.com/tunnel")
	t.Setenv(envAPIKey, "")

	_, err := Load(nil)
	if err == nil {
		t.Fatalf("expected missing API key error")
	}
}
