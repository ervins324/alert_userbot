package telegram

import (
	"bufio"
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
)

// Mode selects the runtime behavior of the userbot.
type Mode int

const (
	ModeDaemon Mode = iota
	ModeTestNotify
	ModeTestForward
)

type forwardTask struct {
	msgID int
	text  string
}

// UserBot is a Telegram MTProto client (logged in as a real user) that
// subscribes to a public channel and forwards its new posts to a destination
// chat while a Kyiv city air alert is active.
type UserBot struct {
	appID         int
	appHash       string
	phone         string
	password      string
	authCode      string
	sessionFile   string
	sourceChannel string
	destStr       string

	state  *alert.KyivAlertState
	filter *filter.TextFilter
	logger *slog.Logger

	client *telegram.Client
	api    *tg.Client

	sourceChannelID int64
	fromPeer        tg.InputPeerClass
	destPeer        tg.InputPeerClass

	queue chan forwardTask
	seen  map[int]struct{}

	forwarded atomic.Int64
	skipped   atomic.Int64
	filtered  atomic.Int64

	wg sync.WaitGroup
}

// NewUserBot creates a new MTProto userbot.
func NewUserBot(
	appID int,
	appHash, phone, password, authCode, sessionFile, sourceChannel, destStr string,
	state *alert.KyivAlertState,
	textFilter *filter.TextFilter,
	queueCapacity int,
	logger *slog.Logger,
) *UserBot {
	if logger == nil {
		logger = slog.Default()
	}
	return &UserBot{
		appID:         appID,
		appHash:       appHash,
		phone:         phone,
		password:      password,
		authCode:      authCode,
		sessionFile:   sessionFile,
		sourceChannel: sourceChannel,
		destStr:       destStr,
		state:         state,
		filter:        textFilter,
		logger:        logger,
		queue:         make(chan forwardTask, queueCapacity),
		seen:          make(map[int]struct{}),
	}
}

// Stats returns counters for forwarded, skipped and filtered messages.
func (u *UserBot) Stats() (forwarded, skipped, filtered int64) {
	return u.forwarded.Load(), u.skipped.Load(), u.filtered.Load()
}

// Run connects, authenticates and runs the userbot until ctx is canceled.
func (u *UserBot) Run(ctx context.Context, mode Mode) error {
	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewChannelMessage) error {
		return u.handleMessage(ctx, update.Message)
	})
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, update *tg.UpdateNewMessage) error {
		return u.handleMessage(ctx, update.Message)
	})

	u.client = telegram.NewClient(u.appID, u.appHash, telegram.Options{
		SessionStorage: session.NewFileStorage(u.sessionFile, 0o600),
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

		if err := u.resolveSource(ctx); err != nil {
			return err
		}
		if err := u.resolveDestPeer(ctx); err != nil {
			return err
		}

		switch mode {
		case ModeTestNotify:
			return u.testNotify(ctx)
		case ModeTestForward:
			return u.testForward(ctx)
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
		return nil
	}
	if u.phone == "" {
		return fmt.Errorf("TG_PHONE is required for first-time login")
	}

	var authenticator auth.UserAuthenticator
	if u.password != "" {
		authenticator = auth.Constant(u.phone, u.password, auth.CodeAuthenticatorFunc(u.askCode))
	} else {
		authenticator = auth.CodeOnly(u.phone, auth.CodeAuthenticatorFunc(u.askCode))
	}

	u.logger.Info("performing first-time Telegram login", slog.String("phone", u.phone))
	return auth.NewFlow(authenticator, auth.SendCodeOptions{}).Run(ctx, u.client.Auth())
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

func (u *UserBot) resolveSource(ctx context.Context) error {
	res, err := u.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: strings.TrimPrefix(u.sourceChannel, "@"),
	})
	if err != nil {
		return fmt.Errorf("resolve source channel %q: %w", u.sourceChannel, err)
	}
	for _, c := range res.Chats {
		if ch, ok := c.(*tg.Channel); ok {
			u.sourceChannelID = ch.ID
			u.fromPeer = ch.AsInputPeer()
			// Ensure membership so we receive channel updates.
			if !ch.Left && !ch.Megagroup {
				if _, err := u.api.ChannelsJoinChannel(ctx, &tg.ChannelsJoinChannelRequest{
					Channel: ch.AsInput(),
				}); err != nil {
					u.logger.Warn("could not join source channel (may already be a member)",
						slog.String("channel", u.sourceChannel), slog.String("err", err.Error()))
				}
			}
			u.logger.Info("source channel resolved", slog.String("channel", u.sourceChannel), slog.Int64("id", ch.ID))
			return nil
		}
	}
	return fmt.Errorf("source channel %q is not a channel", u.sourceChannel)
}

