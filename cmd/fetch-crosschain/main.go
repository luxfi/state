package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	
	"github.com/luxfi/genesis/pkg/crosschain"
)

var (
	cfg = struct {
		Chain       string
		TokenAddr   string
		OutputDir   string
		BlockNumber int64
		CacheDir    string
	}{
		OutputDir: "chaindata",
		CacheDir:  "chaindata/.cache",
	}
)

// Predefined chain configurations
var chainConfigs = map[string]*crosschain.ChainConfig{
	"bsc": {
		Name:    "bsc-mainnet",
		ChainID: big.NewInt(56),
		RPCURLs: []string{
			"https://bsc-dataseed.binance.org/",
			"https://bsc-dataseed1.defibit.io/",
			"https://bsc-dataseed1.ninicoin.io/",
		},
	},
	"eth": {
		Name:    "eth-mainnet",
		ChainID: big.NewInt(1),
		RPCURLs: []string{
			"https://eth.llamarpc.com",
			"https://ethereum.publicnode.com",
			"https://eth-mainnet.public.blastapi.io",
		},
	},
}

// Known token addresses
var tokenAddresses = map[string]map[string]string{
	"bsc": {
		"zoo": "0x7cd05c8f51b89df17e1e3ffe45d0c210b934bf67",
	},
	"eth": {
		"lux-nft": "0x6B813b7Ae93f065ddD8AC9EfDE7C1922A90d2Fb2",
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.Chain, "chain", "", "Chain to fetch from (bsc, eth)")
	rootCmd.PersistentFlags().StringVar(&cfg.TokenAddr, "token", "", "Token contract address")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "Output directory")
	rootCmd.PersistentFlags().Int64Var(&cfg.BlockNumber, "block", 0, "Block number (0 for latest)")
	rootCmd.PersistentFlags().StringVar(&cfg.CacheDir, "cache", cfg.CacheDir, "Cache directory")
}

var rootCmd = &cobra.Command{
	Use:   "fetch-crosschain",
	Short: "Fetch and cache cross-chain data",
	Long:  "Tool for fetching token data from other blockchains with local caching",
}

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch data from specified chain",
	RunE:  runFetch,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available chains and tokens",
	Run:   runList,
}

func runList(cmd *cobra.Command, args []string) {
	fmt.Println("Available chains and tokens:")
	fmt.Println()
	
	for chain, config := range chainConfigs {
		fmt.Printf("%s (Chain ID: %s)\n", chain, config.ChainID)
		if tokens, ok := tokenAddresses[chain]; ok {
			for name, addr := range tokens {
				fmt.Printf("  - %s: %s\n", name, addr)
			}
		}
		fmt.Println()
	}
}

func runFetch(cmd *cobra.Command, args []string) error {
	// Validate chain
	if cfg.Chain == "" {
		return fmt.Errorf("chain is required")
	}
	
	chainConfig, ok := chainConfigs[cfg.Chain]
	if !ok {
		return fmt.Errorf("unknown chain: %s", cfg.Chain)
	}
	
	// Set cache directory
	chainConfig.CacheDir = cfg.CacheDir
	
	// Create client
	client, err := crosschain.NewClient(chainConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()
	
	ctx := context.Background()
	
	// Get block number
	var blockNumber *big.Int
	if cfg.BlockNumber > 0 {
		blockNumber = big.NewInt(cfg.BlockNumber)
	} else {
		blockNumber, err = client.GetLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest block: %w", err)
		}
	}
	
	fmt.Printf("Using block: %s\n", blockNumber)
	
	// Create output directory
	outputDir := filepath.Join(cfg.OutputDir, chainConfig.Name)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}
	
	// Save metadata
	metadata := map[string]interface{}{
		"chain":       chainConfig.Name,
		"chainId":     chainConfig.ChainID,
		"blockNumber": blockNumber,
		"timestamp":   time.Now().Format(time.RFC3339),
		"tokenAddr":   cfg.TokenAddr,
	}
	
	metadataPath := filepath.Join(outputDir, "metadata.json")
	if err := saveJSON(metadataPath, metadata); err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	
	// Fetch token data if address provided
	if cfg.TokenAddr != "" {
		tokenAddr := common.HexToAddress(cfg.TokenAddr)
		
		// Create scanner
		scanner := crosschain.NewEventScanner(client)
		
		// Define scan range
		startBlock := big.NewInt(0)
		
		// For BSC Zoo token, start from deployment block
		if cfg.Chain == "bsc" && strings.ToLower(cfg.TokenAddr) == strings.ToLower(tokenAddresses["bsc"]["zoo"]) {
			startBlock = big.NewInt(14000000) // Approximate Zoo deployment block
		}
		
		// Try to get burn events
		fmt.Printf("Scanning for burn events from block %s to %s...\n", startBlock, blockNumber)
		burns, err := scanner.ScanBurnEvents(ctx, tokenAddr, startBlock, blockNumber)
		if err != nil {
			log.Printf("Failed to scan burn events: %v", err)
			// Fall back to cached method
			burns, _ = client.GetBurnEvents(ctx, tokenAddr, startBlock, blockNumber)
		}
		
		// Save burn events
		if len(burns) > 0 {
			burnsPath := filepath.Join(outputDir, "burn_events.json")
			if err := saveJSON(burnsPath, burns); err != nil {
				log.Printf("Failed to save burns: %v", err)
			}
			
			// Create CSV
			csvPath := filepath.Join(outputDir, "burn_events.csv")
			if err := saveBurnsCSV(csvPath, burns); err != nil {
				log.Printf("Failed to save CSV: %v", err)
			}
			
			// Save burn addresses summary
			burnSummary := make(map[string]*big.Int)
			for _, burn := range burns {
				if current, ok := burnSummary[burn.From.Hex()]; ok {
					burnSummary[burn.From.Hex()] = new(big.Int).Add(current, burn.Amount)
				} else {
					burnSummary[burn.From.Hex()] = burn.Amount
				}
			}
			
			summaryPath := filepath.Join(outputDir, "burn_address_summary.json")
			saveJSON(summaryPath, burnSummary)
		}
		
		fmt.Printf("Found %d burn events\n", len(burns))
		
		// For full holder snapshot, would need to scan all transfers
		// For now, just save a note
		holders := []crosschain.TokenHolder{
			{
				Note: fmt.Sprintf("Full holder snapshot requires scanning all %d blocks", blockNumber),
			},
		}
		
		holdersPath := filepath.Join(outputDir, "token_holders.json")
		if err := saveJSON(holdersPath, holders); err != nil {
			log.Printf("Failed to save holders: %v", err)
		}
	}
	
	fmt.Printf("\nData saved to: %s\n", outputDir)
	return nil
}

func saveJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

func saveBurnsCSV(path string, burns []crosschain.BurnEvent) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header
	if err := writer.Write([]string{"address", "amount", "block_number", "tx_hash"}); err != nil {
		return err
	}
	
	// Write data
	for _, burn := range burns {
		record := []string{
			burn.From.Hex(),
			burn.Amount.String(),
			fmt.Sprintf("%d", burn.BlockNumber),
			burn.TransactionHash.Hex(),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

func main() {
	rootCmd.AddCommand(fetchCmd)
	rootCmd.AddCommand(listCmd)
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}