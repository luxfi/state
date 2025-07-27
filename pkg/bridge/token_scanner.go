package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"strings"

	"github.com/luxfi/geth"
	"github.com/luxfi/geth/accounts/abi"
	"github.com/luxfi/geth/accounts/abi/bind"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
)

// ERC20 ABI events and methods
const erc20ABI = `[
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"},
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}
]`

// TokenScanner handles token scanning from external chains
type TokenScanner struct {
	config      TokenScannerConfig
	client      *ethclient.Client
	contractABI abi.ABI
}

// NewTokenScanner creates a new token scanner
func NewTokenScanner(config TokenScannerConfig) (*TokenScanner, error) {
	if config.ContractAddress == "" {
		return nil, fmt.Errorf("contract address is required")
	}
	if config.ProjectName == "" {
		return nil, fmt.Errorf("project name is required")
	}
	
	// Set default RPC if not provided
	if config.RPC == "" {
		switch config.Chain {
		case "ethereum", "eth":
			config.RPC = "https://eth.llamarpc.com"
		case "bsc", "binance":
			config.RPC = "https://bsc-dataseed.binance.org/"
		case "polygon":
			config.RPC = "https://polygon-rpc.com"
		case "7777", "96369":
			// Local chains
			config.RPC = "http://localhost:9650/ext/bc/C/rpc"
		default:
			return nil, fmt.Errorf("RPC endpoint required for chain: %s", config.Chain)
		}
	}
	
	// Connect to the Ethereum client
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	
	// Parse the ABI
	contractABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	
	return &TokenScanner{
		config:      config,
		client:      client,
		contractABI: contractABI,
	}, nil
}

