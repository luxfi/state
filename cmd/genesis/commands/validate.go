package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewValidateCommand() *cobra.Command {
	var (
		genesisPath string
		networkName string
		chainID     int64
		verbose     bool
		strict      bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate genesis file integrity",
		Long: `Validate a genesis file for correctness, completeness, and compatibility
with Lux Network requirements.`,
		Example: `  # Validate genesis file
  genesis validate --genesis ./genesis/lux-mainnet.json

  # Validate with network requirements
  genesis validate \
    --genesis ./genesis/zoo-mainnet.json \
    --network zoo-mainnet \
    --chain-id 200200

  # Strict validation with verbose output
  genesis validate \
    --genesis ./genesis/network.json \
    --strict \
    --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if genesisPath == "" {
				return fmt.Errorf("genesis file path is required")
			}

			validator, err := genesis.NewValidator(genesis.ValidatorConfig{
				GenesisPath: genesisPath,
				NetworkName: networkName,
				ChainID:     chainID,
				Strict:      strict,
				Verbose:     verbose,
			})
			if err != nil {
				return fmt.Errorf("failed to create validator: %w", err)
			}

			fmt.Printf("Validating genesis file: %s\n\n", genesisPath)

			result, err := validator.Validate()
			if err != nil {
				return fmt.Errorf("validation failed: %w", err)
			}

			// Display results
			fmt.Printf("=== Genesis Validation Results ===\n\n")
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Chain ID: %d\n", result.ChainID)
			fmt.Printf("Network: %s\n", result.NetworkName)

			fmt.Printf("\n📊 Statistics:\n")
			fmt.Printf("  Total Accounts: %d\n", result.TotalAccounts)
			fmt.Printf("  Total Supply: %s\n", result.TotalSupply)
			fmt.Printf("  Contract Accounts: %d\n", result.ContractAccounts)
			fmt.Printf("  EOA Accounts: %d\n", result.EOAAccounts)

			if result.AssetInfo != nil {
				fmt.Printf("\n💎 Assets:\n")
				for _, asset := range result.AssetInfo {
					fmt.Printf("  %s: %d holders, %s supply\n", 
						asset.Name, asset.Holders, asset.TotalSupply)
				}
			}

			// Check results
			fmt.Printf("\n✓ Checks Passed: %d\n", result.ChecksPassed)
			fmt.Printf("✗ Checks Failed: %d\n", result.ChecksFailed)

			if verbose && len(result.Details) > 0 {
				fmt.Printf("\nDetailed Checks:\n")
				for _, check := range result.Details {
					status := "✓"
					if !check.Passed {
						status = "✗"
					}
					fmt.Printf("  %s %s: %s\n", status, check.Name, check.Message)
				}
			}

			// Show errors
			if len(result.Errors) > 0 {
				fmt.Printf("\n❌ Errors:\n")
				for _, err := range result.Errors {
					fmt.Printf("  - %s\n", err)
				}
			}

			// Show warnings
			if len(result.Warnings) > 0 {
				fmt.Printf("\n⚠️  Warnings:\n")
				for _, warning := range result.Warnings {
					fmt.Printf("  - %s\n", warning)
				}
			}

			// Final verdict
			if result.Status == "VALID" {
				fmt.Printf("\n✅ Genesis validation passed!\n")
				if result.ReadyForProduction {
					fmt.Printf("   This genesis is ready for production deployment.\n")
				} else {
					fmt.Printf("   This genesis is valid but may need review for production use.\n")
				}
			} else {
				fmt.Printf("\n❌ Genesis validation failed!\n")
				fmt.Printf("   Please fix the errors above before using this genesis.\n")
				return fmt.Errorf("validation failed with %d errors", len(result.Errors))
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&genesisPath, "genesis", "g", "", "Path to genesis file")
	cmd.Flags().StringVarP(&networkName, "network", "n", "", "Expected network name")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Expected chain ID")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed validation output")
	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict validation mode")

	cmd.MarkFlagRequired("genesis")

	return cmd
}