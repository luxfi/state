package commands

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewGenerateCommand() *cobra.Command {
	var (
		networkName     string
		chainID         int64
		chainType       string
		dataPath        string
		externalPath    string
		templatePath    string
		outputPath      string
		assetPrefix     string
		includeTestData bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate genesis file from extracted data",
		Long: `Generate a complete genesis file by combining extracted blockchain data
with external asset imports and applying network-specific configurations.`,
		Example: `  # Generate LUX mainnet genesis
  genesis generate \
    --network lux-mainnet \
    --data ./data/extracted/lux-96369 \
    --external ./data/external/ \
    --output ./genesis/lux-mainnet-96369.json

  # Generate ZOO mainnet with custom template
  genesis generate \
    --network zoo-mainnet \
    --chain-id 200200 \
    --data ./data/extracted/zoo-200200 \
    --template ./templates/zoo-custom.json \
    --output ./genesis/zoo-mainnet.json

  # Generate test network with test accounts
  genesis generate \
    --network lux-testnet \
    --data ./data/extracted/lux-96368 \
    --include-test-data \
    --output ./genesis/lux-testnet.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if networkName == "" && chainID == 0 {
				return fmt.Errorf("either --network or --chain-id must be specified")
			}
			if dataPath == "" {
				return fmt.Errorf("--data path is required")
			}

			// Create config
			config := genesis.GeneratorConfig{
				NetworkName:     networkName,
				ChainID:         chainID,
				ChainType:       chainType,
				DataPath:        dataPath,
				ExternalPath:    externalPath,
				TemplatePath:    templatePath,
				OutputPath:      outputPath,
				AssetPrefix:     assetPrefix,
				IncludeTestData: includeTestData,
			}

			// Set defaults
			if config.OutputPath == "" {
				config.OutputPath = fmt.Sprintf("./genesis/%s.json", networkName)
			}

			generator, err := genesis.NewGenerator(config)
			if err != nil {
				return fmt.Errorf("failed to create generator: %w", err)
			}

			log.Printf("Generating genesis for %s (chain ID: %d)", networkName, chainID)
			log.Printf("Data source: %s", dataPath)
			
			if externalPath != "" {
				log.Printf("External assets: %s", externalPath)
			}

			result, err := generator.Generate()
			if err != nil {
				return fmt.Errorf("generation failed: %w", err)
			}

			// Display results
			fmt.Printf("\n✅ Genesis generated successfully!\n\n")
			fmt.Printf("Network: %s\n", result.NetworkName)
			fmt.Printf("Chain ID: %d\n", result.ChainID)
			fmt.Printf("Chain Type: %s\n", result.ChainType)
			fmt.Printf("Total Accounts: %d\n", result.TotalAccounts)
			fmt.Printf("Total Balance: %s\n", result.TotalBalance)

			if len(result.Assets) > 0 {
				fmt.Printf("\nAssets Included:\n")
				for _, asset := range result.Assets {
					fmt.Printf("  - %s: %d holders, %s total supply\n", 
						asset.Name, asset.Holders, asset.TotalSupply)
				}
			}

			if len(result.ExternalAssets) > 0 {
				fmt.Printf("\nExternal Assets:\n")
				for _, ext := range result.ExternalAssets {
					fmt.Printf("  - %s from %s: %d items\n", 
						ext.Type, ext.Source, ext.Count)
				}
			}

			fmt.Printf("\nGenesis file written to: %s\n", result.OutputPath)
			fmt.Printf("File size: %s\n", result.FileSize)

			// Validate the generated file
			fmt.Printf("\nValidating genesis file...\n")
			if err := generator.ValidateOutput(); err != nil {
				return fmt.Errorf("genesis validation failed: %w", err)
			}
			fmt.Printf("✅ Genesis validation passed\n")

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&networkName, "network", "n", "", "Network name (e.g., lux-mainnet)")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Chain ID")
	cmd.Flags().StringVar(&chainType, "chain-type", "C", "EVM C-Chain")
	cmd.Flags().StringVarP(&dataPath, "data", "d", "", "Path to extracted blockchain data")
	cmd.Flags().StringVarP(&externalPath, "external", "e", "", "Path to external asset imports")
	cmd.Flags().StringVarP(&templatePath, "template", "t", "", "Genesis template file")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output genesis file path")
	cmd.Flags().StringVar(&assetPrefix, "asset-prefix", "", "Asset name prefix (e.g., LUX, ZOO)")
	cmd.Flags().BoolVar(&includeTestData, "include-test-data", false, "Include test accounts and data")

	cmd.MarkFlagRequired("data")

	return cmd
}