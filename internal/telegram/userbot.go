package telegram

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"alert-userbot/internal/alert"
	"alert-userbot/internal/filter"
	"alert-userbot/internal/notifier"
)

// Mode selects the runtime behavior of the userbot.
type Mode int

const (
	ModeDaemon Mode = iota
	ModeTestNotify
	ModeTestForward
	ModeTestAlert
)

type channelMsgKey struct {
	channelID int64
	msgID     int
}

type forwardTask struct {
	channelID    int64
	msgID        int
	text         string
	photo        *tg.InputPhotoFileLocation
	fwdChannelID int64
	fwdPost      int
}

// UserBot is a Telegram MTProto client (logged in as a real user) that reads
// new posts from public channels while a Kyiv city air alert is active and
// delivers them to a destination chat through a bot.
type UserBot struct {
	appID          int
	appHash        string
	phone          string
	password       string
	authCode       string
	sessionFile    string
	sourceChannels []string

	state     *alert.KyivAlertState
	filter    *filter.TextFilter
	geoFilter *filter.GeoFilter
	bot       *notifier.TelegramBot
	sigStore  *filter.SignatureStore
	logger    *slog.Logger

	forceAlert    bool
	sessionExists bool // set by checkSession, used by ensureAuth for diagnostics

	client *telegram.Client
	api    *tg.Client

	sourceChannelIDs map[int64]struct{}
	channelsByID     map[int64]*tg.Channel
	// channelKeys maps a resolved channel's numeric ID to its normalized key
	// (lowercased username or raw numeric string) used for signature lookups.
	channelKeys map[int64]string
	primaryPeer tg.InputPeerClass

	queue chan forwardTask
	seen  map[channelMsgKey]struct{}

	forwarded atomic.Int64
	skipped   atomic.Int64
	filtered  atomic.Int64

	wg sync.WaitGroup
}

// NewUserBot creates a new MTProto userbot that feeds messages to the bot.
func NewUserBot(
	appID int,
	appHash, phone, password, authCode, sessionFile string,
	sourceChannels []string,
	state *alert.KyivAlertState,
	textFilter *filter.TextFilter,
	geoFilter *filter.GeoFilter,
	bot *notifier.TelegramBot,
	sigStore *filter.SignatureStore,
	queueCapacity int,
	forceAlert bool,
	logger *slog.Logger,
) *UserBot {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserBot{
		appID:            appID,
		appHash:          appHash,
		phone:            phone,
		password:         password,
		authCode:         authCode,
		sessionFile:      sessionFile,
		sourceChannels:   sourceChannels,
		state:            state,
		filter:           textFilter,
		geoFilter:        geoFilter,
		bot:              bot,
		sigStore:         sigStore,
		logger:           logger,
		forceAlert:       forceAlert,
		sourceChannelIDs: make(map[int64]struct{}),
		channelsByID:     make(map[int64]*tg.Channel),
		channelKeys:      make(map[int64]string),
		queue:            make(chan forwardTask, queueCapacity),
		seen:             make(map[channelMsgKey]struct{}),
	}
}

// Stats returns counters for forwarded, skipped and filtered messages.
func (u *UserBot) Stats() (forwarded, skipped, filtered int64) {
	return u.forwarded.Load(), u.skipped.Load(), u.filtered.Load()
}

// Run connects, authenticates and runs the userbot until ctx is canceled.
func (u *UserBot) Run(ctx context.Context, mode Mode) error {
	if err := u.checkSession(); err != nil {
		return err
	}

	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		return u.handleMessage(update.Message)
	})
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		return u.handleMessage(update.Message)
	})

	u.client = telegram.NewClient(u.appID, u.appHash, telegram.Options{
		SessionStorage: &session.FileStorage{Path: u.sessionFile},
		UpdateHandler:  dispatcher,
		Device:         telegram.DeviceTDesktopWindows(),
	})
	u.api = u.client.API()

	return u.client.Run(ctx, func(ctx context.Context) error {
		if err := u.ensureAuth(ctx); err != nil {
			return fmt.Errorf("telegram auth: %w", err)
		}

		self, err := u.client.Self(ctx)
		if err != nil {
			return fmt.Errorf("telegram self: %w", err)
		}
		u.logger.Info("telegram user authenticated", slog.Int64("user_id", self.ID))

		if err := u.resolveSources(ctx); err != nil {
			return err
		}

		switch mode {
		case ModeTestNotify:
			return u.testNotify()
		case ModeTestForward:
			return u.testForward(ctx)
		case ModeTestAlert:
			return u.testAlert(ctx)
		default:
			return u.runDaemon(ctx)
		}
	})
}

