package command

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"alert-userbot/internal/filter"
	"alert-userbot/internal/geomap"
	"alert-userbot/internal/geoparse"
	"alert-userbot/internal/notifier"
	"alert-userbot/internal/telegram"
)

// ChannelManager defines the operations required for dynamic channel management.
type ChannelManager interface {
	MonitoredChannels() []telegram.ChannelInfo
	AddChannel(ctx context.Context, channelRef string) (*telegram.ChannelInfo, error)
	RemoveChannel(channelRef string) (*telegram.ChannelInfo, error)
}

// Handler processes interactive bot commands such as /map, /channels, /addchannel,
// /removechannel, /setsig, /clearsig, /listsig, and /help.
type Handler struct {
	bot          *notifier.TelegramBot
	sigStore     *filter.SignatureStore
	channels     ChannelManager
	adminUserIDs []int64 // if empty, any user in chat may manage channels and signatures
	logger       *slog.Logger
}

// NewHandler creates a new bot command handler.
func NewHandler(
	bot *notifier.TelegramBot,
	sigStore *filter.SignatureStore,
	channels ChannelManager,
	adminUserIDs []int64,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		bot:          bot,
		sigStore:     sigStore,
		channels:     channels,
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
				h.handleMessage(ctx, u.Message)
			}
		}
	}
}

func (h *Handler) handleMessage(ctx context.Context, msg *notifier.BotMessage) {
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
	case "/channels", "/listchannels":
		h.handleListChannels(msg)
	case "/addchannel":
		h.handleAddChannel(ctx, msg, args)
	case "/removechannel", "/delchannel":
		h.handleRemoveChannel(msg, args)
	case "/setsig":
		h.handleSetSig(msg, args)
	case "/clearsig":
		h.handleClearSig(msg, args)
	case "/listsig":
		h.handleListSig(msg)
	case "/help", "/start":
		h.handleHelp(msg)
	}
}

// ── /help ────────────────────────────────────────────────────────────────────

func (h *Handler) handleHelp(msg *notifier.BotMessage) {
	var sb strings.Builder
	sb.WriteString("🤖 <b>Команди бота моніторингу:</b>\n\n")
	sb.WriteString("🗺 <b>Карта:</b>\n")
	sb.WriteString("• <code>/map [район/локація]</code> — згенерувати карту загрози або відповісти на повідомлення\n\n")
	sb.WriteString("📢 <b>Керування каналами:</b>\n")
	sb.WriteString("• <code>/channels</code> — переглянути активні канали моніторингу\n")
	sb.WriteString("• <code>/addchannel &lt;канал&gt;</code> — додати канал (@username або -100... ID)\n")
	sb.WriteString("• <code>/removechannel &lt;канал&gt;</code> — видалити канал з моніторингу\n\n")
	sb.WriteString("✍️ <b>Керування підписами:</b>\n")
	sb.WriteString("• <code>/setsig &lt;текст&gt;</code> — встановити загальний підпис для всіх каналів\n")
	sb.WriteString("• <code>/setsig &lt;канал&gt; &lt;текст&gt;</code> — встановити підпис для конкретного каналу\n")
	sb.WriteString("• <code>/clearsig [канал]</code> — видалити підпис\n")
	sb.WriteString("• <code>/listsig</code> — переглянути всі підписи\n")

	_ = h.bot.SendTextReply(msg.Chat.ID, sb.String(), msg.MessageID)
}

// ── /channels ────────────────────────────────────────────────────────────────

