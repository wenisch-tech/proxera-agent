package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	envServerURL         = "PROXERA_SERVER_URL"
	envAPIKey            = "PROXERA_API_KEY"
	envLogLevel          = "PROXERA_LOG_LEVEL"
	envHeartbeatInterval = "PROXERA_HEARTBEAT_INTERVAL"
	envHeartbeatTimeout  = "PROXERA_HEARTBEAT_TIMEOUT"
	envReconnectBase     = "PROXERA_RECONNECT_BASE"
	envReconnectMax      = "PROXERA_RECONNECT_MAX"
	envRequestTimeout    = "PROXERA_REQUEST_TIMEOUT"
	envConcurrencyLimit  = "PROXERA_CONCURRENCY_LIMIT"
)

type Config struct {
	ServerURL         string
	APIKey            string
	LogLevel          string
	HeartbeatInterval time.Duration
	HeartbeatTimeout  time.Duration
	ReconnectBase     time.Duration
	ReconnectMax      time.Duration
	RequestTimeout    time.Duration
	ConcurrencyLimit  int
}

func Load(args []string) (Config, error) {
	cfg := Config{
		ServerURL:         envOrDefault(envServerURL, ""),
		APIKey:            envOrDefault(envAPIKey, ""),
		LogLevel:          strings.ToLower(envOrDefault(envLogLevel, "info")),
		HeartbeatInterval: mustDurationFromEnv(envHeartbeatInterval, 30*time.Second),
		HeartbeatTimeout:  mustDurationFromEnv(envHeartbeatTimeout, 10*time.Second),
		ReconnectBase:     mustDurationFromEnv(envReconnectBase, 1*time.Second),
		ReconnectMax:      mustDurationFromEnv(envReconnectMax, 60*time.Second),
		RequestTimeout:    mustDurationFromEnv(envRequestTimeout, 30*time.Second),
		ConcurrencyLimit:  mustIntFromEnv(envConcurrencyLimit, 100),
	}

	fs := flag.NewFlagSet("proxera-agent", flag.ContinueOnError)
	fs.StringVar(&cfg.ServerURL, "server-url", cfg.ServerURL, "Proxera server websocket URL, e.g. wss://server/tunnel")
	fs.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "API key used as X-Proxera-Token")
	fs.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level: debug|info|warn|error")
	fs.DurationVar(&cfg.HeartbeatInterval, "heartbeat-interval", cfg.HeartbeatInterval, "Heartbeat ping interval")
	fs.DurationVar(&cfg.HeartbeatTimeout, "heartbeat-timeout", cfg.HeartbeatTimeout, "Heartbeat pong timeout")
	fs.DurationVar(&cfg.ReconnectBase, "reconnect-base", cfg.ReconnectBase, "Reconnect initial backoff duration")
	fs.DurationVar(&cfg.ReconnectMax, "reconnect-max", cfg.ReconnectMax, "Reconnect maximum backoff duration")
	fs.DurationVar(&cfg.RequestTimeout, "request-timeout", cfg.RequestTimeout, "Timeout for proxied local HTTP requests")
	fs.IntVar(&cfg.ConcurrencyLimit, "concurrency-limit", cfg.ConcurrencyLimit, "Max in-flight proxied requests")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("%s or --server-url must be set", envServerURL)
	}
	if !strings.HasPrefix(c.ServerURL, "ws://") && !strings.HasPrefix(c.ServerURL, "wss://") {
		return errors.New("server URL must start with ws:// or wss://")
	}
	if c.APIKey == "" {
		return fmt.Errorf("%s or --api-key must be set", envAPIKey)
	}
	if c.HeartbeatInterval <= 0 || c.HeartbeatTimeout <= 0 {
		return errors.New("heartbeat interval and timeout must be positive")
	}
	if c.ReconnectBase <= 0 || c.ReconnectMax < c.ReconnectBase {
		return errors.New("reconnect-base must be > 0 and reconnect-max must be >= reconnect-base")
	}
	if c.RequestTimeout <= 0 {
		return errors.New("request-timeout must be positive")
	}
	if c.ConcurrencyLimit <= 0 {
		return errors.New("concurrency-limit must be positive")
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func mustDurationFromEnv(key string, fallback time.Duration) time.Duration {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return v
}

func mustIntFromEnv(key string, fallback int) int {
	raw, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return fallback
	}
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return v
}
