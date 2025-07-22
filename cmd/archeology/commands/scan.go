package commands

import (
	"fmt"
	"log"

	"github.com/luxfi/genesis/pkg/scanner"
	"github.com/spf13/cobra"
)

// NewScanCommand creates the scan subcommand for external assets
func NewScanCommand() *cobra.Command {
	var (
		chain           string
		rpcURL          string
		contractAddress string
		contractType    string
		outputPath      string
		blockRange      int64
		projectName     string
		crossRefPath    string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan external EVM chains for NFTs and tokens",
		Long: `Scan external EVM chains (Ethereum, BSC, etc.) for NFTs and tokens.
This command finds all holders of a given contract and prepares data for X-Chain genesis integration.`,
		Example: `  # Scan Lux NFTs on Ethereum
  archeology scan --chain ethereum --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 --project lux --type nft

  # Scan Zoo tokens on BSC (auto-detect type)
  archeology scan --chain bsc --contract 0xADDRESS --project zoo --type auto

  # Scan with custom RPC
  archeology scan --rpc https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY --contract 0xADDRESS --project lux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if contractAddress == "" {
				return fmt.Errorf("--contract is required")
			}

			// Create scanner config
			config := scanner.Config{
				Chain:           chain,
				RPC:             rpcURL,
				ContractAddress: contractAddress,
				ContractType:    contractType,
				OutputPath:      outputPath,
				BlockRange:      blockRange,
				ProjectName:     projectName,
				CrossRefPath:    crossRefPath,
			}

			// Create scanner
			s, err := scanner.New(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}

			// Run scan
			log.Printf("Scanning %s for %s assets...", chain, projectName)
			result, err := s.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Print summary
			log.Printf("\n=== Scan Summary ===")
			log.Printf("Chain: %s", result.Chain)
			log.Printf("Contract: %s", result.ContractAddress)
			log.Printf("Type: %s", result.AssetType)
			log.Printf("Total holders: %d", result.TotalHolders)
			
			if result.AssetType == "NFT" {
				log.Printf("Total NFTs: %d", result.TotalNFTs)
				log.Printf("Collections found:")
				for collection, count := range result.NFTCollections {
					log.Printf("  - %s: %d NFTs", collection, count)
				}
			} else {
				log.Printf("Total supply: %s", result.TotalSupply)
			}

			if result.CrossRefStats != nil {
				log.Printf("\nCross-reference results:")
				log.Printf("  - Already received on-chain: %d", result.CrossRefStats.AlreadyReceived)
				log.Printf("  - Not yet received: %d", result.CrossRefStats.NotYetReceived)
			}

			log.Printf("\nOutput file: %s", result.OutputFile)

			return nil
		},
	}

	// Define flags
	cmd.Flags().StringVar(&chain, "chain", "ethereum", "Chain name (ethereum, bsc, polygon, arbitrum, optimism, avalanche)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "Custom RPC URL (overrides default for chain)")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "Contract address to scan")
	cmd.Flags().StringVar(&contractType, "type", "auto", "Contract type: nft, token, or auto")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output CSV file (auto-generated if empty)")
	cmd.Flags().Int64Var(&blockRange, "blocks", 5000000, "Number of blocks to scan back")
	cmd.Flags().StringVar(&projectName, "project", "lux", "Project name (lux, zoo, spc, hanzo)")
	cmd.Flags().StringVar(&crossRefPath, "crossref", "", "Path to existing chain data for cross-reference")

	// Mark required flags
	cmd.MarkFlagRequired("contract")

	return cmd
}