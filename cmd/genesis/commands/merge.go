package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewMergeCommand() *cobra.Command {
	var (
		inputFiles []string
		outputPath string
		chainType  string
		validate   bool
	)

	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge multiple genesis files",
		Long: `Merge multiple genesis files or data sources into a single unified genesis.
This is useful for combining data from different extraction runs or external sources.`,
		Example: `  # Merge multiple genesis files
  genesis merge \
    --input ./genesis/lux-base.json \
    --input ./genesis/lux-external.json \
    --output ./genesis/lux-complete.json

  # Merge with validation
  genesis merge \
    --input ./data/accounts.json \
    --input ./data/contracts.json \
    --input ./data/external.json \
    --output ./genesis/final.json \
    --validate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(inputFiles) < 2 {
				return fmt.Errorf("at least two input files are required")
			}
			if outputPath == "" {
				return fmt.Errorf("output path is required")
			}

			merger, err := genesis.NewMerger(genesis.MergerConfig{
				InputFiles: inputFiles,
				OutputPath: outputPath,
				ChainType:  chainType,
				Validate:   validate,
			})
			if err != nil {
				return fmt.Errorf("failed to create merger: %w", err)
			}

			fmt.Printf("Merging %d genesis files...\n", len(inputFiles))
			for _, file := range inputFiles {
				fmt.Printf("  - %s\n", file)
			}

			result, err := merger.Merge()
			if err != nil {
				return fmt.Errorf("merge failed: %w", err)
			}

			fmt.Printf("\n✅ Merge completed!\n\n")
			fmt.Printf("Total Accounts: %d\n", result.TotalAccounts)
			fmt.Printf("Total Balance: %s\n", result.TotalBalance)
			fmt.Printf("Assets Merged: %d\n", result.AssetsMerged)
			fmt.Printf("Conflicts Resolved: %d\n", result.ConflictsResolved)

			if len(result.Warnings) > 0 {
				fmt.Printf("\n⚠️  Warnings:\n")
				for _, warning := range result.Warnings {
					fmt.Printf("  - %s\n", warning)
				}
			}

			fmt.Printf("\nOutput written to: %s\n", outputPath)

			if validate {
				fmt.Printf("\n✅ Validation passed\n")
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&inputFiles, "input", "i", nil, "Input genesis files to merge")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for merged genesis")
	cmd.Flags().StringVar(&chainType, "chain-type", "", "Chain type (C-Chain, X-Chain, P-Chain)")
	cmd.Flags().BoolVar(&validate, "validate", true, "Validate merged genesis")

	return cmd
}