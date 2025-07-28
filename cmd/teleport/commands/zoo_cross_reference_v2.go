package commands

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

const (
	// Zoo contract addresses on BSC
	ZooTokenBSC = "0x0a6045b79151d0a54dbd5227082445750a023af2"
	EggNFTBSC   = "0x5bb68cf06289d54efde25155c88003be685356a8"
	EggPurchaseAddr = "0x28dad8427f127664365109c4a9406c8bc7844718"
	ZooPerEgg = 4200000
)

// NewZooCrossReferenceV2Command creates the improved zoo-cross-reference command
func NewZooCrossReferenceV2Command() *cobra.Command {
	var (
		bscRPC           string
		mainnetRPC       string
		fromBlock        uint64
		toBlock          uint64
		outputDir        string
		knownHoldersFile string
	)

	cmd := &cobra.Command{
		Use:   "zoo-full-analysis",
		Short: "Complete Zoo ecosystem analysis using modular scanners",
		Long: `Performs comprehensive analysis of the Zoo ecosystem using modular scanners:

This command will:
1. Scan BSC for all ZOO transfers to the EGG purchase address
2. Scan BSC for all ZOO burns to the dead address
3. Scan BSC for all EGG NFT holders
4. Cross-reference with Zoo mainnet (200200) if RPC provided
5. Generate multiple CSV files and a summary report

Generated files:
- zoo_egg_purchases.csv: All ZOO transfers for EGG purchases
- zoo_burns.csv: All ZOO burns to dead address
- zoo_egg_holders.csv: Current EGG NFT holders
- zoo_cross_chain_balances.csv: Cross-chain balance comparison
- zoo_analysis_report.txt: Complete analysis report`,
		Example: `  # Complete Zoo analysis
  teleport zoo-full-analysis --output-dir ./zoo-analysis

  # With mainnet cross-reference
  teleport zoo-full-analysis \
    --mainnet-rpc http://localhost:9630/ext/bc/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/rpc \
    --output-dir ./zoo-analysis`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create output directory
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			// 1. Scan EGG NFT holders
			log.Printf("\n=== Step 1: Scanning EGG NFT holders ===")
			nftConfig := &scanner.NFTHolderScanConfig{
				RPC:             bscRPC,
				ContractAddress: EggNFTBSC,
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				IncludeTokenIDs: false,
			}

			nftScanner, err := scanner.NewNFTHolderScanner(nftConfig)
			if err != nil {
				return fmt.Errorf("failed to create NFT scanner: %w", err)
			}
			defer nftScanner.Close()

			eggHolders, err := nftScanner.ScanHolders()
			if err != nil {
				return fmt.Errorf("failed to scan NFT holders: %w", err)
			}
			log.Printf("Found %d EGG NFT holders", len(eggHolders))

			// Export EGG holders
			eggHoldersFile := filepath.Join(outputDir, "zoo_egg_holders.csv")
			if err := exportEggHoldersWithZoo(eggHolders, eggHoldersFile); err != nil {
				return fmt.Errorf("failed to export EGG holders: %w", err)
			}

			// 2. Scan ZOO purchases to EGG address
			log.Printf("\n=== Step 2: Scanning ZOO transfers for EGG purchases ===")
			purchaseConfig := &scanner.TokenTransferScanConfig{
				RPC:             bscRPC,
				TokenAddress:    ZooTokenBSC,
				TargetAddresses: []string{EggPurchaseAddr},
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				Direction:       "to",
			}

			purchaseScanner, err := scanner.NewTokenTransferScanner(purchaseConfig)
			if err != nil {
				return fmt.Errorf("failed to create purchase scanner: %w", err)
			}
			defer purchaseScanner.Close()

			purchases, err := purchaseScanner.ScanTransfers()
			if err != nil {
				return fmt.Errorf("failed to scan purchases: %w", err)
			}
			log.Printf("Found %d EGG purchase transactions", len(purchases))

			// Export purchases
			purchasesFile := filepath.Join(outputDir, "zoo_egg_purchases.csv")
			if err := exportPurchasesWithValidation(purchases, eggHolders, purchasesFile); err != nil {
				return fmt.Errorf("failed to export purchases: %w", err)
			}

			// 3. Scan ZOO burns
			log.Printf("\n=== Step 3: Scanning ZOO burns to dead address ===")
			burnConfig := &scanner.TokenBurnScanConfig{
				RPC:          bscRPC,
				TokenAddress: ZooTokenBSC,
				BurnAddress:  scanner.DeadAddress,
				FromBlock:    fromBlock,
				ToBlock:      toBlock,
			}

			burnScanner, err := scanner.NewTokenBurnScanner(burnConfig)
			if err != nil {
				return fmt.Errorf("failed to create burn scanner: %w", err)
			}
			defer burnScanner.Close()

			burns, err := burnScanner.ScanBurns()
			if err != nil {
				return fmt.Errorf("failed to scan burns: %w", err)
			}
			log.Printf("Found %d burn transactions", len(burns))

			burnsByAddress, err := burnScanner.ScanBurnsByAddress()
			if err != nil {
				return fmt.Errorf("failed to aggregate burns: %w", err)
			}

			// 4. Cross-chain balance check if mainnet RPC provided
			var crossChainBalances map[string][]scanner.CrossChainBalance
			if mainnetRPC != "" {
				log.Printf("\n=== Step 4: Checking cross-chain balances ===")
				
				// Get unique burner addresses
				burnerAddresses := []string{}
				for addr := range burnsByAddress {
					burnerAddresses = append(burnerAddresses, addr)
				}

				crossChainConfig := &scanner.CrossChainBalanceScanConfig{
					Chains: []scanner.ChainConfig{
						{
							Name:         "BSC",
							ChainID:      56,
							RPC:          bscRPC,
							TokenAddress: ZooTokenBSC,
						},
						{
							Name:         "Zoo Mainnet",
							ChainID:      200200,
							RPC:          mainnetRPC,
							TokenAddress: ZooTokenBSC, // Adjust if different on mainnet
						},
					},
				}

				balanceScanner, err := scanner.NewCrossChainBalanceScanner(crossChainConfig)
				if err != nil {
					log.Printf("Warning: failed to create balance scanner: %v", err)
				} else {
					defer balanceScanner.Close()
					crossChainBalances, err = balanceScanner.ScanBalances(burnerAddresses)
					if err != nil {
						log.Printf("Warning: failed to scan balances: %v", err)
					} else {
						log.Printf("Checked %d burner addresses across chains", len(burnerAddresses))
					}
				}
			}

			// 5. Generate comprehensive report
			log.Printf("\n=== Step 5: Generating analysis report ===")
			reportFile := filepath.Join(outputDir, "zoo_analysis_report.txt")
			if err := generateZooAnalysisReport(
				eggHolders,
				purchases,
				burns,
				burnsByAddress,
				crossChainBalances,
				reportFile,
			); err != nil {
				return fmt.Errorf("failed to generate report: %w", err)
			}

			// Export burns with cross-chain status
			burnsFile := filepath.Join(outputDir, "zoo_burns.csv")
			if err := exportBurnsWithStatus(burns, crossChainBalances, burnsFile); err != nil {
				return fmt.Errorf("failed to export burns: %w", err)
			}

			// Print summary
			fmt.Printf("\n=== Analysis Complete ===\n")
			fmt.Printf("Files created in %s:\n", outputDir)
			fmt.Printf("  - zoo_egg_holders.csv (%d holders)\n", len(eggHolders))
			fmt.Printf("  - zoo_egg_purchases.csv (%d purchases)\n", len(purchases))
			fmt.Printf("  - zoo_burns.csv (%d burns)\n", len(burns))
			fmt.Printf("  - zoo_analysis_report.txt\n")

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&bscRPC, "bsc-rpc", "", "BSC RPC endpoint (defaults to public RPC)")
	cmd.Flags().StringVar(&mainnetRPC, "mainnet-rpc", "", "Zoo mainnet RPC for cross-referencing")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block for scanning")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "./zoo-analysis", "Output directory")
	cmd.Flags().StringVar(&knownHoldersFile, "known-holders", "", "JSON file with known holders")

	return cmd
}

