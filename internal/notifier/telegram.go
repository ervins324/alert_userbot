package notifier

import (
	"bytes"
	"context"
	"encoding/json"
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

// Update represents a Telegram Bot API update.
type Update struct {
	UpdateID int         `json:"update_id"`
	Message  *BotMessage `json:"message"`
}

// BotMessage represents a message received from Telegram Bot API.
type BotMessage struct {
	MessageID      int         `json:"message_id"`
	Chat           struct {
		ID int64 `json:"id"`
	} `json:"chat"`
	Text           string      `json:"text"`
	Caption        string      `json:"caption"`
	ReplyToMessage *BotMessage `json:"reply_to_message"`
}

type getUpdatesResponse struct {
	Ok     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// SendText sends a plain text message to the destination chat.
func (b *TelegramBot) SendText(text string) error {
	form := url.Values{}
	form.Set("chat_id", b.chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	return b.post("/sendMessage", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

// SendTextReply sends a text reply to a specific message in a chat.
func (b *TelegramBot) SendTextReply(chatID int64, text string, replyToMsgID int) error {
	form := url.Values{}
	targetChat := b.chatID
	if chatID != 0 {
		targetChat = fmt.Sprintf("%d", chatID)
	}
	form.Set("chat_id", targetChat)
	form.Set("text", text)
	if replyToMsgID > 0 {
		form.Set("reply_to_message_id", fmt.Sprintf("%d", replyToMsgID))
	}
	form.Set("disable_web_page_preview", "true")
	return b.post("/sendMessage", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
}

// SendPhoto uploads a photo with an optional caption.
func (b *TelegramBot) SendPhoto(data []byte, caption string) error {
	return b.SendPhotoReply(0, data, caption, 0)
}

// SendPhotoReply uploads a photo with an optional caption and reply-to message ID.
func (b *TelegramBot) SendPhotoReply(chatID int64, data []byte, caption string, replyToMsgID int) error {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	targetChat := b.chatID
	if chatID != 0 {
		targetChat = fmt.Sprintf("%d", chatID)
	}
	if err := w.WriteField("chat_id", targetChat); err != nil {
		return err
	}
	if caption != "" {
		if err := w.WriteField("caption", caption); err != nil {
			return err
		}
	}
	if replyToMsgID > 0 {
		if err := w.WriteField("reply_to_message_id", fmt.Sprintf("%d", replyToMsgID)); err != nil {
			return err
		}
	}
	fw, err := w.CreateFormFile("photo", "map.png")
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

// GetUpdates fetches new updates from Telegram using long polling.
func (b *TelegramBot) GetUpdates(ctx context.Context, offset int, timeoutSec int) ([]Update, error) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=%d&allowed_updates=[\"message\"]",
		b.token, offset, timeoutSec)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	pollClient := &http.Client{
		Timeout: time.Duration(timeoutSec+15) * time.Second,
	}

	resp, err := pollClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("getUpdates returned %d: %s", resp.StatusCode, sanitize(body))
	}

	var res getUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Result, nil
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
