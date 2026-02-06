package process

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/safety"
)

// Manager handles process termination.
type Manager struct {
	safetyMgr *safety.Manager
	timeout   time.Duration
}

// NewManager creates a new process manager.
func NewManager(safetyMgr *safety.Manager, timeout time.Duration) *Manager {
	return &Manager{
		safetyMgr: safetyMgr,
		timeout:   timeout,
	}
}

// Terminate sends SIGTERM to a process, then SIGKILL after timeout.
func (m *Manager) Terminate(pid int, force bool) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if force {
		if err := proc.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("sending SIGKILL to %d: %w", pid, err)
		}
		return nil
	}

	// Send SIGTERM first
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to %d: %w", pid, err)
	}

	// Wait for process to exit, then SIGKILL if still alive
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < int(m.timeout.Seconds()*10); i++ {
			if err := proc.Signal(syscall.Signal(0)); err != nil {
				return // Process is gone
			}
			time.Sleep(100 * time.Millisecond)
		}
		// Process still alive — force kill
		_ = proc.Signal(syscall.SIGKILL)
	}()

	<-done
	return nil
}

// SafeTerminate validates with safety manager before terminating.
func (m *Manager) SafeTerminate(procInfo *monitor.ProcessInfo, force bool) error {
	allowed, reason := m.safetyMgr.ValidateTermination(procInfo)
	if !allowed {
		return fmt.Errorf("termination blocked: %s", reason)
	}
	return m.Terminate(procInfo.PID, force)
}

// Children returns child PIDs of the given PID (from current /proc data).
func (m *Manager) Children(pid int, allProcs []*monitor.ProcessInfo) []int {
	var children []int
	for _, p := range allProcs {
		if p.PPid == pid {
			children = append(children, p.PID)
		}
	}
	return children
}
