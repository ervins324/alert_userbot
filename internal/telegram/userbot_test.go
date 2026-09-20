package telegram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gotd/td/tg"
)

func TestUserBot_ChannelsPersistenceAndMatching(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ub_channels_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	channelsFile := filepath.Join(tmpDir, "channels.json")

	// 1. Create UserBot with initial sources
	ub := NewUserBot(
		123, "hash", "phone", "pass", "code", "session.bin", channelsFile,
		[]string{"-1001929743622", "mon1tor_ua"},
		nil, nil, nil, nil, nil, 10, false, nil,
	)

	// Verify file was created with initial sources
	if _, err := os.Stat(channelsFile); err != nil {
		t.Fatalf("expected channels.json to be created: %v", err)
	}

	// 2. Simulate resolving a channel
	ub.mu.Lock()
	chID := int64(1929743622)
	ub.sourceChannelIDs[chID] = struct{}{}
	ub.channelsByID[chID] = &tg.Channel{
		ID:       chID,
		Title:    "Київ Монітор",
		Username: "kyiv_monitor1",
	}
	ub.channelKeys[chID] = "kyiv_monitor1"
	ub.mu.Unlock()

	monitored := ub.MonitoredChannels()
	if len(monitored) != 1 {
		t.Fatalf("expected 1 resolved channel, got %d", len(monitored))
	}
	if monitored[0].Username != "kyiv_monitor1" {
		t.Errorf("unexpected username: %s", monitored[0].Username)
	}

	// 3. Test channelMatches helper
	chInfo := &ChannelInfo{
		ID:         chID,
		Username:   "kyiv_monitor1",
		Title:      "Київ Монітор",
		ConfigName: "-1001929743622",
	}
	if !channelMatches("@kyiv_monitor1", chInfo) {
		t.Error("expected match by @username")
	}
	if !channelMatches("kyiv_monitor1", chInfo) {
		t.Error("expected match by username")
	}
	if !channelMatches("-1001929743622", chInfo) {
		t.Error("expected match by -100 ID")
	}
	if !channelMatches("1929743622", chInfo) {
		t.Error("expected match by raw numeric ID")
	}
	if !channelMatches("Київ Монітор", chInfo) {
		t.Error("expected match by title")
	}
	if channelMatches("another_channel", chInfo) {
		t.Error("expected no match for different channel")
	}

	// 4. Test RemoveChannel
	info, err := ub.RemoveChannel("@kyiv_monitor1")
	if err != nil {
		t.Fatalf("unexpected error removing channel: %v", err)
	}
	if info.ID != chID {
		t.Errorf("unexpected removed channel ID %d", info.ID)
	}

	// 5. Create new UserBot instance pointing to same file and verify channel is removed
	ub2 := NewUserBot(
		123, "hash", "phone", "pass", "code", "session.bin", channelsFile,
		nil, nil, nil, nil, nil, nil, 10, false, nil,
	)
	for _, sc := range ub2.sourceChannels {
		if sc == "-1001929743622" || sc == "kyiv_monitor1" {
			t.Errorf("expected channel to be removed from persistent storage, found %s", sc)
		}
	}
}
