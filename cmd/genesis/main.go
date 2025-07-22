package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/cmd/genesis/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "genesis",
		Short: "Genesis generation and network launch tool",
		Long: `genesis is the official tool for generating genesis files and launching
Lux Network L1 and L2 chains. It combines extracted blockchain data with
external assets to create complete genesis configurations.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add commands
	rootCmd.AddCommand(
		commands.NewGenerateCommand(),
		commands.NewImportCommand(),
		commands.NewMergeCommand(),
		commands.NewValidateCommand(),
		commands.NewLaunchCommand(),
		commands.NewDeployCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}