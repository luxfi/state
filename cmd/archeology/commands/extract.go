package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/archeology"
)

func NewExtractCommand() *cobra.Command {
	var (
		srcPath      string
		dstPath      string
		chainID      int64
		networkName  string
		includeState bool
		limit        int
		verify       bool
	)

	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract blockchain data from raw database",
		Long: `Extract blockchain data from PebbleDB/LevelDB databases,
removing namespace prefixes and organizing data for genesis generation.`,
		Example: `  # Extract LUX mainnet data
  lux-archeology extract \
    --source /path/to/raw/pebbledb \
    --destination ./extracted/lux-96369 \
    --chain-id 96369 \
    --include-state

  # Extract with verification
  lux-archeology extract \
    --source /path/to/raw/db \
    --destination ./extracted/zoo-200200 \
    --network zoo-mainnet \
    --verify`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if srcPath == "" {
				return fmt.Errorf("source path is required")
			}
			if dstPath == "" {
				return fmt.Errorf("destination path is required")
			}
			if chainID == 0 && networkName == "" {
				return fmt.Errorf("either --chain-id or --network must be specified")
			}

			// Create extractor
			config := archeology.ExtractorConfig{
				SourcePath:   srcPath,
				DestPath:     dstPath,
				ChainID:      chainID,
				NetworkName:  networkName,
				IncludeState: includeState,
				Limit:        limit,
				Verify:       verify,
			}

			extractor, err := archeology.NewExtractor(config)
			if err != nil {
				return fmt.Errorf("failed to create extractor: %w", err)
			}

			// Run extraction
			log.Printf("Starting extraction from %s to %s", srcPath, dstPath)
			result, err := extractor.Extract()
			if err != nil {
				return fmt.Errorf("extraction failed: %w", err)
			}

			// Display results
			fmt.Printf("\nExtraction completed successfully!\n")
			fmt.Printf("Chain ID: %d\n", result.ChainID)
			fmt.Printf("Blocks extracted: %d\n", result.BlockCount)
			fmt.Printf("Accounts found: %d\n", result.AccountCount)
			fmt.Printf("Storage entries: %d\n", result.StorageCount)
			fmt.Printf("Output path: %s\n", result.OutputPath)

			if verify {
				fmt.Printf("\n✅ Data verification passed\n")
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&srcPath, "source", "s", "", "Source database path")
	cmd.Flags().StringVarP(&dstPath, "destination", "d", "", "Destination path for extracted data")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Chain ID for namespace removal")
	cmd.Flags().StringVarP(&networkName, "network", "n", "", "Network name (e.g., lux-mainnet, zoo-mainnet)")
	cmd.Flags().BoolVar(&includeState, "include-state", true, "Include state data (accounts, storage)")
	cmd.Flags().IntVar(&limit, "limit", 0, "Limit number of blocks to extract (0 for all)")
	cmd.Flags().BoolVar(&verify, "verify", false, "Verify extracted data integrity")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("destination")

	return cmd
}