func (h *Handler) handleListChannels(msg *notifier.BotMessage) {
	channels := h.getChannels()
	if len(channels) == 0 {
		_ = h.bot.SendTextReply(msg.Chat.ID, "ℹ️ Наразі немає активних каналів моніторингу.\nДодайте канал: <code>/addchannel @username</code>", msg.MessageID)
		return
	}

	var sb strings.Builder
	sb.WriteString("📢 <b>Активні канали моніторингу:</b>\n\n")
	for i, ch := range channels {
		name := ch.Title
		if name == "" {
			name = ch.ConfigName
		}
		idStr := ""
		if ch.ID != 0 {
			idStr = fmt.Sprintf(" <code>-100%d</code>", ch.ID)
		}
		unameStr := ""
		if ch.Username != "" {
			unameStr = fmt.Sprintf(" (@%s)", html.EscapeString(ch.Username))
		}

		sb.WriteString(fmt.Sprintf("%d. <b>%s</b>%s%s\n", i+1, html.EscapeString(name), unameStr, idStr))
	}

	sb.WriteString("\n💡 <i>Додати:</i> <code>/addchannel &lt;канал&gt;</code>\n<i>Видалити:</i> <code>/removechannel &lt;канал&gt;</code>")
	_ = h.bot.SendTextReply(msg.Chat.ID, sb.String(), msg.MessageID)
}

// ── /addchannel ──────────────────────────────────────────────────────────────

func (h *Handler) handleAddChannel(ctx context.Context, msg *notifier.BotMessage, args string) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу додавати канали.", msg.MessageID)
		return
	}

	raw := strings.TrimSpace(args)
	if raw == "" {
		hint := "ℹ️ <b>Використання:</b> <code>/addchannel &lt;канал&gt;</code>\n\nПриклади:\n• <code>/addchannel @kyiv_monitor1</code>\n• <code>/addchannel -1001929743622</code>\n• <code>/addchannel https://t.me/mon1tor_ua</code>"
		_ = h.bot.SendTextReply(msg.Chat.ID, hint, msg.MessageID)
		return
	}

	_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("⏳ Підключаю та перевіряю канал %s...", html.EscapeString(raw)), msg.MessageID)

	info, err := h.channels.AddChannel(ctx, raw)
	if err != nil {
		h.logger.Error("failed to add channel", slog.String("raw", raw), slog.String("err", err.Error()))
		_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("❌ Не вдалося додати канал %s:\n%s", html.EscapeString(raw), html.EscapeString(err.Error())), msg.MessageID)
		return
	}

	title := info.Title
	if title == "" {
		title = raw
	}
	uname := ""
	if info.Username != "" {
		uname = fmt.Sprintf(" (@%s)", html.EscapeString(info.Username))
	}
	idStr := ""
	if info.ID != 0 {
		idStr = fmt.Sprintf(" (ID: <code>-100%d</code>)", info.ID)
	}

	reply := fmt.Sprintf("✅ Канал <b>%s</b>%s%s успішно додано до моніторингу!\n\n(💾 Збережено на диску — моніторинг активний і переживе перезапуск)", html.EscapeString(title), uname, idStr)
	_ = h.bot.SendTextReply(msg.Chat.ID, reply, msg.MessageID)
}

// ── /removechannel ───────────────────────────────────────────────────────────

