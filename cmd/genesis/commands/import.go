package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	chainDataPath string
	networkID     uint32
	blockchainID  string
)

// NewImportCommand returns the import command with subcommands
func NewImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import blockchain data into genesis",
		Long:  `Import existing blockchain data from various sources into genesis format`,
	}

	// Add historic import subcommand
	historicCmd := &cobra.Command{
		Use:   "historic",
		Short: "Import historic blockchain data",
		Long:  `Import historic blockchain data from PebbleDB for migration to new network`,
		RunE:  runImportHistoric,
	}

	historicCmd.Flags().StringVar(&chainDataPath, "chain-data", "", "Path to chain data directory")
	historicCmd.Flags().Uint32Var(&networkID, "network-id", 96369, "Network ID")
	historicCmd.Flags().StringVar(&blockchainID, "blockchain-id", "", "Blockchain ID")
	historicCmd.MarkFlagRequired("chain-data")

	cmd.AddCommand(historicCmd)
	return cmd
}

func runImportHistoric(cmd *cobra.Command, args []string) error {
	// Verify chain data exists
	if _, err := os.Stat(chainDataPath); os.IsNotExist(err) {
		return fmt.Errorf("chain data path does not exist: %s", chainDataPath)
	}

	fmt.Printf("Importing historic data from %s\n", chainDataPath)
	fmt.Printf("Network ID: %d\n", networkID)
	
	// Map network IDs to blockchain IDs if not provided
	if blockchainID == "" {
		switch networkID {
		case 96369:
			blockchainID = "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"
		case 96368:
			blockchainID = "2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG"
		case 200200:
			blockchainID = "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM"
		case 200201:
			blockchainID = "2usKC5aApgWQWwanB4LL6QPoqxR1bWWjPCtemBYbZvxkNfcnbj"
		case 36911:
			blockchainID = "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1"
		default:
			return fmt.Errorf("unknown network ID %d, please provide --blockchain-id", networkID)
		}
		fmt.Printf("Using blockchain ID: %s\n", blockchainID)
	}

	// Check if data exists for this blockchain
	dataPath := filepath.Join(chainDataPath, blockchainID)
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		// Try pebbledb subdirectory
		dataPath = filepath.Join(chainDataPath, blockchainID, "db", "pebbledb")
		if _, err := os.Stat(dataPath); os.IsNotExist(err) {
			return fmt.Errorf("no data found for blockchain %s in %s", blockchainID, chainDataPath)
		}
	}

	fmt.Printf("Found blockchain data at: %s\n", dataPath)
	
	// TODO: Implement actual import logic
	// This would involve:
	// 1. Reading the PebbleDB data
	// 2. Extracting state and blocks
	// 3. Converting to genesis format
	// 4. Merging with existing genesis if needed
	
	fmt.Println("✅ Import functionality prepared - integration with luxd/geth import pending")
	return nil
}