package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iamgilwell/aura/internal/config"
)

var (
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "aura",
	Short: "Aura - AI-Powered Process Optimizer",
	Long: `Aura (AI-Powered Process Optimizer) monitors Linux processes,
uses Anthropic Claude AI for intelligent termination decisions,
and provides an htop-like TUI for interactive management.

Run 'aura interactive' for the full TUI experience, or
'aura monitor' for a simple text-based process view.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "enable verbose output")

	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(yoloCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(logsCmd)
}

func initConfig() {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if verbose {
		cfg.Notifications.Verbose = true
	}
	config.Global = cfg
}
