package config

import (
	"os"
	"testing"
)

func TestLoadConfigSuccess(t *testing.T) {
	os.Setenv("TG_API_ID", "12345")
	os.Setenv("TG_API_HASH", "0123456789abcdef0123456789abcdef")
	os.Setenv("DESTINATION_CHAT_ID", "1134564474")
	defer os.Unsetenv("TG_API_ID")
	defer os.Unsetenv("TG_API_HASH")
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
	if cfg.DestinationChatID != "1134564474" {
		t.Errorf("unexpected dest chat %s", cfg.DestinationChatID)
	}
	if cfg.TelegramAPIID != 12345 {
		t.Errorf("unexpected api id %d", cfg.TelegramAPIID)
	}
	if cfg.SessionFile != "session.bin" {
		t.Errorf("unexpected session file %s", cfg.SessionFile)
	}
}

func TestLoadConfigMissingAPIID(t *testing.T) {
	os.Unsetenv("TG_API_ID")
	os.Unsetenv("TG_API_HASH")
	os.Setenv("DESTINATION_CHAT_ID", "123")
	defer os.Unsetenv("DESTINATION_CHAT_ID")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing TG_API_ID")
	}
}

func TestLoadConfigMissingAPIHash(t *testing.T) {
	os.Setenv("TG_API_ID", "12345")
	os.Unsetenv("TG_API_HASH")
	os.Setenv("DESTINATION_CHAT_ID", "123")
	defer os.Unsetenv("TG_API_ID")
	defer os.Unsetenv("DESTINATION_CHAT_ID")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing TG_API_HASH")
	}
}

func TestLoadConfigMissingChatID(t *testing.T) {
	os.Setenv("TG_API_ID", "12345")
	os.Setenv("TG_API_HASH", "hash")
	os.Unsetenv("DESTINATION_CHAT_ID")
	os.Unsetenv("TELEGRAM_CHAT_ID")
	defer os.Unsetenv("TG_API_ID")
	defer os.Unsetenv("TG_API_HASH")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for missing chat id")
	}
}

func TestLoadConfigCustomQueueCapacity(t *testing.T) {
	os.Setenv("TG_API_ID", "12345")
	os.Setenv("TG_API_HASH", "hash")
	os.Setenv("DESTINATION_CHAT_ID", "123")
	os.Setenv("QUEUE_CAPACITY", "500")
	defer func() {
		os.Unsetenv("TG_API_ID")
		os.Unsetenv("TG_API_HASH")
		os.Unsetenv("DESTINATION_CHAT_ID")
		os.Unsetenv("QUEUE_CAPACITY")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.QueueCapacity != 500 {
		t.Errorf("expected queue capacity 500, got %d", cfg.QueueCapacity)
	}
}
