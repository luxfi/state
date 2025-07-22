package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewVerifyCommand() *cobra.Command {
	var (
		scanFile       string
		genesisFile    string
		chainData      string
		verifyBalances bool
		verifyHolders  bool
		verifyMetadata bool
		outputReport   string
	)

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify scanned assets against on-chain data",
		Long: `Verify that scanned asset data matches on-chain state and genesis requirements.
This ensures data integrity before migration.`,
		Example: `  # Verify scan against blockchain
  teleport verify \
    --scan ./scans/tokens.json \
    --chain-data ./data/extracted/lux-96369

  # Verify against genesis file
  teleport verify \
    --scan ./scans/nfts.json \
    --genesis ./genesis/network.json \
    --verify-holders

  # Full verification with report
  teleport verify \
    --scan ./scans/assets.json \
    --chain-data ./data/extracted/zoo-200200 \
    --verify-balances \
    --verify-metadata \
    --output ./reports/verification.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if scanFile == "" {
				return fmt.Errorf("scan file is required")
			}
			if genesisFile == "" && chainData == "" {
				return fmt.Errorf("either --genesis or --chain-data is required")
			}

			verifier, err := bridge.NewVerifier(bridge.VerifierConfig{
				ScanFile:       scanFile,
				GenesisFile:    genesisFile,
				ChainData:      chainData,
				VerifyBalances: verifyBalances,
				VerifyHolders:  verifyHolders,
				VerifyMetadata: verifyMetadata,
			})
			if err != nil {
				return fmt.Errorf("failed to create verifier: %w", err)
			}

			fmt.Printf("Verifying scan data: %s\n", scanFile)
			if genesisFile != "" {
				fmt.Printf("Against genesis: %s\n", genesisFile)
			}
			if chainData != "" {
				fmt.Printf("Against chain data: %s\n", chainData)
			}

			result, err := verifier.Verify()
			if err != nil {
				return fmt.Errorf("verification failed: %w", err)
			}

			// Display results
			fmt.Printf("\n=== Verification Results ===\n\n")
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Records Verified: %d\n", result.RecordsVerified)
			fmt.Printf("Checks Performed: %d\n", result.ChecksPerformed)

			// Show check results
			fmt.Printf("\n✓ Passed: %d\n", result.ChecksPassed)
			fmt.Printf("✗ Failed: %d\n", result.ChecksFailed)
			fmt.Printf("⚠ Warnings: %d\n", len(result.Warnings))

			if verifyBalances && result.BalanceCheck != nil {
				fmt.Printf("\n💰 Balance Verification:\n")
				fmt.Printf("  Accounts Checked: %d\n", result.BalanceCheck.AccountsChecked)
				fmt.Printf("  Matches: %d\n", result.BalanceCheck.Matches)
				fmt.Printf("  Mismatches: %d\n", result.BalanceCheck.Mismatches)
				if result.BalanceCheck.TotalDifference != "0" {
					fmt.Printf("  Total Difference: %s\n", result.BalanceCheck.TotalDifference)
				}
			}

			if verifyHolders && result.HolderCheck != nil {
				fmt.Printf("\n👥 Holder Verification:\n")
				fmt.Printf("  Expected Holders: %d\n", result.HolderCheck.ExpectedHolders)
				fmt.Printf("  Found Holders: %d\n", result.HolderCheck.FoundHolders)
				fmt.Printf("  Missing: %d\n", result.HolderCheck.Missing)
				fmt.Printf("  Extra: %d\n", result.HolderCheck.Extra)
			}

			// Show discrepancies
			if len(result.Discrepancies) > 0 {
				fmt.Printf("\n❌ Discrepancies Found:\n")
				for i, disc := range result.Discrepancies {
					if i >= 10 {
						fmt.Printf("  ... and %d more\n", len(result.Discrepancies)-10)
						break
					}
					fmt.Printf("  - %s: %s\n", disc.Type, disc.Description)
				}
			}

			// Show warnings
			if len(result.Warnings) > 0 {
				fmt.Printf("\n⚠️  Warnings:\n")
				for _, warning := range result.Warnings {
					fmt.Printf("  - %s\n", warning)
				}
			}

			// Save report if requested
			if outputReport != "" {
				if err := verifier.SaveReport(outputReport); err != nil {
					return fmt.Errorf("failed to save report: %w", err)
				}
				fmt.Printf("\nVerification report saved to: %s\n", outputReport)
			}

			// Final verdict
			if result.Status == "VERIFIED" {
				fmt.Printf("\n✅ Verification passed!\n")
				fmt.Printf("   Data is ready for migration.\n")
			} else if result.Status == "PARTIAL" {
				fmt.Printf("\n⚠️  Partial verification!\n")
				fmt.Printf("   Some checks failed. Review discrepancies before proceeding.\n")
			} else {
				fmt.Printf("\n❌ Verification failed!\n")
				fmt.Printf("   Please fix the issues above before migration.\n")
				return fmt.Errorf("verification failed")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&scanFile, "scan", "s", "", "Scan file to verify")
	cmd.Flags().StringVarP(&genesisFile, "genesis", "g", "", "Genesis file to verify against")
	cmd.Flags().StringVar(&chainData, "chain-data", "", "Extracted chain data to verify against")
	cmd.Flags().BoolVar(&verifyBalances, "verify-balances", false, "Verify account balances")
	cmd.Flags().BoolVar(&verifyHolders, "verify-holders", true, "Verify holder addresses")
	cmd.Flags().BoolVar(&verifyMetadata, "verify-metadata", false, "Verify NFT metadata")
	cmd.Flags().StringVarP(&outputReport, "output", "o", "", "Save verification report")

	cmd.MarkFlagRequired("scan")

	return cmd
}