package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TelegramNotifier sends low-latency notifications to Telegram.
type TelegramNotifier struct {
	token      string
	chatID     string
	apiURL     string
	httpClient *http.Client
	queue      chan alertTask
	bufferPool sync.Pool
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *slog.Logger
	closed     atomic.Bool
}

type alertTask struct {
	text      string
	parseMode string
	retries   int
}

type telegramRequest struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// NewTelegramNotifier creates a new low-latency Telegram notifier instance.
func NewTelegramNotifier(token, chatID string, workerCount, queueCapacity int, timeout time.Duration, logger *slog.Logger) *TelegramNotifier {
	if logger == nil {
		logger = slog.Default()
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 3 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	ctx, cancel := context.WithCancel(context.Background())

	tn := &TelegramNotifier{
		token:      token,
		chatID:     chatID,
		apiURL:     fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token),
		httpClient: client,
		queue:      make(chan alertTask, queueCapacity),
		bufferPool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}

	// Start worker pool
	for i := 0; i < workerCount; i++ {
		tn.wg.Add(1)
		go tn.worker(i)
	}

	// Pre-warm TCP/TLS connections
	go tn.prewarm()

	return tn
}

// Notify enqueues a notification text immediately without blocking (0ms delay).
func (tn *TelegramNotifier) Notify(text string) bool {
	if tn.closed.Load() {
		return false
	}

	task := alertTask{
		text:      text,
		parseMode: "HTML",
		retries:   0,
	}

	select {
	case tn.queue <- task:
		return true
	default:
		tn.logger.Error("Telegram notification queue full, message dropped to prevent latency degradation",
			slog.String("text_snippet", snippet(text, 50)))
		return false
	}
}

// NotifyRaw constructs and enqueues a notification from raw NEPTUN alert bytes.
func (tn *TelegramNotifier) NotifyRaw(payload []byte) bool {
	// Construct concise HTML message
	text := fmt.Sprintf("🚨 <b>AIR DEFENSE ALERT — KYIV REGION</b> 🚨\n\n<code>%s</code>", escapeHTML(string(payload)))
	return tn.Notify(text)
}

func (tn *TelegramNotifier) worker(id int) {
	defer tn.wg.Done()
	for {
		select {
		case <-tn.ctx.Done():
			// Drain remaining tasks in queue without re-queuing
			for {
				select {
				case task, ok := <-tn.queue:
					if !ok {
						return
					}
					_ = tn.sendHTTP(task)
				default:
					return
				}
			}
		case task, ok := <-tn.queue:
			if !ok {
				return
			}
			if err := tn.sendHTTP(task); err != nil {
				tn.logger.Error("Failed to send Telegram notification", slog.Int("worker_id", id), slog.String("error", err.Error()))
				if task.retries < 2 {
					task.retries++
					select {
					case <-tn.ctx.Done():
					case tn.queue <- task:
					default:
					}
				}
			}
		}
	}
}

func (tn *TelegramNotifier) sendHTTP(task alertTask) error {
	buf := tn.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer tn.bufferPool.Put(buf)

	reqPayload := telegramRequest{
		ChatID:    tn.chatID,
		Text:      task.text,
		ParseMode: task.parseMode,
	}

	if err := json.NewEncoder(buf).Encode(reqPayload); err != nil {
		return fmt.Errorf("json encode error: %w", err)
	}

	req, err := http.NewRequestWithContext(tn.ctx, http.MethodPost, tn.apiURL, buf)
	if err != nil {
		return fmt.Errorf("http request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tn.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do failed: %s", tn.sanitize(err.Error()))
	}
	defer resp.Body.Close()

	// Discard body to enable HTTP keep-alive reuse
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status code %d", resp.StatusCode)
	}

	return nil
}

// prewarm periodically sends lightweight GET requests to ensure TLS connections stay warm.
func (tn *TelegramNotifier) sanitize(msg string) string {
	return strings.ReplaceAll(msg, tn.token, "<REDACTED>")
}

func (tn *TelegramNotifier) prewarm() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	getMeURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", tn.token)

	// Perform initial warm up call
	tn.pingTelegram(getMeURL)

	for {
		select {
		case <-tn.ctx.Done():
			return
		case <-ticker.C:
			tn.pingTelegram(getMeURL)
		}
	}
}

func (tn *TelegramNotifier) pingTelegram(url string) {
	ctx, cancel := context.WithTimeout(tn.ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	resp, err := tn.httpClient.Do(req)
	if err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// Close gracefully stops workers and flushes pending notifications.
func (tn *TelegramNotifier) Close() {
	if !tn.closed.CompareAndSwap(false, true) {
		return
	}
	tn.cancel()
	tn.wg.Wait()
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func escapeHTML(s string) string {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '&':
			buf.WriteString("&amp;")
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}
