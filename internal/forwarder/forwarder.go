package forwarder

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"alert-userbot/internal/alert"
	"alert-userbot/internal/notifier"
	"alert-userbot/internal/scraper"
)

// Forwarder connects the channel scraper, the Kyiv alert state and the
// Telegram bot: messages are forwarded to the destination chat only while a
// Kyiv city air alert is active.
type Forwarder struct {
	state    *alert.KyivAlertState
	bot      *notifier.TelegramBot
	client   *http.Client
	logger   *slog.Logger
	forwarded int64
	skipped   int64
}

// New creates a Forwarder.
func New(state *alert.KyivAlertState, bot *notifier.TelegramBot, timeout time.Duration, logger *slog.Logger) *Forwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Forwarder{
		state:  state,
		bot:    bot,
		logger: logger,
		client: &http.Client{Timeout: timeout},
	}
}

// Stats returns counts of forwarded and skipped messages.
func (f *Forwarder) Stats() (forwarded, skipped int64) {
	return f.forwarded, f.skipped
}

// Run consumes scraped messages and forwards them while the alert is active.
func (f *Forwarder) Run(ctx context.Context, in <-chan scraper.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-in:
			if !ok {
				return
			}
			f.handle(ctx, msg)
		}
	}
}

// Forward processes a single scraped message: it is sent to the destination
// chat only while a Kyiv city air alert is active.
func (f *Forwarder) Forward(ctx context.Context, msg scraper.Message) {
	f.handle(ctx, msg)
}

func (f *Forwarder) handle(ctx context.Context, msg scraper.Message) {
	if !f.state.IsActive() {
		f.skipped++
		f.logger.Debug("message skipped (no active alert)",
			slog.Int("msg_id", msg.ID))
		return
	}

	if msg.PhotoURL != "" {
		img, err := f.downloadPhoto(ctx, msg.PhotoURL)
		if err != nil {
			f.logger.Error("failed to download channel photo",
				slog.Int("msg_id", msg.ID), slog.String("err", err.Error()))
			return
		}
		if !f.bot.SendPhoto(msg.Text, img, fmt.Sprintf("photo_%d.jpg", msg.ID)) {
			return
		}
	} else {
		if !f.bot.SendMessage(msg.Text) {
			return
		}
	}

	f.forwarded++
	f.logger.Info("forwarded channel message",
		slog.Int("msg_id", msg.ID), slog.String("snippet", snippet(msg.Text, 60)))
}

func (f *Forwarder) downloadPhoto(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("photo download status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
