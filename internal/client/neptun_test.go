package client

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"alert-userbot/internal/filter"
	"alert-userbot/internal/notifier"
)

func TestNeptunClientConnectionAndFiltering(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	var connectedClient *websocket.Conn
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connectedClient = conn

		// Send test messages
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"region": "Львівська область", "status": "active"}`))
		time.Sleep(20 * time.Millisecond)
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"region": "м. Київ", "status": "missile alert"}`))
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	kyivFilter := filter.NewKyivFilter()
	telegramNotifier := notifier.NewTelegramNotifier("dummy-token", "12345", 1, 10, 1*time.Second, logger)
	defer telegramNotifier.Close()

	client := NewNeptunClient(wsURL, 10*time.Millisecond, 100*time.Millisecond, kyivFilter, telegramNotifier, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go client.Start(ctx)

	// Wait for processing
	time.Sleep(200 * time.Millisecond)

	processed, matched := client.GetStats()
	if processed < 2 {
		t.Errorf("expected at least 2 processed messages, got %d", processed)
	}
	if matched < 1 {
		t.Errorf("expected at least 1 matched message for Kyiv, got %d", matched)
	}

	if connectedClient != nil {
		_ = connectedClient.Close()
	}
}