func (h *Handler) handleRemoveChannel(msg *notifier.BotMessage, args string) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу видаляти канали.", msg.MessageID)
		return
	}

	raw := strings.TrimSpace(args)
	if raw == "" {
		hint := "ℹ️ <b>Використання:</b> <code>/removechannel &lt;канал&gt;</code>\nВкажіть назву, @username або ID зі списку <code>/channels</code>."
		_ = h.bot.SendTextReply(msg.Chat.ID, hint, msg.MessageID)
		return
	}

	info, err := h.channels.RemoveChannel(raw)
	if err != nil {
		h.logger.Error("failed to remove channel", slog.String("raw", raw), slog.String("err", err.Error()))
		_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("❌ Помилка: %s", err.Error()), msg.MessageID)
		return
	}

	title := info.Title
	if title == "" {
		title = raw
	}
	idStr := ""
	if info.ID != 0 {
		idStr = fmt.Sprintf(" (ID: -100%d)", info.ID)
	}

	reply := fmt.Sprintf("🗑 Канал <b>%s</b>%s успішно видалено з моніторингу.\n\n(💾 Оновлений список збережено на диску)", html.EscapeString(title), idStr)
	_ = h.bot.SendTextReply(msg.Chat.ID, reply, msg.MessageID)
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

	args = strings.TrimSpace(args)
	if args == "" {
		var sb strings.Builder
		sb.WriteString("ℹ️ <b>Використання команди /setsig:</b>\n\n")
		sb.WriteString("• <code>/setsig &lt;текст&gt;</code> — встановити загальний підпис для всіх каналів\n")
		sb.WriteString("• <code>/setsig default &lt;текст&gt;</code> — встановити загальний підпис\n")
		sb.WriteString("• <code>/setsig &lt;канал&gt; &lt;текст&gt;</code> — встановити підпис для конкретного каналу\n")
		sb.WriteString("• <code>/clearsig [канал]</code> — видалити підпис\n")
		sb.WriteString("• <code>/listsig</code> — переглянути активні підписи\n")

		channels := h.getChannels()
		if len(channels) > 0 {
			sb.WriteString("\n📢 <b>Канали моніторингу:</b>\n")
			for _, ch := range channels {
				name := ch.Title
				if name == "" {
					name = ch.ConfigName
				}
				idStr := ""
				if ch.ID != 0 {
					idStr = fmt.Sprintf(" (ID: -100%d)", ch.ID)
				}
				unameStr := ""
				if ch.Username != "" {
					unameStr = fmt.Sprintf(" @%s", html.EscapeString(ch.Username))
				}
				sb.WriteString(fmt.Sprintf("• %s%s%s\n", html.EscapeString(name), unameStr, idStr))
			}
		}

		_ = h.bot.SendTextReply(msg.Chat.ID, sb.String(), msg.MessageID)
		return
	}

	firstWord, rest := splitFirstWord(args)
	firstLower := strings.ToLower(firstWord)

	// 1. Explicit global default keyword
	if firstLower == "default" || firstLower == "all" || firstLower == "*" || firstLower == "загальний" {
		if rest == "" {
			cur := h.sigStore.Get("default")
			if cur == "" {
				_ = h.bot.SendTextReply(msg.Chat.ID, "ℹ️ Загальний підпис наразі не встановлено.", msg.MessageID)
			} else {
				_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("📝 Поточний загальний підпис:\n%s", html.EscapeString(cur)), msg.MessageID)
			}
			return
		}
		_ = h.sigStore.Set("default", rest)
		h.logger.Info("default signature set", slog.String("sig", rest), slog.Int64("by_user", msg.From.ID))
		_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("✅ Встановлено загальний підпис для всіх повідомлень:\n%s", html.EscapeString(rest)), msg.MessageID)
		return
	}

	// 2. Check if firstWord matches one of our monitored channels
	channels := h.getChannels()
	var matchedChannel *telegram.ChannelInfo
	for i := range channels {
		ch := &channels[i]
		if channelMatches(firstWord, ch) {
			matchedChannel = ch
			break
		}
	}

	if matchedChannel != nil {
		targetKey := channelKey(matchedChannel)
		if rest == "" {
			cur := h.sigStore.Get(targetKey)
			if cur == "" {
				cur = h.sigStore.Get("default")
				if cur != "" {
					_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("ℹ️ Спеціальний підпис не встановлено. Використовується загальний:\n%s", html.EscapeString(cur)), msg.MessageID)
					return
				}
				_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("ℹ️ Підпис для каналу %q не встановлено.", targetKey), msg.MessageID)
			} else {
				_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("📝 Поточний підпис для %q:\n%s", targetKey, html.EscapeString(cur)), msg.MessageID)
			}
			return
		}

		_ = h.sigStore.Set(targetKey, rest)
		h.logger.Info("channel signature set",
			slog.String("channel_key", targetKey),
			slog.String("sig", rest),
			slog.Int64("by_user", msg.From.ID))
		_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("✅ Підпис для каналу %q встановлено:\n%s", targetKey, html.EscapeString(rest)), msg.MessageID)
		return
	}

	// 3. Check if firstWord looks like an explicit channel username or numeric ID
	if looksLikeChannelIdentifier(firstWord) && rest != "" {
		targetKey := filter.NormalizeChannelKey(firstWord)
		_ = h.sigStore.Set(targetKey, rest)
		h.logger.Info("custom signature set for identifier",
			slog.String("key", targetKey),
			slog.String("sig", rest),
			slog.Int64("by_user", msg.From.ID))
		_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("✅ Підпис для %q встановлено:\n%s", targetKey, html.EscapeString(rest)), msg.MessageID)
		return
	}

	// 4. Otherwise, the user entered text directly without specifying a channel name!
	_ = h.sigStore.Set("default", args)
	if len(channels) == 1 {
		_ = h.sigStore.Set(channelKey(&channels[0]), args)
	}

	h.logger.Info("signature set as default", slog.String("sig", args), slog.Int64("by_user", msg.From.ID))
	_ = h.bot.SendTextReply(msg.Chat.ID,
		fmt.Sprintf("✅ Встановлено загальний підпис для всіх повідомлень:\n%s\n\n(💾 Збережено на диску)", html.EscapeString(args)),
		msg.MessageID)
}

