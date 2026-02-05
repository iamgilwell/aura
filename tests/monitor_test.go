package tests

import (
	"testing"
	"time"

	"github.com/iamgilwell/aura/internal/monitor"
)

func TestGetSystemMetrics(t *testing.T) {
	metrics := monitor.GetSystemMetrics()

	if metrics.TotalMemMB <= 0 {
		t.Error("expected positive total memory")
	}
	if metrics.FreeMemMB < 0 {
		t.Error("expected non-negative free memory")
	}
	if metrics.TotalMemory < 0 || metrics.TotalMemory > 100 {
		t.Errorf("memory percentage out of range: %.1f", metrics.TotalMemory)
	}
	if metrics.Uptime <= 0 {
		t.Error("expected positive uptime")
	}
}

func TestProcessMonitorCreation(t *testing.T) {
	mon := monitor.NewProcessMonitor(
		2*time.Second,
		100,
		[]string{"systemd", "init"},
	)

	if mon == nil {
		t.Fatal("expected non-nil monitor")
	}

	// Initial process list should be empty before Start
	procs := mon.Processes()
	if len(procs) != 0 {
		t.Errorf("expected 0 processes before start, got %d", len(procs))
	}
}

func TestProcessCategoryString(t *testing.T) {
	tests := []struct {
		cat  monitor.ProcessCategory
		want string
	}{
		{monitor.CategoryUser, "User"},
		{monitor.CategorySystem, "System"},
		{monitor.CategoryKernel, "Kernel"},
		{monitor.CategoryEssential, "Essential"},
	}

	for _, tt := range tests {
		got := tt.cat.String()
		if got != tt.want {
			t.Errorf("category %d: got %q, want %q", tt.cat, got, tt.want)
		}
	}
}

func TestClassifier(t *testing.T) {
	c := monitor.NewClassifier([]string{"sshd", "nginx"})

	tests := []struct {
		name string
		proc *monitor.ProcessInfo
		want monitor.ProcessCategory
	}{
		{
			name: "PID 1 is kernel",
			proc: &monitor.ProcessInfo{PID: 1, Name: "systemd", PPid: 0, UID: 0, Cmdline: "/sbin/init"},
			want: monitor.CategoryKernel,
		},
		{
			name: "kthreadd child is kernel",
			proc: &monitor.ProcessInfo{PID: 100, Name: "kworker/0:0", PPid: 2, UID: 0, Cmdline: "[kworker/0:0]"},
			want: monitor.CategoryKernel,
		},
		{
			name: "protected process is essential",
			proc: &monitor.ProcessInfo{PID: 500, Name: "sshd", PPid: 1, UID: 0, Cmdline: "/usr/sbin/sshd"},
			want: monitor.CategoryEssential,
		},
		{
			name: "root daemon is system",
			proc: &monitor.ProcessInfo{PID: 600, Name: "cron", PPid: 1, UID: 0, Cmdline: "/usr/sbin/cron"},
			want: monitor.CategorySystem,
		},
		{
			name: "user process",
			proc: &monitor.ProcessInfo{PID: 1000, Name: "firefox", PPid: 800, UID: 1000, Cmdline: "/usr/bin/firefox"},
			want: monitor.CategoryUser,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Classify(tt.proc)
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
