package filter

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// SignatureStore holds per-channel and global custom signatures, persisting
// them to a JSON file so that restarts do not wipe configured signatures.
//
// Keys are normalized channel identifiers (lowercase username without leading
// '@', numeric channel ID, or "default" for the global fallback).
type SignatureStore struct {
	filePath string
	logger   *slog.Logger
	mu       sync.RWMutex
	sigs     map[string]string
}

// NewSignatureStore creates a SignatureStore loaded from filePath if it exists,
// seeding from seed (e.g. CHANNEL_SIGNATURES env var) as defaults.
func NewSignatureStore(filePath string, seed map[string]string, logger *slog.Logger) *SignatureStore {
	if logger == nil {
		logger = slog.Default()
	}

	s := &SignatureStore{
		filePath: filePath,
		logger:   logger,
		sigs:     make(map[string]string),
	}

	// 1. Seed from environment / defaults
	for k, v := range seed {
		norm := NormalizeChannelKey(k)
		if norm != "" && v != "" {
			s.sigs[norm] = v
		}
	}

	// 2. Load from disk file if available (takes precedence over env defaults)
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			var fileSigs map[string]string
			if err := json.Unmarshal(data, &fileSigs); err == nil {
				for k, v := range fileSigs {
					norm := NormalizeChannelKey(k)
					if norm != "" && v != "" {
						s.sigs[norm] = v
					}
				}
				s.logger.Info("loaded custom signatures from file",
					slog.String("path", filePath),
					slog.Int("count", len(fileSigs)))
			} else {
				s.logger.Warn("failed to parse signatures file, starting with seed",
					slog.String("path", filePath),
					slog.String("err", err.Error()))
			}
		} else if len(s.sigs) > 0 {
			// File doesn't exist yet, but seed has items — persist them
			_ = s.saveLocked()
		}
	}

	return s
}

// Set stores or updates the signature for the given channel key and persists to disk.
func (s *SignatureStore) Set(key, text string) error {
	norm := NormalizeChannelKey(key)
	if norm == "" {
		norm = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.sigs[norm] = text
	return s.saveLocked()
}

// Get returns the signature for the given channel key, or "" if not set.
func (s *SignatureStore) Get(key string) string {
	norm := NormalizeChannelKey(key)
	if norm == "" {
		norm = "default"
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sigs[norm]
}

// Clear removes the signature for the given channel key and persists to disk.
func (s *SignatureStore) Clear(key string) error {
	norm := NormalizeChannelKey(key)
	if norm == "" {
		norm = "default"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sigs, norm)
	// If clearing numeric ID, also clear with/without -100 prefix
	if strings.HasPrefix(norm, "-100") {
		delete(s.sigs, strings.TrimPrefix(norm, "-100"))
	} else if isNumeric(norm) {
		delete(s.sigs, "-100"+norm)
	}

	return s.saveLocked()
}

// GetForChannel searches the store using candidate channel identifiers in order.
// If any candidate matches, its signature is returned. If none match, it falls
// back to the global default signature ("default", "all", "*").
// Returns (signature, matchedKey). If no signature is configured, returns ("", "").
func (s *SignatureStore) GetForChannel(candidates ...string) (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Try channel-specific candidates
	for _, c := range candidates {
		if c == "" {
			continue
		}
		norm := NormalizeChannelKey(c)
		if norm == "" {
			continue
		}

		if sig, ok := s.sigs[norm]; ok && sig != "" {
			return sig, norm
		}

		// Also check with/without -100 prefix for channel IDs
		if strings.HasPrefix(norm, "-100") {
			trimmed := strings.TrimPrefix(norm, "-100")
			if sig, ok := s.sigs[trimmed]; ok && sig != "" {
				return sig, trimmed
			}
		} else if isNumeric(norm) {
			prefixed := "-100" + norm
			if sig, ok := s.sigs[prefixed]; ok && sig != "" {
				return sig, prefixed
			}
		}
	}

	// 2. Try global default signature fallback
	for _, fallbackKey := range []string{"default", "all", "*"} {
		if sig, ok := s.sigs[fallbackKey]; ok && sig != "" {
			return sig, "default"
		}
	}

	return "", ""
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

func (s *SignatureStore) saveLocked() error {
	if s.filePath == "" {
		return nil
	}

	dir := filepath.Dir(s.filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			s.logger.Error("failed to create signatures directory",
				slog.String("dir", dir),
				slog.String("err", err.Error()))
			return err
		}
	}

	data, err := json.MarshalIndent(s.sigs, "", "  ")
	if err != nil {
		s.logger.Error("failed to marshal signatures", slog.String("err", err.Error()))
		return err
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		s.logger.Error("failed to write signatures file",
			slog.String("path", s.filePath),
			slog.String("err", err.Error()))
		return err
	}

	s.logger.Info("saved custom signatures to file",
		slog.String("path", s.filePath),
		slog.Int("count", len(s.sigs)))
	return nil
}

// NormalizeChannelKey returns the canonical key for a channel identifier:
// strips URL prefixes, leading '@', whitespace, and lowercases.
func NormalizeChannelKey(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"'`)

	for _, prefix := range []string{
		"https://t.me/", "http://t.me/",
		"https://telegram.me/", "http://telegram.me/",
		"t.me/", "telegram.me/",
	} {
		if strings.HasPrefix(strings.ToLower(s), prefix) {
			s = s[len(prefix):]
			break
		}
	}

	s = strings.TrimPrefix(s, "@")
	s = strings.TrimSuffix(s, "/")
	return strings.ToLower(strings.TrimSpace(s))
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
