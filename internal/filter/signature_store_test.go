package filter

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSignatureStore_SetGet(t *testing.T) {
	s := NewSignatureStore("", nil, nil)

	// Initially empty
	if got := s.Get("mon1tor_ua"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	if err := s.Set("mon1tor_ua", "🔔 My sig"); err != nil {
		t.Fatalf("unexpected error setting signature: %v", err)
	}
	if got := s.Get("mon1tor_ua"); got != "🔔 My sig" {
		t.Errorf("expected %q, got %q", "🔔 My sig", got)
	}

	// Leading '@' should normalize to same key
	if got := s.Get("@mon1tor_ua"); got != "🔔 My sig" {
		t.Errorf("expected %q with @, got %q", "🔔 My sig", got)
	}

	// Update
	if err := s.Set("mon1tor_ua", "Updated sig"); err != nil {
		t.Fatalf("unexpected error updating signature: %v", err)
	}
	if got := s.Get("mon1tor_ua"); got != "Updated sig" {
		t.Errorf("expected %q, got %q", "Updated sig", got)
	}
}

func TestSignatureStore_Clear(t *testing.T) {
	s := NewSignatureStore("", map[string]string{"ch1": "foo"}, nil)
	if err := s.Clear("ch1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := s.Get("ch1"); got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
	// Clearing non-existent key is a no-op
	if err := s.Clear("nonexistent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSignatureStore_FilePersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sigstore_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "signatures.json")

	// 1. Create store with initial seed
	s1 := NewSignatureStore(filePath, map[string]string{"default": "Global Sig"}, nil)
	if err := s1.Set("mon1tor_ua", "Channel Specific Sig"); err != nil {
		t.Fatalf("failed to set sig: %v", err)
	}

	// 2. Create brand new store pointing to same file
	s2 := NewSignatureStore(filePath, nil, nil)
	if got := s2.Get("default"); got != "Global Sig" {
		t.Errorf("expected persisted default sig 'Global Sig', got %q", got)
	}
	if got := s2.Get("mon1tor_ua"); got != "Channel Specific Sig" {
		t.Errorf("expected persisted channel sig, got %q", got)
	}

	// 3. Clear in s2 and verify file updated
	if err := s2.Clear("mon1tor_ua"); err != nil {
		t.Fatalf("failed to clear sig: %v", err)
	}

	s3 := NewSignatureStore(filePath, nil, nil)
	if got := s3.Get("mon1tor_ua"); got != "" {
		t.Errorf("expected cleared sig to be empty in s3, got %q", got)
	}
	if got := s3.Get("default"); got != "Global Sig" {
		t.Errorf("expected default sig to remain in s3, got %q", got)
	}
}

func TestSignatureStore_GetForChannel(t *testing.T) {
	s := NewSignatureStore("", map[string]string{
		"default":        "Default Footer",
		"mon1tor_ua":     "Monitor Footer",
		"-1001929743622": "Private 1 Footer",
		"1855211672":     "Private 2 Footer (saved without -100)",
	}, nil)

	// Case 1: Specific match by username
	sig, key := s.GetForChannel("mon1tor_ua", "-100123456", "123456")
	if sig != "Monitor Footer" || key != "mon1tor_ua" {
		t.Errorf("expected Monitor Footer, got %q (key=%q)", sig, key)
	}

	// Case 2: Specific match by -100 ID
	sig, key = s.GetForChannel("test_ch", "-1001929743622", "1929743622")
	if sig != "Private 1 Footer" {
		t.Errorf("expected Private 1 Footer, got %q (key=%q)", sig, key)
	}

	// Case 3: Match when saved without -100 but candidate has -100
	sig, key = s.GetForChannel("-1001855211672", "1855211672")
	if sig != "Private 2 Footer (saved without -100)" {
		t.Errorf("expected Private 2 Footer, got %q (key=%q)", sig, key)
	}

	// Case 4: Match when saved with -100 but candidate only has positive numeric ID
	sig, key = s.GetForChannel("1929743622")
	if sig != "Private 1 Footer" {
		t.Errorf("expected Private 1 Footer, got %q (key=%q)", sig, key)
	}

	// Case 5: Fallback to global default when channel has no specific signature
	sig, key = s.GetForChannel("unknown_channel", "-1009999999999", "9999999999")
	if sig != "Default Footer" || key != "default" {
		t.Errorf("expected Default Footer fallback, got %q (key=%q)", sig, key)
	}

	// Case 6: No match when default is cleared
	s.Clear("default")
	sig, key = s.GetForChannel("unknown_channel")
	if sig != "" || key != "" {
		t.Errorf("expected empty result when no match and no default, got %q (key=%q)", sig, key)
	}
}

func TestSignatureStore_List(t *testing.T) {
	seed := map[string]string{"a": "aaa", "b": "bbb"}
	s := NewSignatureStore("", seed, nil)
	list := s.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(list))
	}
	if list["a"] != "aaa" || list["b"] != "bbb" {
		t.Errorf("unexpected list contents: %v", list)
	}
	// Mutating snapshot does not affect store
	list["a"] = "mutated"
	if s.Get("a") != "aaa" {
		t.Error("store was mutated through List() snapshot")
	}
}

func TestSignatureStore_Concurrent(t *testing.T) {
	s := NewSignatureStore("", nil, nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); _ = s.Set("k", "v") }()
		go func() { defer wg.Done(); _ = s.Get("k") }()
		go func() { defer wg.Done(); _, _ = s.GetForChannel("k", "default") }()
		go func() { defer wg.Done(); _ = s.List() }()
	}
	wg.Wait()
}

func TestNormalizeChannelKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"mon1tor_ua", "mon1tor_ua"},
		{"@Mon1tor_UA", "mon1tor_ua"},
		{"t.me/Mon1tor_UA", "mon1tor_ua"},
		{"https://t.me/Mon1tor_UA", "mon1tor_ua"},
		{"http://t.me/Mon1tor_UA", "mon1tor_ua"},
		{"https://telegram.me/Chan/", "chan"},
		{"-1001234567890", "-1001234567890"},
		{`"default"`, "default"},
		{"", ""},
	}
	for _, tc := range cases {
		got := NormalizeChannelKey(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeChannelKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
