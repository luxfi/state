package commands

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

// NewCheckCrossChainBalancesCommand creates the check-cross-chain-balances command
func NewCheckCrossChainBalancesCommand() *cobra.Command {
	var (
		sourceChain   string
		sourceRPC     string
		sourceToken   string
		targetChain   string
		targetRPC     string
		targetToken   string
		addressFile   string
		addresses     []string
		outputCSV     string
		outputJSON    string
		compareMode   bool
	)

	cmd := &cobra.Command{
		Use:   "check-cross-chain-balances",
		Short: "Check token balances across multiple chains",
		Long: `Checks token balances for addresses across multiple chains.

This is useful for:
- Verifying token migrations between chains
- Checking if burners have received tokens on target chain
- Comparing balances between source and destination chains`,
		Example: `  # Check if BSC burners have ZOO on mainnet
  teleport check-cross-chain-balances \
    --source-chain BSC --source-rpc https://bsc-rpc \
    --source-token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
    --target-chain "Zoo Mainnet" --target-rpc http://localhost:9650/ext/bc/zoo/rpc \
    --target-token 0x... \
    --address-file burners.txt \
    --compare

  # Check balances for specific addresses
  teleport check-cross-chain-balances \
    --source-chain BSC --source-rpc https://bsc-rpc \
    --source-token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
    --address 0xaddr1 --address 0xaddr2 \
    --output balances.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load addresses
			allAddresses := addresses
			if addressFile != "" {
				fileAddresses, err := loadAddressesFromFile(addressFile)
				if err != nil {
					return fmt.Errorf("failed to load addresses: %w", err)
				}
				allAddresses = append(allAddresses, fileAddresses...)
			}

			if len(allAddresses) == 0 {
				return fmt.Errorf("no addresses provided")
			}

			log.Printf("Checking balances for %d addresses", len(allAddresses))

			// Create scanner config
			chains := []scanner.ChainConfig{
				{
					Name:         sourceChain,
					ChainID:      getChainID(sourceChain),
					RPC:          sourceRPC,
					TokenAddress: sourceToken,
				},
			}

			if targetRPC != "" {
				chains = append(chains, scanner.ChainConfig{
					Name:         targetChain,
					ChainID:      getChainID(targetChain),
					RPC:          targetRPC,
					TokenAddress: targetToken,
				})
			}

			config := &scanner.CrossChainBalanceScanConfig{
				Chains: chains,
			}

			// Create scanner
			balanceScanner, err := scanner.NewCrossChainBalanceScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create balance scanner: %w", err)
			}
			defer balanceScanner.Close()

			// Scan balances
			balances, err := balanceScanner.ScanBalances(allAddresses)
			if err != nil {
				return fmt.Errorf("failed to scan balances: %w", err)
			}

			// Print summary
			fmt.Printf("\n=== Balance Summary ===\n")
			fmt.Printf("Addresses checked: %d\n", len(allAddresses))
			fmt.Printf("Addresses with balances: %d\n", len(balances))

			// Count by chain
			chainCounts := make(map[int64]int)
			for _, balanceList := range balances {
				for _, balance := range balanceList {
					chainCounts[balance.ChainID]++
				}
			}
			for _, chain := range chains {
				fmt.Printf("%s: %d addresses with balance\n", chain.Name, chainCounts[chain.ChainID])
			}

			// Compare mode
			if compareMode && len(chains) >= 2 {
				fmt.Printf("\n=== Balance Comparison ===\n")
				
				// Find addresses with balance on source but not target
				missingOnTarget := 0
				presentOnBoth := 0
				onlyOnTarget := 0

				for addr, balanceList := range balances {
					hasSource := false
					hasTarget := false
					for _, balance := range balanceList {
						if balance.ChainID == chains[0].ChainID {
							hasSource = true
						}
						if balance.ChainID == chains[1].ChainID {
							hasTarget = true
						}
					}

					if hasSource && !hasTarget {
						missingOnTarget++
						if missingOnTarget <= 10 {
							fmt.Printf("  %s: Has balance on %s but not %s\n", addr, chains[0].Name, chains[1].Name)
						}
					} else if hasSource && hasTarget {
						presentOnBoth++
					} else if !hasSource && hasTarget {
						onlyOnTarget++
					}
				}

				fmt.Printf("\nSummary:\n")
				fmt.Printf("Present on both chains: %d\n", presentOnBoth)
				fmt.Printf("Only on %s: %d\n", chains[0].Name, missingOnTarget)
				fmt.Printf("Only on %s: %d\n", chains[1].Name, onlyOnTarget)

				if missingOnTarget > 10 {
					fmt.Printf("(Showing first 10 addresses missing on target)\n")
				}
			}

			// Export to CSV
			if outputCSV != "" {
				if err := scanner.ExportCrossChainBalancesToCSV(balances, outputCSV); err != nil {
					return fmt.Errorf("failed to export CSV: %w", err)
				}
				log.Printf("Exported balances to %s", outputCSV)
			}

			// Export to JSON
			if outputJSON != "" {
				summary := map[string]interface{}{
					"addressesChecked": len(allAddresses),
					"chains":          chains,
					"balances":        balances,
				}
				if err := scanner.ExportToJSON(summary, outputJSON); err != nil {
					return fmt.Errorf("failed to export JSON: %w", err)
				}
				log.Printf("Exported summary to %s", outputJSON)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&sourceChain, "source-chain", "BSC", "Source chain name")
	cmd.Flags().StringVar(&sourceRPC, "source-rpc", "", "Source chain RPC")
	cmd.Flags().StringVar(&sourceToken, "source-token", "", "Source token address")
	cmd.Flags().StringVar(&targetChain, "target-chain", "Zoo Mainnet", "Target chain name")
	cmd.Flags().StringVar(&targetRPC, "target-rpc", "", "Target chain RPC")
	cmd.Flags().StringVar(&targetToken, "target-token", "", "Target token address")
	cmd.Flags().StringVar(&addressFile, "address-file", "", "File containing addresses (one per line)")
	cmd.Flags().StringSliceVar(&addresses, "address", nil, "Individual addresses to check")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().BoolVar(&compareMode, "compare", false, "Compare balances between chains")

	cmd.MarkFlagRequired("source-rpc")
	cmd.MarkFlagRequired("source-token")

	return cmd
}

// getChainID returns chain ID for known chains
func getChainID(chainName string) int64 {
	chainIDs := map[string]int64{
		"Ethereum":      1,
		"BSC":          56,
		"Polygon":      137,
		"Lux Mainnet":  96369,
		"Zoo Mainnet":  200200,
		"SPC Mainnet":  36911,
		"Hanzo Mainnet": 36963,
	}

	if id, ok := chainIDs[chainName]; ok {
		return id
	}

	// Try to parse as number
	var id int64
	fmt.Sscanf(chainName, "%d", &id)
	return id
}

// loadAddressesFromFile loads addresses from a text file
func loadAddressesFromFile(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	addresses := []string{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		addr := strings.TrimSpace(line)
		if addr != "" && strings.HasPrefix(addr, "0x") {
			addresses = append(addresses, addr)
		}
	}

	return addresses, nil
}