package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/config"
	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/safety"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current system and APO status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	cfg := config.Global

	metrics := monitor.GetSystemMetrics()

	fmt.Println("╔══════════════════════════════════════════╗")
	fmt.Println("║     APO - AI-Powered Process Optimizer   ║")
	fmt.Println("╚══════════════════════════════════════════╝")
	fmt.Println()

	// Daemon status
	pidFile := filepath.Join(os.TempDir(), "apo.pid")
	if data, err := os.ReadFile(pidFile); err == nil {
		fmt.Printf("Daemon:     Running (PID: %s)\n", string(data))
	} else {
		fmt.Println("Daemon:     Not running")
	}
	fmt.Println()

	// System metrics
	fmt.Println("System Metrics:")
	fmt.Printf("  CPU Usage:    %.1f%%\n", metrics.TotalCPU)
	fmt.Printf("  Memory:       %.1f%% (%.0f MB / %.0f MB)\n",
		metrics.TotalMemory, metrics.TotalMemMB-metrics.FreeMemMB, metrics.TotalMemMB)
	fmt.Printf("  Load Average: %.2f, %.2f, %.2f\n",
		metrics.LoadAvg1, metrics.LoadAvg5, metrics.LoadAvg15)
	fmt.Printf("  Uptime:       %s\n", metrics.Uptime)
	fmt.Println()

	// Config summary
	fmt.Println("Configuration:")
	fmt.Printf("  AI Enabled:     %v\n", cfg.AI.Enabled)
	fmt.Printf("  AI Model:       %s\n", cfg.Anthropic.Model)
	fmt.Printf("  Consent Level:  %d (%s)\n", cfg.Safety.ConsentLevel,
		safety.LevelDescription(cfg.Safety.ConsentLevel))
	fmt.Printf("  Scan Interval:  %s\n", cfg.Monitoring.ScanInterval)
	fmt.Printf("  CPU Threshold:  %.1f%%\n", cfg.Monitoring.CPUThreshold)
	fmt.Printf("  Mem Threshold:  %.1f%%\n", cfg.Monitoring.MemoryThreshold)
	fmt.Printf("  Aggressiveness: %d/10\n", cfg.AI.Aggressiveness)

	apiKeySet := cfg.Anthropic.APIKey != ""
	fmt.Printf("  API Key Set:    %v\n", apiKeySet)
	fmt.Println()

	fmt.Printf("Protected Processes: %d configured\n", len(cfg.Safety.ProtectedProcs))
	fmt.Printf("Never Terminate:     %d configured\n", len(cfg.Safety.NeverTerminate))

	return nil
}
