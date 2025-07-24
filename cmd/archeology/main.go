package main

import (
	"fmt"
	"log"
	"os"

	"github.com/luxfi/genesis/cmd/archaeology/commands"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "archaeology",
		Short: "Blockchain Archaeology - Extract and analyze historical blockchain data",
		Long: `Blockchain Archaeology is a comprehensive tool for extracting, analyzing, and migrating
historical blockchain data from various EVM chains. It supports data extraction from
LevelDB and PebbleDB databases, external asset scanning from other chains, and
genesis file generation for Lux Network.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add subcommands
	rootCmd.AddCommand(
		// Core archaeology commands
		commands.NewExtractCommand(),
		commands.NewAnalyzeCommand(),
		commands.NewScanCommand(),
		commands.NewGenesisCommand(),
		commands.NewListCommand(),
		
		// Import commands
		commands.NewImportNFTCommand(),
		commands.NewImportTokenCommand(),
		
		// Scanner commands (modular blockchain scanners)
		commands.NewScanBurnsCommand(),
		commands.NewScanHoldersCommand(),
		commands.NewScanTransfersCommand(),
		commands.NewScanCurrentHoldersCommand(),
		commands.NewScanBurnsCachedCommand(),
	)

	// Execute
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}