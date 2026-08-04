package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigSuccess(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
	os.Setenv("TELEGRAM_CHAT_ID", "-100123456789")
	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.NeptunWSURL != "wss://neptun.in.ua/api/v1/stream" {
		t.Errorf("expected default neptun url, got %s", cfg.NeptunWSURL)
	}
	if cfg.TelegramBotToken != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Errorf("expected bot token, got %s", cfg.TelegramBotToken)
	}
	if cfg.TelegramChatID != "-100123456789" {
		t.Errorf("expected chat id, got %s", cfg.TelegramChatID)
	}
}

func TestLoadConfigMissingToken(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Setenv("TELEGRAM_CHAT_ID", "-100123456789")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing token, got nil")
	}
}

func TestLoadConfigInvalidURL(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "123456:ABC")
	os.Setenv("TELEGRAM_CHAT_ID", "123")
	os.Setenv("NEPTUN_WS_URL", "http://invalid-scheme.com")
	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")
	defer os.Unsetenv("NEPTUN_WS_URL")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for invalid scheme, got nil")
	}
}

func TestCustomDurationsAndInts(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "token")
	os.Setenv("TELEGRAM_CHAT_ID", "chat")
	os.Setenv("MIN_RECONNECT_INTERVAL", "500ms")
	os.Setenv("MAX_RECONNECT_INTERVAL", "5s")
	os.Setenv("WORKER_COUNT", "8")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("TELEGRAM_CHAT_ID")
		os.Unsetenv("MIN_RECONNECT_INTERVAL")
		os.Unsetenv("MAX_RECONNECT_INTERVAL")
		os.Unsetenv("WORKER_COUNT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MinReconnectInterval != 500*time.Millisecond {
		t.Errorf("expected 500ms, got %v", cfg.MinReconnectInterval)
	}
	if cfg.MaxReconnectInterval != 5*time.Second {
		t.Errorf("expected 5s, got %v", cfg.MaxReconnectInterval)
	}
	if cfg.WorkerCount != 8 {
		t.Errorf("expected worker count 8, got %d", cfg.WorkerCount)
	}
}
