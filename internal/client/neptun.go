package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"alert-userbot/internal/filter"
	"alert-userbot/internal/notifier"
)

// NeptunClient manages the WebSocket connection to the NEPTUN Air Defense API.
type NeptunClient struct {
	url            string
	minBackoff     time.Duration
	maxBackoff     time.Duration
	filter         *filter.KyivFilter
	notifier       *notifier.TelegramNotifier
	logger         *slog.Logger
	connected      int32 // atomic boolean (1 for connected, 0 for disconnected)
	totalProcessed int64
	totalMatched   int64
}

// NewNeptunClient creates a new Neptun WebSocket client instance.
func NewNeptunClient(
	url string,
	minBackoff, maxBackoff time.Duration,
	kyivFilter *filter.KyivFilter,
	telegramNotifier *notifier.TelegramNotifier,
	logger *slog.Logger,
) *NeptunClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &NeptunClient{
		url:        url,
		minBackoff: minBackoff,
		maxBackoff: maxBackoff,
		filter:     kyivFilter,
		notifier:   telegramNotifier,
		logger:     logger,
	}
}

// IsConnected returns true if the client is currently connected to the WebSocket stream.
func (c *NeptunClient) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// GetStats returns processing statistics.
func (c *NeptunClient) GetStats() (processed, matched int64) {
	return atomic.LoadInt64(&c.totalProcessed), atomic.LoadInt64(&c.totalMatched)
}

// Start runs the WebSocket client loop with auto-reconnection and exponential backoff.
func (c *NeptunClient) Start(ctx context.Context) {
	backoff := c.minBackoff

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		NetDialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Neptun WebSocket client stopped by context cancellation")
			return
		default:
		}

		c.logger.Info("Connecting to Neptun Air Defense API stream", slog.String("url", c.url))

		conn, resp, err := dialer.DialContext(ctx, c.url, nil)
		if err != nil {
			statusCode := 0
			if resp != nil {
				statusCode = resp.StatusCode
			}
			c.logger.Error("Failed to connect to Neptun WebSocket",
				slog.String("error", err.Error()),
				slog.Int("status_code", statusCode),
				slog.Duration("retry_in", backoff))

			if !c.sleepContext(ctx, backoff) {
				return
			}
			backoff *= 2
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
			continue
		}

		// Reset backoff on successful connection
		backoff = c.minBackoff
		atomic.StoreInt32(&c.connected, 1)
		c.logger.Info("Successfully connected to Neptun Air Defense API stream")

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

		// Run read and ping loop
		connErr := c.handleConnection(ctx, conn)
		atomic.StoreInt32(&c.connected, 0)

		if ctx.Err() != nil {
			c.logger.Info("Neptun client disconnected due to context cancellation")
			return
		}

		c.logger.Warn("Neptun WebSocket connection lost",
			slog.String("error", formatError(connErr)),
			slog.Duration("reconnecting_in", backoff))

		if !c.sleepContext(ctx, backoff) {
			return
		}
		backoff *= 2
		if backoff > c.maxBackoff {
			backoff = c.maxBackoff
		}
	}
}

func (c *NeptunClient) handleConnection(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	// Configure ping/pong handlers for liveness
	pongWait := 60 * time.Second
	pingInterval := (pongWait * 9) / 10

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Setup ping ticker
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	// Error channel for background reader/writer coordination
	errChan := make(chan error, 2)

	// Background writer for ping messages
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pingTicker.C:
				_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					errChan <- fmt.Errorf("ping write error: %w", err)
					return
				}
			}
		}
	}()

	// Reader loop (hot path for threat detection)
	go func() {
		for {
			msgType, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					errChan <- fmt.Errorf("unexpected websocket close: %w", err)
				} else {
					errChan <- err
				}
				return
			}

			if msgType == websocket.TextMessage || msgType == websocket.BinaryMessage {
				atomic.AddInt64(&c.totalProcessed, 1)

				// Blazing fast zero-allocation filter check (< 1ms target)
				if c.filter.IsKyivTarget(message) {
					atomic.AddInt64(&c.totalMatched, 1)
					c.logger.Info("🚨 Target alert matched for Kyiv / Kyiv Oblast!",
						slog.String("payload_snippet", truncate(string(message), 120)))

					// Dispatch non-blocking to Telegram notifier
					c.notifier.NotifyRaw(message)
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}

func (c *NeptunClient) sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func formatError(err error) string {
	if err == nil {
		return "unknown"
	}
	return err.Error()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

var ErrDisconnected = errors.New("websocket client disconnected")
