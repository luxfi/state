package commands

import (
	"fmt"
	"log"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewScanTokenCommand() *cobra.Command {
	var (
		chain           string
		chainID         int64
		rpcURL          string
		contractAddress string
		projectName     string
		outputPath      string
		fromBlock       uint64
		toBlock         uint64
		minBalance      string
		includeZero     bool
		crossReference  string
	)

	cmd := &cobra.Command{
		Use:   "scan-token",
		Short: "Scan ERC20 tokens from any EVM chain",
		Long: `Scan ERC20 token holders from any EVM-compatible blockchain.
This command creates a complete snapshot of token holders for genesis inclusion.`,
		Example: `  # Scan USDC from Ethereum
  teleport scan-token \
    --chain ethereum \
    --contract 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
    --project usdc \
    --output ./external/usdc-holders.json

  # Scan historic ZOO token from BSC
  teleport scan-token \
    --chain bsc \
    --contract 0xZOO_TOKEN_ADDRESS \
    --project zoo \
    --min-balance 1000000000000000000

  # Scan from local chain (7777)
  teleport scan-token \
    --chain local \
    --chain-id 7777 \
    --contract 0xLOCAL_TOKEN \
    --project lux-legacy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if contractAddress == "" {
				return fmt.Errorf("contract address is required")
			}
			if projectName == "" {
				return fmt.Errorf("project name is required")
			}
			if chain == "" && rpcURL == "" {
				return fmt.Errorf("either --chain or --rpc must be specified")
			}

			// Special handling for local chains
			if chain == "local" || chain == "7777" || chain == "lux-7777" {
				if rpcURL == "" {
					rpcURL = "http://localhost:9650/ext/bc/C/rpc"
				}
				if chainID == 0 {
					chainID = 7777
				}
			} else if chain == "96369" || chain == "lux-mainnet" {
				if rpcURL == "" {
					rpcURL = "http://localhost:9650/ext/bc/C/rpc"
				}
				if chainID == 0 {
					chainID = 96369
				}
			}

			config := bridge.TokenScannerConfig{
				Chain:           chain,
				ChainID:         chainID,
				RPCURL:          rpcURL,
				ContractAddress: contractAddress,
				ProjectName:     projectName,
				FromBlock:       fromBlock,
				ToBlock:         toBlock,
				MinBalance:      minBalance,
				IncludeZero:     includeZero,
				CrossReference:  crossReference,
			}

			scanner, err := bridge.NewTokenScanner(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}

			log.Printf("Scanning ERC20 token from %s", contractAddress)
			log.Printf("Chain: %s (ID: %d)", chain, chainID)
			log.Printf("Project: %s", projectName)

			// Run scan
			result, err := scanner.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Display results
			fmt.Printf("\n✅ Token scan completed!\n\n")
			fmt.Printf("Contract: %s\n", result.ContractAddress)
			fmt.Printf("Name: %s\n", result.TokenName)
			fmt.Printf("Symbol: %s\n", result.Symbol)
			fmt.Printf("Decimals: %d\n", result.Decimals)
			fmt.Printf("Total Supply: %s\n", result.TotalSupply)
			fmt.Printf("Unique Holders: %d\n", result.UniqueHolders)
			fmt.Printf("Blocks Scanned: %d to %d\n", result.FromBlock, result.ToBlock)

			// Show distribution
			if len(result.Distribution) > 0 {
				fmt.Printf("\nToken Distribution:\n")
				for _, tier := range result.Distribution {
					fmt.Printf("  %s: %d holders (%.2f%% of supply)\n", 
						tier.Range, tier.Count, tier.Percentage)
				}
			}

			// Show top holders
			if len(result.TopHolders) > 0 {
				fmt.Printf("\nTop 10 Holders:\n")
				for i, holder := range result.TopHolders {
					if i >= 10 {
						break
					}
					balance := new(big.Int)
					balance.SetString(holder.Balance, 10)
					fmt.Printf("  %d. %s: %s %s (%.2f%%)\n", 
						i+1, holder.Address, holder.BalanceFormatted, 
						result.Symbol, holder.Percentage)
				}
			}

			// Cross-reference results
			if crossReference != "" && result.CrossReferenceResult != nil {
				fmt.Printf("\nCross-Reference Results:\n")
				fmt.Printf("  Addresses on target chain: %d\n", result.CrossReferenceResult.FoundOnChain)
				fmt.Printf("  New addresses: %d\n", result.CrossReferenceResult.NewAddresses)
				fmt.Printf("  Missing from target: %d\n", result.CrossReferenceResult.MissingFromChain)
			}

			// Export results
			if outputPath == "" {
				outputPath = fmt.Sprintf("./token-scan-%s-%s.json", projectName, chain)
			}

			if err := scanner.Export(outputPath); err != nil {
				return fmt.Errorf("failed to export results: %w", err)
			}

			fmt.Printf("\nResults exported to: %s\n", outputPath)

			// Show migration readiness
			if result.MigrationInfo != nil {
				fmt.Printf("\n📊 Migration Readiness:\n")
				fmt.Printf("  Total holders to migrate: %d\n", result.MigrationInfo.HoldersToMigrate)
				fmt.Printf("  Total balance to migrate: %s %s\n", 
					result.MigrationInfo.BalanceToMigrate, result.Symbol)
				if result.MigrationInfo.RecommendedLayer != "" {
					fmt.Printf("  Recommended deployment: %s\n", result.MigrationInfo.RecommendedLayer)
				}
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVarP(&chain, "chain", "c", "", "Blockchain name (ethereum, bsc, polygon, local)")
	cmd.Flags().Int64Var(&chainID, "chain-id", 0, "Chain ID")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Custom RPC URL")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "Token contract address")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output file path")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block (0 for earliest)")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 for latest)")
	cmd.Flags().StringVar(&minBalance, "min-balance", "0", "Minimum balance to include (in wei)")
	cmd.Flags().BoolVar(&includeZero, "include-zero", false, "Include zero balance holders")
	cmd.Flags().StringVar(&crossReference, "cross-reference", "", "Cross-reference with extracted chain data")

	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("project")

	return cmd
}