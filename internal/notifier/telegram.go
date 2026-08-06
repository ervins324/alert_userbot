package notifier

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TelegramBot sends messages via the Telegram Bot API. It is used to deliver
// content read by the userbot to the destination chat from a real bot.
type TelegramBot struct {
	token  string
	chatID string
	http   *http.Client
	logger *slog.Logger
}

// NewTelegramBot creates a Bot API sender.
func NewTelegramBot(token, chatID string, timeout time.Duration, logger *slog.Logger) *TelegramBot {
	if logger == nil {
		logger = slog.Default()
	}
	return &TelegramBot{
		token:  token,
		chatID: chatID,
		http:   &http.Client{Timeout: timeout},
		logger: logger,
	}
}

// SendText sends a plain text message to the destination chat.
func (b *TelegramBot) SendText(text string) error {
	form := url.Values{}
	form.Set("chat_id", b.chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	return b.post("/sendMessage", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

// SendPhoto uploads a photo with an optional caption.
func (b *TelegramBot) SendPhoto(data []byte, caption string) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if err := w.WriteField("chat_id", b.chatID); err != nil {
		return err
	}
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return err
		}
	}
	fw, err := w.CreateFormFile("photo", "photo.jpg")
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return b.post("/sendPhoto", &body, w.FormDataContentType())
}

// post performs the API call with a small retry loop.
func (b *TelegramBot) post(method string, body io.Reader, contentType string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s%s", b.token, method)
	safeURL := fmt.Sprintf("https://api.telegram.org/bot***%s", method)

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, body)
		if err != nil {
			return err
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := b.http.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("telegram api %s returned %d: %s", method, resp.StatusCode, sanitize(respBody))
		time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
	}
	return fmt.Errorf("telegram api %s failed after retries: %w", safeURL, lastErr)
}

func sanitize(b []byte) string {
	return strings.TrimSpace(string(b))
}
