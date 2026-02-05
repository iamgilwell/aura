package cmd

import (
	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/ai"
	"github.com/iamgilwell/aura/internal/config"
	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/notification"
	"github.com/iamgilwell/aura/internal/power"
	"github.com/iamgilwell/aura/internal/process"
	"github.com/iamgilwell/aura/internal/safety"
	"github.com/iamgilwell/aura/internal/ui"
)

var interactiveCmd = &cobra.Command{
	Use:   "interactive",
	Short: "Launch htop-like interactive TUI",
	Long:  `Launches the full interactive terminal UI with process table, AI decisions panel, and keyboard controls.`,
	RunE:  runInteractive,
}

func runInteractive(cmd *cobra.Command, args []string) error {
	cfg := config.Global

	notifier, err := notification.NewNotifier(cfg.Notifications.LogFile, cfg.Notifications.ColorEnabled, cfg.Notifications.Verbose)
	if err != nil {
		return err
	}
	defer notifier.Close()

	auditor, err := notification.NewAuditor(cfg.Notifications.AuditFile)
	if err != nil {
		return err
	}
	defer auditor.Close()

	safetyMgr := safety.NewManager(cfg.Safety.ProtectedProcs, cfg.Safety.NeverTerminate, cfg.Safety.ConsentLevel)
	procMgr := process.NewManager(safetyMgr, cfg.Safety.TerminateTimeout)

	cache := ai.NewCache(cfg.AI.CacheSize, cfg.AI.CacheTTL)
	var aiEngine *ai.Engine
	if cfg.AI.Enabled && cfg.Anthropic.APIKey != "" {
		aiEngine = ai.NewEngine(cfg.Anthropic.APIKey, cfg.Anthropic.Model, cache, cfg.AI.ConfidenceThreshold, cfg.AI.Aggressiveness)
	}

	powerCalc := power.NewCalculator(cfg.Power.CPUWattPerPercent, cfg.Power.MemoryWattPerMB, cfg.Power.DiskWattPerMBps)
	powerMetrics := power.NewMetrics()

	mon := monitor.NewProcessMonitor(
		cfg.Monitoring.ScanInterval,
		cfg.Monitoring.HistorySize,
		cfg.Safety.ProtectedProcs,
	)

	app := ui.NewApp(cfg, mon, aiEngine, safetyMgr, procMgr, powerCalc, powerMetrics, notifier, auditor)
	return app.Run()
}
