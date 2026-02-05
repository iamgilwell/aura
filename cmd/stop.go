package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the APO daemon",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	pidFile := filepath.Join(os.TempDir(), "apo.pid")

	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("APO daemon is not running (no PID file found)")
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return fmt.Errorf("invalid PID file: %w", err)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process %d: %w", pid, err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		// Process might already be dead
		os.Remove(pidFile)
		return fmt.Errorf("sending SIGTERM to PID %d: %w (PID file cleaned up)", pid, err)
	}

	os.Remove(pidFile)
	fmt.Printf("APO daemon stopped (PID: %d)\n", pid)
	return nil
}
