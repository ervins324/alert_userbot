package notifier

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendMessage(t *testing.T) {
	var received int32
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTEST/sendMessage" {
			w.WriteHeader(http.StatusOK)
			return
		}
		atomic.AddInt32(&received, 1)
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot := NewTelegramBot("TEST", "1134564474", 2, 10, 2*time.Second, logger)
	bot.apiURL = server.URL

	if !bot.SendMessage("Hello <b>world</b>") {
		t.Fatal("SendMessage returned false")
	}

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&received) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	bot.Close()

	if atomic.LoadInt32(&received) != 1 {
		t.Fatalf("expected 1 request, got %d", received)
	}
	if body["chat_id"] != "1134564474" || body["parse_mode"] != "HTML" {
		t.Errorf("unexpected body: %+v", body)
	}
	if body["text"] != "Hello <b>world</b>" {
		t.Errorf("unexpected text: %v", body["text"])
	}
}

func TestSendPhoto(t *testing.T) {
	var received int32
	var contentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTEST/sendPhoto" {
			w.WriteHeader(http.StatusOK)
			return
		}
		atomic.AddInt32(&received, 1)
		contentType = r.Header.Get("Content-Type")
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot := NewTelegramBot("TEST", "1134564474", 2, 10, 2*time.Second, logger)
	bot.apiURL = server.URL

	if !bot.SendPhoto("caption", []byte("fakeimage"), "photo.jpg") {
		t.Fatal("SendPhoto returned false")
	}

	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&received) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	bot.Close()

	if atomic.LoadInt32(&received) != 1 {
		t.Fatalf("expected 1 request, got %d", received)
	}
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Errorf("expected multipart content type, got %q", contentType)
	}
}

func TestQueueFullDrops(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot := NewTelegramBot("TEST", "1134564474", 0, 1, 2*time.Second, logger)
	defer bot.Close()

	if !bot.SendMessage("first") {
		t.Error("first enqueue should succeed")
	}
	if bot.SendMessage("second") {
		t.Error("second enqueue should fail (queue full)")
	}
}

func TestTokenSanitized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bot := NewTelegramBot("SECRETTOKEN", "1", 0, 1, time.Second, logger)
	defer bot.Close()

	msg := bot.sanitize("https://api.telegram.org/botSECRETTOKEN/sendMessage timeout")
	if strings.Contains(msg, "SECRETTOKEN") {
		t.Errorf("token leaked: %s", msg)
	}
}
