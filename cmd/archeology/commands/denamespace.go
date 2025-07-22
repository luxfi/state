package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/archeology"
)

func NewDenameSpaceCommand() *cobra.Command {
	var (
		srcPath  string
		dstPath  string
		chainID  int64
		dryRun   bool
		progress bool
	)

	cmd := &cobra.Command{
		Use:   "denamespace",
		Short: "Remove namespace prefixes from PebbleDB",
		Long: `Remove the 33-byte namespace prefixes from PebbleDB/LevelDB databases.
This is required to make the data readable by standard Ethereum tools.`,
		Example: `  # Remove namespacing from LUX mainnet
  lux-archeology denamespace \
    --source /path/to/raw/pebbledb \
    --destination ./denamespaced/lux-96369 \
    --chain-id 96369

  # Dry run to see what would be processed
  lux-archeology denamespace \
    --source /path/to/raw/db \
    --destination ./output \
    --chain-id 200200 \
    --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if srcPath == "" {
				return fmt.Errorf("source path is required")
			}
			if dstPath == "" {
				return fmt.Errorf("destination path is required")
			}
			if chainID == 0 {
				return fmt.Errorf("chain ID is required")
			}

			config := archeology.DenamespacerConfig{
				SourcePath:   srcPath,
				DestPath:     dstPath,
				ChainID:      chainID,
				DryRun:       dryRun,
				ShowProgress: progress,
			}

			denamespacer, err := archeology.NewDenamespacer(config)
			if err != nil {
				return fmt.Errorf("failed to create denamespacer: %w", err)
			}

			log.Printf("Starting denamespace operation...")
			log.Printf("Source: %s", srcPath)
			log.Printf("Destination: %s", dstPath)
			log.Printf("Chain ID: %d", chainID)

			if dryRun {
				log.Printf("DRY RUN MODE - No changes will be made")
			}

			result, err := denamespacer.Process()
			if err != nil {
				return fmt.Errorf("denamespace failed: %w", err)
			}

			fmt.Printf("\nDenamespace completed successfully!\n")
			fmt.Printf("Keys processed: %d\n", result.KeysProcessed)
			fmt.Printf("Keys with namespace: %d\n", result.KeysWithNamespace)
			fmt.Printf("Keys without namespace: %d\n", result.KeysWithoutNamespace)
			fmt.Printf("Errors: %d\n", result.Errors)

			if dryRun {
				fmt.Printf("\nThis was a dry run - no data was written\n")
			} else {
				fmt.Printf("\nOutput written to: %s\n", dstPath)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&srcPath, "source", "s", "", "Source database path")
	cmd.Flags().StringVarP(&dstPath, "destination", "d", "", "Destination path")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Chain ID for namespace")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without writing")
	cmd.Flags().BoolVar(&progress, "progress", true, "Show progress during processing")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("destination")
	cmd.MarkFlagRequired("chain-id")

	return cmd
}