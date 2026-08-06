package alert

import (
	"log/slog"
	"sync/atomic"
)

// KyivAlertState tracks whether an air alert is currently active for Kyiv city.
type KyivAlertState struct {
	active atomic.Bool
	logger *slog.Logger
}

// NewKyivAlertState creates a new alert state machine.
func NewKyivAlertState(logger *slog.Logger) *KyivAlertState {
	if logger == nil {
		logger = slog.Default()
	}
	return &KyivAlertState{logger: logger}
}

// IsActive reports whether a Kyiv city air alert is currently active.
func (s *KyivAlertState) IsActive() bool {
	return s.active.Load()
}

// SetActive updates the state and logs the transition. Repeated calls with the
// same value are no-ops (deduplication).
func (s *KyivAlertState) SetActive(v bool) {
	if s.active.Load() == v {
		return
	}
	s.active.Store(v)
	if v {
		s.logger.Info("AIR ALERT ACTIVE for Kyiv city")
	} else {
		s.logger.Info("Air alert cleared for Kyiv city")
	}
}
