package commands

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/luxfi/genesis/pkg/scanner"
	"github.com/spf13/cobra"
)

// NewImportNFTCommand creates the import-nft command
func NewImportNFTCommand() *cobra.Command {
	var (
		network         string
		chainID         int64
		contractAddress string
		rpcURL          string
		outputPath      string
		blockRange      int64
		projectName     string
		crossRefPath    string
	)

	cmd := &cobra.Command{
		Use:   "import-nft",
		Short: "Import NFTs from any EVM chain",
		Long: `Import NFTs from any EVM chain by specifying network parameters.
This command scans the specified contract on the given network and exports
NFT holder data in a format ready for genesis integration.`,
		Example: `  # Import Lux NFTs from Ethereum
  archeology import-nft \
    --network ethereum \
    --chain-id 1 \
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
    --project lux

  # Import Zoo NFTs from BSC
  archeology import-nft \
    --network bsc \
    --chain-id 56 \
    --contract 0xYOUR_CONTRACT_ADDRESS \
    --project zoo

  # Import from custom RPC
  archeology import-nft \
    --rpc https://your-rpc-endpoint.com \
    --chain-id 137 \
    --contract 0xYOUR_CONTRACT_ADDRESS \
    --project custom`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if contractAddress == "" {
				return fmt.Errorf("contract address is required")
			}
			if projectName == "" {
				return fmt.Errorf("project name is required")
			}

			// Set up RPC URL
			if rpcURL == "" && network != "" {
				// Try to get default RPC for known networks
				switch strings.ToLower(network) {
				case "ethereum", "eth", "mainnet":
					rpcURL = "https://eth.llamarpc.com"
					if chainID == 0 {
						chainID = 1
					}
				case "bsc", "binance", "bnb":
					rpcURL = "https://bsc-dataseed.binance.org"
					if chainID == 0 {
						chainID = 56
					}
				case "polygon", "matic":
					rpcURL = "https://polygon-rpc.com"
					if chainID == 0 {
						chainID = 137
					}
				case "arbitrum", "arb":
					rpcURL = "https://arb1.arbitrum.io/rpc"
					if chainID == 0 {
						chainID = 42161
					}
				case "optimism", "op":
					rpcURL = "https://mainnet.optimism.io"
					if chainID == 0 {
						chainID = 10
					}
				case "avalanche", "avax":
					rpcURL = "https://api.avax.network/ext/bc/C/rpc"
					if chainID == 0 {
						chainID = 43114
					}
				default:
					return fmt.Errorf("unknown network '%s' and no RPC URL provided", network)
				}
			}

			if rpcURL == "" {
				return fmt.Errorf("either --network or --rpc must be provided")
			}

			// Set default output path
			if outputPath == "" {
				networkName := network
				if networkName == "" {
					networkName = fmt.Sprintf("chain-%d", chainID)
				}
				outputPath = fmt.Sprintf("exports/%s-nfts-%s.csv", projectName, networkName)
			}

			// Create scanner config
			config := scanner.Config{
				Chain:           network,
				RPC:             rpcURL,
				ContractAddress: contractAddress,
				ContractType:    "nft",
				OutputPath:      outputPath,
				BlockRange:      blockRange,
				ProjectName:     projectName,
				CrossRefPath:    crossRefPath,
			}

			// Log configuration
			log.Printf("Import NFT Configuration:")
			log.Printf("  Network: %s", network)
			log.Printf("  Chain ID: %d", chainID)
			log.Printf("  RPC URL: %s", rpcURL)
			log.Printf("  Contract: %s", contractAddress)
			log.Printf("  Project: %s", projectName)
			log.Printf("  Output: %s", outputPath)
			log.Printf("  Block Range: %d", blockRange)

			// Create scanner
			s, err := scanner.New(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}

			// Run scan
			log.Printf("\nStarting NFT import...")
			result, err := s.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Print results
			fmt.Printf("\n✅ NFT Import Complete!\n")
			fmt.Printf("Chain: %s (ID: %d)\n", result.Chain, chainID)
			fmt.Printf("Contract: %s\n", result.ContractAddress)
			fmt.Printf("Total Holders: %d\n", result.TotalHolders)
			fmt.Printf("Total NFTs: %d\n", result.TotalNFTs)

			if len(result.NFTCollections) > 0 {
				fmt.Printf("\nNFT Collections:\n")
				for collection, count := range result.NFTCollections {
					fmt.Printf("  - %s: %d NFTs\n", collection, count)
				}
			}

			if result.CrossRefStats != nil {
				fmt.Printf("\nCross-Reference Stats:\n")
				fmt.Printf("  Already on chain: %d\n", result.CrossRefStats.AlreadyReceived)
				fmt.Printf("  New holders: %d\n", result.CrossRefStats.NotYetReceived)
			}

			fmt.Printf("\nOutput saved to: %s\n", result.OutputFile)

			// Create Makefile snippet for easy re-run
			makefileSnippet := fmt.Sprintf(`
# Add this to your Makefile for easy re-run:
import-%s-nfts:
	@./bin/archeology import-nft \
		--network %s \
		--chain-id %d \
		--contract %s \
		--project %s \
		--output %s`,
				projectName,
				network,
				chainID,
				contractAddress,
				projectName,
				outputPath)

			fmt.Printf("\n%s\n", makefileSnippet)

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&network, "network", "n", "", "Network name (ethereum, bsc, polygon, arbitrum, optimism, avalanche)")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Chain ID (e.g., 1 for Ethereum, 56 for BSC)")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "NFT contract address (required)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "RPC endpoint URL (overrides network defaults)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output CSV file path")
	cmd.Flags().Int64Var(&blockRange, "block-range", 500000, "Number of blocks to scan backwards from current")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name (lux, zoo, spc, hanzo) (required)")
	cmd.Flags().StringVar(&crossRefPath, "cross-ref", "", "Cross-reference CSV file to check existing holders")

	// Mark required flags
	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("project")

	return cmd
}

// Helper function to validate Ethereum address
func isValidAddress(address string) bool {
	if !strings.HasPrefix(address, "0x") {
		return false
	}
	// Remove 0x prefix and check length
	addr := strings.TrimPrefix(address, "0x")
	if len(addr) != 40 {
		return false
	}
	// Check if hex
	_, err := strconv.ParseUint(addr, 16, 64)
	return err == nil
}

// NetworkInfo contains network configuration
type NetworkInfo struct {
	Name    string
	ChainID int64
	RPC     string
}

// GetKnownNetworks returns a list of known networks
func GetKnownNetworks() []NetworkInfo {
	return []NetworkInfo{
		{Name: "ethereum", ChainID: 1, RPC: "https://eth.llamarpc.com"},
		{Name: "bsc", ChainID: 56, RPC: "https://bsc-dataseed.binance.org"},
		{Name: "polygon", ChainID: 137, RPC: "https://polygon-rpc.com"},
		{Name: "arbitrum", ChainID: 42161, RPC: "https://arb1.arbitrum.io/rpc"},
		{Name: "optimism", ChainID: 10, RPC: "https://mainnet.optimism.io"},
		{Name: "avalanche", ChainID: 43114, RPC: "https://api.avax.network/ext/bc/C/rpc"},
		{Name: "fantom", ChainID: 250, RPC: "https://rpc.ftm.tools"},
		{Name: "gnosis", ChainID: 100, RPC: "https://rpc.gnosischain.com"},
		{Name: "base", ChainID: 8453, RPC: "https://mainnet.base.org"},
		{Name: "zksync", ChainID: 324, RPC: "https://mainnet.era.zksync.io"},
	}
}