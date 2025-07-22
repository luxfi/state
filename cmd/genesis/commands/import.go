package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import external assets into genesis",
		Long:  `Import external assets from teleport scans into genesis configuration.`,
	}

	cmd.AddCommand(
		newImportAssetsCommand(),
		newImportValidatorsCommand(),
	)

	return cmd
}

func newImportAssetsCommand() *cobra.Command {
	var (
		assetFiles  []string
		genesisPath string
		outputPath  string
		merge       bool
	)

	return &cobra.Command{
		Use:   "assets",
		Short: "Import external assets into genesis",
		Example: `  # Import single asset file
  genesis import assets \
    --asset ./external/lux-nfts-ethereum.json \
    --genesis ./genesis/lux-mainnet.json

  # Import multiple assets
  genesis import assets \
    --asset ./external/nfts.json \
    --asset ./external/tokens.json \
    --genesis ./genesis/network.json \
    --merge`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(assetFiles) == 0 {
				return fmt.Errorf("at least one asset file is required")
			}
			if genesisPath == "" {
				return fmt.Errorf("genesis file is required")
			}

			importer, err := genesis.NewImporter(genesis.ImporterConfig{
				GenesisPath: genesisPath,
				OutputPath:  outputPath,
				Merge:       merge,
			})
			if err != nil {
				return fmt.Errorf("failed to create importer: %w", err)
			}

			fmt.Printf("Importing %d asset file(s) into genesis...\n", len(assetFiles))

			for _, assetFile := range assetFiles {
				fmt.Printf("  - %s\n", filepath.Base(assetFile))
				if err := importer.ImportAssetFile(assetFile); err != nil {
					return fmt.Errorf("failed to import %s: %w", assetFile, err)
				}
			}

			result, err := importer.Complete()
			if err != nil {
				return fmt.Errorf("import failed: %w", err)
			}

			fmt.Printf("\n✅ Import completed!\n")
			fmt.Printf("Assets imported: %d\n", result.AssetsImported)
			fmt.Printf("Accounts added: %d\n", result.AccountsAdded)
			fmt.Printf("Output: %s\n", result.OutputPath)

			return nil
		},
	}
}

func newImportValidatorsCommand() *cobra.Command {
	var (
		validatorFile string
		genesisPath   string
		outputPath    string
	)

	return &cobra.Command{
		Use:   "validators",
		Short: "Import validator set into genesis",
		Example: `  genesis import validators \
    --validators ./validators.json \
    --genesis ./genesis/network.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if validatorFile == "" {
				return fmt.Errorf("validator file is required")
			}
			if genesisPath == "" {
				return fmt.Errorf("genesis file is required")
			}

			// Implementation would go here
			fmt.Printf("Importing validators from %s\n", validatorFile)
			fmt.Printf("This feature is coming soon!\n")
			
			return nil
		},
	}
}