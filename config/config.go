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
	TelegramAPIID        int
	TelegramAPIHash      string
	TelegramPhone        string
	TelegramPassword     string
	TelegramAuthCode     string
	SessionFile          string
	DestinationChatID    string
	SourceChannel        string
	SkipPatterns         []string
	ExcludedRegions      []string
	MinReconnectInterval time.Duration
	MaxReconnectInterval time.Duration
	QueueCapacity        int
	HTTPTimeout          time.Duration
	ForceAlert           bool
}

// Load loads and validates configuration from environment variables / .env.
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; real env vars take precedence

	cfg := &Config{
		NeptunWSURL:          getEnv("NEPTUN_WS_URL", "wss://neptun.in.ua/api/v1/stream"),
		TelegramBotToken:     strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		TelegramAPIID:        getEnvInt("TG_API_ID", 0),
		TelegramAPIHash:      strings.TrimSpace(os.Getenv("TG_API_HASH")),
		TelegramPhone:        strings.TrimSpace(os.Getenv("TG_PHONE")),
		TelegramPassword:     os.Getenv("TG_PASSWORD"),
		TelegramAuthCode:     strings.TrimSpace(os.Getenv("TG_AUTH_CODE")),
		SessionFile:          getEnv("SESSION_FILE", "session.bin"),
		DestinationChatID:    strings.TrimSpace(firstNonEmpty(os.Getenv("DESTINATION_CHAT_ID"), os.Getenv("TELEGRAM_CHAT_ID"))),
		SourceChannel:        getEnv("SOURCE_CHANNEL", "mon1tor_ua"),
		SkipPatterns:         getEnvList("SKIP_PATTERNS"),
		ExcludedRegions:      getEnvList("EXCLUDED_REGIONS"),
		MinReconnectInterval: getEnvDuration("MIN_RECONNECT_INTERVAL", 1*time.Second),
		MaxReconnectInterval: getEnvDuration("MAX_RECONNECT_INTERVAL", 30*time.Second),
		QueueCapacity:        getEnvInt("QUEUE_CAPACITY", 1000),
		HTTPTimeout:          getEnvDuration("HTTP_TIMEOUT", 10*time.Second),
		ForceAlert:           getEnvBool("FORCE_ALERT", false),
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
	if c.TelegramAPIID <= 0 {
		return fmt.Errorf("TG_API_ID is required (get it at my.telegram.org)")
	}
	if c.TelegramAPIHash == "" {
		return fmt.Errorf("TG_API_HASH is required (get it at my.telegram.org)")
	}
	if c.DestinationChatID == "" {
		return ErrMissingChatID
	}
	if c.TelegramBotToken == "" {
		return ErrMissingTelegramToken
	}
	if strings.TrimSpace(c.SourceChannel) == "" {
		return fmt.Errorf("SOURCE_CHANNEL cannot be empty")
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
	return nil
}

var (
	ErrMissingNeptunURL    = errors.New("NEPTUN_WS_URL cannot be empty")
	ErrMissingChatID       = errors.New("DESTINATION_CHAT_ID (or TELEGRAM_CHAT_ID) environment variable is required")
	ErrMissingTelegramToken = errors.New("TELEGRAM_BOT_TOKEN environment variable is required")
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

func getEnvBool(key string, fallback bool) bool {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		switch strings.ToLower(val) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func getEnvList(key string) []string {	raw := strings.TrimSpace(os.Getenv(key))
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
