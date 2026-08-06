package config

import (
	"os"
	"testing"
)

const testToken = "123456:ABC"

func setTestEnv(t *testing.T, extra map[string]string) {
	t.Helper()
	base := map[string]string{
		"TG_API_ID":          "12345",
		"TG_API_HASH":        "0123456789abcdef0123456789abcdef",
		"TELEGRAM_BOT_TOKEN": testToken,
	}
	for k, v := range base {
		os.Setenv(k, v)
	}
	for k, v := range extra {
		os.Setenv(k, v)
	}
	os.Unsetenv("TG_PHONE")
	os.Unsetenv("TG_PASSWORD")
	os.Unsetenv("TG_AUTH_CODE")
	os.Unsetenv("SESSION_FILE")
}

func cleanupEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TG_API_ID", "TG_API_HASH", "TELEGRAM_BOT_TOKEN", "DESTINATION_CHAT_ID",
		"TELEGRAM_CHAT_ID", "QUEUE_CAPACITY", "TG_PHONE", "TG_PASSWORD",
		"TG_AUTH_CODE", "SESSION_FILE",
	} {
		os.Unsetenv(k)
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	setTestEnv(t, map[string]string{"DESTINATION_CHAT_ID": "1134564474"})
	defer cleanupEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.NeptunWSURL != "wss://neptun.in.ua/api/v1/stream" {
		t.Errorf("unexpected neptun url %s", cfg.NeptunWSURL)
	}
	if cfg.SourceChannel != "mon1tor_ua" {
		t.Errorf("unexpected source channel %s", cfg.SourceChannel)
	}
	if cfg.DestinationChatID != "1134564474" {
		t.Errorf("unexpected dest chat %s", cfg.DestinationChatID)
	}
	if cfg.TelegramAPIID != 12345 {
		t.Errorf("unexpected api id %d", cfg.TelegramAPIID)
	}
	if cfg.TelegramBotToken != testToken {
		t.Errorf("unexpected bot token %s", cfg.TelegramBotToken)
	}
	if cfg.SessionFile != "session.bin" {
		t.Errorf("unexpected session file %s", cfg.SessionFile)
	}
}

func TestLoadConfigFallsBackToTelegramChatID(t *testing.T) {
	setTestEnv(t, map[string]string{"TELEGRAM_CHAT_ID": "1134564474"})
	defer cleanupEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DestinationChatID != "1134564474" {
		t.Errorf("expected fallback chat id, got %s", cfg.DestinationChatID)
	}
}

func TestLoadConfigMissingAPIID(t *testing.T) {
	setTestEnv(t, map[string]string{"DESTINATION_CHAT_ID": "123"})
	os.Unsetenv("TG_API_ID")
	defer cleanupEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing TG_API_ID")
	}
}

func TestLoadConfigMissingAPIHash(t *testing.T) {
	setTestEnv(t, map[string]string{"DESTINATION_CHAT_ID": "123"})
	os.Unsetenv("TG_API_HASH")
	defer cleanupEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing TG_API_HASH")
	}
}

func TestLoadConfigMissingToken(t *testing.T) {
	setTestEnv(t, map[string]string{"DESTINATION_CHAT_ID": "123"})
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer cleanupEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing bot token")
	}
}

func TestLoadConfigMissingChatID(t *testing.T) {
	setTestEnv(t, nil)
	os.Unsetenv("DESTINATION_CHAT_ID")
	os.Unsetenv("TELEGRAM_CHAT_ID")
	defer cleanupEnv(t)

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing chat id")
	}
}

func TestLoadConfigCustomQueueCapacity(t *testing.T) {
	setTestEnv(t, map[string]string{"DESTINATION_CHAT_ID": "123", "QUEUE_CAPACITY": "500"})
	defer cleanupEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.QueueCapacity != 500 {
		t.Errorf("expected queue capacity 500, got %d", cfg.QueueCapacity)
	}
}
