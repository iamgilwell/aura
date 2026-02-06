package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/config"
)

var followLogs bool

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View Aura log files",
	RunE:  runLogs,
}

func init() {
	logsCmd.Flags().BoolVar(&followLogs, "follow", false, "follow log output (like tail -f)")
}

func runLogs(cmd *cobra.Command, args []string) error {
	cfg := config.Global
	logFile := cfg.Notifications.LogFile

	f, err := os.Open(logFile)
	if err != nil {
		return fmt.Errorf("opening log file %s: %w", logFile, err)
	}
	defer f.Close()

	if !followLogs {
		// Print entire log file
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		return scanner.Err()
	}

	// Follow mode: read existing content then tail
	if _, err := io.Copy(os.Stdout, f); err != nil {
		return err
	}

	fmt.Println("--- Following log output (Ctrl+C to stop) ---")

	for {
		line := make([]byte, 4096)
		n, err := f.Read(line)
		if n > 0 {
			os.Stdout.Write(line[:n])
		}
		if err != nil {
			time.Sleep(500 * time.Millisecond)
		}
	}
}
