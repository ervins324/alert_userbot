package client

import (
	"bytes"
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

	"alert-userbot/internal/alert"
)

// NeptunClient maintains the WebSocket connection to the NEPTUN Air Defense
// API and feeds the Kyiv alert state machine from "alerts" frames.
type NeptunClient struct {
	url            string
	minBackoff     time.Duration
	maxBackoff     time.Duration
	state          *alert.KyivAlertState
	logger         *slog.Logger
	connected      int32
	totalProcessed int64
	totalAlerts    int64
}

// NewNeptunClient creates a new Neptun WebSocket client.
func NewNeptunClient(
	url string,
	minBackoff, maxBackoff time.Duration,
	state *alert.KyivAlertState,
	logger *slog.Logger,
) *NeptunClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &NeptunClient{
		url:        url,
		minBackoff: minBackoff,
		maxBackoff: maxBackoff,
		state:      state,
		logger:     logger,
	}
}

// IsConnected reports whether the client is currently connected.
func (c *NeptunClient) IsConnected() bool {
	return atomic.LoadInt32(&c.connected) == 1
}

// GetStats returns processing statistics.
func (c *NeptunClient) GetStats() (processed, alerts int64) {
	return atomic.LoadInt64(&c.totalProcessed), atomic.LoadInt64(&c.totalAlerts)
}

// Start runs the WebSocket client loop with auto-reconnection and exponential
// backoff until ctx is canceled.
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
			backoff = c.nextBackoff(backoff)
			continue
		}

		backoff = c.minBackoff
		atomic.StoreInt32(&c.connected, 1)
		c.logger.Info("Successfully connected to Neptun Air Defense API stream")

		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}

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
		backoff = c.nextBackoff(backoff)
	}
}

func (c *NeptunClient) nextBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > c.maxBackoff {
		return c.maxBackoff
	}
	return next
}

func (c *NeptunClient) handleConnection(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	pongWait := 60 * time.Second
	pingInterval := (pongWait * 9) / 10

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	errChan := make(chan error, 2)

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
				c.handleFrame(message)
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

// handleFrame processes a single WebSocket frame. Only "alerts" frames drive
// the Kyiv alert state.
func (c *NeptunClient) handleFrame(frame []byte) {
	atomic.AddInt64(&c.totalProcessed, 1)

	// Fast path: only parse frames that are alert snapshots.
	if !bytes.Contains(frame, []byte(`"type":"alerts"`)) {
		return
	}

	atomic.AddInt64(&c.totalAlerts, 1)
	c.state.SetActive(alert.KyivAlertActive(frame))
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

var ErrDisconnected = errors.New("websocket client disconnected")