// ── /clearsig ─────────────────────────────────────────────────────────────────

func (h *Handler) handleClearSig(msg *notifier.BotMessage, args string) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу керувати підписами.", msg.MessageID)
		return
	}

	target := strings.TrimSpace(args)
	if target == "" || strings.ToLower(target) == "default" {
		_ = h.sigStore.Clear("default")
		channels := h.getChannels()
		if len(channels) == 1 {
			_ = h.sigStore.Clear(channelKey(&channels[0]))
		}
		_ = h.bot.SendTextReply(msg.Chat.ID, "🗑 Загальний підпис видалено.", msg.MessageID)
		return
	}

	if strings.ToLower(target) == "all" {
		all := h.sigStore.List()
		for k := range all {
			_ = h.sigStore.Clear(k)
		}
		_ = h.bot.SendTextReply(msg.Chat.ID, "🗑 Усі підписи видалено.", msg.MessageID)
		return
	}

	channels := h.getChannels()
	var keyToClear string
	for _, ch := range channels {
		if channelMatches(target, &ch) {
			keyToClear = channelKey(&ch)
			break
		}
	}
	if keyToClear == "" {
		keyToClear = filter.NormalizeChannelKey(target)
	}

	_ = h.sigStore.Clear(keyToClear)
	h.logger.Info("channel signature cleared", slog.String("channel_key", keyToClear), slog.Int64("by_user", msg.From.ID))
	_ = h.bot.SendTextReply(msg.Chat.ID, fmt.Sprintf("🗑 Підпис для %q видалено.", keyToClear), msg.MessageID)
}

// ── /listsig ──────────────────────────────────────────────────────────────────

