package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/cmd/teleport/commands"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "teleport",
		Short: "External blockchain asset scanner and importer",
		Long: `teleport scans external blockchains (Ethereum, BSC, etc.) for NFTs and tokens
that should be included in Lux Network genesis. It handles the complete workflow
of discovering, validating, and formatting external assets for genesis inclusion.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add commands
	rootCmd.AddCommand(
		// Original commands
		commands.NewMigrateCommand(),
		commands.NewScanNFTCommand(),
		commands.NewScanTokenCommand(),
		commands.NewZooMigrateCommand(),
		commands.NewScanEggHoldersCommand(),
		commands.NewExportCommand(),
		commands.NewVerifyCommand(),
		commands.NewListCommand(),
		
		// New modular scanner commands
		commands.NewScanTokenBurnsCommand(),
		commands.NewScanNFTHoldersCommand(),
		commands.NewScanTokenTransfersCommand(),
		commands.NewCheckCrossChainBalancesCommand(),
		commands.NewZooCrossReferenceCommand(),
		commands.NewZooCrossReferenceV2Command(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}