func (u *UserBot) ensureAuth(ctx context.Context) error {
	status, err := u.client.Auth().Status(ctx)
	if err != nil {
		return err
	}
	if status.Authorized {
		u.logger.Info("session restored from file (already authorized)",
			slog.String("session_file", u.sessionFile))
		return nil
	}

	// Session is NOT authorized — either first time or session expired.
	if u.sessionExists {
		u.logger.Warn("session file exists but is not authorized — "+
			"session may have expired or been revoked by Telegram",
			slog.String("session_file", u.sessionFile))
	}

	if u.phone == "" {
		if u.sessionExists {
			return fmt.Errorf("session file %q exists but the session is expired; "+
				"set TG_PHONE in .env to re-authenticate", u.sessionFile)
		}
		return fmt.Errorf("TG_PHONE is required for first-time login")
	}

	var authenticator auth.UserAuthenticator
	if u.password != "" {
		authenticator = auth.Constant(u.phone, u.password, auth.CodeAuthenticatorFunc(u.askCode))
	} else {
		authenticator = auth.CodeOnly(u.phone, auth.CodeAuthenticatorFunc(u.askCode))
	}

	u.logger.Info("performing Telegram login", slog.String("phone", u.phone))
	return auth.NewFlow(authenticator, auth.SendCodeOptions{}).Run(ctx, u.client.Auth())
}

// checkSession verifies whether the session file exists and validates that
// the required credentials are available for first-time login.
func (u *UserBot) checkSession() error {
	info, err := os.Stat(u.sessionFile)
	if os.IsNotExist(err) {
		u.sessionExists = false
		u.logger.Warn("session file not found — first-time login required",
			slog.String("session_file", u.sessionFile))
		if u.phone == "" {
			return fmt.Errorf("session file %q does not exist and TG_PHONE is not set; "+
				"cannot perform first-time login — set TG_PHONE in .env", u.sessionFile)
		}
		u.logger.Info("TG_PHONE is set, will proceed with interactive authentication")
		return nil
	}
	if err != nil {
		return fmt.Errorf("cannot access session file %q: %w", u.sessionFile, err)
	}
	u.sessionExists = true
	u.logger.Info("existing session file found",
		slog.String("session_file", u.sessionFile),
		slog.Int64("size_bytes", info.Size()))
	return nil
}

