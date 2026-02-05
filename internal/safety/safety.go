package safety

import (
	"fmt"
	"sync"

	"github.com/iamgilwell/aura/internal/monitor"
)

// Manager validates process termination safety.
type Manager struct {
	mu             sync.RWMutex
	protectedProcs map[string]bool
	neverTerminate map[string]bool
	consentMgr     *ConsentManager
}

// NewManager creates a new safety manager.
func NewManager(protectedNames, neverTermNames []string, consentLevel int) *Manager {
	protected := make(map[string]bool, len(protectedNames))
	for _, n := range protectedNames {
		protected[n] = true
	}
	never := make(map[string]bool, len(neverTermNames))
	for _, n := range neverTermNames {
		never[n] = true
	}
	return &Manager{
		protectedProcs: protected,
		neverTerminate: never,
		consentMgr:     NewConsentManager(consentLevel),
	}
}

// IsProtected returns true if the process must not be terminated.
func (m *Manager) IsProtected(proc *monitor.ProcessInfo) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// PID 1 and 2 are always protected
	if proc.PID <= 2 {
		return true
	}

	// Kernel threads are always protected
	if proc.Category == monitor.CategoryKernel {
		return true
	}

	// Check never-terminate list
	if m.neverTerminate[proc.Name] {
		return true
	}

	// Check protected processes list
	if m.protectedProcs[proc.Name] {
		return true
	}

	return false
}

// ValidateTermination checks if a process can be safely terminated.
func (m *Manager) ValidateTermination(proc *monitor.ProcessInfo) (bool, string) {
	if proc.PID <= 2 {
		return false, fmt.Sprintf("PID %d is a critical system process", proc.PID)
	}

	if proc.Category == monitor.CategoryKernel {
		return false, fmt.Sprintf("process '%s' is a kernel thread", proc.Name)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.neverTerminate[proc.Name] {
		return false, fmt.Sprintf("process '%s' is on the never-terminate list", proc.Name)
	}

	if m.protectedProcs[proc.Name] {
		return false, fmt.Sprintf("process '%s' is protected", proc.Name)
	}

	return true, "termination allowed"
}

// ConsentLevel returns the current consent level.
func (m *Manager) ConsentLevel() int {
	return m.consentMgr.Level()
}

// SetConsentLevel updates the consent level.
func (m *Manager) SetConsentLevel(level int) {
	m.consentMgr.SetLevel(level)
}

// NeedsConfirmation checks if the consent level requires user confirmation.
func (m *Manager) NeedsConfirmation(proc *monitor.ProcessInfo) bool {
	return m.consentMgr.NeedsConfirmation(proc)
}

// IsMonitorOnly returns true if consent level is 3 (monitor only).
func (m *Manager) IsMonitorOnly() bool {
	return m.consentMgr.Level() == ConsentMonitorOnly
}
