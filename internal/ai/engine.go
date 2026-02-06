package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/iamgilwell/aura/internal/monitor"
)

// Engine is the AI decision engine using Anthropic Claude.
type Engine struct {
	client           anthropic.Client
	model            anthropic.Model
	cache            *Cache
	confidenceThresh float64
	aggressiveness   int

	mu         sync.RWMutex
	history    []*DecisionResponse
	maxHistory int
}

// NewEngine creates a new AI decision engine.
func NewEngine(apiKey, model string, cache *Cache, confidenceThresh float64, aggressiveness int) *Engine {
	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}

	return &Engine{
		client:           anthropic.NewClient(opts...),
		model:            anthropic.Model(model),
		cache:            cache,
		confidenceThresh: confidenceThresh,
		aggressiveness:   aggressiveness,
		maxHistory:       100,
	}
}

// EvaluateProcess asks the AI whether a process should be terminated.
func (e *Engine) EvaluateProcess(ctx context.Context, proc *monitor.ProcessInfo, state *monitor.SystemMetrics) (*DecisionResponse, error) {
	// Check cache first
	sig := ProcessSignature(proc)
	if cached, ok := e.cache.Get(sig); ok {
		e.addToHistory(cached)
		return cached, nil
	}

	prompt := e.buildPrompt(proc, state)

	msg, err := e.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     e.model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt(e.aggressiveness)},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return e.fallbackDecision(proc, fmt.Errorf("API call failed: %w", err)), nil
	}

	// Extract text response
	var responseText string
	for _, block := range msg.Content {
		if block.Type == "text" {
			responseText = block.Text
			break
		}
	}

	decision, err := e.parseResponse(responseText, proc)
	if err != nil {
		return e.fallbackDecision(proc, err), nil
	}

	// Cache the decision
	e.cache.Put(sig, decision)
	e.addToHistory(decision)

	return decision, nil
}

// DecisionHistory returns recent AI decisions.
func (e *Engine) DecisionHistory() []*DecisionResponse {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*DecisionResponse, len(e.history))
	copy(result, e.history)
	return result
}

func (e *Engine) addToHistory(d *DecisionResponse) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history = append(e.history, d)
	if len(e.history) > e.maxHistory {
		e.history = e.history[len(e.history)-e.maxHistory:]
	}
}

func systemPrompt(aggressiveness int) string {
	return fmt.Sprintf(`You are Aura, an AI-powered Linux process optimizer. Your job is to evaluate running processes and decide whether they should be terminated to save resources and power.

Aggressiveness level: %d/10 (1=very conservative, only terminate clearly wasteful processes; 10=aggressive, terminate anything not essential)

RULES:
- Never recommend terminating kernel threads, init/systemd, or critical system services
- Consider process dependencies — terminating a parent may orphan children
- Factor in: CPU usage, memory usage, IO activity, process age, user vs system
- Provide confidence score (0.0-1.0) and risk assessment (0.0-1.0)
- Estimate power savings in watts

Respond ONLY with valid JSON in this exact format:
{
  "action": "terminate|keep|notify|throttle",
  "confidence": 0.0-1.0,
  "reason": "brief explanation",
  "risk_score": 0.0-1.0,
  "savings_watt": 0.0
}`, aggressiveness)
}

func (e *Engine) buildPrompt(proc *monitor.ProcessInfo, state *monitor.SystemMetrics) string {
	var sb strings.Builder
	sb.WriteString("Evaluate this process for potential termination:\n\n")
	sb.WriteString(fmt.Sprintf("Process: %s (PID: %d)\n", proc.Name, proc.PID))
	sb.WriteString(fmt.Sprintf("User: %s (UID: %d)\n", proc.User, proc.UID))
	sb.WriteString(fmt.Sprintf("CPU: %.1f%%\n", proc.CPU))
	sb.WriteString(fmt.Sprintf("Memory: %.1f%% (%.1f MB)\n", proc.Memory, proc.MemoryMB))
	sb.WriteString(fmt.Sprintf("IO Read: %d bytes, IO Write: %d bytes\n", proc.IORead, proc.IOWrite))
	sb.WriteString(fmt.Sprintf("State: %s\n", proc.State))
	sb.WriteString(fmt.Sprintf("Parent PID: %d\n", proc.PPid))
	sb.WriteString(fmt.Sprintf("Category: %s\n", proc.Category))
	sb.WriteString(fmt.Sprintf("Command: %s\n", proc.Cmdline))
	sb.WriteString(fmt.Sprintf("CPU Trend: %+.1f%%\n", proc.CPUTrend))
	sb.WriteString(fmt.Sprintf("Memory Trend: %+.1f%%\n", proc.MemoryTrend))

	sb.WriteString("\nSystem State:\n")
	sb.WriteString(fmt.Sprintf("Total CPU Usage: %.1f%%\n", state.TotalCPU))
	sb.WriteString(fmt.Sprintf("Memory Usage: %.1f%% (%.0f MB used / %.0f MB total)\n",
		state.TotalMemory, state.TotalMemMB-state.FreeMemMB, state.TotalMemMB))
	sb.WriteString(fmt.Sprintf("Load Average: %.2f, %.2f, %.2f\n",
		state.LoadAvg1, state.LoadAvg5, state.LoadAvg15))
	sb.WriteString(fmt.Sprintf("Total Processes: %d\n", state.NumProcs))

	return sb.String()
}

func (e *Engine) parseResponse(text string, proc *monitor.ProcessInfo) (*DecisionResponse, error) {
	// Find JSON in response
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := text[start : end+1]

	var raw struct {
		Action      string  `json:"action"`
		Confidence  float64 `json:"confidence"`
		Reason      string  `json:"reason"`
		RiskScore   float64 `json:"risk_score"`
		SavingsWatt float64 `json:"savings_watt"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON response: %w", err)
	}

	return &DecisionResponse{
		ProcessPID:  proc.PID,
		ProcessName: proc.Name,
		Action:      Action(raw.Action),
		Confidence:  raw.Confidence,
		Reason:      raw.Reason,
		RiskScore:   raw.RiskScore,
		SavingsWatt: raw.SavingsWatt,
		Timestamp:   time.Now(),
	}, nil
}

func (e *Engine) fallbackDecision(proc *monitor.ProcessInfo, err error) *DecisionResponse {
	return &DecisionResponse{
		ProcessPID:  proc.PID,
		ProcessName: proc.Name,
		Action:      ActionKeep,
		Confidence:  0.0,
		Reason:      fmt.Sprintf("AI unavailable (%v) - defaulting to keep", err),
		RiskScore:   0.0,
		SavingsWatt: 0.0,
		Timestamp:   time.Now(),
	}
}
