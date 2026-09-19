package filter

import (
	"sync"
)

// SignatureStore holds per-channel custom signatures in a thread-safe map.
// Keys are normalized channel identifiers (lowercase username without leading
// '@', or raw numeric string for private channels).
type SignatureStore struct {
	mu   sync.RWMutex
	sigs map[string]string
}

// NewSignatureStore creates a SignatureStore pre-seeded from a map.
// The provided map is copied; passing nil is safe.
func NewSignatureStore(seed map[string]string) *SignatureStore {
	s := &SignatureStore{
		sigs: make(map[string]string, len(seed)),
	}
	for k, v := range seed {
		if k != "" && v != "" {
			s.sigs[k] = v
		}
	}
	return s
}

// Set stores or updates the signature for the given channel key.
func (s *SignatureStore) Set(key, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sigs[key] = text
}

// Get returns the signature for the given channel key, or "" if not set.
func (s *SignatureStore) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sigs[key]
}

// Clear removes the signature for the given channel key.
func (s *SignatureStore) Clear(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sigs, key)
}

// List returns a snapshot copy of all key → signature mappings.
func (s *SignatureStore) List() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.sigs))
	for k, v := range s.sigs {
		out[k] = v
	}
	return out
}

// NormalizeChannelKey returns the canonical key for a channel identifier:
// strips https://, http://, t.me/, telegram.me/ prefixes and leading '@',
// then lowercases the result.
func NormalizeChannelKey(raw string) string {
	s := raw
	for _, prefix := range []string{
		"https://t.me/", "http://t.me/",
		"https://telegram.me/", "http://telegram.me/",
		"t.me/", "telegram.me/",
	} {
		if len(s) > len(prefix) && s[:len(prefix)] == prefix {
			s = s[len(prefix):]
			break
		}
	}
	if len(s) > 0 && s[0] == '@' {
		s = s[1:]
	}
	// lowercase for consistent matching
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}
