package filter

import (
	"sync"
	"testing"
)

func TestSignatureStore_SetGet(t *testing.T) {
	s := NewSignatureStore(nil)

	// Initially empty
	if got := s.Get("mon1tor_ua"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	s.Set("mon1tor_ua", "🔔 My sig")
	if got := s.Get("mon1tor_ua"); got != "🔔 My sig" {
		t.Errorf("expected %q, got %q", "🔔 My sig", got)
	}

	// Update
	s.Set("mon1tor_ua", "Updated sig")
	if got := s.Get("mon1tor_ua"); got != "Updated sig" {
		t.Errorf("expected %q, got %q", "Updated sig", got)
	}
}

func TestSignatureStore_Clear(t *testing.T) {
	s := NewSignatureStore(map[string]string{"ch1": "foo"})
	s.Clear("ch1")
	if got := s.Get("ch1"); got != "" {
		t.Errorf("expected empty after clear, got %q", got)
	}
	// Clearing non-existent key is a no-op
	s.Clear("nonexistent")
}

func TestSignatureStore_List(t *testing.T) {
	seed := map[string]string{"a": "aaa", "b": "bbb"}
	s := NewSignatureStore(seed)
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

func TestSignatureStore_Seed(t *testing.T) {
	// Empty values / keys in seed are ignored
	s := NewSignatureStore(map[string]string{"": "val", "key": ""})
	if len(s.List()) != 0 {
		t.Error("expected empty store from empty seed entries")
	}
}

func TestSignatureStore_Concurrent(t *testing.T) {
	s := NewSignatureStore(nil)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); s.Set("k", "v") }()
		go func() { defer wg.Done(); _ = s.Get("k") }()
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
		{"https://telegram.me/Chan", "chan"},
		{"-1001234567890", "-1001234567890"},
		{"", ""},
	}
	for _, tc := range cases {
		got := NormalizeChannelKey(tc.in)
		if got != tc.want {
			t.Errorf("NormalizeChannelKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
