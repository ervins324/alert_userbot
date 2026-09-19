package command

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"alert-userbot/internal/filter"
	"alert-userbot/internal/geomap"
	"alert-userbot/internal/geoparse"
	"alert-userbot/internal/notifier"
)

// Handler processes interactive bot commands such as /map, /setsig, /clearsig,
// and /listsig.
type Handler struct {
	bot          *notifier.TelegramBot
	sigStore     *filter.SignatureStore
	adminUserIDs []int64 // if empty, any user may manage signatures
	logger       *slog.Logger
}

// NewHandler creates a new bot command handler.
func NewHandler(bot *notifier.TelegramBot, sigStore *filter.SignatureStore, adminUserIDs []int64, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		bot:          bot,
		sigStore:     sigStore,
		adminUserIDs: adminUserIDs,
		logger:       logger,
	}
}

// Start runs the long-polling command listener loop until ctx is canceled.
func (h *Handler) Start(ctx context.Context) {
	h.logger.Info("Telegram command listener started (long polling mode)")
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

	cmd, args := parseCommand(text)

	switch cmd {
	case "/map":
		h.handleMap(msg, args)
	case "/setsig":
		h.handleSetSig(msg, args)
	case "/clearsig":
		h.handleClearSig(msg, args)
	case "/listsig":
		h.handleListSig(msg)
	}
}

// ── /map ─────────────────────────────────────────────────────────────────────

func (h *Handler) handleMap(msg *notifier.BotMessage, args string) {
	h.logger.Info("received /map command",
		slog.Int64("chat_id", msg.Chat.ID),
		slog.Int("msg_id", msg.MessageID),
		slog.Bool("has_reply", msg.ReplyToMessage != nil))

	var targetText string
	if args != "" {
		targetText = args
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

	loc := geoparse.ExtractLocation(targetText)
	if loc == nil || (len(loc.MatchedRaions) == 0 && len(loc.Points) == 0) {
		noLocMsg := "⚠️ Не вдалося розпізнати район або орієнтир у Києві.\nСпробуйте уточнити: /map [назва району/масиву] (наприклад: /map Дарницький або /map Борщагівка)"
		_ = h.bot.SendTextReply(msg.Chat.ID, noLocMsg, msg.MessageID)
		return
	}

	imgData, err := geomap.RenderKyivMap(loc)
	if err != nil {
		h.logger.Error("failed to render map", slog.String("err", err.Error()))
		_ = h.bot.SendTextReply(msg.Chat.ID, "❌ Помилка генерації карти", msg.MessageID)
		return
	}

	caption := fmt.Sprintf("📍 %s", loc.Description)
	if err := h.bot.SendPhotoReply(msg.Chat.ID, imgData, caption, msg.MessageID); err != nil {
		h.logger.Error("failed to send map photo reply", slog.String("err", err.Error()))
	} else {
		h.logger.Info("sent map reply", slog.String("location", loc.Description), slog.Int64("chat_id", msg.Chat.ID))
	}
}

// ── /setsig ──────────────────────────────────────────────────────────────────

func (h *Handler) handleSetSig(msg *notifier.BotMessage, args string) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу керувати підписами.", msg.MessageID)
		return
	}

	// args is everything after "/setsig": "<channel> [text]"
	channelKey, sigText, hasText := parseChannelAndRest(args)
	if channelKey == "" {
		_ = h.bot.SendTextReply(msg.Chat.ID,
			"ℹ️ Використання:\n/setsig <канал> <текст підпису>  — встановити підпис\n/setsig <канал>                  — показати поточний підпис",
			msg.MessageID)
		return
	}

	key := filter.NormalizeChannelKey(channelKey)

	if !hasText {
		// Show current signature
		current := h.sigStore.Get(key)
		if current == "" {
			_ = h.bot.SendTextReply(msg.Chat.ID,
				fmt.Sprintf("ℹ️ Підпис для каналу %q не встановлено.", key),
				msg.MessageID)
		} else {
			_ = h.bot.SendTextReply(msg.Chat.ID,
				fmt.Sprintf("📝 Поточний підпис для %q:\n%s", key, current),
				msg.MessageID)
		}
		return
	}

	h.sigStore.Set(key, sigText)
	h.logger.Info("channel signature set",
		slog.String("channel_key", key),
		slog.String("sig", sigText),
		slog.Int64("by_user", msg.From.ID))
	_ = h.bot.SendTextReply(msg.Chat.ID,
		fmt.Sprintf("✅ Підпис для каналу %q встановлено:\n%s", key, sigText),
		msg.MessageID)
}

// ── /clearsig ─────────────────────────────────────────────────────────────────

func (h *Handler) handleClearSig(msg *notifier.BotMessage, args string) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу керувати підписами.", msg.MessageID)
		return
	}

	channelKey := strings.TrimSpace(args)
	if channelKey == "" {
		_ = h.bot.SendTextReply(msg.Chat.ID,
			"ℹ️ Використання: /clearsig <канал>",
			msg.MessageID)
		return
	}

	key := filter.NormalizeChannelKey(channelKey)
	h.sigStore.Clear(key)
	h.logger.Info("channel signature cleared",
		slog.String("channel_key", key),
		slog.Int64("by_user", msg.From.ID))
	_ = h.bot.SendTextReply(msg.Chat.ID,
		fmt.Sprintf("🗑 Підпис для каналу %q видалено.", key),
		msg.MessageID)
}

// ── /listsig ──────────────────────────────────────────────────────────────────

func (h *Handler) handleListSig(msg *notifier.BotMessage) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу керувати підписами.", msg.MessageID)
		return
	}

	sigs := h.sigStore.List()
	if len(sigs) == 0 {
		_ = h.bot.SendTextReply(msg.Chat.ID, "ℹ️ Підписи не встановлені.", msg.MessageID)
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 Поточні підписи каналів:\n")
	for k, v := range sigs {
		sb.WriteString(fmt.Sprintf("\n• %s:\n  %s\n", k, v))
	}
	_ = h.bot.SendTextReply(msg.Chat.ID, sb.String(), msg.MessageID)
}

// ── helpers ──────────────────────────────────────────────────────────────────

// isAdmin returns true when the message sender is allowed to manage signatures.
// If adminUserIDs is empty, everyone is allowed.
func (h *Handler) isAdmin(msg *notifier.BotMessage) bool {
	if len(h.adminUserIDs) == 0 {
		return true
	}
	for _, id := range h.adminUserIDs {
		if msg.From.ID == id {
			return true
		}
	}
	return false
}

// parseCommand splits a bot message text into the command and its arguments.
// The command is lowercased and any "@BotName" suffix is stripped.
func parseCommand(text string) (cmd, args string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}
	raw := strings.ToLower(fields[0])
	// Strip @BotUsername suffix
	if idx := strings.Index(raw, "@"); idx > 0 {
		raw = raw[:idx]
	}
	cmd = raw
	if len(fields) > 1 {
		args = strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return cmd, args
}

// parseChannelAndRest splits "/setsig args" remainder into the channel
// identifier (first token) and the rest of the string (signature text).
// hasText is false when no text follows the channel identifier.
func parseChannelAndRest(args string) (channelKey, rest string, hasText bool) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", "", false
	}
	idx := strings.IndexByte(args, ' ')
	if idx < 0 {
		return args, "", false
	}
	return args[:idx], strings.TrimSpace(args[idx+1:]), true
}
