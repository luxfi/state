package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

// NewZooMigrateCommand creates the zoo-migrate command
func NewZooMigrateCommand() *cobra.Command {
	var (
		rpc            string
		fromBlock      uint64
		toBlock        uint64
		includeBurns   bool
		includeEggNFTs bool
		outputPath     string
	)

	cmd := &cobra.Command{
		Use:   "zoo-migrate",
		Short: "Scan and migrate Zoo tokens from BSC including burns",
		Long: `Scans the BSC blockchain for Zoo token holders and includes those who burned
tokens to the dead address (0x000000000000000000000000000000000000dEaD).

This special migration command:
- Tracks all current Zoo token holders
- Includes users who burned tokens to the dead address
- Optionally includes EGG NFT holders for additional benefits
- Generates genesis allocations with burn amounts included`,
		Example: `  # Scan Zoo tokens with burns included
  teleport zoo-migrate --include-burns --output zoo-genesis.json

  # Include EGG NFT holders
  teleport zoo-migrate --include-burns --include-egg-nfts --output zoo-complete.json

  # Scan specific block range
  teleport zoo-migrate --from-block 20000000 --to-block 25000000 --include-burns`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := bridge.ZooMigrationConfig{
				RPC:            rpc,
				FromBlock:      fromBlock,
				ToBlock:        toBlock,
				IncludeBurns:   includeBurns,
				IncludeEggNFTs: includeEggNFTs,
				OutputPath:     outputPath,
			}

			// Create scanner
			scanner, err := bridge.NewZooMigrationScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}
			defer scanner.Close()

			// Perform scan
			log.Printf("Starting Zoo token migration scan on BSC...")
			result, err := scanner.ScanZooMigration()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Export results
			if err := scanner.Export(result); err != nil {
				return fmt.Errorf("failed to export results: %w", err)
			}

			// Print summary
			fmt.Printf("\nZoo Migration Summary:\n")
			fmt.Printf("====================\n")
			fmt.Printf("Token Address: %s\n", result.TokenAddress)
			fmt.Printf("Total Supply: %s\n", result.TotalSupply)
			fmt.Printf("Burned Supply: %s\n", result.BurnedSupply)
			fmt.Printf("Unique Holders: %d\n", result.UniqueHolders)
			fmt.Printf("Holders with Burns: %d\n", result.HoldersWithBurns)
			if includeEggNFTs {
				fmt.Printf("EGG NFT Holders: %d\n", result.EggNFTHolders)
			}
			fmt.Printf("\nResults exported to: %s\n", outputPath)

			// Show top holders
			fmt.Printf("\nTop 10 Holders (including burns):\n")
			limit := 10
			if len(result.Holders) < limit {
				limit = len(result.Holders)
			}
			for i := 0; i < limit; i++ {
				holder := result.Holders[i]
				fmt.Printf("%2d. %s: %s", i+1, holder.Address[:10]+"..."+holder.Address[len(holder.Address)-8:], holder.TotalAllocation)
				if holder.BurnedAmount != "" {
					fmt.Printf(" (burned: %s)", holder.BurnedAmount)
				}
				if holder.HasEggNFT {
					fmt.Printf(" [%d EGG NFTs]", holder.EggNFTCount)
				}
				fmt.Println()
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "BSC RPC endpoint (default: BSC public RPC)")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block for scanning")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block for scanning (default: latest)")
	cmd.Flags().BoolVar(&includeBurns, "include-burns", true, "Include burned amounts in allocations")
	cmd.Flags().BoolVar(&includeEggNFTs, "include-egg-nfts", false, "Include EGG NFT holder data")
	cmd.Flags().StringVar(&outputPath, "output", "zoo-migration.json", "Output file path")

	return cmd
}