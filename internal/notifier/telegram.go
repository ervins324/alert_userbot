package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// TelegramBot sends messages (text and photos) to a destination chat via the
// Telegram Bot API using a non-blocking worker pool.
type TelegramBot struct {
	token      string
	chatID     string
	apiURL     string
	httpClient *http.Client
	queue      chan notifyTask
	bufferPool sync.Pool
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *slog.Logger
	closed     atomic.Bool
}

type notifyKind int

const (
	taskMessage notifyKind = iota
	taskPhoto
)

type notifyTask struct {
	kind     notifyKind
	text     string
	image    []byte
	filename string
	retries  int
}

// NewTelegramBot creates a new Telegram Bot API client.
func NewTelegramBot(token, chatID string, workerCount, queueCapacity int, timeout time.Duration, logger *slog.Logger) *TelegramBot {
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

	tb := &TelegramBot{
		token:      token,
		chatID:     chatID,
		apiURL:     "https://api.telegram.org",
		httpClient: client,
		queue:      make(chan notifyTask, queueCapacity),
		bufferPool: sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}

	for i := 0; i < workerCount; i++ {
		tb.wg.Add(1)
		go tb.worker(i)
	}

	go tb.prewarm()

	return tb
}

// SendMessage enqueues a text message to the destination chat. Returns false
// if the notifier is closed or the queue is full (never blocks).
func (tb *TelegramBot) SendMessage(text string) bool {
	return tb.enqueue(notifyTask{kind: taskMessage, text: text})
}

// SendPhoto enqueues a photo (with optional caption) to the destination chat.
func (tb *TelegramBot) SendPhoto(caption string, image []byte, filename string) bool {
	return tb.enqueue(notifyTask{kind: taskPhoto, text: caption, image: image, filename: filename})
}

func (tb *TelegramBot) enqueue(task notifyTask) bool {
	if tb.closed.Load() {
		return false
	}
	select {
	case tb.queue <- task:
		return true
	default:
		tb.logger.Error("Telegram queue full, message dropped", slog.String("snippet", snippet(task.text, 50)))
		return false
	}
}

func (tb *TelegramBot) worker(id int) {
	defer tb.wg.Done()
	for {
		select {
		case <-tb.ctx.Done():
			for {
				select {
				case task, ok := <-tb.queue:
					if !ok {
						return
					}
					_ = tb.send(task)
				default:
					return
				}
			}
		case task, ok := <-tb.queue:
			if !ok {
				return
			}
			if err := tb.send(task); err != nil {
				tb.logger.Error("Failed to send Telegram message",
					slog.Int("worker_id", id),
					slog.String("err", tb.sanitize(err.Error())))
				if task.retries < 2 {
					task.retries++
					select {
					case <-tb.ctx.Done():
					case tb.queue <- task:
					default:
					}
				}
			}
		}
	}
}

func (tb *TelegramBot) send(task notifyTask) error {
	switch task.kind {
	case taskMessage:
		return tb.sendMessage(task.text)
	case taskPhoto:
		return tb.sendPhoto(task.text, task.image, task.filename)
	}
	return fmt.Errorf("unknown task kind")
}

type sendMessageReq struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

func (tb *TelegramBot) sendMessage(text string) error {
	buf := tb.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer tb.bufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(sendMessageReq{
		ChatID:    tb.chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		return fmt.Errorf("json encode: %w", err)
	}

	req, err := http.NewRequestWithContext(tb.ctx, http.MethodPost,
		fmt.Sprintf("%s/bot%s/sendMessage", tb.apiURL, tb.token), buf)
	if err != nil {
		return fmt.Errorf("request creation: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return tb.do(req)
}

func (tb *TelegramBot) sendPhoto(caption string, image []byte, filename string) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", tb.chatID)
	if caption != "" {
		_ = writer.WriteField("caption", caption)
		_ = writer.WriteField("parse_mode", "HTML")
	}

	part, err := writer.CreateFormFile("photo", filename)
	if err != nil {
		return fmt.Errorf("multipart create: %w", err)
	}
	if _, err := part.Write(image); err != nil {
		return fmt.Errorf("multipart write: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("multipart close: %w", err)
	}

	req, err := http.NewRequestWithContext(tb.ctx, http.MethodPost,
		fmt.Sprintf("%s/bot%s/sendPhoto", tb.apiURL, tb.token), body)
	if err != nil {
		return fmt.Errorf("request creation: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return tb.do(req)
}

func (tb *TelegramBot) do(req *http.Request) error {
	resp, err := tb.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}
	return nil
}

func (tb *TelegramBot) sanitize(msg string) string {
	return strings.ReplaceAll(msg, tb.token, "<REDACTED>")
}

func (tb *TelegramBot) prewarm() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	getMeURL := fmt.Sprintf("%s/bot%s/getMe", tb.apiURL, tb.token)

	for {
		ctx, cancel := context.WithTimeout(tb.ctx, 2*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, getMeURL, nil)
		if err == nil {
			if resp, err := tb.httpClient.Do(req); err == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
		cancel()

		select {
		case <-tb.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// Close stops workers after draining pending tasks.
func (tb *TelegramBot) Close() {
	if !tb.closed.CompareAndSwap(false, true) {
		return
	}
	tb.cancel()
	tb.wg.Wait()
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
