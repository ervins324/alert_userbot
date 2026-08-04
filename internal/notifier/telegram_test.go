package notifier

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestTelegramNotifierSend(t *testing.T) {
	var receivedCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&receivedCount, 1)

		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		var req map[string]interface{}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to parse json: %v", err)
		}

		if req["chat_id"] != "12345" {
			t.Errorf("expected chat_id 12345, got %v", req["chat_id"])
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tn := NewTelegramNotifier("dummy-token", "12345", 2, 10, 2*time.Second, logger)
	tn.apiURL = server.URL // Override API URL to test server

	// Test non-blocking enqueue
	ok := tn.Notify("Test alert message")
	if !ok {
		t.Fatalf("expected Notify to return true")
	}

	// Give worker time to process task
	time.Sleep(100 * time.Millisecond)
	tn.Close()

	if atomic.LoadInt32(&receivedCount) != 1 {
		t.Errorf("expected 1 received request, got %d", receivedCount)
	}
}

func TestTelegramNotifierQueueFull(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	// 0 workers, capacity 1 to test queue saturation
	tn := NewTelegramNotifier("dummy-token", "12345", 0, 1, 2*time.Second, logger)
	defer tn.Close()

	first := tn.Notify("Msg 1")
	if !first {
		t.Errorf("expected first notify to succeed")
	}

	second := tn.Notify("Msg 2")
	if second {
		t.Errorf("expected second notify to fail (queue full)")
	}
}

func TestEscapeHTML(t *testing.T) {
	input := "Alert <test> & dangerous"
	expected := "Alert &lt;test&gt; &amp; dangerous"
	got := escapeHTML(input)
	if got != expected {
		t.Errorf("escapeHTML(%q) = %q; want %q", input, got, expected)
	}
}
