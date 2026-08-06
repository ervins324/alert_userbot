package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration parameters.
type Config struct {
	NeptunWSURL          string
	TelegramBotToken     string
	DestinationChatID    string
	SourceChannel        string
	SkipPatterns         []string
	PollInterval         time.Duration
	MinReconnectInterval time.Duration
	MaxReconnectInterval time.Duration
	HTTPTimeout          time.Duration
	QueueCapacity        int
	WorkerCount          int
}

// Load loads and validates configuration from environment variables / .env.
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; real env vars take precedence

	cfg := &Config{
		NeptunWSURL:          getEnv("NEPTUN_WS_URL", "wss://neptun.in.ua/api/v1/stream"),
		TelegramBotToken:     strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		DestinationChatID:    strings.TrimSpace(firstNonEmpty(os.Getenv("DESTINATION_CHAT_ID"), os.Getenv("TELEGRAM_CHAT_ID"))),
		SourceChannel:        getEnv("SOURCE_CHANNEL", "mon1tor_ua"),
		SkipPatterns:         getEnvList("SKIP_PATTERNS"),
		PollInterval:         getEnvDuration("POLL_INTERVAL", 10*time.Second),
		MinReconnectInterval: getEnvDuration("MIN_RECONNECT_INTERVAL", 1*time.Second),
		MaxReconnectInterval: getEnvDuration("MAX_RECONNECT_INTERVAL", 30*time.Second),
		HTTPTimeout:          getEnvDuration("HTTP_TIMEOUT", 10*time.Second),
		QueueCapacity:        getEnvInt("QUEUE_CAPACITY", 1000),
		WorkerCount:          getEnvInt("WORKER_COUNT", 4),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate checks required configuration values.
func (c *Config) Validate() error {
	if c.NeptunWSURL == "" {
		return ErrMissingNeptunURL
	}
	if !strings.HasPrefix(c.NeptunWSURL, "ws://") && !strings.HasPrefix(c.NeptunWSURL, "wss://") {
		return fmt.Errorf("invalid NEPTUN_WS_URL scheme, must start with ws:// or wss://: %s", c.NeptunWSURL)
	}
	if c.TelegramBotToken == "" {
		return ErrMissingTelegramToken
	}
	if c.DestinationChatID == "" {
		return ErrMissingChatID
	}
	if strings.TrimSpace(c.SourceChannel) == "" {
		return fmt.Errorf("SOURCE_CHANNEL cannot be empty")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("POLL_INTERVAL must be positive")
	}
	if c.MinReconnectInterval <= 0 {
		return fmt.Errorf("MIN_RECONNECT_INTERVAL must be positive")
	}
	if c.MaxReconnectInterval < c.MinReconnectInterval {
		return fmt.Errorf("MAX_RECONNECT_INTERVAL must be >= MIN_RECONNECT_INTERVAL")
	}
	if c.QueueCapacity <= 0 {
		return fmt.Errorf("QUEUE_CAPACITY must be positive")
	}
	if c.WorkerCount <= 0 {
		return fmt.Errorf("WORKER_COUNT must be positive")
	}
	return nil
}

var (
	ErrMissingNeptunURL = errors.New("NEPTUN_WS_URL cannot be empty")
	ErrMissingTelegramToken = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
	ErrMissingChatID        = errors.New("DESTINATION_CHAT_ID (or TELEGRAM_CHAT_ID) environment variable is required")
)

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func getEnv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil && i > 0 {
			return i
		}
	}
	return fallback
}

func getEnvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
