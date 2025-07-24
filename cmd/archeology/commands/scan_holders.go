package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/scanner"
)

// NewScanHoldersCommand creates the scan-holders command
func NewScanHoldersCommand() *cobra.Command {
	var (
		rpc             string
		contractAddress string
		contractType    string
		fromBlock       uint64
		toBlock         uint64
		outputCSV       string
		outputJSON      string
		includeTokenIDs bool
		topN            int
		showDistribution bool
	)

	cmd := &cobra.Command{
		Use:   "scan-holders",
		Short: "Scan for current token/NFT holders",
		Long: `Scans blockchain to find all current holders by processing Transfer events.

Supports both:
- ERC20 tokens (fungible)
- ERC721 NFTs (non-fungible)

This command tracks all transfers from contract deployment to build current ownership state.`,
		Example: `  # Scan EGG NFT holders on BSC
  archaeology scan-holders \
    --rpc https://bsc-dataseed.binance.org/ \
    --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
    --type nft \
    --output egg-holders.csv

  # Scan token holders with distribution
  archaeology scan-holders \
    --rpc https://bsc-dataseed.binance.org/ \
    --contract 0x0a6045b79151d0a54dbd5227082445750a023af2 \
    --type token \
    --top 20 --show-distribution`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if contractType == "nft" || contractType == "erc721" {
				return scanNFTHolders(rpc, contractAddress, fromBlock, toBlock, outputCSV, outputJSON, includeTokenIDs, topN, showDistribution)
			} else if contractType == "token" || contractType == "erc20" {
				return scanTokenHolders(rpc, contractAddress, fromBlock, toBlock, outputCSV, outputJSON, topN, showDistribution)
			} else {
				return fmt.Errorf("invalid contract type: %s (must be 'nft' or 'token')", contractType)
			}
		},
	}

	// Add flags
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "Contract address")
	cmd.Flags().StringVar(&contractType, "type", "nft", "Contract type: nft or token")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().BoolVar(&includeTokenIDs, "include-token-ids", false, "Include token IDs in NFT output")
	cmd.Flags().IntVar(&topN, "top", 0, "Show top N holders")
	cmd.Flags().BoolVar(&showDistribution, "show-distribution", false, "Show holder distribution")

	cmd.MarkFlagRequired("rpc")
	cmd.MarkFlagRequired("contract")

	return cmd
}

func scanNFTHolders(rpc, contractAddress string, fromBlock, toBlock uint64, outputCSV, outputJSON string, includeTokenIDs bool, topN int, showDistribution bool) error {
	// Create scanner config
	config := &scanner.NFTHolderScanConfig{
		RPC:             rpc,
		ContractAddress: contractAddress,
		FromBlock:       fromBlock,
		ToBlock:         toBlock,
		IncludeTokenIDs: includeTokenIDs,
	}

	// Create scanner
	nftScanner, err := scanner.NewNFTHolderScanner(config)
	if err != nil {
		return fmt.Errorf("failed to create NFT scanner: %w", err)
	}
	defer nftScanner.Close()

	log.Printf("Scanning NFT contract %s", contractAddress)

	// Scan holders
	holders, err := nftScanner.ScanHolders()
	if err != nil {
		return fmt.Errorf("failed to scan holders: %w", err)
	}

	log.Printf("Found %d unique holders", len(holders))

	// Calculate total NFTs
	totalNFTs := 0
	for _, holder := range holders {
		totalNFTs += holder.TokenCount
	}
	log.Printf("Total NFTs held: %d", totalNFTs)

	// Export to CSV if requested
	if outputCSV != "" {
		metadata := map[string]string{
			"Contract": contractAddress,
			"Type":     "NFT",
		}
		if err := scanner.ExportNFTHoldersToCSV(holders, outputCSV, metadata); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		log.Printf("Exported holders to %s", outputCSV)
	}

	// Export to JSON if requested
	if outputJSON != "" {
		data := map[string]interface{}{
			"contract":     contractAddress,
			"type":         "NFT",
			"totalHolders": len(holders),
			"totalNFTs":    totalNFTs,
			"holders":      holders,
		}
		if err := scanner.ExportToJSON(data, outputJSON); err != nil {
			return fmt.Errorf("failed to export JSON: %w", err)
		}
		log.Printf("Exported holders to %s", outputJSON)
	}

	// Show distribution if requested
	if showDistribution {
		distribution := scanner.GetHolderDistribution(holders)
		fmt.Printf("\n=== Holder Distribution ===\n")
		for category, count := range distribution {
			fmt.Printf("%-15s: %d holders\n", category, count)
		}
	}

	// Show top holders if requested
	if topN > 0 {
		topHolders, err := nftScanner.GetTopHolders(topN)
		if err != nil {
			return fmt.Errorf("failed to get top holders: %w", err)
		}

		fmt.Printf("\n=== Top %d Holders ===\n", topN)
		for i, holder := range topHolders {
			fmt.Printf("%2d. %s: %d NFTs\n", i+1, holder.Address, holder.TokenCount)
		}
	}

	// Summary
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Contract: %s\n", contractAddress)
	fmt.Printf("Total holders: %d\n", len(holders))
	fmt.Printf("Total NFTs: %d\n", totalNFTs)
	if len(holders) > 0 {
		avgHolding := float64(totalNFTs) / float64(len(holders))
		fmt.Printf("Average holding: %.2f NFTs\n", avgHolding)
	}

	return nil
}

