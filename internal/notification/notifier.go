package notification

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/iamgilwell/aura/internal/ai"
)

// Color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// Notifier handles terminal output and file logging.
type Notifier struct {
	mu           sync.Mutex
	logFile      *os.File
	logger       *log.Logger
	colorEnabled bool
	verbose      bool
}

// NewNotifier creates a new notifier.
func NewNotifier(logFilePath string, colorEnabled, verbose bool) (*Notifier, error) {
	n := &Notifier{
		colorEnabled: colorEnabled,
		verbose:      verbose,
	}

	if logFilePath != "" {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		n.logFile = f
		n.logger = log.New(f, "", log.LstdFlags)
	}

	return n, nil
}

// Close closes the log file.
func (n *Notifier) Close() {
	if n.logFile != nil {
		n.logFile.Close()
	}
}

// Info logs an informational message.
func (n *Notifier) Info(msg string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.colorEnabled {
		fmt.Printf("%s[INFO]%s %s\n", colorGreen, colorReset, msg)
	} else {
		fmt.Printf("[INFO] %s\n", msg)
	}

	if n.logger != nil {
		n.logger.Printf("[INFO] %s", msg)
	}
}

// Warn logs a warning message.
func (n *Notifier) Warn(msg string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.colorEnabled {
		fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, msg)
	} else {
		fmt.Printf("[WARN] %s\n", msg)
	}

	if n.logger != nil {
		n.logger.Printf("[WARN] %s", msg)
	}
}

// Error logs an error message.
func (n *Notifier) Error(msg string) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.colorEnabled {
		fmt.Printf("%s[ERROR]%s %s\n", colorRed, colorReset, msg)
	} else {
		fmt.Printf("[ERROR] %s\n", msg)
	}

	if n.logger != nil {
		n.logger.Printf("[ERROR] %s", msg)
	}
}

// Debug logs a debug message (only if verbose).
func (n *Notifier) Debug(msg string) {
	if !n.verbose {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.colorEnabled {
		fmt.Printf("%s[DEBUG]%s %s\n", colorCyan, colorReset, msg)
	} else {
		fmt.Printf("[DEBUG] %s\n", msg)
	}

	if n.logger != nil {
		n.logger.Printf("[DEBUG] %s", msg)
	}
}

// Decision logs an AI decision with color coding.
func (n *Notifier) Decision(d *ai.DecisionResponse) {
	n.mu.Lock()
	defer n.mu.Unlock()

	var color string
	switch d.Action {
	case ai.ActionTerminate:
		color = colorRed
	case ai.ActionNotify:
		color = colorYellow
	case ai.ActionKeep:
		color = colorGreen
	default:
		color = colorBlue
	}

	if n.colorEnabled {
		fmt.Printf("%s[AI]%s %s%-10s%s PID=%-7d %-20s conf=%.2f risk=%.2f save=%.1fW %s\n",
			colorBold, colorReset,
			color, string(d.Action), colorReset,
			d.ProcessPID, d.ProcessName,
			d.Confidence, d.RiskScore, d.SavingsWatt,
			d.Reason)
	} else {
		fmt.Printf("[AI] %-10s PID=%-7d %-20s conf=%.2f risk=%.2f save=%.1fW %s\n",
			string(d.Action), d.ProcessPID, d.ProcessName,
			d.Confidence, d.RiskScore, d.SavingsWatt,
			d.Reason)
	}

	if n.logger != nil {
		n.logger.Printf("[AI] %s PID=%d %s conf=%.2f risk=%.2f save=%.1fW %s",
			string(d.Action), d.ProcessPID, d.ProcessName,
			d.Confidence, d.RiskScore, d.SavingsWatt,
			d.Reason)
	}
}

// FormatTimestamp formats a time for display.
func FormatTimestamp(t time.Time) string {
	return t.Format("15:04:05")
}
