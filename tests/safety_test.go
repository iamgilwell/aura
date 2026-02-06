package tests

import (
	"testing"

	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/safety"
)

func TestSafetyManagerIsProtected(t *testing.T) {
	mgr := safety.NewManager(
		[]string{"sshd", "nginx"},
		[]string{"systemd", "init"},
		safety.ConsentConfirmAll,
	)

	tests := []struct {
		name      string
		proc      *monitor.ProcessInfo
		protected bool
	}{
		{
			name:      "PID 1 always protected",
			proc:      &monitor.ProcessInfo{PID: 1, Name: "systemd", Category: monitor.CategoryKernel},
			protected: true,
		},
		{
			name:      "PID 2 always protected",
			proc:      &monitor.ProcessInfo{PID: 2, Name: "kthreadd", Category: monitor.CategoryKernel},
			protected: true,
		},
		{
			name:      "kernel thread protected",
			proc:      &monitor.ProcessInfo{PID: 100, Name: "kworker", Category: monitor.CategoryKernel},
			protected: true,
		},
		{
			name:      "never-terminate list",
			proc:      &monitor.ProcessInfo{PID: 500, Name: "systemd", Category: monitor.CategorySystem},
			protected: true,
		},
		{
			name:      "protected process",
			proc:      &monitor.ProcessInfo{PID: 600, Name: "sshd", Category: monitor.CategoryEssential},
			protected: true,
		},
		{
			name:      "user process not protected",
			proc:      &monitor.ProcessInfo{PID: 1000, Name: "firefox", Category: monitor.CategoryUser},
			protected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.IsProtected(tt.proc)
			if got != tt.protected {
				t.Errorf("IsProtected = %v, want %v", got, tt.protected)
			}
		})
	}
}

func TestSafetyValidateTermination(t *testing.T) {
	mgr := safety.NewManager(
		[]string{"sshd"},
		[]string{"systemd"},
		safety.ConsentAutomatic,
	)

	// Should block PID 1
	allowed, reason := mgr.ValidateTermination(&monitor.ProcessInfo{PID: 1, Name: "systemd", Category: monitor.CategoryKernel})
	if allowed {
		t.Error("should not allow terminating PID 1")
	}
	if reason == "" {
		t.Error("expected a reason for blocking")
	}

	// Should allow user process
	allowed, _ = mgr.ValidateTermination(&monitor.ProcessInfo{PID: 1000, Name: "firefox", Category: monitor.CategoryUser})
	if !allowed {
		t.Error("should allow terminating user process")
	}
}

func TestConsentLevels(t *testing.T) {
	tests := []struct {
		level       int
		description string
	}{
		{safety.ConsentAutomatic, "Fully Automatic"},
		{safety.ConsentNotifySystem, "Notify for System"},
		{safety.ConsentConfirmAll, "Confirm All"},
		{safety.ConsentMonitorOnly, "Monitor Only"},
	}

	for _, tt := range tests {
		got := safety.LevelDescription(tt.level)
		if got != tt.description {
			t.Errorf("LevelDescription(%d) = %q, want %q", tt.level, got, tt.description)
		}
	}
}

func TestConsentNeedsConfirmation(t *testing.T) {
	userProc := &monitor.ProcessInfo{PID: 1000, Name: "firefox", Category: monitor.CategoryUser}
	sysProc := &monitor.ProcessInfo{PID: 500, Name: "cron", Category: monitor.CategorySystem}

	// Level 0: automatic — no confirmation needed
	mgr := safety.NewManager(nil, nil, safety.ConsentAutomatic)
	if mgr.NeedsConfirmation(userProc) {
		t.Error("level 0 should not need confirmation for user process")
	}

	// Level 1: notify for system
	mgr = safety.NewManager(nil, nil, safety.ConsentNotifySystem)
	if mgr.NeedsConfirmation(userProc) {
		t.Error("level 1 should not need confirmation for user process")
	}
	if !mgr.NeedsConfirmation(sysProc) {
		t.Error("level 1 should need confirmation for system process")
	}

	// Level 2: confirm all
	mgr = safety.NewManager(nil, nil, safety.ConsentConfirmAll)
	if !mgr.NeedsConfirmation(userProc) {
		t.Error("level 2 should need confirmation for all processes")
	}

	// Level 3: monitor only
	mgr = safety.NewManager(nil, nil, safety.ConsentMonitorOnly)
	if !mgr.IsMonitorOnly() {
		t.Error("level 3 should be monitor only")
	}
}