func scanTokenHolders(rpc, contractAddress string, fromBlock, toBlock uint64, outputCSV, outputJSON string, topN int, showDistribution bool) error {
	// For ERC20 tokens, we can use the transfer scanner to build holder balances
	config := &scanner.TokenTransferScanConfig{
		RPC:          rpc,
		TokenAddress: contractAddress,
		FromBlock:    fromBlock,
		ToBlock:      toBlock,
	}

	transferScanner, err := scanner.NewTokenTransferScanner(config)
	if err != nil {
		return fmt.Errorf("failed to create transfer scanner: %w", err)
	}
	defer transferScanner.Close()

	log.Printf("Scanning token contract %s", contractAddress)

	// Scan all transfers
	transfers, err := transferScanner.ScanTransfers()
	if err != nil {
		return fmt.Errorf("failed to scan transfers: %w", err)
	}

	log.Printf("Found %d transfers", len(transfers))

	// Calculate balance changes
	balances := scanner.GetBalanceChanges(transfers)
	
	// Filter out zero/negative balances
	holders := []scanner.NFTHolder{} // Reuse NFTHolder struct for consistency
	for addr, balance := range balances {
		if balance.Sign() > 0 {
			holders = append(holders, scanner.NFTHolder{
				Address:    addr,
				TokenCount: 1, // For sorting purposes
				TokenIDs:   []string{balance.String()}, // Store balance as "token ID"
			})
		}
	}

	log.Printf("Found %d holders with positive balances", len(holders))

	// Export if requested
	if outputCSV != "" {
		metadata := map[string]string{
			"Contract": contractAddress,
			"Type":     "Token",
		}
		if err := scanner.ExportNFTHoldersToCSV(holders, outputCSV, metadata); err != nil {
			return fmt.Errorf("failed to export CSV: %w", err)
		}
		log.Printf("Exported holders to %s", outputCSV)
	}

	if outputJSON != "" {
		data := map[string]interface{}{
			"contract":     contractAddress,
			"type":         "Token",
			"totalHolders": len(holders),
			"holders":      holders,
		}
		if err := scanner.ExportToJSON(data, outputJSON); err != nil {
			return fmt.Errorf("failed to export JSON: %w", err)
		}
		log.Printf("Exported holders to %s", outputJSON)
	}

	// Summary
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Contract: %s\n", contractAddress)
	fmt.Printf("Total holders with positive balance: %d\n", len(holders))

	return nil
}