// Scan performs the token scan
func (s *TokenScanner) Scan() (*TokenScanResult, error) {
	ctx := context.Background()
	
	// Get contract address
	contractAddr := common.HexToAddress(s.config.ContractAddress)
	
	// Get current block number
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}
	
	// Set ToBlock if not specified
	if s.config.ToBlock == 0 {
		s.config.ToBlock = currentBlock
	}
	
	// Create contract instance
	contract := bind.NewBoundContract(contractAddr, s.contractABI, s.client, s.client, s.client)
	
	// Get token info
	var name, symbol string
	var decimals uint8
	var totalSupply *big.Int
	
	// Get name
	results := []interface{}{&name}
	err = contract.Call(nil, &results, "name")
	if err != nil {
		log.Printf("Warning: failed to get token name: %v", err)
		name = "Unknown Token"
	}
	
	// Get symbol
	results = []interface{}{&symbol}
	err = contract.Call(nil, &results, "symbol")
	if err != nil {
		log.Printf("Warning: failed to get token symbol: %v", err)
		symbol = "UNKNOWN"
	}
	
	// Get decimals
	results = []interface{}{&decimals}
	err = contract.Call(nil, &results, "decimals")
	if err != nil {
		log.Printf("Warning: failed to get decimals, defaulting to 18: %v", err)
		decimals = 18
	}
	
	// Get total supply
	results = []interface{}{&totalSupply}
	err = contract.Call(nil, &results, "totalSupply")
	if err != nil {
		return nil, fmt.Errorf("failed to get total supply: %w", err)
	}
	
	// Scan Transfer events to find all token holders
	balances := make(map[common.Address]*big.Int)
	
	// Define the Transfer event signature
	transferEventSig := s.contractABI.Events["Transfer"].ID
	
	// Set FromBlock if not specified
	fromBlock := s.config.FromBlock
	if fromBlock == 0 {
		// Default to scanning last 1M blocks
		fromBlock = currentBlock - 1000000
		if fromBlock < 0 {
			fromBlock = 0
		}
	}
	
	chunkSize := uint64(10000)
	totalScanned := 0
	
	log.Printf("Scanning token %s (%s) from block %d to %d", name, symbol, fromBlock, s.config.ToBlock)
	
	for startBlock := fromBlock; startBlock <= s.config.ToBlock; startBlock += chunkSize {
		endBlock := startBlock + chunkSize - 1
		if endBlock > s.config.ToBlock {
			endBlock = s.config.ToBlock
		}
		
		// Create filter query
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{contractAddr},
			Topics:    [][]common.Hash{{transferEventSig}},
		}
		
		// Get logs
		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}
		
		// Process Transfer events
		for _, vLog := range logs {
			var from, to common.Address
			var value *big.Int
			
			// Parse indexed topics (from and to)
			if len(vLog.Topics) >= 3 {
				from = common.HexToAddress(vLog.Topics[1].Hex())
				to = common.HexToAddress(vLog.Topics[2].Hex())
			}
			
			// Parse data (value)
			if len(vLog.Data) >= 32 {
				value = new(big.Int).SetBytes(vLog.Data)
			} else {
				continue
			}
			
			// Update balances
			if from != common.HexToAddress("0x0") {
				if balances[from] == nil {
					balances[from] = new(big.Int)
				}
				balances[from].Sub(balances[from], value)
			}
			
			if to != common.HexToAddress("0x0") {
				if balances[to] == nil {
					balances[to] = new(big.Int)
				}
				balances[to].Add(balances[to], value)
			}
		}
		
		totalScanned += len(logs)
		
		// Progress indicator
		if (endBlock-fromBlock)%100000 == 0 {
			log.Printf("Scanned up to block %d/%d (processed %d transfers)", endBlock, s.config.ToBlock, totalScanned)
		}
	}
	
	// Clean up zero or negative balances
	holders := make([]TokenHolder, 0)
	totalHeldSupply := new(big.Int)
	
	for addr, balance := range balances {
		if balance.Cmp(big.NewInt(0)) > 0 {
			// Calculate formatted balance
			divisor := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
			balanceFloat := new(big.Float).SetInt(balance)
			balanceFloat.Quo(balanceFloat, divisor)
			
			// Format with thousands separators
			balanceStr := fmt.Sprintf("%.2f", balanceFloat)
			formatted := formatNumber(balanceStr) + " " + symbol
			
			// Calculate percentage
			percentage := new(big.Float).SetInt(balance)
			percentage.Mul(percentage, big.NewFloat(100))
			percentage.Quo(percentage, new(big.Float).SetInt(totalSupply))
			percentFloat, _ := percentage.Float64()
			
			holders = append(holders, TokenHolder{
				Address:          addr.Hex(),
				Balance:          balance.String(),
				BalanceFormatted: formatted,
				Percentage:       percentFloat,
			})
			
			totalHeldSupply.Add(totalHeldSupply, balance)
		}
	}
	
	// Sort holders by balance
	sort.Slice(holders, func(i, j int) bool {
		balI := new(big.Int)
		balJ := new(big.Int)
		balI.SetString(holders[i].Balance, 10)
		balJ.SetString(holders[j].Balance, 10)
		return balI.Cmp(balJ) > 0
	})
	
	// Get top holders
	topHolders := holders
	if len(topHolders) > 20 {
		topHolders = holders[:20]
	}
	
	// Calculate distribution tiers
	distribution := calculateDistribution(holders, decimals)
	
	// Determine migration info
	migrationInfo := &MigrationInfo{
		HoldersToMigrate: len(holders),
		BalanceToMigrate: totalHeldSupply.String(),
		RecommendedLayer: "L2", // Default recommendation
	}
	
	// Adjust recommendation based on holder count and value
	if len(holders) < 1000 {
		migrationInfo.RecommendedLayer = "L1"
	} else if len(holders) > 10000 {
		migrationInfo.RecommendedLayer = "L3"
	}
	
	result := &TokenScanResult{
		ContractAddress:  s.config.ContractAddress,
		TokenName:        name,
		Symbol:           symbol,
		Decimals:         int(decimals),
		TotalSupply:      totalSupply.String(),
		UniqueHolders:    len(holders),
		FromBlock:        fromBlock,
		ToBlock:          s.config.ToBlock,
		Distribution:     distribution,
		TopHolders:       topHolders,
		MigrationInfo:    migrationInfo,
		Holders:          holders, // Include all holders
	}
	
	log.Printf("Scan complete: found %d token holders for %s (%s)", len(holders), name, symbol)
	
	return result, nil
}