// resolveDestPeer resolves the destination chat: self, username, or a
// numeric ID found in the account's dialogs.
func (u *UserBot) resolveDestPeer(ctx context.Context) error {
	s := strings.TrimSpace(u.destStr)
	if strings.HasPrefix(s, "@") {
		s = s[1:]
	}

	self, err := u.client.Self(ctx)
	if err != nil {
		return fmt.Errorf("telegram self: %w", err)
	}

	if !isNumericID(s) {
		peer, err := u.resolveByUsername(ctx, s)
		if err != nil {
			return fmt.Errorf("resolve destination %q: %w", u.destStr, err)
		}
		u.destPeer = peer
		u.logger.Info("destination resolved", slog.String("dest", u.destStr))
		return nil
	}

	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid DESTINATION_CHAT_ID %q", u.destStr)
	}

	if id == self.ID {
		u.destPeer = &tg.InputPeerSelf{}
		u.logger.Info("destination is the authenticated user (Saved Messages)")
		return nil
	}

	peer, err := u.findPeerInDialogs(ctx, s)
	if err != nil {
		return fmt.Errorf("resolve destination %q from dialogs: %w", u.destStr, err)
	}
	u.destPeer = peer
	u.logger.Info("destination resolved from dialogs", slog.String("dest", u.destStr))
	return nil
}

func (u *UserBot) resolveByUsername(ctx context.Context, username string) (tg.InputPeerClass, error) {
	res, err := u.api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{Username: username})
	if err != nil {
		return nil, err
	}
	for _, c := range res.Chats {
		switch e := c.(type) {
		case *tg.Channel:
			if p, ok := res.Peer.(*tg.PeerChannel); ok && p.ChannelID == e.ID {
				return e.AsInputPeer(), nil
			}
		case *tg.Chat:
			if p, ok := res.Peer.(*tg.PeerChat); ok && p.ChatID == e.ID {
				return e.AsInputPeer(), nil
			}
		}
	}
	for _, us := range res.Users {
		if e, ok := us.(*tg.User); ok {
			if p, ok := res.Peer.(*tg.PeerUser); ok && p.UserID == e.ID {
				return e.AsInputPeer(), nil
			}
		}
	}
	return nil, fmt.Errorf("peer %q not resolved", username)
}

// findPeerInDialogs searches the account dialogs for a chat/channel/user
// matching the numeric ID. Supports -100-prefixed channel IDs.
func (u *UserBot) findPeerInDialogs(ctx context.Context, s string) (tg.InputPeerClass, error) {
	channelID, isChannel := parseChannelID(s)

	req := &tg.MessagesGetDialogsRequest{
		ExcludePinned: true,
		OffsetDate:    0,
		OffsetID:      0,
		OffsetPeer:    &tg.InputPeerEmpty{},
		Limit:         100,
		Hash:          0,
	}

	for page := 0; page < 30; page++ {
		res, err := u.api.MessagesGetDialogs(ctx, req)
		if err != nil {
			return nil, err
		}

		switch d := res.(type) {
		case *tg.MessagesDialogs:
			if peer := scanDialogs(d.Chats, d.Users, channelID, isChannel); peer != nil {
				return peer, nil
			}
			return nil, fmt.Errorf("peer %s not found in dialogs", s)
		case *tg.MessagesDialogsSlice:
			if peer := scanDialogs(d.Chats, d.Users, channelID, isChannel); peer != nil {
				return peer, nil
			}
			if len(d.Dialogs) == 0 {
				return nil, fmt.Errorf("peer %s not found in dialogs", s)
			}
			// Page forward using the last message as the offset.
			if len(d.Messages) > 0 {
				last := d.Messages[len(d.Messages)-1]
				req.OffsetDate = last.GetDate()
				req.OffsetID = last.GetID()
			}
		default:
			return nil, fmt.Errorf("unexpected dialogs response %T", res)
		}
	}
	return nil, fmt.Errorf("peer %s not found in dialogs (too many pages)", s)
}

func scanDialogs(chats []tg.ChatClass, users []tg.UserClass, channelID int64, isChannel bool) tg.InputPeerClass {
	for _, c := range chats {
		switch e := c.(type) {
		case *tg.Channel:
			if isChannel && e.ID == channelID {
				return e.AsInputPeer()
			}
		case *tg.Chat:
			if !isChannel && e.ID == channelID {
				return e.AsInputPeer()
			}
		}
	}
	if !isChannel {
		for _, us := range users {
			if e, ok := us.(*tg.User); ok && e.ID == channelID {
				return e.AsInputPeer()
			}
		}
	}
	return nil
}

