package monitor

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"
)

// ProcessMonitor scans /proc and tracks process metrics.
type ProcessMonitor struct {
	mu           sync.RWMutex
	processes    map[int]*ProcessInfo
	classifier   *Classifier
	scanInterval time.Duration
	historySize  int
	metrics      *SystemMetrics
	onUpdate     func([]*ProcessInfo, *SystemMetrics)

	prevProcs map[int]*ProcessInfo // previous scan for delta calculation
}

// NewProcessMonitor creates a new monitor.
func NewProcessMonitor(scanInterval time.Duration, historySize int, protectedNames []string) *ProcessMonitor {
	return &ProcessMonitor{
		processes:    make(map[int]*ProcessInfo),
		prevProcs:    make(map[int]*ProcessInfo),
		classifier:   NewClassifier(protectedNames),
		scanInterval: scanInterval,
		historySize:  historySize,
		metrics:      &SystemMetrics{},
	}
}

// OnUpdate sets a callback invoked after each scan.
func (m *ProcessMonitor) OnUpdate(fn func([]*ProcessInfo, *SystemMetrics)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onUpdate = fn
}

// Start begins the scanning loop.
func (m *ProcessMonitor) Start(ctx context.Context) error {
	// Perform initial scan
	m.scan()

	ticker := time.NewTicker(m.scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			m.scan()
		}
	}
}

// Stop is a no-op; use context cancellation.
func (m *ProcessMonitor) Stop() {}

// Processes returns a snapshot of current processes sorted by CPU desc.
func (m *ProcessMonitor) Processes() []*ProcessInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	procs := make([]*ProcessInfo, 0, len(m.processes))
	for _, p := range m.processes {
		procs = append(procs, p)
	}
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].CPU > procs[j].CPU
	})
	return procs
}

// SystemMetrics returns the latest system metrics.
func (m *ProcessMonitor) SystemMetrics() *SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

func (m *ProcessMonitor) scan() {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return
	}

	newProcs := make(map[int]*ProcessInfo)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		proc, err := parseProcessInfo(pid)
		if err != nil {
			continue
		}

		// Carry forward previous jiffies for CPU calculation
		if prev, ok := m.prevProcs[pid]; ok {
			proc.prevUtime = prev.prevUtime
			proc.prevStime = prev.prevStime
			proc.lastScan = prev.lastScan

			// Re-parse stat to get proper CPU delta
			_ = proc.parseStat()

			// Calculate trends
			proc.CPUTrend = proc.CPU - prev.CPU
			proc.MemoryTrend = proc.Memory - prev.Memory
		}

		// Classify
		proc.Category = m.classifier.Classify(proc)

		newProcs[pid] = proc
	}

	// Get system metrics
	sysMetrics := GetSystemMetrics()
	sysMetrics.NumProcs = len(newProcs)

	// Calculate total CPU from all processes
	var totalCPU float64
	for _, p := range newProcs {
		totalCPU += p.CPU
	}
	sysMetrics.TotalCPU = totalCPU

	m.mu.Lock()
	m.prevProcs = m.processes
	m.processes = newProcs
	m.metrics = sysMetrics
	callback := m.onUpdate
	m.mu.Unlock()

	if callback != nil {
		procs := m.Processes()
		callback(procs, sysMetrics)
	}
}

// FormatProcessLine formats a process for text output.
func FormatProcessLine(p *ProcessInfo) string {
	return fmt.Sprintf("%7d %-20s %-10s %6.1f%% %6.1f%% %8.1fMB %-10s %s",
		p.PID, truncate(p.Name, 20), p.User,
		p.CPU, p.Memory, p.MemoryMB,
		p.Category, truncate(p.Cmdline, 40))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
