package commands

import (
	"fmt"
	"log"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

// NewScanTransfersCommand creates the scan-transfers command
func NewScanTransfersCommand() *cobra.Command {
	var (
		rpc             string
		tokenAddress    string
		targetAddresses []string
		direction       string
		fromBlock       uint64
		toBlock         uint64
		outputCSV       string
		outputJSON      string
		showBalances    bool
	)

	cmd := &cobra.Command{
		Use:   "scan-transfers",
		Short: "Scan for token transfers to/from specific addresses",
		Long: `Scans blockchain for ERC20 token transfers involving specific addresses.

Direction options:
- "to": Only transfers TO the target addresses
- "from": Only transfers FROM the target addresses  
- "both": All transfers involving the target addresses (default)

This is useful for:
- Tracking payments to specific addresses (e.g., purchase addresses)
- Analyzing token flows from distribution wallets
- Building transfer history for specific addresses`,
		Example: `  # Scan transfers TO a purchase address
  archaeology scan-transfers \
    --rpc https://bsc-dataseed.binance.org/ \
    --token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
    --target 0x28dad8427f127664365109c4a9406c8bc7844718 \
    --direction to \
    --output purchases.csv

  # Scan all transfers for multiple addresses with balance calculation
  archaeology scan-transfers \
    --rpc https://bsc-dataseed.binance.org/ \
    --token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
    --target 0xaddr1 --target 0xaddr2 \
    --show-balances`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create scanner config
			config := &scanner.TokenTransferScanConfig{
				RPC:             rpc,
				TokenAddress:    tokenAddress,
				TargetAddresses: targetAddresses,
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				Direction:       direction,
			}

			// Create scanner
			transferScanner, err := scanner.NewTokenTransferScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create transfer scanner: %w", err)
			}
			defer transferScanner.Close()

			log.Printf("Scanning transfers of token %s", tokenAddress)
			if len(targetAddresses) > 0 {
				log.Printf("Target addresses: %v", targetAddresses)
				log.Printf("Direction: %s", direction)
			} else {
				log.Printf("Scanning all transfers")
			}

			// Scan transfers
			transfers, err := transferScanner.ScanTransfers()
			if err != nil {
				return fmt.Errorf("failed to scan transfers: %w", err)
			}

			log.Printf("Found %d transfers", len(transfers))

			// Export to CSV if requested
			if outputCSV != "" {
				if err := scanner.ExportTokenTransfersToCSV(transfers, outputCSV); err != nil {
					return fmt.Errorf("failed to export CSV: %w", err)
				}
				log.Printf("Exported transfers to %s", outputCSV)
			}

			// Calculate and show balance changes if requested
			if showBalances {
				balanceChanges := scanner.GetBalanceChanges(transfers)
				
				fmt.Printf("\n=== Balance Changes ===\n")
				fmt.Printf("Addresses affected: %d\n", len(balanceChanges))

				// Find addresses with positive and negative balances
				positive := 0
				negative := 0
				zero := 0
				for _, balance := range balanceChanges {
					switch balance.Cmp(big.NewInt(0)) {
					case 1:
						positive++
					case -1:
						negative++
					case 0:
						zero++
					}
				}
				fmt.Printf("Positive balances: %d\n", positive)
				fmt.Printf("Negative balances: %d\n", negative)
				fmt.Printf("Zero balances: %d\n", zero)

				// Show top receivers and senders
				type balanceEntry struct {
					addr    string
					balance *big.Int
				}
				entries := []balanceEntry{}
				for addr, balance := range balanceChanges {
					entries = append(entries, balanceEntry{addr, balance})
				}

				// Sort by balance (descending)
				for i := 0; i < len(entries); i++ {
					for j := i + 1; j < len(entries); j++ {
						if entries[j].balance.Cmp(entries[i].balance) > 0 {
							entries[i], entries[j] = entries[j], entries[i]
						}
					}
				}

				// Show top receivers
				fmt.Printf("\nTop 10 Receivers:\n")
				decimals := big.NewInt(1e18)
				for i := 0; i < 10 && i < len(entries); i++ {
					if entries[i].balance.Cmp(big.NewInt(0)) <= 0 {
						break
					}
					decimalAmount := new(big.Float).SetInt(entries[i].balance)
					decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(decimals))
					fmt.Printf("%2d. %s: +%s tokens\n", i+1, entries[i].addr, decimalAmount.Text('f', 6))
				}

				// Show top senders (reverse order)
				fmt.Printf("\nTop 10 Senders:\n")
				count := 0
				for i := len(entries) - 1; i >= 0 && count < 10; i-- {
					if entries[i].balance.Cmp(big.NewInt(0)) >= 0 {
						break
					}
					decimalAmount := new(big.Float).SetInt(entries[i].balance)
					decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(decimals))
					fmt.Printf("%2d. %s: %s tokens\n", count+1, entries[i].addr, decimalAmount.Text('f', 6))
					count++
				}
			}

			// Export to JSON if requested
			if outputJSON != "" {
				summary := map[string]interface{}{
					"token":          tokenAddress,
					"totalTransfers": len(transfers),
					"transfers":      transfers,
				}
				if showBalances {
					summary["balanceChanges"] = scanner.GetBalanceChanges(transfers)
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
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint")
	cmd.Flags().StringVar(&tokenAddress, "token", "", "Token contract address")
	cmd.Flags().StringSliceVar(&targetAddresses, "target", nil, "Target addresses to filter (can specify multiple)")
	cmd.Flags().StringVar(&direction, "direction", "both", "Transfer direction: to, from, or both")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().BoolVar(&showBalances, "show-balances", false, "Calculate and show balance changes")

	cmd.MarkFlagRequired("rpc")
	cmd.MarkFlagRequired("token")

	return cmd
}