func parseChannelID(s string) (int64, bool) {
	if strings.HasPrefix(s, "-100") && len(s) > 4 {
		id, err := strconv.ParseInt(s[4:], 10, 64)
		if err != nil {
			return 0, false
		}
		return id, true
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, false
}

func isNumericID(s string) bool {
	_, err := strconv.ParseInt(s, 10, 64)
	return err == nil
}

// handleMessage processes a channel update and queues matching messages.
func (u *UserBot) handleMessage(ctx context.Context, msgClass tg.MessageClass) error {
	m, ok := msgClass.(*tg.Message)
	if !ok {
		return nil // ignore service messages
	}

	peer := m.GetPeerID()
	ch, ok := peer.(*tg.PeerChannel)
	if !ok || ch.ChannelID != u.sourceChannelID {
		return nil
	}

	msgID := m.GetID()
	if _, dup := u.seen[msgID]; dup {
		return nil
	}
	u.seen[msgID] = struct{}{}

	select {
	case u.queue <- forwardTask{msgID: msgID, text: m.GetMessage()}:
	default:
		u.logger.Error("forward queue full, message dropped", slog.Int("msg_id", msgID))
	}
	return nil
}

func (u *UserBot) runDaemon(ctx context.Context) error {
	u.wg.Add(1)
	go u.worker(ctx)

	u.logger.Info("userbot ready: forwarding channel posts while Kyiv alert is active",
		slog.String("source", u.sourceChannel), slog.String("dest", u.destStr))

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
	if u.filter != nil && u.filter.ShouldSkip(task.text) {
		u.filtered.Add(1)
		u.logger.Info("message filtered out", slog.Int("msg_id", task.msgID))
		return
	}
	if !u.state.IsActive() {
		u.skipped.Add(1)
		u.logger.Debug("message skipped (no active alert)", slog.Int("msg_id", task.msgID))
		return
	}
	if err := u.forward(ctx, task.msgID); err != nil {
		u.logger.Error("forward failed", slog.Int("msg_id", task.msgID), slog.String("err", err.Error()))
		return
	}
	u.forwarded.Add(1)
	u.logger.Info("forwarded channel message", slog.Int("msg_id", task.msgID))
}

func (u *UserBot) forward(ctx context.Context, msgID int) error {
	randomID, err := u.client.RandInt64()
	if err != nil {
		return err
	}
	_, err = u.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: u.fromPeer,
		ID:       []int{msgID},
		RandomID: []int64{randomID},
		ToPeer:   u.destPeer,
	})
	return err
}

func (u *UserBot) testNotify(ctx context.Context) error {
	randomID, err := u.client.RandInt64()
	if err != nil {
		return err
	}
	_, err = u.api.MessagesSendMessage(ctx, &tg.MessagesSendMessageRequest{
		Peer:     u.destPeer,
		Message:  "🔔 <b>Test notification</b> — Neptun MTProto forwarder is working. If you receive this, your userbot + destination chat are configured correctly.",
		RandomID: randomID,
	})
	if err != nil {
		return fmt.Errorf("send test message: %w", err)
	}
	u.logger.Info("test notification sent. Check your Telegram chat.")
	return nil
}

func (u *UserBot) testForward(ctx context.Context) error {
	// Simulate an active alert and forward the latest source-channel post.
	u.state.SetActive(true)

	res, err := u.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     u.fromPeer,
		OffsetID: 0,
		OffsetDate: 0,
		AddOffset: 0,
		Limit:    5,
		MaxID:    0,
		MinID:    0,
		Hash:     0,
	})
	if err != nil {
		return fmt.Errorf("get channel history: %w", err)
	}

	var msgID int
	switch msgs := res.(type) {
	case *tg.MessagesChannelMessages:
		for _, m := range msgs.Messages {
			if mm, ok := m.(*tg.Message); ok {
				msgID = mm.GetID()
				break
			}
		}
	case *tg.MessagesMessages:
		for _, m := range msgs.Messages {
			if mm, ok := m.(*tg.Message); ok {
				msgID = mm.GetID()
				break
			}
		}
	default:
		return fmt.Errorf("unexpected history response %T", res)
	}

	if msgID == 0 {
		return fmt.Errorf("no messages found in channel %q", u.sourceChannel)
	}

	u.logger.Info("test forward: forwarding latest channel message (alert simulated as active)", slog.Int("msg_id", msgID))
	return u.forward(ctx, msgID)
}
