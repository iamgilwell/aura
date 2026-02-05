package safety

import (
	"sync"

	"github.com/iamgilwell/aura/internal/monitor"
)

// Consent levels.
const (
	ConsentAutomatic   = 0 // Fully automatic
	ConsentNotifySystem = 1 // Notify for system processes
	ConsentConfirmAll  = 2 // Confirm all terminations
	ConsentMonitorOnly = 3 // Monitoring only, no terminations
)

// ConsentManager handles user consent levels for process termination.
type ConsentManager struct {
	mu    sync.RWMutex
	level int
}

// NewConsentManager creates a consent manager with the given level.
func NewConsentManager(level int) *ConsentManager {
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	return &ConsentManager{level: level}
}

// Level returns the current consent level.
func (c *ConsentManager) Level() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.level
}

// SetLevel sets the consent level (0-3).
func (c *ConsentManager) SetLevel(level int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if level < 0 {
		level = 0
	}
	if level > 3 {
		level = 3
	}
	c.level = level
}

// NeedsConfirmation returns whether user confirmation is needed.
func (c *ConsentManager) NeedsConfirmation(proc *monitor.ProcessInfo) bool {
	c.mu.RLock()
	level := c.level
	c.mu.RUnlock()

	switch level {
	case ConsentAutomatic:
		return false
	case ConsentNotifySystem:
		return proc.Category == monitor.CategorySystem || proc.Category == monitor.CategoryEssential
	case ConsentConfirmAll:
		return true
	case ConsentMonitorOnly:
		return true // Always block in monitor-only mode
	default:
		return true
	}
}

// LevelDescription returns a human-readable description of a consent level.
func LevelDescription(level int) string {
	switch level {
	case ConsentAutomatic:
		return "Fully Automatic"
	case ConsentNotifySystem:
		return "Notify for System"
	case ConsentConfirmAll:
		return "Confirm All"
	case ConsentMonitorOnly:
		return "Monitor Only"
	default:
		return "Unknown"
	}
}