func (h *Handler) handleListSig(msg *notifier.BotMessage) {
	if !h.isAdmin(msg) {
		_ = h.bot.SendTextReply(msg.Chat.ID, "⛔ У вас немає дозволу керувати підписами.", msg.MessageID)
		return
	}

	sigs := h.sigStore.List()
	channels := h.getChannels()

	var sb strings.Builder
	sb.WriteString("📋 <b>Налаштування підписів каналів:</b>\n\n")

	// 1. Default signature
	defSig := h.sigStore.Get("default")
	if defSig != "" {
		sb.WriteString(fmt.Sprintf("⭐ <b>Загальний підпис (за замовчуванням):</b>\n%s\n\n", html.EscapeString(defSig)))
	} else {
		sb.WriteString("⭐ <b>Загальний підпис:</b> (не встановлено)\n\n")
	}

	// 2. Monitored channels
	if len(channels) > 0 {
		sb.WriteString("📢 <b>Канали моніторингу:</b>\n")
		for _, ch := range channels {
			name := ch.Title
			if name == "" {
				name = ch.ConfigName
			}
			idStr := ""
			if ch.ID != 0 {
				idStr = fmt.Sprintf(" [-100%d]", ch.ID)
			}
			unameStr := ""
			if ch.Username != "" {
				unameStr = fmt.Sprintf(" (@%s)", html.EscapeString(ch.Username))
			}

			sig, matchedKey := h.sigStore.GetForChannel(
				ch.ConfigName,
				fmt.Sprintf("-100%d", ch.ID),
				fmt.Sprintf("%d", ch.ID),
				ch.Username,
				ch.Title,
			)

			if sig != "" {
				if matchedKey == "default" {
					sb.WriteString(fmt.Sprintf("• <b>%s</b>%s%s:\n  ↳ <i>[Використовується загальний підпис]</i>\n", html.EscapeString(name), unameStr, idStr))
				} else {
					sb.WriteString(fmt.Sprintf("• <b>%s</b>%s%s:\n  ↳ <i>%s</i>\n", html.EscapeString(name), unameStr, idStr, html.EscapeString(sig)))
				}
			} else {
				sb.WriteString(fmt.Sprintf("• <b>%s</b>%s%s:\n  ↳ (немає підпису)\n", html.EscapeString(name), unameStr, idStr))
			}
		}
		sb.WriteString("\n")
	}

	// 3. Any additional explicit keys in store
	extraCount := 0
	for k, v := range sigs {
		if k == "default" || k == "all" || k == "*" {
			continue
		}
		isChannelKey := false
		for _, ch := range channels {
			if channelMatches(k, &ch) {
				isChannelKey = true
				break
			}
		}
		if !isChannelKey {
			if extraCount == 0 {
				sb.WriteString("🔖 <b>Інші збережені підписи:</b>\n")
			}
			sb.WriteString(fmt.Sprintf("• %s:\n  %s\n", html.EscapeString(k), html.EscapeString(v)))
			extraCount++
		}
	}

	sb.WriteString("💡 <i>Змінити:</i> <code>/setsig &lt;текст&gt;</code> або <code>/setsig &lt;канал&gt; &lt;текст&gt;</code>")
	_ = h.bot.SendTextReply(msg.Chat.ID, sb.String(), msg.MessageID)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func (h *Handler) getChannels() []telegram.ChannelInfo {
	if h.channels == nil {
		return nil
	}
	return h.channels.MonitoredChannels()
}

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

func parseCommand(text string) (cmd, args string) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", ""
	}
	raw := strings.ToLower(fields[0])
	if idx := strings.Index(raw, "@"); idx > 0 {
		raw = raw[:idx]
	}
	cmd = raw
	if len(fields) > 1 {
		args = strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return cmd, args
}

func splitFirstWord(s string) (first, rest string) {
	s = strings.TrimSpace(s)
	idx := strings.IndexByte(s, ' ')
	if idx < 0 {
		return s, ""
	}
	return s[:idx], strings.TrimSpace(s[idx+1:])
}

func channelMatches(query string, ch *telegram.ChannelInfo) bool {
	norm := filter.NormalizeChannelKey(query)
	if norm == "" {
		return false
	}
	if norm == filter.NormalizeChannelKey(ch.Username) && ch.Username != "" {
		return true
	}
	if norm == filter.NormalizeChannelKey(ch.ConfigName) && ch.ConfigName != "" {
		return true
	}
	if ch.ID != 0 {
		if norm == fmt.Sprintf("-100%d", ch.ID) || norm == fmt.Sprintf("%d", ch.ID) {
			return true
		}
	}
	if ch.Title != "" && strings.EqualFold(strings.TrimSpace(query), strings.TrimSpace(ch.Title)) {
		return true
	}
	return false
}

func channelKey(ch *telegram.ChannelInfo) string {
	if ch.Username != "" {
		return filter.NormalizeChannelKey(ch.Username)
	}
	if ch.ID != 0 {
		return fmt.Sprintf("-100%d", ch.ID)
	}
	return filter.NormalizeChannelKey(ch.ConfigName)
}

func looksLikeChannelIdentifier(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "@") || strings.HasPrefix(s, "-100") || strings.HasPrefix(s, "t.me/") {
		return true
	}
	if len(s) > 3 {
		allDigits := true
		for i := 0; i < len(s); i++ {
			if s[i] < '0' || s[i] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}
	return false
}
