package notification

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/iamgilwell/aura/internal/ai"
)

// AuditEntry is a single audit log entry.
type AuditEntry struct {
	Timestamp   time.Time          `json:"timestamp"`
	Event       string             `json:"event"`
	Decision    *ai.DecisionResponse `json:"decision,omitempty"`
	Details     string             `json:"details,omitempty"`
}

// Auditor writes an append-only audit trail.
type Auditor struct {
	mu   sync.Mutex
	file *os.File
}

// NewAuditor creates a new auditor.
func NewAuditor(filePath string) (*Auditor, error) {
	if filePath == "" {
		return &Auditor{}, nil
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening audit file: %w", err)
	}

	return &Auditor{file: f}, nil
}

// Close closes the audit file.
func (a *Auditor) Close() {
	if a.file != nil {
		a.file.Close()
	}
}

// LogDecision records an AI decision to the audit trail.
func (a *Auditor) LogDecision(decision *ai.DecisionResponse) {
	a.log(AuditEntry{
		Timestamp: time.Now(),
		Event:     "ai_decision",
		Decision:  decision,
	})
}

// LogTermination records a process termination.
func (a *Auditor) LogTermination(pid int, name string, reason string) {
	a.log(AuditEntry{
		Timestamp: time.Now(),
		Event:     "termination",
		Details:   fmt.Sprintf("pid=%d name=%s reason=%s", pid, name, reason),
	})
}

// LogEvent records a general event.
func (a *Auditor) LogEvent(event, details string) {
	a.log(AuditEntry{
		Timestamp: time.Now(),
		Event:     event,
		Details:   details,
	})
}

func (a *Auditor) log(entry AuditEntry) {
	if a.file == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	a.file.Write(data)
	a.file.Write([]byte("\n"))
}
