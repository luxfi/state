package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewExportCommand() *cobra.Command {
	var (
		inputFiles   []string
		outputPath   string
		format       string // genesis, csv, json
		chainType    string // C-Chain, X-Chain, P-Chain
		mergeMode    string // combine, separate
		includeProof bool
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export scanned assets to various formats",
		Long: `Export scanned assets from teleport to different formats suitable for
genesis generation, analysis, or cross-chain verification.`,
		Example: `  # Export to genesis format
  teleport export \
    --input ./scans/nfts.json \
    --input ./scans/tokens.json \
    --format genesis \
    --output ./exports/assets-genesis.json

  # Export to CSV for analysis
  teleport export \
    --input ./scans/holders.json \
    --format csv \
    --output ./exports/holders.csv

  # Export with merkle proofs
  teleport export \
    --input ./scans/snapshot.json \
    --format genesis \
    --chain-type C-Chain \
    --include-proof \
    --output ./exports/verified-genesis.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(inputFiles) == 0 {
				return fmt.Errorf("at least one input file is required")
			}
			if outputPath == "" {
				return fmt.Errorf("output path is required")
			}

			exporter, err := bridge.NewExporter(bridge.ExporterConfig{
				InputFiles:   inputFiles,
				OutputPath:   outputPath,
				Format:       format,
				ChainType:    chainType,
				MergeMode:    mergeMode,
				IncludeProof: includeProof,
			})
			if err != nil {
				return fmt.Errorf("failed to create exporter: %w", err)
			}

			fmt.Printf("Exporting %d file(s) to %s format...\n", len(inputFiles), format)
			for _, file := range inputFiles {
				fmt.Printf("  - %s\n", filepath.Base(file))
			}

			result, err := exporter.Export()
			if err != nil {
				return fmt.Errorf("export failed: %w", err)
			}

			// Display results
			fmt.Printf("\n✅ Export completed!\n\n")
			fmt.Printf("Format: %s\n", result.Format)
			fmt.Printf("Records Exported: %d\n", result.RecordsExported)
			fmt.Printf("Total Value: %s\n", result.TotalValue)

			if result.AssetsSummary != nil {
				fmt.Printf("\nAssets Summary:\n")
				for assetType, count := range result.AssetsSummary {
					fmt.Printf("  %s: %d\n", assetType, count)
				}
			}

			if includeProof && result.ProofInfo != nil {
				fmt.Printf("\nMerkle Proof Info:\n")
				fmt.Printf("  Root Hash: %s\n", result.ProofInfo.RootHash)
				fmt.Printf("  Tree Height: %d\n", result.ProofInfo.TreeHeight)
				fmt.Printf("  Proofs Generated: %d\n", result.ProofInfo.ProofsGenerated)
			}

			fmt.Printf("\nOutput files:\n")
			for _, file := range result.OutputFiles {
				fmt.Printf("  - %s\n", file)
			}

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&inputFiles, "input", "i", nil, "Input files from scans")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path")
	cmd.Flags().StringVarP(&format, "format", "f", "genesis", "Export format (genesis, csv, json)")
	cmd.Flags().StringVar(&chainType, "chain-type", "C-Chain", "Chain type for genesis format")
	cmd.Flags().StringVar(&mergeMode, "merge-mode", "combine", "How to merge multiple inputs (combine, separate)")
	cmd.Flags().BoolVar(&includeProof, "include-proof", false, "Include merkle proofs for verification")

	return cmd
}