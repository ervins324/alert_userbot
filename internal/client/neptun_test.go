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

	"alert-userbot/internal/alert"
)

func TestNeptunClientDrivesAlertState(t *testing.T) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// No Kyiv alert.
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"alerts","data":{"raions":[{"key":"львівський","name":"Львівський район","oblast":"Львівська область"}]}}`))
		time.Sleep(20 * time.Millisecond)
		// Kyiv city alert starts.
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"alerts","data":{"raions":[{"key":"оболонський","name":"Оболонський район","oblast":"місто Київ"}]}}`))
		time.Sleep(20 * time.Millisecond)
		// Kyiv alert cleared.
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"alerts","data":{"raions":[]}}`))
		time.Sleep(20 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	state := alert.NewKyivAlertState(logger)
	client := NewNeptunClient(wsURL, 10*time.Millisecond, 100*time.Millisecond, state, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go client.Start(ctx)

	waitForState := func(want bool) bool {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if state.IsActive() == want {
				return true
			}
			time.Sleep(10 * time.Millisecond)
		}
		return false
	}

	if !waitForState(true) {
		t.Error("expected alert state to be active after Kyiv frame")
	}
	if !waitForState(false) {
		t.Error("expected alert state to be inactive after clear frame")
	}
}
