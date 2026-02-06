package power

import (
	"sync"
	"time"
)

// Metrics tracks power savings over time.
type Metrics struct {
	mu          sync.RWMutex
	totalWattsSaved float64
	savingsLog  []SavingsEntry
	startTime   time.Time
}

// SavingsEntry records a single power savings event.
type SavingsEntry struct {
	Timestamp   time.Time
	ProcessName string
	PID         int
	WattsSaved  float64
	Reason      string
}

// NewMetrics creates a new power metrics tracker.
func NewMetrics() *Metrics {
	return &Metrics{
		startTime: time.Now(),
	}
}

// RecordSaving logs a power savings event.
func (m *Metrics) RecordSaving(processName string, pid int, watts float64, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalWattsSaved += watts
	m.savingsLog = append(m.savingsLog, SavingsEntry{
		Timestamp:   time.Now(),
		ProcessName: processName,
		PID:         pid,
		WattsSaved:  watts,
		Reason:      reason,
	})
}

// TotalSaved returns total watts saved this session.
func (m *Metrics) TotalSaved() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalWattsSaved
}

// SessionDuration returns how long the session has been running.
func (m *Metrics) SessionDuration() time.Duration {
	return time.Since(m.startTime)
}

// MonthlyProjection projects monthly kWh savings based on current rate.
func (m *Metrics) MonthlyProjection() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	elapsed := time.Since(m.startTime).Hours()
	if elapsed < 0.001 {
		return 0
	}

	wattsPerHour := m.totalWattsSaved / elapsed
	hoursPerMonth := 24.0 * 30.0
	return wattsPerHour * hoursPerMonth / 1000.0 // kWh
}

// RecentSavings returns the most recent n savings entries.
func (m *Metrics) RecentSavings(n int) []SavingsEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n > len(m.savingsLog) {
		n = len(m.savingsLog)
	}
	result := make([]SavingsEntry, n)
	copy(result, m.savingsLog[len(m.savingsLog)-n:])
	return result
}

// Count returns the number of savings events.
func (m *Metrics) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.savingsLog)
}