// Export exports the scan results
func (s *TokenScanner) Export(outputPath string) error {
	result, err := s.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}
	
	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	// Export as JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	
	log.Printf("Exported scan results to %s", outputPath)
	
	return nil
}

// GetTopHolders returns only the top N holders
func (s *TokenScanner) GetTopHolders(limit int) ([]TokenHolder, error) {
	result, err := s.Scan()
	if err != nil {
		return nil, err
	}
	
	if limit > len(result.TopHolders) {
		limit = len(result.TopHolders)
	}
	
	return result.TopHolders[:limit], nil
}

// GetHoldersByMinBalance returns holders with balance above threshold
func (s *TokenScanner) GetHoldersByMinBalance(minBalance string) ([]TokenHolder, error) {
	result, err := s.Scan()
	if err != nil {
		return nil, err
	}
	
	threshold := new(big.Int)
	if _, ok := threshold.SetString(minBalance, 10); !ok {
		return nil, fmt.Errorf("invalid minimum balance: %s", minBalance)
	}
	
	filtered := []TokenHolder{}
	for _, holder := range result.Holders {
		balance := new(big.Int)
		balance.SetString(holder.Balance, 10)
		if balance.Cmp(threshold) >= 0 {
			filtered = append(filtered, holder)
		}
	}
	
	return filtered, nil
}

// calculateDistribution calculates token distribution tiers
func calculateDistribution(holders []TokenHolder, decimals uint8) []DistributionTier {
	// Define tier thresholds (in token units)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	
	tiers := []struct {
		name      string
		threshold *big.Int
	}{
		{"Whale (>1M)", new(big.Int).Mul(big.NewInt(1000000), divisor)},
		{"Large (100K-1M)", new(big.Int).Mul(big.NewInt(100000), divisor)},
		{"Medium (10K-100K)", new(big.Int).Mul(big.NewInt(10000), divisor)},
		{"Small (1K-10K)", new(big.Int).Mul(big.NewInt(1000), divisor)},
		{"Micro (<1K)", big.NewInt(0)},
	}
	
	distribution := make([]DistributionTier, len(tiers))
	totalValue := new(big.Int)
	
	// Calculate total value
	for _, holder := range holders {
		balance := new(big.Int)
		balance.SetString(holder.Balance, 10)
		totalValue.Add(totalValue, balance)
	}
	
	// Count holders and value in each tier
	for i, tier := range tiers {
		count := 0
		tierValue := new(big.Int)
		
		for _, holder := range holders {
			balance := new(big.Int)
			balance.SetString(holder.Balance, 10)
			
			inTier := false
			if i == 0 {
				// Top tier
				inTier = balance.Cmp(tier.threshold) >= 0
			} else if i < len(tiers)-1 {
				// Middle tiers
				inTier = balance.Cmp(tier.threshold) >= 0 && balance.Cmp(tiers[i-1].threshold) < 0
			} else {
				// Bottom tier
				inTier = balance.Cmp(tiers[i-1].threshold) < 0
			}
			
			if inTier {
				count++
				tierValue.Add(tierValue, balance)
			}
		}
		
		// Calculate percentage of total value
		percentage := new(big.Float).SetInt(tierValue)
		percentage.Mul(percentage, big.NewFloat(100))
		if totalValue.Cmp(big.NewInt(0)) > 0 {
			percentage.Quo(percentage, new(big.Float).SetInt(totalValue))
		}
		percentFloat, _ := percentage.Float64()
		
		distribution[i] = DistributionTier{
			Range:      tier.name,
			Count:      count,
			Percentage: percentFloat,
		}
	}
	
	return distribution
}

// formatNumber adds thousands separators to a number string
func formatNumber(s string) string {
	parts := strings.Split(s, ".")
	intPart := parts[0]
	
	// Add commas
	n := len(intPart)
	if n <= 3 {
		return s
	}
	
	var result strings.Builder
	for i, digit := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}
	
	if len(parts) > 1 {
		result.WriteRune('.')
		result.WriteString(parts[1])
	}
	
	return result.String()
}

// Close closes the scanner connection
func (s *TokenScanner) Close() {
	if s.client != nil {
		s.client.Close()
	}
}