// exportEggHoldersWithZoo exports EGG holders with ZOO equivalents
func exportEggHoldersWithZoo(holders []scanner.NFTHolder, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"Address", "EggCount", "ZooEquivalent"})

	// Write data
	for _, holder := range holders {
		zooEquiv := holder.TokenCount * ZooPerEgg
		writer.Write([]string{
			holder.Address,
			fmt.Sprintf("%d", holder.TokenCount),
			fmt.Sprintf("%d", zooEquiv),
		})
	}

	return nil
}

// exportPurchasesWithValidation exports purchases with validation against actual holdings
func exportPurchasesWithValidation(purchases []scanner.TokenTransfer, holders []scanner.NFTHolder, filename string) error {
	// Build holder map
	holderMap := make(map[string]int)
	for _, holder := range holders {
		holderMap[strings.ToLower(holder.Address)] = holder.TokenCount
	}

	// Aggregate purchases by address
	purchasesByAddr := make(map[string]*big.Int)
	for _, purchase := range purchases {
		addr := strings.ToLower(purchase.From)
		amount := new(big.Int)
		amount.SetString(purchase.Amount, 10)
		
		if existing, ok := purchasesByAddr[addr]; ok {
			existing.Add(existing, amount)
		} else {
			purchasesByAddr[addr] = amount
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"TxHash", "BlockNumber", "Timestamp", "From", "Amount",
		"ExpectedEggs", "ActualEggs", "Matched",
	})

	// Write purchases with validation
	decimals := big.NewInt(1e18)
	zooPerEggWei := new(big.Int).Mul(big.NewInt(ZooPerEgg), decimals)

	for _, purchase := range purchases {
		amount := new(big.Int)
		amount.SetString(purchase.Amount, 10)
		
		expectedEggs := new(big.Int).Div(amount, zooPerEggWei)
		actualEggs := holderMap[strings.ToLower(purchase.From)]
		matched := actualEggs >= int(expectedEggs.Int64())

		writer.Write([]string{
			purchase.TxHash,
			fmt.Sprintf("%d", purchase.BlockNumber),
			purchase.Timestamp.Format("2006-01-02 15:04:05"),
			purchase.From,
			purchase.Amount,
			expectedEggs.String(),
			fmt.Sprintf("%d", actualEggs),
			fmt.Sprintf("%v", matched),
		})
	}

	return nil
}