// askCode returns the login code from TG_AUTH_CODE env or prompts on stdin.
func (u *UserBot) askCode(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	if u.authCode != "" {
		return u.authCode, nil
	}
	fmt.Print("Enter Telegram login code: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (u *UserBot) resolveSources(ctx context.Context) error {
	for i, chName := range u.sourceChannels {
		ch, err := u.resolveSourceSingle(ctx, chName)
		if err != nil {
			return fmt.Errorf("resolve source channel %q: %w", chName, err)
		}
		u.sourceChannelIDs[ch.ID] = struct{}{}
		u.channelsByID[ch.ID] = ch
		// Build normalized key for signature store lookups.
		u.channelKeys[ch.ID] = filter.NormalizeChannelKey(chName)
		if i == 0 {
			u.primaryPeer = ch.AsInputPeer()
		}
		u.logger.Info("source channel resolved",
			slog.String("channel", chName),
			slog.Int64("id", ch.ID),
			slog.String("title", ch.Title),
			slog.String("sig_key", u.channelKeys[ch.ID]))
	}
	return nil
}

func (u *UserBot) resolveSourceSingle(ctx context.Context, source string) (*tg.Channel, error) {
	clean := strings.TrimSpace(source)
	clean = strings.TrimPrefix(clean, "https://")
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "t.me/")
	clean = strings.TrimPrefix(clean, "telegram.me/")
	clean = strings.TrimPrefix(clean, "@")

	// Private channel: SOURCE_CHANNELS is a numeric -100-prefixed ID.
	if id, ok := parseChannelID(clean); ok {
		ch, err := u.findChannelInDialogs(ctx, id)
		if err != nil {
			return nil, err
		}
		return ch, nil
	}

	res, err := u.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: clean,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve username %q: %w", clean, err)
	}
	for _, c := range res.Chats {
		if ch, ok := c.(*tg.Channel); ok {
			// Ensure membership so we receive channel updates.
			if !ch.Left && !ch.Megagroup {
				if _, err := u.api.ChannelsJoinChannel(ctx, ch.AsInput()); err != nil {
					u.logger.Warn("could not join source channel (may already be a member)",
						slog.String("channel", source), slog.String("err", err.Error()))
				}
			}
			return ch, nil
		}
	}
	return nil, fmt.Errorf("source %q is not a channel", source)
}

// parseChannelID parses a Telegram chat ID. Channel IDs come as -100<id>.
func parseChannelID(s string) (int64, bool) {
	if strings.HasPrefix(s, "-100") && len(s) > 4 {
		id, err := strconv.ParseInt(s[4:], 10, 64)
		if err != nil {
			return 0, false
		}
		return id, true
	}
	return 0, false
}

// findChannelInDialogs pages the account dialogs looking for a channel by ID.
func (u *UserBot) findChannelInDialogs(ctx context.Context, channelID int64) (*tg.Channel, error) {
	req := &tg.MessagesGetDialogsRequest{
		ExcludePinned: true,
		OffsetDate:    0,
		OffsetID:      0,
		OffsetPeer:    &tg.InputPeerEmpty{},
		Limit:         100,
		Hash:          0,
	}

	for page := 0; page < 50; page++ {
		res, err := u.api.MessagesGetDialogs(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("get dialogs: %w", err)
		}

		switch d := res.(type) {
		case *tg.MessagesDialogs:
			if ch := scanChannels(d.Chats, channelID); ch != nil {
				return ch, nil
			}
			return nil, fmt.Errorf("channel %d not found in dialogs", channelID)
		case *tg.MessagesDialogsSlice:
			if ch := scanChannels(d.Chats, channelID); ch != nil {
				return ch, nil
			}
			if len(d.Messages) == 0 {
				return nil, fmt.Errorf("channel %d not found in dialogs", channelID)
			}
			last := d.Messages[len(d.Messages)-1]
			if lm, ok := last.(*tg.Message); ok {
				req.OffsetDate = lm.GetDate()
				req.OffsetID = lm.GetID()
			}
		default:
			return nil, fmt.Errorf("unexpected dialogs response %T", res)
		}
	}
	return nil, fmt.Errorf("channel %d not found in dialogs (too many pages)", channelID)
}

func scanChannels(chats []tg.ChatClass, channelID int64) *tg.Channel {
	for _, c := range chats {
		if ch, ok := c.(*tg.Channel); ok && ch.ID == channelID {
			return ch
		}
	}
	return nil
}

// handleMessage processes a channel update and queues matching messages.
func (u *UserBot) handleMessage(msgClass tg.MessageClass) error {
	m, ok := msgClass.(*tg.Message)
	if !ok {
		return nil // ignore service messages
	}

	ch, ok := m.GetPeerID().(*tg.PeerChannel)
	if !ok {
		return nil
	}
	if _, found := u.sourceChannelIDs[ch.ChannelID]; !found {
		return nil
	}

	msgID := m.GetID()
	key := channelMsgKey{channelID: ch.ChannelID, msgID: msgID}
	if _, dup := u.seen[key]; dup {
		return nil
	}
	u.seen[key] = struct{}{}

	select {
	case u.queue <- forwardTask{
		channelID:    ch.ChannelID,
		msgID:        msgID,
		text:         m.GetMessage(),
		photo:        photoLocation(m),
		fwdChannelID: fwdChannelID(m),
		fwdPost:      fwdChannelPost(m),
	}:
	default:
		u.logger.Error("forward queue full, message dropped",
			slog.Int64("channel_id", ch.ChannelID),
			slog.Int("msg_id", msgID))
	}
	return nil
}

