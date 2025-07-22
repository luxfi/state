package commands

import (
	"fmt"
	"log"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

// NewScanBurnsCommand creates the scan-burns command
func NewScanBurnsCommand() *cobra.Command {
	var (
		rpc           string
		tokenAddress  string
		burnAddresses []string
		fromBlock     uint64
		toBlock       uint64
		outputCSV     string
		outputJSON    string
		summarize     bool
	)

	cmd := &cobra.Command{
		Use:   "scan-burns",
		Short: "Scan for token burns to specific addresses",
		Long: `Scans blockchain for ERC20 token burns to dead addresses.

Common burn addresses:
- 0x000000000000000000000000000000000000dEaD (dead address)
- 0x0000000000000000000000000000000000000000 (zero address)

This command is useful for:
- Tracking tokens burned for migration
- Analyzing deflationary token mechanics
- Preparing genesis allocations that include burned amounts`,
		Example: `  # Scan for ZOO token burns on BSC
  archeology scan-burns \
    --rpc https://bsc-dataseed.binance.org/ \
    --token 0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13 \
    --burn-address 0x000000000000000000000000000000000000dEaD \
    --output burns.csv

  # Scan with summary
  archeology scan-burns \
    --rpc https://bsc-dataseed.binance.org/ \
    --token 0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13 \
    --summarize --output-json burn-summary.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default burn address if none specified
			if len(burnAddresses) == 0 {
				burnAddresses = []string{scanner.DeadAddress}
			}

			// Create scanner config
			config := &scanner.TokenBurnScanConfig{
				RPC:           rpc,
				TokenAddress:  tokenAddress,
				BurnAddress:   burnAddresses[0],
				BurnAddresses: burnAddresses,
				FromBlock:     fromBlock,
				ToBlock:       toBlock,
			}

			// Create scanner
			burnScanner, err := scanner.NewTokenBurnScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create burn scanner: %w", err)
			}
			defer burnScanner.Close()

			log.Printf("Scanning for burns of token %s", tokenAddress)
			log.Printf("Burn addresses: %v", burnAddresses)

			// Scan burns
			burns, err := burnScanner.ScanBurns()
			if err != nil {
				return fmt.Errorf("failed to scan burns: %w", err)
			}

			log.Printf("Found %d burn transactions", len(burns))

			// Export raw burns if requested
			if outputCSV != "" {
				if err := scanner.ExportTokenBurnsToCSV(burns, outputCSV); err != nil {
					return fmt.Errorf("failed to export CSV: %w", err)
				}
				log.Printf("Exported burns to %s", outputCSV)
			}

			// Generate summary if requested
			if summarize {
				burnsByAddress, err := burnScanner.ScanBurnsByAddress()
				if err != nil {
					return fmt.Errorf("failed to aggregate burns: %w", err)
				}

				// Print summary
				fmt.Printf("\n=== Burn Summary ===\n")
				fmt.Printf("Total unique burners: %d\n", len(burnsByAddress))
				
				// Calculate total burned
				totalBurned := big.NewInt(0)
				for _, amount := range burnsByAddress {
					totalBurned.Add(totalBurned, amount)
				}
				
				// Convert to decimal (assuming 18 decimals)
				decimals := big.NewInt(1e18)
				totalDecimal := new(big.Float).SetInt(totalBurned)
				totalDecimal.Quo(totalDecimal, new(big.Float).SetInt(decimals))
				
				fmt.Printf("Total burned: %s wei (%s tokens)\n", totalBurned.String(), totalDecimal.Text('f', 6))

				// Export summary
				if outputJSON != "" {
					summary := map[string]interface{}{
						"totalBurns":        len(burns),
						"uniqueBurners":     len(burnsByAddress),
						"totalBurnedWei":    totalBurned.String(),
						"totalBurnedTokens": totalDecimal.Text('f', 6),
						"burnsByAddress":    burnsByAddress,
					}
					if err := scanner.ExportToJSON(summary, outputJSON); err != nil {
						return fmt.Errorf("failed to export JSON: %w", err)
					}
					log.Printf("Exported summary to %s", outputJSON)
				}

				// Show top burners
				fmt.Printf("\nTop 10 burners:\n")
				type burner struct {
					addr   string
					amount *big.Int
				}
				burners := []burner{}
				for addr, amount := range burnsByAddress {
					burners = append(burners, burner{addr, amount})
				}
				// Sort by amount
				for i := 0; i < len(burners); i++ {
					for j := i + 1; j < len(burners); j++ {
						if burners[j].amount.Cmp(burners[i].amount) > 0 {
							burners[i], burners[j] = burners[j], burners[i]
						}
					}
				}
				for i := 0; i < 10 && i < len(burners); i++ {
					decimalAmount := new(big.Float).SetInt(burners[i].amount)
					decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(decimals))
					fmt.Printf("%2d. %s: %s tokens\n", i+1, burners[i].addr, decimalAmount.Text('f', 6))
				}
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint")
	cmd.Flags().StringVar(&tokenAddress, "token", "", "Token contract address")
	cmd.Flags().StringSliceVar(&burnAddresses, "burn-address", nil, "Burn addresses to scan (can specify multiple)")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file for raw burns")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file for summary")
	cmd.Flags().BoolVar(&summarize, "summarize", false, "Generate burn summary")

	cmd.MarkFlagRequired("rpc")
	cmd.MarkFlagRequired("token")

	return cmd
}