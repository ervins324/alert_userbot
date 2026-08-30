package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"alert-userbot/internal/geomap"
	"alert-userbot/internal/geoparse"
	"alert-userbot/internal/notifier"
)

// Handler processes interactive bot commands such as /map.
type Handler struct {
	bot    *notifier.TelegramBot
	logger *slog.Logger
}

// NewHandler creates a new bot command handler.
func NewHandler(bot *notifier.TelegramBot, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		bot:    bot,
		logger: logger,
	}
}

// Start runs the long-polling command listener loop until ctx is canceled.
func (h *Handler) Start(ctx context.Context) {
	h.logger.Info("Telegram /map command listener started (long polling mode)")
	offset := 0

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("Telegram command listener stopped")
			return
		default:
		}

		updates, err := h.bot.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.logger.Debug("getUpdates error (will retry)", slog.String("err", err.Error()))
			time.Sleep(2 * time.Second)
			continue
		}

		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Message != nil {
				h.handleMessage(u.Message)
			}
		}
	}
}

func (h *Handler) handleMessage(msg *notifier.BotMessage) {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		text = strings.TrimSpace(msg.Caption)
	}
	if text == "" {
		return
	}

	// Check if message is a /map command
	if !isMapCommand(text) {
		return
	}

	h.logger.Info("received /map command",
		slog.Int64("chat_id", msg.Chat.ID),
		slog.Int("msg_id", msg.MessageID),
		slog.Bool("has_reply", msg.ReplyToMessage != nil))

	// 1. Determine target text to parse
	var targetText string
	cmdArgs := extractCommandArgs(text)

	if cmdArgs != "" {
		targetText = cmdArgs
	} else if msg.ReplyToMessage != nil {
		targetText = msg.ReplyToMessage.Text
		if targetText == "" {
			targetText = msg.ReplyToMessage.Caption
		}
	}

	if targetText == "" {
		hint := "📍 Щоб отримати карту, надішліть /map у відповідь на повідомлення або напишіть:\n/map [район/локація] (наприклад: /map Позняки або /map Оболонь)"
		_ = h.bot.SendTextReply(msg.Chat.ID, hint, msg.MessageID)
		return
	}

	// 2. Extract geographic location
	loc := geoparse.ExtractLocation(targetText)
	if loc == nil || (len(loc.MatchedRaions) == 0 && len(loc.Points) == 0) {
		noLocMsg := "⚠️ Не вдалося розпізнати район або орієнтир у Києві.\nСпробуйте уточнити: /map [назва району/масиву] (наприклад: /map Дарницький або /map Борщагівка)"
		_ = h.bot.SendTextReply(msg.Chat.ID, noLocMsg, msg.MessageID)
		return
	}

	// 3. Render map image
	imgData, err := geomap.RenderKyivMap(loc)
	if err != nil {
		h.logger.Error("failed to render map", slog.String("err", err.Error()))
		_ = h.bot.SendTextReply(msg.Chat.ID, "❌ Помилка генерації карти", msg.MessageID)
		return
	}

	// 4. Send photo reply
	caption := fmt.Sprintf("📍 %s", loc.Description)
	if err := h.bot.SendPhotoReply(msg.Chat.ID, imgData, caption, msg.MessageID); err != nil {
		h.logger.Error("failed to send map photo reply", slog.String("err", err.Error()))
	} else {
		h.logger.Info("sent map reply", slog.String("location", loc.Description), slog.Int64("chat_id", msg.Chat.ID))
	}
}

func isMapCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	cmd := strings.ToLower(fields[0])
	return cmd == "/map" || strings.HasPrefix(cmd, "/map@")
}

func extractCommandArgs(text string) string {
	fields := strings.Fields(text)
	if len(fields) <= 1 {
		return ""
	}
	return strings.TrimSpace(strings.Join(fields[1:], " "))
}
