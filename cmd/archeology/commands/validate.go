package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/archeology"
)

func NewValidateCommand() *cobra.Command {
	var (
		dbPath      string
		checkState  bool
		checkBlocks bool
		verbose     bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate extracted blockchain data integrity",
		Long: `Validate the integrity of extracted blockchain data,
checking for consistency, completeness, and correctness.`,
		Example: `  # Validate all data
  lux-archeology validate --db ./extracted/lux-96369

  # Validate with detailed checks
  lux-archeology validate \
    --db ./extracted/zoo-200200 \
    --check-state \
    --check-blocks \
    --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dbPath == "" {
				return fmt.Errorf("database path is required")
			}

			config := archeology.ValidatorConfig{
				DatabasePath: dbPath,
				CheckState:   checkState,
				CheckBlocks:  checkBlocks,
				Verbose:      verbose,
			}

			validator, err := archeology.NewValidator(config)
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}

			fmt.Printf("Validating database: %s\n\n", dbPath)

			result, err := validator.Validate()
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			// Display results
			fmt.Printf("=== Validation Results ===\n\n")
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Blocks Validated: %d\n", result.BlocksValidated)
			fmt.Printf("Accounts Validated: %d\n", result.AccountsValidated)
			fmt.Printf("Errors Found: %d\n", len(result.Errors))
			fmt.Printf("Warnings: %d\n", len(result.Warnings))

			if checkBlocks && result.BlockchainIntegrity != nil {
				fmt.Printf("\nBlockchain Integrity:\n")
				fmt.Printf("  Continuous: %v\n", result.BlockchainIntegrity.Continuous)
				fmt.Printf("  Hash Chain Valid: %v\n", result.BlockchainIntegrity.HashChainValid)
				fmt.Printf("  First Block: %d\n", result.BlockchainIntegrity.FirstBlock)
				fmt.Printf("  Last Block: %d\n", result.BlockchainIntegrity.LastBlock)
				fmt.Printf("  Missing Blocks: %d\n", len(result.BlockchainIntegrity.MissingBlocks))
			}

			if checkState && result.StateIntegrity != nil {
				fmt.Printf("\nState Integrity:\n")
				fmt.Printf("  State Root Valid: %v\n", result.StateIntegrity.StateRootValid)
				fmt.Printf("  Account Hashes Valid: %v\n", result.StateIntegrity.AccountHashesValid)
				fmt.Printf("  Storage Hashes Valid: %v\n", result.StateIntegrity.StorageHashesValid)
			}

			// Show errors
			if len(result.Errors) > 0 {
				fmt.Printf("\n❌ Errors:\n")
				for i, err := range result.Errors {
					if !verbose && i >= 10 {
						fmt.Printf("  ... and %d more errors\n", len(result.Errors)-10)
						break
					}
					fmt.Printf("  - %s\n", err)
				}
			}

			// Show warnings
			if len(result.Warnings) > 0 {
				fmt.Printf("\n⚠️  Warnings:\n")
				for i, warn := range result.Warnings {
					if !verbose && i >= 10 {
						fmt.Printf("  ... and %d more warnings\n", len(result.Warnings)-10)
						break
					}
					fmt.Printf("  - %s\n", warn)
				}
			}

			if result.Status == "VALID" {
				fmt.Printf("\n✅ Database validation passed!\n")
			} else {
				fmt.Printf("\n❌ Database validation failed!\n")
				return fmt.Errorf("validation failed with %d errors", len(result.Errors))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&dbPath, "db", "d", "", "Path to extracted database")
	cmd.Flags().BoolVar(&checkState, "check-state", true, "Validate state integrity")
	cmd.Flags().BoolVar(&checkBlocks, "check-blocks", true, "Validate blockchain integrity")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed validation output")

	cmd.MarkFlagRequired("db")

	return cmd
}