// exportBurnsWithStatus exports burns with mainnet delivery status
func exportBurnsWithStatus(burns []scanner.TokenBurn, crossChainBalances map[string][]scanner.CrossChainBalance, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	writer.Write([]string{
		"TxHash", "BlockNumber", "Timestamp", "From",
		"BurnedAmount", "HasMainnetBalance", "MainnetBalance",
	})

	// Write burns with status
	for _, burn := range burns {
		hasMainnet := false
		mainnetBalance := "0"
		
		if balances, ok := crossChainBalances[strings.ToLower(burn.From)]; ok {
			for _, balance := range balances {
				if balance.ChainID == 200200 { // Zoo mainnet
					hasMainnet = true
					mainnetBalance = balance.Balance
					break
				}
			}
		}

		writer.Write([]string{
			burn.TxHash,
			fmt.Sprintf("%d", burn.BlockNumber),
			burn.Timestamp.Format("2006-01-02 15:04:05"),
			burn.From,
			burn.Amount,
			fmt.Sprintf("%v", hasMainnet),
			mainnetBalance,
		})
	}

	return nil
}

// generateZooAnalysisReport generates comprehensive analysis report
func generateZooAnalysisReport(
	holders []scanner.NFTHolder,
	purchases []scanner.TokenTransfer,
	burns []scanner.TokenBurn,
	burnsByAddress map[string]*big.Int,
	crossChainBalances map[string][]scanner.CrossChainBalance,
	filename string,
) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Zoo Ecosystem Analysis Report\n")
	fmt.Fprintf(file, "============================\n\n")

	// EGG NFT Summary
	totalEggs := 0
	for _, holder := range holders {
		totalEggs += holder.TokenCount
	}
	fmt.Fprintf(file, "EGG NFT Summary:\n")
	fmt.Fprintf(file, "----------------\n")
	fmt.Fprintf(file, "Total holders: %d\n", len(holders))
	fmt.Fprintf(file, "Total EGGs: %d\n", totalEggs)
	fmt.Fprintf(file, "Total ZOO equivalent: %d\n\n", totalEggs*ZooPerEgg)

	// Purchase Summary
	fmt.Fprintf(file, "EGG Purchase Summary:\n")
	fmt.Fprintf(file, "--------------------\n")
	fmt.Fprintf(file, "Total purchase transactions: %d\n", len(purchases))
	
	// Calculate total purchased
	totalPurchased := big.NewInt(0)
	uniquePurchasers := make(map[string]bool)
	for _, purchase := range purchases {
		amount := new(big.Int)
		amount.SetString(purchase.Amount, 10)
		totalPurchased.Add(totalPurchased, amount)
		uniquePurchasers[strings.ToLower(purchase.From)] = true
	}
	
	decimals := big.NewInt(1e18)
	totalPurchasedDecimal := new(big.Float).SetInt(totalPurchased)
	totalPurchasedDecimal.Quo(totalPurchasedDecimal, new(big.Float).SetInt(decimals))
	
	fmt.Fprintf(file, "Unique purchasers: %d\n", len(uniquePurchasers))
	fmt.Fprintf(file, "Total ZOO spent: %s tokens\n\n", totalPurchasedDecimal.Text('f', 2))

	// Burn Summary
	fmt.Fprintf(file, "ZOO Burn Summary:\n")
	fmt.Fprintf(file, "-----------------\n")
	fmt.Fprintf(file, "Total burn transactions: %d\n", len(burns))
	fmt.Fprintf(file, "Unique burners: %d\n", len(burnsByAddress))
	
	totalBurned := big.NewInt(0)
	for _, amount := range burnsByAddress {
		totalBurned.Add(totalBurned, amount)
	}
	totalBurnedDecimal := new(big.Float).SetInt(totalBurned)
	totalBurnedDecimal.Quo(totalBurnedDecimal, new(big.Float).SetInt(decimals))
	
	fmt.Fprintf(file, "Total ZOO burned: %s tokens\n\n", totalBurnedDecimal.Text('f', 2))

	// Cross-chain Summary
	if crossChainBalances != nil {
		fmt.Fprintf(file, "Cross-Chain Status:\n")
		fmt.Fprintf(file, "-------------------\n")
		
		burnersWithMainnet := 0
		for addr := range burnsByAddress {
			if balances, ok := crossChainBalances[addr]; ok {
				for _, balance := range balances {
					if balance.ChainID == 200200 {
						burnersWithMainnet++
						break
					}
				}
			}
		}
		
		fmt.Fprintf(file, "Burners with mainnet balance: %d/%d (%.1f%%)\n",
			burnersWithMainnet, len(burnsByAddress),
			float64(burnersWithMainnet)/float64(len(burnsByAddress))*100)
		fmt.Fprintf(file, "Burners needing delivery: %d\n\n", len(burnsByAddress)-burnersWithMainnet)
	}

	// Top holders
	fmt.Fprintf(file, "Top 20 EGG Holders:\n")
	fmt.Fprintf(file, "-------------------\n")
	limit := 20
	if len(holders) < limit {
		limit = len(holders)
	}
	// Sort by count
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].TokenCount > holders[i].TokenCount {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(file, "%2d. %s: %d EGGs (%d ZOO)\n",
			i+1, holders[i].Address, holders[i].TokenCount, holders[i].TokenCount*ZooPerEgg)
	}

	return nil
}