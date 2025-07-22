package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

// NewScanNFTHoldersCommand creates the scan-nft-holders command
func NewScanNFTHoldersCommand() *cobra.Command {
	var (
		rpc             string
		contractAddress string
		fromBlock       uint64
		toBlock         uint64
		outputCSV       string
		outputJSON      string
		includeTokenIDs bool
		topN            int
		showDistribution bool
	)

	cmd := &cobra.Command{
		Use:   "scan-nft-holders",
		Short: "Scan for current NFT holders",
		Long: `Scans blockchain to find all current NFT holders by processing Transfer events.

This command:
- Tracks all NFT transfers from contract deployment
- Builds current ownership state
- Can export holder lists with token counts
- Shows distribution statistics`,
		Example: `  # Scan EGG NFT holders on BSC
  teleport scan-nft-holders \
    --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
    --output egg-holders.csv

  # Show top holders and distribution
  teleport scan-nft-holders \
    --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
    --top 20 --show-distribution

  # Include token IDs in output
  teleport scan-nft-holders \
    --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
    --include-token-ids --output-json holders-detailed.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create scanner config
			config := &scanner.NFTHolderScanConfig{
				RPC:             rpc,
				ContractAddress: contractAddress,
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				IncludeTokenIDs: includeTokenIDs,
			}

			// Create scanner
			nftScanner, err := scanner.NewNFTHolderScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create NFT scanner: %w", err)
			}
			defer nftScanner.Close()

			log.Printf("Scanning NFT contract %s", contractAddress)

			// Scan holders
			holders, err := nftScanner.ScanHolders()
			if err != nil {
				return fmt.Errorf("failed to scan holders: %w", err)
			}

			log.Printf("Found %d unique holders", len(holders))

			// Calculate total NFTs
			totalNFTs := 0
			for _, holder := range holders {
				totalNFTs += holder.TokenCount
			}
			log.Printf("Total NFTs held: %d", totalNFTs)

			// Export to CSV if requested
			if outputCSV != "" {
				metadata := map[string]string{
					"Contract": contractAddress,
				}
				if err := scanner.ExportNFTHoldersToCSV(holders, outputCSV, metadata); err != nil {
					return fmt.Errorf("failed to export CSV: %w", err)
				}
				log.Printf("Exported holders to %s", outputCSV)
			}

			// Export to JSON if requested
			if outputJSON != "" {
				data := map[string]interface{}{
					"contract":     contractAddress,
					"totalHolders": len(holders),
					"totalNFTs":    totalNFTs,
					"holders":      holders,
				}
				if err := scanner.ExportToJSON(data, outputJSON); err != nil {
					return fmt.Errorf("failed to export JSON: %w", err)
				}
				log.Printf("Exported holders to %s", outputJSON)
			}

			// Show distribution if requested
			if showDistribution {
				distribution := scanner.GetHolderDistribution(holders)
				fmt.Printf("\n=== Holder Distribution ===\n")
				for category, count := range distribution {
					fmt.Printf("%-15s: %d holders\n", category, count)
				}
			}

			// Show top holders if requested
			if topN > 0 {
				topHolders, err := nftScanner.GetTopHolders(topN)
				if err != nil {
					return fmt.Errorf("failed to get top holders: %w", err)
				}

				fmt.Printf("\n=== Top %d Holders ===\n", topN)
				for i, holder := range topHolders {
					fmt.Printf("%2d. %s: %d NFTs\n", i+1, holder.Address, holder.TokenCount)
				}
			}

			// Summary
			fmt.Printf("\n=== Summary ===\n")
			fmt.Printf("Contract: %s\n", contractAddress)
			fmt.Printf("Total holders: %d\n", len(holders))
			fmt.Printf("Total NFTs: %d\n", totalNFTs)
			if len(holders) > 0 {
				avgHolding := float64(totalNFTs) / float64(len(holders))
				fmt.Printf("Average holding: %.2f NFTs\n", avgHolding)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "NFT contract address")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().BoolVar(&includeTokenIDs, "include-token-ids", false, "Include token IDs in output")
	cmd.Flags().IntVar(&topN, "top", 0, "Show top N holders")
	cmd.Flags().BoolVar(&showDistribution, "show-distribution", false, "Show holder distribution")

	cmd.MarkFlagRequired("contract")

	return cmd
}