// fwdChannelID returns the ID of the channel a message was forwarded from.
func fwdChannelID(m *tg.Message) int64 {
	if pc, ok := m.FwdFrom.FromID.(*tg.PeerChannel); ok {
		return pc.ChannelID
	}
	return 0
}

// fwdChannelPost returns the original message ID in the forwarded-from channel.
func fwdChannelPost(m *tg.Message) int {
	if _, ok := m.FwdFrom.FromID.(*tg.PeerChannel); ok {
		return m.FwdFrom.ChannelPost
	}
	return 0
}

// photoLocation extracts the original photo location from a message, if any.
// The thumb size must be a real PhotoSize type, otherwise Telegram rejects the
// download with FILE_REFERENCE_EXPIRED.
func photoLocation(m *tg.Message) *tg.InputPhotoFileLocation {
	if media, ok := m.GetMedia(); ok {
		if mp, ok := media.(*tg.MessageMediaPhoto); ok {
			if pc, ok := mp.GetPhoto(); ok {
				if p, ok := pc.AsNotEmpty(); ok {
					return p.AsInputPhotoFileLocation(bestPhotoSizeType(p))
				}
			}
		}
	}
	return nil
}

// bestPhotoSizeType returns the type of the largest photo size available.
func bestPhotoSizeType(p *tg.Photo) string {
	bestType := ""
	bestArea := 0
	for _, s := range p.Sizes {
		switch ps := s.(type) {
		case *tg.PhotoSize:
			if a := ps.W * ps.H; a > bestArea {
				bestArea = a
				bestType = ps.Type
			}
		case *tg.PhotoSizeProgressive:
			if a := ps.W * ps.H; a > bestArea {
				bestArea = a
				bestType = ps.Type
			}
		}
	}
	return bestType
}

func (u *UserBot) runDaemon(ctx context.Context) error {
	u.wg.Add(1)
	go u.worker(ctx)

	u.logger.Info("userbot ready: forwarding channel posts while Kyiv alert is active",
		slog.Any("sources", u.sourceChannels),
		slog.Bool("force_alert", u.forceAlert))

	<-ctx.Done()
	u.wg.Wait()
	u.logger.Info("userbot stopped")
	return nil
}

func (u *UserBot) worker(ctx context.Context) {
	defer u.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-u.queue:
			if !ok {
				return
			}
			u.process(ctx, task)
		}
	}
}

func (u *UserBot) process(ctx context.Context, task forwardTask) {
	text := task.text
	if cleaned, removed := filter.RemoveSignature(text); removed {
		u.logger.Debug("channel signature removed", slog.Int("msg_id", task.msgID))
		text = cleaned
	}

	// Nothing left to send (signature-only post without media).
	if text == "" && task.photo == nil {
		u.skipped.Add(1)
		u.logger.Debug("message skipped (empty after signature removal)", slog.Int("msg_id", task.msgID))
		return
	}

	if u.filter != nil && u.filter.ShouldSkip(text) {
		u.filtered.Add(1)
		u.logger.Info("message filtered out", slog.Int("msg_id", task.msgID))
		return
	}
	if u.geoFilter != nil && u.geoFilter.ShouldSkip(text) {
		u.filtered.Add(1)
		u.logger.Info("message filtered by geography", slog.Int("msg_id", task.msgID))
		return
	}
	if !u.state.IsActive() && !u.forceAlert {
		u.skipped.Add(1)
		u.logger.Debug("message skipped (no active alert)", slog.Int("msg_id", task.msgID))
		return
	}

	// Append custom per-channel signature if one is configured.
	if u.sigStore != nil {
		if key, ok := u.channelKeys[task.channelID]; ok {
			if sig := u.sigStore.Get(key); sig != "" {
				if text != "" {
					text = text + "\n\n" + sig
				} else {
					text = sig
				}
				u.logger.Debug("custom signature appended",
					slog.Int("msg_id", task.msgID),
					slog.String("channel_key", key))
			}
		}
	}

	var err error
	if task.photo != nil {
		u.logger.Info("downloading photo",
			slog.Int("msg_id", task.msgID),
			slog.String("loc", describePhoto(task.photo)),
			slog.Int64("fwd_channel", task.fwdChannelID),
			slog.Int("fwd_post", task.fwdPost))
		var data []byte
		data, err = u.downloadPhoto(ctx, task)
		if err != nil {
			u.logger.Error("photo download failed", slog.Int("msg_id", task.msgID), slog.String("err", err.Error()))
			return
		}
		err = u.bot.SendPhoto(data, text)
	} else {
		err = u.bot.SendText(text)
	}
	if err != nil {
		u.logger.Error("send failed", slog.Int("msg_id", task.msgID), slog.String("err", err.Error()))
		return
	}
	u.forwarded.Add(1)
	u.logger.Info("sent channel message", slog.Int("msg_id", task.msgID))
}

