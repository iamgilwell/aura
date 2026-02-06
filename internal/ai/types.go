package ai

import (
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/iamgilwell/aura/internal/monitor"
)

// Action is the AI-recommended action for a process.
type Action string

const (
	ActionTerminate Action = "terminate"
	ActionKeep      Action = "keep"
	ActionNotify    Action = "notify"
	ActionThrottle  Action = "throttle"
)

// DecisionRequest contains the data sent to the AI for evaluation.
type DecisionRequest struct {
	Process      *monitor.ProcessInfo
	SystemState  *monitor.SystemMetrics
	ProcessList  []*monitor.ProcessInfo // top consumers for context
	History      []*DecisionResponse    // recent decisions for context
}

// DecisionResponse is the AI's evaluation of a process.
type DecisionResponse struct {
	ProcessPID   int       `json:"process_pid"`
	ProcessName  string    `json:"process_name"`
	Action       Action    `json:"action"`
	Confidence   float64   `json:"confidence"`
	Reason       string    `json:"reason"`
	RiskScore    float64   `json:"risk_score"`
	SavingsWatt  float64   `json:"savings_watt"`
	Timestamp    time.Time `json:"timestamp"`
	FromCache    bool      `json:"from_cache"`
}

// ProcessSignature generates a cache key for a process based on its name and resource pattern.
func ProcessSignature(proc *monitor.ProcessInfo) string {
	// Bucket CPU and memory to avoid cache misses from tiny fluctuations
	cpuBucket := int(proc.CPU / 5) * 5   // Round to nearest 5%
	memBucket := int(proc.Memory / 5) * 5

	raw := fmt.Sprintf("%s|%s|%d|%d|%s",
		proc.Name,
		proc.User,
		cpuBucket,
		memBucket,
		proc.Category,
	)

	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash[:8])
}
