package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/luxfi/genesis/pkg/scanner"
	"github.com/spf13/cobra"
)

// NewImportTokenCommand creates the import-token command
func NewImportTokenCommand() *cobra.Command {
	var (
		network         string
		chainID         int64
		contractAddress string
		rpcURL          string
		outputPath      string
		blockRange      int64
		projectName     string
		crossRefPath    string
		tokenSymbol     string
		tokenDecimals   int
	)

	cmd := &cobra.Command{
		Use:   "import-token",
		Short: "Import ERC20 tokens from any EVM chain",
		Long: `Import ERC20 tokens from any EVM chain by specifying network parameters.
This command scans the specified token contract on the given network and exports
token holder data in a format ready for X-Chain genesis integration.`,
		Example: `  # Import Zoo tokens from BSC
  archeology import-token \
    --network bsc \
    --chain-id 56 \
    --contract 0xYOUR_ZOO_TOKEN_ADDRESS \
    --project zoo \
    --symbol ZOO

  # Import tokens from local 7777 chain
  archeology import-token \
    --rpc http://localhost:9650/ext/bc/C/rpc \
    --chain-id 7777 \
    --contract 0xTOKEN_ADDRESS \
    --project lux \
    --symbol LUX

  # Import from Ethereum
  archeology import-token \
    --network ethereum \
    --chain-id 1 \
    --contract 0xUSDC_ADDRESS \
    --project usdc \
    --symbol USDC \
    --decimals 6`,
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
				case "7777", "lux-7777", "local":
					rpcURL = "http://localhost:9650/ext/bc/C/rpc"
					if chainID == 0 {
						chainID = 7777
					}
				case "96369", "lux-mainnet":
					rpcURL = "http://localhost:9650/ext/bc/C/rpc"
					if chainID == 0 {
						chainID = 96369
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
				tokenName := tokenSymbol
				if tokenName == "" {
					tokenName = "token"
				}
				outputPath = fmt.Sprintf("exports/%s-%s-%s.csv", projectName, strings.ToLower(tokenName), networkName)
			}

			// Create scanner config
			config := scanner.Config{
				Chain:           network,
				RPC:             rpcURL,
				ContractAddress: contractAddress,
				ContractType:    "token",
				OutputPath:      outputPath,
				BlockRange:      blockRange,
				ProjectName:     projectName,
				CrossRefPath:    crossRefPath,
			}

			// Log configuration
			log.Printf("Import Token Configuration:")
			log.Printf("  Network: %s", network)
			log.Printf("  Chain ID: %d", chainID)
			log.Printf("  RPC URL: %s", rpcURL)
			log.Printf("  Contract: %s", contractAddress)
			log.Printf("  Project: %s", projectName)
			if tokenSymbol != "" {
				log.Printf("  Symbol: %s", tokenSymbol)
			}
			if tokenDecimals > 0 {
				log.Printf("  Decimals: %d", tokenDecimals)
			}
			log.Printf("  Output: %s", outputPath)
			log.Printf("  Block Range: %d", blockRange)

			// Create scanner
			s, err := scanner.New(config)
			if err != nil {
				return fmt.Errorf("failed to create scanner: %w", err)
			}

			// Run scan
			log.Printf("\nStarting token import...")
			result, err := s.Scan()
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			// Print results
			fmt.Printf("\n✅ Token Import Complete!\n")
			fmt.Printf("Chain: %s (ID: %d)\n", result.Chain, chainID)
			fmt.Printf("Contract: %s\n", result.ContractAddress)
			fmt.Printf("Total Holders: %d\n", result.TotalHolders)
			if result.TotalSupply != "" {
				fmt.Printf("Total Supply: %s tokens\n", result.TotalSupply)
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
import-%s-tokens:
	@./bin/archeology import-token \
		--network %s \
		--chain-id %d \
		--contract %s \
		--project %s`,
				projectName,
				network,
				chainID,
				contractAddress,
				projectName)

			if tokenSymbol != "" {
				makefileSnippet += fmt.Sprintf(" \\\n\t\t--symbol %s", tokenSymbol)
			}
			if tokenDecimals > 0 {
				makefileSnippet += fmt.Sprintf(" \\\n\t\t--decimals %d", tokenDecimals)
			}
			makefileSnippet += fmt.Sprintf(" \\\n\t\t--output %s", outputPath)

			fmt.Printf("\n%s\n", makefileSnippet)

			// Special note for local chains
			if chainID == 7777 || chainID == 96369 || strings.Contains(rpcURL, "localhost") {
				fmt.Printf("\n💡 Tip: For local chain imports, make sure your node is running with:\n")
				fmt.Printf("   make run network=%d\n", chainID)
				fmt.Printf("   Or: luxd --network-id=%d --http-host=0.0.0.0\n", chainID)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&network, "network", "n", "", "Network name (ethereum, bsc, polygon, 7777, 96369, etc.)")
	cmd.Flags().Int64VarP(&chainID, "chain-id", "c", 0, "Chain ID (e.g., 1 for Ethereum, 56 for BSC, 7777 for old Lux)")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "Token contract address (required)")
	cmd.Flags().StringVar(&rpcURL, "rpc", "", "RPC endpoint URL (overrides network defaults)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output CSV file path")
	cmd.Flags().Int64Var(&blockRange, "block-range", 500000, "Number of blocks to scan backwards from current")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Project name (lux, zoo, spc, hanzo, usdc, etc.) (required)")
	cmd.Flags().StringVar(&crossRefPath, "cross-ref", "", "Cross-reference CSV file to check existing holders")
	cmd.Flags().StringVarP(&tokenSymbol, "symbol", "s", "", "Token symbol (optional, for documentation)")
	cmd.Flags().IntVar(&tokenDecimals, "decimals", 18, "Token decimals (default: 18)")

	// Mark required flags
	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("project")

	return cmd
}

// TokenNetworkInfo contains token-specific network configuration
type TokenNetworkInfo struct {
	Name      string
	ChainID   int64
	RPC       string
	LocalPort int // For local chains
}

// GetTokenNetworks returns a list of networks commonly used for tokens
func GetTokenNetworks() []TokenNetworkInfo {
	return []TokenNetworkInfo{
		// Public networks
		{Name: "ethereum", ChainID: 1, RPC: "https://eth.llamarpc.com"},
		{Name: "bsc", ChainID: 56, RPC: "https://bsc-dataseed.binance.org"},
		{Name: "polygon", ChainID: 137, RPC: "https://polygon-rpc.com"},
		{Name: "arbitrum", ChainID: 42161, RPC: "https://arb1.arbitrum.io/rpc"},
		{Name: "optimism", ChainID: 10, RPC: "https://mainnet.optimism.io"},
		{Name: "avalanche", ChainID: 43114, RPC: "https://api.avax.network/ext/bc/C/rpc"},
		
		// Local Lux chains
		{Name: "lux-7777", ChainID: 7777, RPC: "http://localhost:9650/ext/bc/C/rpc", LocalPort: 9650},
		{Name: "lux-96369", ChainID: 96369, RPC: "http://localhost:9650/ext/bc/C/rpc", LocalPort: 9650},
		{Name: "lux-96368", ChainID: 96368, RPC: "http://localhost:9650/ext/bc/C/rpc", LocalPort: 9650},
		
		// Lux subnets (local)
		{Name: "zoo-200200", ChainID: 200200, RPC: "http://localhost:9650/ext/bc/zoo/rpc", LocalPort: 9650},
		{Name: "spc-36911", ChainID: 36911, RPC: "http://localhost:9650/ext/bc/spc/rpc", LocalPort: 9650},
		{Name: "hanzo-36963", ChainID: 36963, RPC: "http://localhost:9650/ext/bc/hanzo/rpc", LocalPort: 9650},
	}
}