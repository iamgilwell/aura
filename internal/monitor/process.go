package monitor

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// ProcessCategory classifies a process.
type ProcessCategory int

const (
	CategoryUser ProcessCategory = iota
	CategorySystem
	CategoryKernel
	CategoryEssential
)

func (c ProcessCategory) String() string {
	switch c {
	case CategoryUser:
		return "User"
	case CategorySystem:
		return "System"
	case CategoryKernel:
		return "Kernel"
	case CategoryEssential:
		return "Essential"
	default:
		return "Unknown"
	}
}

// ProcessInfo holds information about a running process.
type ProcessInfo struct {
	PID       int
	Name      string
	User      string
	UID       int
	CPU       float64
	Memory    float64
	MemoryMB  float64
	IORead    int64
	IOWrite   int64
	State     string
	PPid      int
	Cmdline   string
	StartTime time.Time
	Category  ProcessCategory

	// Deltas tracked across scans
	CPUTrend    float64
	MemoryTrend float64

	// Raw jiffies for CPU calculation
	prevUtime uint64
	prevStime uint64
	lastScan  time.Time
}

// SystemMetrics holds system-wide resource metrics.
type SystemMetrics struct {
	TotalCPU    float64
	TotalMemory float64
	TotalMemMB  float64
	FreeMemMB   float64
	LoadAvg1    float64
	LoadAvg5    float64
	LoadAvg15   float64
	NumProcs    int
	Uptime      time.Duration
}

// parseProcessInfo reads /proc/[pid] and returns ProcessInfo.
func parseProcessInfo(pid int) (*ProcessInfo, error) {
	proc := &ProcessInfo{PID: pid}

	// Parse /proc/[pid]/stat
	if err := proc.parseStat(); err != nil {
		return nil, fmt.Errorf("parsing stat for pid %d: %w", pid, err)
	}

	// Parse /proc/[pid]/status for UID and memory
	proc.parseStatus()

	// Parse /proc/[pid]/cmdline
	proc.parseCmdline()

	// Parse /proc/[pid]/io (may fail without root)
	proc.parseIO()

	// Resolve username from UID
	proc.resolveUser()

	return proc, nil
}

func (p *ProcessInfo) parseStat() error {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", p.PID))
	if err != nil {
		return err
	}

	content := string(data)

	// Name is between parentheses — handle names containing spaces/parens
	nameStart := strings.IndexByte(content, '(')
	nameEnd := strings.LastIndexByte(content, ')')
	if nameStart < 0 || nameEnd < 0 || nameEnd <= nameStart {
		return fmt.Errorf("invalid stat format for pid %d", p.PID)
	}

	p.Name = content[nameStart+1 : nameEnd]

	// Fields after the closing paren
	fields := strings.Fields(content[nameEnd+2:])
	if len(fields) < 20 {
		return fmt.Errorf("insufficient stat fields for pid %d", p.PID)
	}

	p.State = fields[0]

	ppid, _ := strconv.Atoi(fields[1])
	p.PPid = ppid

	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)

	starttime, _ := strconv.ParseUint(fields[19], 10, 64)
	clkTck := uint64(100) // sysconf(_SC_CLK_TCK) is typically 100 on Linux
	bootTime := getBootTime()
	p.StartTime = time.Unix(int64(bootTime+starttime/clkTck), 0)

	// Calculate CPU usage from jiffies delta
	now := time.Now()
	if !p.lastScan.IsZero() {
		elapsed := now.Sub(p.lastScan).Seconds()
		if elapsed > 0 {
			utimeDelta := float64(utime - p.prevUtime)
			stimeDelta := float64(stime - p.prevStime)
			p.CPU = ((utimeDelta + stimeDelta) / float64(clkTck)) / elapsed * 100.0
		}
	}

	p.prevUtime = utime
	p.prevStime = stime
	p.lastScan = now

	return nil
}

func (p *ProcessInfo) parseStatus() {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", p.PID))
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "Uid":
			fields := strings.Fields(val)
			if len(fields) > 0 {
				p.UID, _ = strconv.Atoi(fields[0])
			}
		case "VmRSS":
			// VmRSS is in kB
			fields := strings.Fields(val)
			if len(fields) > 0 {
				kb, _ := strconv.ParseFloat(fields[0], 64)
				p.MemoryMB = kb / 1024.0
			}
		}
	}

	// Calculate memory percentage
	totalMem := getTotalMemoryMB()
	if totalMem > 0 {
		p.Memory = (p.MemoryMB / totalMem) * 100.0
	}
}

func (p *ProcessInfo) parseCmdline() {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", p.PID))
	if err != nil {
		return
	}
	// Replace null bytes with spaces
	p.Cmdline = strings.ReplaceAll(strings.TrimRight(string(data), "\x00"), "\x00", " ")
	if p.Cmdline == "" {
		p.Cmdline = fmt.Sprintf("[%s]", p.Name)
	}
}

func (p *ProcessInfo) parseIO() {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/io", p.PID))
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		val, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		switch parts[0] {
		case "read_bytes":
			p.IORead = val
		case "write_bytes":
			p.IOWrite = val
		}
	}
}

func (p *ProcessInfo) resolveUser() {
	// Simple UID-to-name mapping via /etc/passwd
	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		p.User = strconv.Itoa(p.UID)
		return
	}

	uidStr := strconv.Itoa(p.UID)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		if len(fields) >= 3 && fields[2] == uidStr {
			p.User = fields[0]
			return
		}
	}
	p.User = uidStr
}

func getBootTime() uint64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "btime ") {
			val, _ := strconv.ParseUint(strings.Fields(line)[1], 10, 64)
			return val
		}
	}
	return 0
}

func getTotalMemoryMB() float64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseFloat(fields[1], 64)
				return kb / 1024.0
			}
		}
	}
	return 0
}

// GetSystemMetrics reads system-wide metrics.
func GetSystemMetrics() *SystemMetrics {
	m := &SystemMetrics{}

	totalMem := getTotalMemoryMB()
	m.TotalMemMB = totalMem

	// Free memory
	data, _ := os.ReadFile("/proc/meminfo")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, _ := strconv.ParseFloat(fields[1], 64)
				m.FreeMemMB = kb / 1024.0
			}
			break
		}
	}

	if totalMem > 0 {
		m.TotalMemory = ((totalMem - m.FreeMemMB) / totalMem) * 100.0
	}

	// Load average
	loadData, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		fields := strings.Fields(string(loadData))
		if len(fields) >= 3 {
			m.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
			m.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
			m.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}

	// Uptime
	uptimeData, err := os.ReadFile("/proc/uptime")
	if err == nil {
		fields := strings.Fields(string(uptimeData))
		if len(fields) >= 1 {
			secs, _ := strconv.ParseFloat(fields[0], 64)
			m.Uptime = time.Duration(secs * float64(time.Second))
		}
	}

	return m
}
