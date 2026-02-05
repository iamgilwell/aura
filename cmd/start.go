package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

var daemon bool

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start APO in background daemon mode",
	RunE:  runStart,
}

func init() {
	startCmd.Flags().BoolVar(&daemon, "daemon", false, "run as background daemon")
}

func runStart(cmd *cobra.Command, args []string) error {
	pidFile := filepath.Join(os.TempDir(), "apo.pid")

	// Check if already running
	if data, err := os.ReadFile(pidFile); err == nil {
		pidStr := string(data)
		if pid, err := strconv.Atoi(pidStr); err == nil {
			if process, err := os.FindProcess(pid); err == nil {
				// Check if process is alive
				if err := process.Signal(os.Signal(nil)); err == nil {
					return fmt.Errorf("APO daemon already running (PID: %d)", pid)
				}
			}
		}
	}

	if !daemon {
		fmt.Println("Starting APO in foreground monitoring mode...")
		fmt.Println("Use --daemon flag to run in background.")
		return runMonitor(cmd, args)
	}

	// Start as background process
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	daemonArgs := []string{"monitor"}
	if cfgFile != "" {
		daemonArgs = append(daemonArgs, "--config", cfgFile)
	}

	proc := exec.Command(executable, daemonArgs...)
	proc.Stdout = nil
	proc.Stderr = nil
	proc.Stdin = nil

	if err := proc.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	// Write PID file
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(proc.Process.Pid)), 0644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}

	fmt.Printf("APO daemon started (PID: %d)\n", proc.Process.Pid)
	fmt.Printf("PID file: %s\n", pidFile)
	fmt.Println("Use 'apo stop' to stop the daemon.")
	fmt.Println("Use 'apo logs --follow' to watch the log output.")

	return nil
}