// downloadPhoto downloads a photo, retrying once with a refreshed file
// reference if the original expired. Forwarded photos are re-fetched from
// their original channel.
func (u *UserBot) downloadPhoto(ctx context.Context, task forwardTask) ([]byte, error) {
	loc := task.photo
	for attempt := 0; attempt < 2; attempt++ {
		var buf bytes.Buffer
		_, err := u.client.Download(loc).Stream(ctx, &buf)
		if err == nil {
			u.logger.Debug("photo downloaded", slog.Int("msg_id", task.msgID), slog.Int("bytes", buf.Len()))
			return buf.Bytes(), nil
		}
		u.logger.Error("photo download attempt failed",
			slog.Int("attempt", attempt), slog.Int("msg_id", task.msgID),
			slog.String("err", err.Error()), slog.String("loc", describePhoto(loc)))
		if attempt == 0 && isFileReferenceError(err) {
			u.logger.Warn("file reference expired, refreshing", slog.Int("msg_id", task.msgID))
			fresh, ferr := u.refreshPhotoLocation(ctx, task)
			if ferr != nil {
				return nil, fmt.Errorf("refresh file reference: %w", ferr)
			}
			u.logger.Info("fresh photo reference obtained", slog.Int("msg_id", task.msgID), slog.String("loc", describePhoto(fresh)))
			loc = fresh
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("photo download failed")
}

// describePhoto returns a compact description of a photo location for logs.
func describePhoto(loc *tg.InputPhotoFileLocation) string {
	if loc == nil {
		return "nil"
	}
	return fmt.Sprintf("id=%d access_hash=%d ref_len=%d thumb=%q", loc.ID, loc.AccessHash, len(loc.FileReference), loc.ThumbSize)
}

func isFileReferenceError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FILE_REFERENCE")
}

// refreshPhotoLocation obtains a valid file reference, preferring the
// original channel for forwarded messages.
func (u *UserBot) refreshPhotoLocation(ctx context.Context, task forwardTask) (*tg.InputPhotoFileLocation, error) {
	if task.fwdChannelID != 0 && task.fwdPost != 0 {
		loc, err := u.photoFromOriginalChannel(ctx, task.fwdChannelID, task.fwdPost)
		if err != nil {
			u.logger.Warn("original channel lookup failed, falling back to source channel",
				slog.Int64("channel_id", task.fwdChannelID), slog.Int("post_id", task.fwdPost), slog.String("err", err.Error()))
		} else {
			return loc, nil
		}
	}
	chObj, ok := u.channelsByID[task.channelID]
	if !ok {
		return nil, fmt.Errorf("source channel %d not found", task.channelID)
	}
	return u.photoFromChannel(ctx, chObj.AsInput(), task.msgID)
}

// photoFromOriginalChannel fetches the original forwarded post from the
// channel it was originally published in.
func (u *UserBot) photoFromOriginalChannel(ctx context.Context, channelID int64, postID int) (*tg.InputPhotoFileLocation, error) {
	ch, err := u.findChannelInDialogs(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("find original channel %d: %w", channelID, err)
	}
	return u.photoFromChannel(ctx, ch.AsInput(), postID)
}

// photoFromChannel fetches a message from a channel and extracts its photo.
func (u *UserBot) photoFromChannel(ctx context.Context, channel tg.InputChannelClass, msgID int) (*tg.InputPhotoFileLocation, error) {
	res, err := u.api.ChannelsGetMessages(ctx, &tg.ChannelsGetMessagesRequest{
		Channel: channel,
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: msgID}},
	})
	if err != nil {
		return nil, err
	}
	if msgs, ok := res.(*tg.MessagesChannelMessages); ok {
		for _, m := range msgs.Messages {
			if mm, ok := m.(*tg.Message); ok && mm.GetID() == msgID {
				if loc := photoLocation(mm); loc != nil {
					return loc, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("message %d not found when refreshing file reference", msgID)
}

func (u *UserBot) testAlert(ctx context.Context) error {
	// Simulate a real NEPTUN alerts frame where the city of Kyiv is delivered
	// through data.oblasts (key "м. київ"), exactly as the live feed does.
	frame := []byte(`{"type":"alerts","data":{"raions":[{"key":"бахмутський","name":"Бахмутський район","oblast":"Донецька область","since":"2026-08-05T11:44:53Z"}],"oblasts":[{"key":"м. київ","name":"м. Київ","oblast":"","since":"2026-08-08T00:03:00Z"}]}}`)
	active := alert.KyivAlertActive(frame)
	u.state.SetActive(active)
	if !active {
		return fmt.Errorf("KyivAlertActive did NOT detect the simulated alert frame (oblasts not parsed?)")
	}
	u.logger.Info("simulated Kyiv alert frame detected via data.oblasts (key \"м. київ\")")
	return u.testForward(ctx)
}

func (u *UserBot) testNotify() error {
	msg := "🔔 Test notification— Neptun forwarder is working. If you receive this, your bot + chat are configured correctly."
	if err := u.bot.SendText(msg); err != nil {
		return fmt.Errorf("send test message: %w", err)
	}
	u.logger.Info("test notification sent. Check your Telegram chat.")
	return nil
}

func (u *UserBot) testForward(ctx context.Context) error {
	// Simulate an active alert and send the latest source-channel post.
	u.state.SetActive(true)

	if u.primaryPeer == nil {
		return fmt.Errorf("no source channels available for test forward")
	}

	res, err := u.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     u.primaryPeer,
		Limit:    5,
		MaxID:    0,
		MinID:    0,
		OffsetID: 0,
	})
	if err != nil {
		return fmt.Errorf("get channel history: %w", err)
	}

	var latest *tg.Message
	switch msgs := res.(type) {
	case *tg.MessagesChannelMessages:
		for _, m := range msgs.Messages {
			if mm, ok := m.(*tg.Message); ok {
				latest = mm
				break
			}
		}
	case *tg.MessagesMessages:
		for _, m := range msgs.Messages {
			if mm, ok := m.(*tg.Message); ok {
				latest = mm
				break
			}
		}
	default:
		return fmt.Errorf("unexpected history response %T", res)
	}
	if latest == nil {
		return fmt.Errorf("no messages found in source channels")
	}

	var channelID int64
	if ch, ok := latest.GetPeerID().(*tg.PeerChannel); ok {
		channelID = ch.ChannelID
	}

	u.logger.Info("test forward: sending latest channel message (alert simulated as active)", slog.Int("msg_id", latest.GetID()))
	u.process(ctx, forwardTask{
		channelID:    channelID,
		msgID:        latest.GetID(),
		text:         latest.GetMessage(),
		photo:        photoLocation(latest),
		fwdChannelID: fwdChannelID(latest),
		fwdPost:      fwdChannelPost(latest),
	})
	return nil
}
