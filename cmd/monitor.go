package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/config"
	"github.com/iamgilwell/aura/internal/monitor"
	"github.com/iamgilwell/aura/internal/notification"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Foreground process monitoring with text output",
	Long:  `Monitors system processes and displays them in a text table, refreshing at the configured interval.`,
	RunE:  runMonitor,
}

func runMonitor(cmd *cobra.Command, args []string) error {
	cfg := config.Global

	notifier, err := notification.NewNotifier(cfg.Notifications.LogFile, cfg.Notifications.ColorEnabled, cfg.Notifications.Verbose)
	if err != nil {
		return fmt.Errorf("creating notifier: %w", err)
	}
	defer notifier.Close()

	notifier.Info("Aura Monitor starting...")

	mon := monitor.NewProcessMonitor(
		cfg.Monitoring.ScanInterval,
		cfg.Monitoring.HistorySize,
		cfg.Safety.ProtectedProcs,
	)

	mon.OnUpdate(func(procs []*monitor.ProcessInfo, metrics *monitor.SystemMetrics) {
		// Clear screen
		fmt.Print("\033[H\033[2J")

		// Header
		fmt.Printf("\033[1mAura Process Monitor\033[0m | Processes: %d | CPU: %.1f%% | Mem: %.1f%% | Load: %.2f\n",
			metrics.NumProcs, metrics.TotalCPU, metrics.TotalMemory, metrics.LoadAvg1)
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────────────")
		fmt.Printf("%7s %-20s %-10s %6s %6s %8s %-10s %s\n",
			"PID", "NAME", "USER", "CPU%", "MEM%", "MEM(MB)", "CATEGORY", "COMMAND")
		fmt.Println("─────────────────────────────────────────────────────────────────────────────────────────────────")

		// Show top 30 processes
		limit := 30
		if len(procs) < limit {
			limit = len(procs)
		}
		for _, p := range procs[:limit] {
			fmt.Println(monitor.FormatProcessLine(p))
		}

		fmt.Printf("\nPress Ctrl+C to exit | Scan interval: %s\n", cfg.Monitoring.ScanInterval)
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		notifier.Info("Shutting down...")
		cancel()
	}()

	return mon.Start(ctx)
}
