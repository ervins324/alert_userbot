package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigSuccess(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11")
	os.Setenv("DESTINATION_CHAT_ID", "-100123456789")
	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("DESTINATION_CHAT_ID")

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
	if cfg.DestinationChatID != "-100123456789" {
		t.Errorf("unexpected dest chat %s", cfg.DestinationChatID)
	}
}

func TestLoadConfigFallsBackToTelegramChatID(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "token")
	os.Setenv("TELEGRAM_CHAT_ID", "1134564474")
	os.Unsetenv("DESTINATION_CHAT_ID")
	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Unsetenv("TELEGRAM_CHAT_ID")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DestinationChatID != "1134564474" {
		t.Errorf("expected fallback chat id, got %s", cfg.DestinationChatID)
	}
}

func TestLoadConfigMissingToken(t *testing.T) {
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Setenv("DESTINATION_CHAT_ID", "123")
	defer os.Unsetenv("DESTINATION_CHAT_ID")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing token")
	}
}

func TestLoadConfigMissingChatID(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "token")
	os.Unsetenv("DESTINATION_CHAT_ID")
	os.Unsetenv("TELEGRAM_CHAT_ID")
	defer os.Unsetenv("TELEGRAM_BOT_TOKEN")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing chat id")
	}
}

func TestCustomDurationsAndInts(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "token")
	os.Setenv("DESTINATION_CHAT_ID", "123")
	os.Setenv("POLL_INTERVAL", "5s")
	os.Setenv("WORKER_COUNT", "8")
	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("DESTINATION_CHAT_ID")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("WORKER_COUNT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.PollInterval != 5*time.Second {
		t.Errorf("expected poll interval 5s, got %v", cfg.PollInterval)
	}
	if cfg.WorkerCount != 8 {
		t.Errorf("expected worker count 8, got %d", cfg.WorkerCount)
	}
}
