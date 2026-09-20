package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// parseChannelSignatures parses CHANNEL_SIGNATURES env var of the form:
// "key1:sig text one,key2:sig text two". The key runs up to the first colon.
func parseChannelSignatures(raw string) map[string]string {
	out := make(map[string]string)
	if raw == "" {
		return out
	}
	// Split on comma, but the value may itself contain commas, so we split
	// on the FIRST colon to get key, then everything after is the value.
	// Multiple entries are newline-separated OR comma-separated at the top level:
	// we use "|" as the entry separator to avoid ambiguity with commas in text.
	// Format: "key1:text one|key2:text two"
	// Fallback: also try comma if no "|" found.
	sep := "|"
	if !strings.Contains(raw, "|") {
		sep = ","
	}
	for _, entry := range strings.Split(raw, sep) {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		idx := strings.Index(entry, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(entry[:idx])
		val := strings.TrimSpace(entry[idx+1:])
		if key != "" && val != "" {
			out[key] = val
		}
	}
	return out
}

// parseAdminUserIDs parses a comma-separated list of int64 Telegram user IDs.
func parseAdminUserIDs(raw string) []int64 {
	if raw == "" {
		return nil
	}
	var ids []int64
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		id, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

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
	SourceChannels       []string
	SourceChannel        string
	SkipPatterns         []string
	ExcludedRegions      []string
	MinReconnectInterval time.Duration
	MaxReconnectInterval time.Duration
	QueueCapacity        int
	HTTPTimeout          time.Duration
	ForceAlert           bool
	// SignaturesFile is the path to the JSON file where runtime signatures are persisted.
	// Defaults to signatures.json next to SessionFile.
	SignaturesFile string
	// ChannelsFile is the path to the JSON file where runtime monitored channels are persisted.
	// Defaults to channels.json next to SessionFile.
	ChannelsFile string
	// ChannelSignatures maps normalized channel keys to their custom signature
	// text. Loaded from CHANNEL_SIGNATURES env var.
	ChannelSignatures map[string]string
	// AdminUserIDs is the optional list of Telegram user IDs allowed to manage
	// signatures via bot commands. If empty, any chat member may do so.
	AdminUserIDs []int64
}

func defaultSignaturesFile(sessionFile string) string {
	dir := filepath.Dir(sessionFile)
	if dir == "" || dir == "." {
		return "signatures.json"
	}
	return filepath.Join(dir, "signatures.json")
}

func defaultChannelsFile(sessionFile string) string {
	dir := filepath.Dir(sessionFile)
	if dir == "" || dir == "." {
		return "channels.json"
	}
	return filepath.Join(dir, "channels.json")
}

// Load loads and validates configuration from environment variables / .env.
func Load() (*Config, error) {
	_ = godotenv.Load() // optional; real env vars take precedence

	sourceRaw := firstNonEmpty(os.Getenv("SOURCE_CHANNELS"), os.Getenv("SOURCE_CHANNEL"))
	if sourceRaw == "" {
		sourceRaw = "mon1tor_ua"
	}
	channels := splitCommaList(sourceRaw)
	firstChannel := ""
	if len(channels) > 0 {
		firstChannel = channels[0]
	}

	sessionFile := getEnv("SESSION_FILE", "session.bin")
	signaturesFile := getEnv("SIGNATURES_FILE", defaultSignaturesFile(sessionFile))
	channelsFile := getEnv("CHANNELS_FILE", defaultChannelsFile(sessionFile))

	cfg := &Config{
		NeptunWSURL:          getEnv("NEPTUN_WS_URL", "wss://neptun.in.ua/api/v1/stream"),
		TelegramBotToken:     strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		TelegramAPIID:        getEnvInt("TG_API_ID", 0),
		TelegramAPIHash:      strings.TrimSpace(os.Getenv("TG_API_HASH")),
		TelegramPhone:        strings.TrimSpace(os.Getenv("TG_PHONE")),
		TelegramPassword:     os.Getenv("TG_PASSWORD"),
		TelegramAuthCode:     strings.TrimSpace(os.Getenv("TG_AUTH_CODE")),
		SessionFile:          sessionFile,
		SignaturesFile:       signaturesFile,
		ChannelsFile:         channelsFile,
		DestinationChatID:    strings.TrimSpace(firstNonEmpty(os.Getenv("DESTINATION_CHAT_ID"), os.Getenv("TELEGRAM_CHAT_ID"))),
		SourceChannels:       channels,
		SourceChannel:        firstChannel,
		SkipPatterns:         getEnvList("SKIP_PATTERNS"),
		ExcludedRegions:      getEnvList("EXCLUDED_REGIONS"),
		MinReconnectInterval: getEnvDuration("MIN_RECONNECT_INTERVAL", 1*time.Second),
		MaxReconnectInterval: getEnvDuration("MAX_RECONNECT_INTERVAL", 30*time.Second),
		QueueCapacity:        getEnvInt("QUEUE_CAPACITY", 1000),
		HTTPTimeout:          getEnvDuration("HTTP_TIMEOUT", 10*time.Second),
		ForceAlert:           getEnvBool("FORCE_ALERT", false),
		ChannelSignatures:    parseChannelSignatures(os.Getenv("CHANNEL_SIGNATURES")),
		AdminUserIDs:         parseAdminUserIDs(os.Getenv("ADMIN_USER_IDS")),
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
	if len(c.SourceChannels) == 0 {
		return fmt.Errorf("SOURCE_CHANNELS (or SOURCE_CHANNEL) cannot be empty")
	}
	for _, ch := range c.SourceChannels {
		if strings.TrimSpace(ch) == "" {
			return fmt.Errorf("SOURCE_CHANNELS contains empty channel name")
		}
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

func getEnvList(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}
	return splitCommaList(raw)
}

func splitCommaList(raw string) []string {
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

