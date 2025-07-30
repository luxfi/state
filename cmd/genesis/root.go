package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "genesis",
	Short: "Genesis migration and blockchain management tool",
	Long: `Genesis is a comprehensive tool for managing blockchain data migration,
inspection, and manipulation for the Lux Network ecosystem.

It provides subcommands for:
- Importing and migrating blockchain data
- Inspecting and analyzing database contents  
- Fixing and cleaning up data issues
- Converting between different formats
- Launching nodes with migrated data`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Initialize all subcommands
	rootCmd.AddCommand(
		newImportCmd(),
		newInspectCmd(),
		newAnalyzeCmd(),
		newFixCmd(),
		newMigrateCmd(),
		newDebugCmd(),
		newLaunchCmd(),
		newGenerateCmd(),
		newValidatorsCmd(),
		newExtractCmd(),
		newToolsCmd(),
		newCopyCmd(), // copy-to-node command
	)
}