package commands

import (
	"fmt"
	"log"

	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/spf13/cobra"
)

// NewGenesisCommand creates the genesis subcommand
func NewGenesisCommand() *cobra.Command {
	var (
		nftCSV      string
		tokenCSV    string
		accountsCSV string
		outputPath  string
		chainType   string
		assetPrefix string
	)

	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Generate X-Chain or P-Chain genesis with external assets",
		Long: `Generate genesis files for X-Chain or P-Chain that include external NFTs,
tokens, and migrated account balances. This integrates data from the scan command
and creates a complete genesis file with all historical assets.`,
		Example: `  # Generate X-Chain genesis with all assets
  archeology genesis --nft-csv exports/lux-nfts-ethereum.csv --token-csv exports/zoo-tokens-bsc.csv --accounts-csv exports/7777-accounts.csv --output configs/xchain-genesis-complete.json

  # Generate with only NFTs for validator staking
  archeology genesis --nft-csv exports/lux-nfts-ethereum.csv --chain p-chain --output configs/pchain-genesis.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if nftCSV == "" && tokenCSV == "" && accountsCSV == "" {
				return fmt.Errorf("at least one CSV input is required (--nft-csv, --token-csv, or --accounts-csv)")
			}

			// Create genesis config
			config := genesis.Config{
				NFTDataPath:      nftCSV,
				TokenDataPath:    tokenCSV,
				AccountsDataPath: accountsCSV,
				OutputPath:       outputPath,
				ChainType:        chainType,
				AssetPrefix:      assetPrefix,
			}

			// Create genesis generator
			gen, err := genesis.NewGenerator(config)
			if err != nil {
				return fmt.Errorf("failed to create genesis generator: %w", err)
			}

			// Generate genesis
			log.Printf("Generating %s genesis with external assets...", chainType)
			result, err := gen.Generate()
			if err != nil {
				return fmt.Errorf("genesis generation failed: %w", err)
			}

			// Print summary
			log.Printf("\n=== Genesis Generation Summary ===")
			log.Printf("Chain type: %s", result.ChainType)
			log.Printf("Genesis timestamp: %s", result.Timestamp)
			log.Printf("Total asset types: %d", result.TotalAssetTypes)
			
			if len(result.NFTCollections) > 0 {
				log.Printf("\nNFT Collections:")
				for collection, stats := range result.NFTCollections {
					log.Printf("  - %s: %d NFTs, %d holders", collection, stats.Count, stats.Holders)
					if stats.StakingEnabled {
						log.Printf("    ✓ Staking enabled (power: %s)", stats.StakingPower)
					}
				}
			}

			if len(result.TokenAssets) > 0 {
				log.Printf("\nToken Assets:")
				for asset, stats := range result.TokenAssets {
					log.Printf("  - %s: %s total supply, %d holders", asset, stats.TotalSupply, stats.Holders)
				}
			}

			if result.AccountsMigrated > 0 {
				log.Printf("\nAccount Migration:")
				log.Printf("  - Total accounts: %d", result.AccountsMigrated)
				log.Printf("  - Validator eligible: %d", result.ValidatorEligible)
				log.Printf("  - Total balance: %s", result.TotalBalance)
			}

			log.Printf("\n✅ Genesis file generated: %s", result.OutputFile)

			// Additional notes
			if chainType == "x-chain" && len(result.NFTCollections) > 0 {
				log.Printf("\nNote: NFT holders can stake their NFTs as validators")
				log.Printf("      Each NFT type has different staking power equivalent to LUX tokens")
			}

			return nil
		},
	}

	// Define flags
	cmd.Flags().StringVar(&nftCSV, "nft-csv", "", "Path to scanned NFT data CSV")
	cmd.Flags().StringVar(&tokenCSV, "token-csv", "", "Path to scanned token data CSV")
	cmd.Flags().StringVar(&accountsCSV, "accounts-csv", "", "Path to account balances CSV (e.g., 7777 export)")
	cmd.Flags().StringVar(&outputPath, "output", "configs/genesis-complete.json", "Output genesis file path")
	cmd.Flags().StringVar(&chainType, "chain", "x-chain", "Chain type: x-chain or p-chain")
	cmd.Flags().StringVar(&assetPrefix, "asset-prefix", "LUX", "Asset name prefix (LUX, ZOO, SPC, HANZO)")

	return cmd
}