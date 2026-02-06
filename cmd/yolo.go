package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/ai"
	"github.com/iamgilwell/aura/internal/config"
	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/notification"
	"github.com/iamgilwell/aura/internal/power"
	"github.com/iamgilwell/aura/internal/process"
	"github.com/iamgilwell/aura/internal/safety"
)

var yoloCmd = &cobra.Command{
	Use:   "yolo",
	Short: "AI-driven automatic process optimization",
	Long: `YOLO mode: AI evaluates all user processes and automatically terminates
those deemed wasteful. Use with caution! Sets consent level to 0 (automatic).`,
	RunE: runYolo,
}

func runYolo(cmd *cobra.Command, args []string) error {
	cfg := config.Global

	if cfg.Anthropic.APIKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is required for yolo mode. Set it in config or environment")
	}

	notifier, err := notification.NewNotifier(cfg.Notifications.LogFile, cfg.Notifications.ColorEnabled, cfg.Notifications.Verbose)
	if err != nil {
		return fmt.Errorf("creating notifier: %w", err)
	}
	defer notifier.Close()

	auditor, err := notification.NewAuditor(cfg.Notifications.AuditFile)
	if err != nil {
		return fmt.Errorf("creating auditor: %w", err)
	}
	defer auditor.Close()

	notifier.Warn("YOLO MODE ACTIVATED - AI will automatically terminate wasteful processes!")
	auditor.LogEvent("yolo_start", "YOLO mode activated")

	// Set consent to automatic
	safetyMgr := safety.NewManager(cfg.Safety.ProtectedProcs, cfg.Safety.NeverTerminate, safety.ConsentAutomatic)
	procMgr := process.NewManager(safetyMgr, cfg.Safety.TerminateTimeout)

	cache := ai.NewCache(cfg.AI.CacheSize, cfg.AI.CacheTTL)
	aiEngine := ai.NewEngine(cfg.Anthropic.APIKey, cfg.Anthropic.Model, cache, cfg.AI.ConfidenceThreshold, cfg.AI.Aggressiveness)

	powerCalc := power.NewCalculator(cfg.Power.CPUWattPerPercent, cfg.Power.MemoryWattPerMB, cfg.Power.DiskWattPerMBps)
	powerMetrics := power.NewMetrics()

	mon := monitor.NewProcessMonitor(
		cfg.Monitoring.ScanInterval,
		cfg.Monitoring.HistorySize,
		cfg.Safety.ProtectedProcs,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mon.OnUpdate(func(procs []*monitor.ProcessInfo, metrics *monitor.SystemMetrics) {
		// Evaluate user processes with high resource usage
		for _, proc := range procs {
			if proc.Category != monitor.CategoryUser {
				continue
			}
			if proc.CPU < cfg.Monitoring.CPUThreshold && proc.Memory < cfg.Monitoring.MemoryThreshold {
				continue
			}

			decision, err := aiEngine.EvaluateProcess(ctx, proc, metrics)
			if err != nil {
				notifier.Error(fmt.Sprintf("AI evaluation failed for PID %d: %v", proc.PID, err))
				continue
			}

			notifier.Decision(decision)
			auditor.LogDecision(decision)

			if decision.Action == ai.ActionTerminate && decision.Confidence >= cfg.AI.ConfidenceThreshold {
				if safetyMgr.IsProtected(proc) {
					notifier.Warn(fmt.Sprintf("Skipping protected process: %s (PID %d)", proc.Name, proc.PID))
					continue
				}

				notifier.Warn(fmt.Sprintf("Terminating: %s (PID %d) - %s", proc.Name, proc.PID, decision.Reason))
				if err := procMgr.SafeTerminate(proc, false); err != nil {
					notifier.Error(fmt.Sprintf("Failed to terminate PID %d: %v", proc.PID, err))
				} else {
					savings := powerCalc.EstimateSavings(proc)
					powerMetrics.RecordSaving(proc.Name, proc.PID, savings, decision.Reason)
					auditor.LogTermination(proc.PID, proc.Name, decision.Reason)
					notifier.Info(fmt.Sprintf("Terminated PID %d, saved %.2fW (total: %.2fW)",
						proc.PID, savings, powerMetrics.TotalSaved()))
				}
			}
		}

		// Periodic status
		fmt.Printf("\n[%s] Processes: %d | CPU: %.1f%% | Mem: %.1f%% | Power saved: %.2fW | Monthly projection: %.2f kWh\n",
			time.Now().Format("15:04:05"),
			metrics.NumProcs, metrics.TotalCPU, metrics.TotalMemory,
			powerMetrics.TotalSaved(), powerMetrics.MonthlyProjection())
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		notifier.Info("Shutting down YOLO mode...")
		auditor.LogEvent("yolo_stop", fmt.Sprintf("Total power saved: %.2fW", powerMetrics.TotalSaved()))
		cancel()
	}()

	return mon.Start(ctx)
}
