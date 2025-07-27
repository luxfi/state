package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/luxfi/geth/ethclient"
	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

// NewZooCrossReferenceCommand creates the zoo-cross-reference command
func NewZooCrossReferenceCommand() *cobra.Command {
	var (
		rpc              string
		fromBlock        uint64
		toBlock          uint64
		outputDir        string
		mainnetRPC       string
		mainnetFromBlock uint64
		mainnetToBlock   uint64
		knownHoldersFile string
	)

	cmd := &cobra.Command{
		Use:   "zoo-cross-reference",
		Short: "Cross-reference Zoo EGG purchases, burns, and mainnet delivery",
		Long: `Performs comprehensive cross-referencing of Zoo ecosystem:

This command will:
1. Scan BSC for all ZOO transfers to the EGG purchase address (0x28dad8427f127664365109c4a9406c8bc7844718)
2. Scan BSC for all ZOO burns to the dead address (0x000000000000000000000000000000000000dEaD)
3. Scan BSC for all EGG NFT holders (0x5bb68cf06289d54efde25155c88003be685356a8)
4. Cross-reference with Zoo mainnet (200200) to check delivery status
5. Generate CSV files for analysis

The following CSVs will be generated:
- zoo_egg_purchases.csv: All ZOO transfers for EGG purchases
- zoo_burns.csv: All ZOO burns to dead address with delivery status
- zoo_egg_nft_holders.csv: Current EGG NFT holders with ZOO equivalents
- zoo_cross_reference_report.txt: Summary report`,
		Example: `  # Basic cross-reference scan
  teleport zoo-cross-reference --output-dir ./zoo-analysis

  # Scan specific block range on BSC
  teleport zoo-cross-reference --from-block 20000000 --to-block 25000000 --output-dir ./zoo-analysis

  # Include mainnet cross-reference
  teleport zoo-cross-reference --mainnet-rpc http://localhost:9650/ext/bc/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/rpc

  # Use known holders file for validation
  teleport zoo-cross-reference --known-holders egg-holders.json --output-dir ./zoo-analysis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// Connect to BSC
			log.Printf("Connecting to BSC...")
			bscClient, err := ethclient.Dial(rpc)
			if err != nil {
				return fmt.Errorf("failed to connect to BSC: %w", err)
			}
			defer bscClient.Close()

			// Get latest block if not specified
			if toBlock == 0 {
				header, err := bscClient.HeaderByNumber(cmd.Context(), nil)
				if err != nil {
					return fmt.Errorf("failed to get latest block: %w", err)
				}
				toBlock = header.Number.Uint64()
			}

			log.Printf("Scanning BSC blocks %d to %d", fromBlock, toBlock)

			// 1. Scan EGG purchases
			log.Printf("\n=== Scanning ZOO transfers for EGG purchases ===")
			purchases, err := bridge.ScanZooEggPurchases(bscClient, bridge.ZooTokenAddress, fromBlock, toBlock)
			if err != nil {
				return fmt.Errorf("failed to scan egg purchases: %w", err)
			}
			log.Printf("Found %d EGG purchase transactions", len(purchases))

			// 2. Scan ZOO burns
			log.Printf("\n=== Scanning ZOO burns to dead address ===")
			burns, err := bridge.ScanZooBurns(bscClient, bridge.ZooTokenAddress, fromBlock, toBlock)
			if err != nil {
				return fmt.Errorf("failed to scan burns: %w", err)
			}
			log.Printf("Found %d burn transactions", len(burns))

			// 3. Scan EGG NFT holders
			log.Printf("\n=== Scanning EGG NFT holders ===")
			nftConfig := bridge.NFTScannerConfig{
				Chain:           "bsc",
				ChainID:         56,
				RPC:             rpc,
				ContractAddress: bridge.EggNFTAddress,
				ProjectName:     "egg",
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
			}

			scanner, err := bridge.NewNFTScanner(nftConfig)
			if err != nil {
				return fmt.Errorf("failed to create NFT scanner: %w", err)
			}
			defer scanner.Close()

			nftResult, err := scanner.Scan()
			if err != nil {
				return fmt.Errorf("failed to scan NFTs: %w", err)
			}

			// Build holder map
			eggHolders := make(map[string]int)
			for _, nft := range nftResult.NFTs {
				addr := strings.ToLower(nft.Owner)
				eggHolders[addr]++
			}
			log.Printf("Found %d unique EGG holders with %d total EGGs", len(eggHolders), nftResult.TotalNFTs)

			// 4. Cross-reference with mainnet if RPC provided
			var mainnetBalances map[string]string
			if mainnetRPC != "" {
				log.Printf("\n=== Cross-referencing with Zoo mainnet ===")
				mainnetBalances, err = scanMainnetBalances(mainnetRPC, mainnetFromBlock, mainnetToBlock)
				if err != nil {
					log.Printf("Warning: failed to scan mainnet: %v", err)
				} else {
					log.Printf("Found %d addresses on mainnet", len(mainnetBalances))
					// Update burn delivery status
					bridge.CrossReferenceWithMainnet(burns, mainnetBalances)
				}
			}

			// 5. Validate purchases against holdings
			log.Printf("\n=== Validating EGG purchases ===")
			validationReport := bridge.ValidateEggPurchases(purchases, eggHolders)

			// Calculate totals for burns
			totalBurnedAmount := "0"
			deliveredCount := 0
			undeliveredCount := 0
			for _, burn := range burns {
				// TODO: Add burn amounts properly
				if burn.DeliveredMainnet {
					deliveredCount++
				} else {
					undeliveredCount++
				}
			}
			validationReport.TotalBurns = len(burns)
			validationReport.TotalBurnedAmount = totalBurnedAmount
			validationReport.DeliveredBurns = deliveredCount
			validationReport.UndeliveredBurns = undeliveredCount

			// 6. Load known holders if provided
			var knownHolders map[string]int
			if knownHoldersFile != "" {
				knownHolders, err = loadKnownHolders(knownHoldersFile)
				if err != nil {
					log.Printf("Warning: failed to load known holders: %v", err)
				}
			}

			// Create cross-reference data structure
			crossRef := &bridge.ZooEggCrossReference{
				EggPurchases:     purchases,
				ZooBurns:         burns,
				EggNFTHolders:    eggHolders,
				MainnetBalances:  mainnetBalances,
				ValidationReport: validationReport,
			}

			// 7. Export to CSV files
			log.Printf("\n=== Exporting results ===")
			basePath := filepath.Join(outputDir, "zoo")
			if err := bridge.ExportZooEggDataToCSV(crossRef, basePath); err != nil {
				return fmt.Errorf("failed to export CSV files: %w", err)
			}

			// 8. Generate summary report
			reportPath := filepath.Join(outputDir, "zoo_cross_reference_report.txt")
			if err := generateSummaryReport(crossRef, knownHolders, reportPath); err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			// Print summary
			fmt.Printf("\n=== Cross-Reference Summary ===\n")
			fmt.Printf("EGG Purchases: %d transactions\n", len(purchases))
			fmt.Printf("Expected EGGs: %d\n", validationReport.TotalExpectedEggs)
			fmt.Printf("Actual EGGs: %d\n", validationReport.TotalActualEggs)
			fmt.Printf("Matched purchases: %d\n", validationReport.MatchedPurchases)
			fmt.Printf("Mismatched purchases: %d\n", validationReport.MismatchedPurchases)
			fmt.Printf("\n")
			fmt.Printf("ZOO Burns: %d transactions\n", len(burns))
			fmt.Printf("Delivered on mainnet: %d\n", deliveredCount)
			fmt.Printf("Not delivered: %d\n", undeliveredCount)
			fmt.Printf("\n")
			fmt.Printf("Files created:\n")
			fmt.Printf("  - %s_egg_purchases.csv\n", basePath)
			fmt.Printf("  - %s_zoo_burns.csv\n", basePath)
			fmt.Printf("  - %s_egg_nft_holders.csv\n", basePath)
			fmt.Printf("  - %s\n", reportPath)

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "BSC RPC endpoint (default: BSC public RPC)")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block for BSC scanning")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block for BSC scanning (default: latest)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "./zoo-cross-reference", "Output directory for CSV files")
	cmd.Flags().StringVar(&mainnetRPC, "mainnet-rpc", "", "Zoo mainnet RPC for cross-referencing")
	cmd.Flags().Uint64Var(&mainnetFromBlock, "mainnet-from-block", 0, "Start block for mainnet scanning")
	cmd.Flags().Uint64Var(&mainnetToBlock, "mainnet-to-block", 0, "End block for mainnet scanning")
	cmd.Flags().StringVar(&knownHoldersFile, "known-holders", "", "JSON file with known EGG holders for validation")

	return cmd
}

// scanMainnetBalances scans Zoo mainnet for current balances
func scanMainnetBalances(rpc string, fromBlock, toBlock uint64) (map[string]string, error) {
	// TODO: Implement mainnet balance scanning
	// This would scan the 200200 chain for ZOO token balances
	// For now, return empty map
	log.Printf("Mainnet scanning not yet implemented")
	return make(map[string]string), nil
}

// loadKnownHolders loads known EGG holders from JSON file
func loadKnownHolders(filename string) (map[string]int, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var holders map[string]int
	if err := json.Unmarshal(data, &holders); err != nil {
		return nil, err
	}

	return holders, nil
}

// generateSummaryReport generates a text summary report
func generateSummaryReport(data *bridge.ZooEggCrossReference, knownHolders map[string]int, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Zoo Ecosystem Cross-Reference Report\n")
	fmt.Fprintf(file, "====================================\n\n")

	// Validation summary
	fmt.Fprintf(file, "EGG Purchase Validation:\n")
	fmt.Fprintf(file, "------------------------\n")
	fmt.Fprintf(file, "Total purchases: %d\n", data.ValidationReport.TotalPurchases)
	fmt.Fprintf(file, "Expected EGGs: %d\n", data.ValidationReport.TotalExpectedEggs)
	fmt.Fprintf(file, "Actual EGGs: %d\n", data.ValidationReport.TotalActualEggs)
	fmt.Fprintf(file, "Matched: %d\n", data.ValidationReport.MatchedPurchases)
	fmt.Fprintf(file, "Mismatched: %d\n", data.ValidationReport.MismatchedPurchases)
	if len(data.ValidationReport.UnexpectedHolders) > 0 {
		fmt.Fprintf(file, "\nUnexpected holders (have EGGs but no purchase record):\n")
		for _, addr := range data.ValidationReport.UnexpectedHolders {
			fmt.Fprintf(file, "  - %s\n", addr)
		}
	}
	fmt.Fprintf(file, "\n")

	// Burns summary
	fmt.Fprintf(file, "ZOO Burns Summary:\n")
	fmt.Fprintf(file, "------------------\n")
	fmt.Fprintf(file, "Total burns: %d\n", data.ValidationReport.TotalBurns)
	fmt.Fprintf(file, "Total burned amount: %s\n", data.ValidationReport.TotalBurnedAmount)
	fmt.Fprintf(file, "Delivered on mainnet: %d\n", data.ValidationReport.DeliveredBurns)
	fmt.Fprintf(file, "Not delivered: %d\n", data.ValidationReport.UndeliveredBurns)
	fmt.Fprintf(file, "\n")

	// Known holders validation if provided
	if knownHolders != nil {
		fmt.Fprintf(file, "Known Holders Validation:\n")
		fmt.Fprintf(file, "-------------------------\n")
		matches := 0
		mismatches := 0
		for addr, expected := range knownHolders {
			actual := data.EggNFTHolders[strings.ToLower(addr)]
			if actual == expected {
				matches++
			} else {
				mismatches++
				if mismatches <= 10 { // Only show first 10 mismatches
					fmt.Fprintf(file, "  %s: expected %d, found %d\n", addr, expected, actual)
				}
			}
		}
		fmt.Fprintf(file, "Matched: %d/%d\n", matches, len(knownHolders))
		if mismatches > 10 {
			fmt.Fprintf(file, "  (showing first 10 mismatches of %d)\n", mismatches)
		}
		fmt.Fprintf(file, "\n")
	}

	// Top holders
	fmt.Fprintf(file, "Top 20 EGG Holders:\n")
	fmt.Fprintf(file, "-------------------\n")
	type holder struct {
		addr  string
		count int
	}
	holders := []holder{}
	for addr, count := range data.EggNFTHolders {
		holders = append(holders, holder{addr, count})
	}
	// Sort by count descending
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].count > holders[i].count {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	for i := 0; i < 20 && i < len(holders); i++ {
		zooEquiv := holders[i].count * bridge.ZooPerEggNFT
		fmt.Fprintf(file, "%2d. %s: %d EGGs (%d ZOO)\n", i+1, holders[i].addr, holders[i].count, zooEquiv)
	}

	return nil
}