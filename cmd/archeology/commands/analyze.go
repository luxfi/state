package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/archeology"
)

func NewAnalyzeCommand() *cobra.Command {
	var (
		dbPath      string
		accountAddr string
		blockNumber int64
		outputPath  string
		networkName string
	)

	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze extracted blockchain data",
		Long: `Analyze extracted blockchain data to find accounts, balances,
storage entries, and other important information.`,
		Example: `  # Analyze all data
  lux-archeology analyze --db ./extracted/lux-96369

  # Find specific account
  lux-archeology analyze \
    --db ./extracted/zoo-200200 \
    --account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714

  # Analyze specific block
  lux-archeology analyze \
    --db ./extracted/spc-36911 \
    --block 1000000

  # Save analysis to file
  lux-archeology analyze \
    --db ./extracted/lux-96369 \
    --output analysis-report.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("database path is required")
			}

			// Check if path exists
			if _, err := os.Stat(dbPath); os.IsNotExist(err) {
				return fmt.Errorf("database path does not exist: %s", dbPath)
			}

			config := archeology.AnalyzerConfig{
				DatabasePath: dbPath,
				AccountAddr:  accountAddr,
				BlockNumber:  blockNumber,
				NetworkName:  networkName,
			}

			analyzer, err := archeology.NewAnalyzer(config)
			if err != nil {
				return fmt.Errorf("failed to create analyzer: %w", err)
			}

			fmt.Printf("Analyzing database: %s\n\n", dbPath)

			result, err := analyzer.Analyze()
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Display results
			fmt.Printf("=== Blockchain Analysis Results ===\n\n")
			fmt.Printf("Chain ID: %d\n", result.ChainID)
			fmt.Printf("Latest Block: %d\n", result.LatestBlock)
			fmt.Printf("Total Accounts: %d\n", result.TotalAccounts)
			fmt.Printf("Contract Accounts: %d\n", result.ContractAccounts)
			fmt.Printf("Total Balance: %s\n", result.TotalBalance)

			if result.GenesisBlock != nil {
				fmt.Printf("\nGenesis Block:\n")
				fmt.Printf("  Number: %d\n", result.GenesisBlock.Number)
				fmt.Printf("  Hash: %s\n", result.GenesisBlock.Hash)
				fmt.Printf("  Timestamp: %d\n", result.GenesisBlock.Timestamp)
			}

			if accountAddr != "" && result.AccountInfo != nil {
				fmt.Printf("\nAccount %s:\n", accountAddr)
				fmt.Printf("  Balance: %s\n", result.AccountInfo.Balance)
				fmt.Printf("  Nonce: %d\n", result.AccountInfo.Nonce)
				fmt.Printf("  Is Contract: %v\n", result.AccountInfo.IsContract)
				if result.AccountInfo.IsContract {
					fmt.Printf("  Code Size: %d bytes\n", result.AccountInfo.CodeSize)
					fmt.Printf("  Storage Entries: %d\n", result.AccountInfo.StorageCount)
				}
			}

			if len(result.TopAccounts) > 0 {
				fmt.Printf("\nTop %d Accounts by Balance:\n", len(result.TopAccounts))
				for i, acc := range result.TopAccounts {
					fmt.Printf("  %d. %s: %s\n", i+1, acc.Address, acc.Balance)
				}
			}

			// Save to file if requested
			if outputPath != "" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal results: %w", err)
				}

				if err := os.WriteFile(outputPath, data, 0644); err != nil {
					return fmt.Errorf("failed to write output file: %w", err)
				}

				fmt.Printf("\nAnalysis saved to: %s\n", outputPath)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&dbPath, "db", "d", "", "Path to extracted database")
	cmd.Flags().StringVarP(&accountAddr, "account", "a", "", "Analyze specific account")
	cmd.Flags().Int64VarP(&blockNumber, "block", "b", 0, "Analyze specific block")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Save analysis to JSON file")
	cmd.Flags().StringVarP(&networkName, "network", "n", "", "Network name for context")

	cmd.MarkFlagRequired("db")

	return cmd
}