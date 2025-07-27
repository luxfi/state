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
	"time"

	"github.com/luxfi/geth"
	"github.com/luxfi/geth/accounts/abi"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
)

const (
	// Zoo token contract on BSC
	ZooTokenAddress = "0x0a6045b79151d0a54dbd5227082445750a023af2"
	// Dead address for burns
	DeadAddress = "0x000000000000000000000000000000000000dEaD"
	// EGG NFT contract on BSC
	EggNFTAddress = "0x5bb68cf06289d54efde25155c88003be685356a8"
)

// ZooMigrationScanner handles the special Zoo token migration logic
type ZooMigrationScanner struct {
	client      *ethclient.Client
	contractABI abi.ABI
	config      ZooMigrationConfig
}

// ZooMigrationConfig contains configuration for Zoo migration scanning
type ZooMigrationConfig struct {
	RPC             string
	FromBlock       uint64
	ToBlock         uint64
	IncludeBurns    bool
	IncludeEggNFTs  bool
	OutputPath      string
}

// ZooHolder represents a Zoo token holder including burn amounts
type ZooHolder struct {
	Address         string `json:"address"`
	Balance         string `json:"balance"`
	BurnedAmount    string `json:"burnedAmount,omitempty"`
	TotalAllocation string `json:"totalAllocation"`
	HasEggNFT       bool   `json:"hasEggNFT,omitempty"`
	EggNFTCount     int    `json:"eggNFTCount,omitempty"`
}

// ZooMigrationResult contains the complete migration data
type ZooMigrationResult struct {
	TokenAddress    string       `json:"tokenAddress"`
	TotalSupply     string       `json:"totalSupply"`
	CirculatingSupply string    `json:"circulatingSupply"`
	BurnedSupply    string       `json:"burnedSupply"`
	UniqueHolders   int          `json:"uniqueHolders"`
	HoldersWithBurns int         `json:"holdersWithBurns"`
	EggNFTHolders   int          `json:"eggNFTHolders"`
	Holders         []ZooHolder  `json:"holders"`
	ScanBlock       uint64       `json:"scanBlock"`
	Timestamp       int64        `json:"timestamp"`
}

// NewZooMigrationScanner creates a new Zoo migration scanner
func NewZooMigrationScanner(config ZooMigrationConfig) (*ZooMigrationScanner, error) {
	if config.RPC == "" {
		config.RPC = "https://bsc-dataseed.binance.org/"
	}
	
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to BSC: %w", err)
	}
	
	contractABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	
	return &ZooMigrationScanner{
		client:      client,
		contractABI: contractABI,
		config:      config,
	}, nil
}

// ScanZooMigration performs the complete Zoo token migration scan
func (s *ZooMigrationScanner) ScanZooMigration() (*ZooMigrationResult, error) {
	ctx := context.Background()
	
	// Get current block
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}
	
	if s.config.ToBlock == 0 {
		s.config.ToBlock = currentBlock
	}
	
	log.Printf("Scanning Zoo token migration data from BSC...")
	log.Printf("Token: %s", ZooTokenAddress)
	log.Printf("Dead address: %s", DeadAddress)
	
	// Track balances and burns
	balances := make(map[common.Address]*big.Int)
	burns := make(map[common.Address]*big.Int)
	totalBurned := big.NewInt(0)
	
	// Scan Transfer events
	contractAddr := common.HexToAddress(ZooTokenAddress)
	deadAddr := common.HexToAddress(DeadAddress)
	
	transferEventSig := s.contractABI.Events["Transfer"].ID
	
	fromBlock := s.config.FromBlock
	if fromBlock == 0 {
		// BSC mainnet launched around block 0, but Zoo token was deployed later
		// Start from a reasonable block number to optimize scanning
		fromBlock = 10000000 // Adjust based on actual deployment block
	}
	
	chunkSize := uint64(5000) // Smaller chunks for BSC
	
	for startBlock := fromBlock; startBlock <= s.config.ToBlock; startBlock += chunkSize {
		endBlock := startBlock + chunkSize - 1
		if endBlock > s.config.ToBlock {
			endBlock = s.config.ToBlock
		}
		
		// Query for all Transfer events
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{contractAddr},
			Topics:    [][]common.Hash{{transferEventSig}},
		}
		
		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}
		
		// Process transfers
		for _, vLog := range logs {
			var from, to common.Address
			var value *big.Int
			
			if len(vLog.Topics) >= 3 {
				from = common.HexToAddress(vLog.Topics[1].Hex())
				to = common.HexToAddress(vLog.Topics[2].Hex())
			}
			
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
				
				// Track burns to dead address
				if to == deadAddr && from != common.HexToAddress("0x0") {
					if burns[from] == nil {
						burns[from] = new(big.Int)
					}
					burns[from].Add(burns[from], value)
					totalBurned.Add(totalBurned, value)
				}
			}
		}
		
		if (endBlock-startBlock)%50000 == 0 {
			log.Printf("Scanned up to block %d/%d", endBlock, s.config.ToBlock)
		}
	}
	
	// Get EGG NFT holders if requested
	eggHolders := make(map[common.Address]int)
	if s.config.IncludeEggNFTs {
		log.Printf("Scanning EGG NFT holders...")
		eggHolders, err = s.scanEggNFTHolders()
		if err != nil {
			log.Printf("Warning: failed to scan EGG NFTs: %v", err)
		}
	}
	
	// Build holder list with burns included
	holders := []ZooHolder{}
	holdersWithBurns := 0
	eggNFTHolderCount := 0
	
	for addr, balance := range balances {
		if balance.Cmp(big.NewInt(0)) > 0 || (s.config.IncludeBurns && burns[addr] != nil && burns[addr].Cmp(big.NewInt(0)) > 0) {
			burnAmount := burns[addr]
			if burnAmount == nil {
				burnAmount = big.NewInt(0)
			}
			
			// Calculate total allocation (balance + burns)
			totalAllocation := new(big.Int).Set(balance)
			if s.config.IncludeBurns && burnAmount.Cmp(big.NewInt(0)) > 0 {
				totalAllocation.Add(totalAllocation, burnAmount)
				holdersWithBurns++
			}
			
			// Skip if total allocation is 0 or negative
			if totalAllocation.Cmp(big.NewInt(0)) <= 0 {
				continue
			}
			
			holder := ZooHolder{
				Address:         addr.Hex(),
				Balance:         balance.String(),
				TotalAllocation: totalAllocation.String(),
			}
			
			if burnAmount.Cmp(big.NewInt(0)) > 0 {
				holder.BurnedAmount = burnAmount.String()
			}
			
			// Check EGG NFT ownership
			if eggCount, hasEgg := eggHolders[addr]; hasEgg {
				holder.HasEggNFT = true
				holder.EggNFTCount = eggCount
				eggNFTHolderCount++
			}
			
			holders = append(holders, holder)
		}
	}
	
	// Sort by total allocation
	sort.Slice(holders, func(i, j int) bool {
		allocI := new(big.Int)
		allocJ := new(big.Int)
		allocI.SetString(holders[i].TotalAllocation, 10)
		allocJ.SetString(holders[j].TotalAllocation, 10)
		return allocI.Cmp(allocJ) > 0
	})
	
	// Get total supply
	totalSupply := big.NewInt(0)
	for _, balance := range balances {
		if balance.Cmp(big.NewInt(0)) > 0 {
			totalSupply.Add(totalSupply, balance)
		}
	}
	
	circulatingSupply := new(big.Int).Sub(totalSupply, balances[deadAddr])
	
	result := &ZooMigrationResult{
		TokenAddress:      ZooTokenAddress,
		TotalSupply:       totalSupply.String(),
		CirculatingSupply: circulatingSupply.String(),
		BurnedSupply:      totalBurned.String(),
		UniqueHolders:     len(holders),
		HoldersWithBurns:  holdersWithBurns,
		EggNFTHolders:     eggNFTHolderCount,
		Holders:           holders,
		ScanBlock:         s.config.ToBlock,
		Timestamp:         time.Now().Unix(),
	}
	
	log.Printf("Zoo migration scan complete:")
	log.Printf("- Total holders: %d", len(holders))
	log.Printf("- Holders who burned: %d", holdersWithBurns)
	log.Printf("- Total burned: %s", totalBurned.String())
	log.Printf("- EGG NFT holders: %d", eggNFTHolderCount)
	
	return result, nil
}

// scanEggNFTHolders scans for EGG NFT holders
func (s *ZooMigrationScanner) scanEggNFTHolders() (map[common.Address]int, error) {
	holders := make(map[common.Address]int)
	
	// Create NFT scanner config
	nftConfig := NFTScannerConfig{
		Chain:           "bsc",
		RPC:             s.config.RPC,
		ContractAddress: EggNFTAddress,
		ProjectName:     "egg",
		FromBlock:       s.config.FromBlock,
		ToBlock:         s.config.ToBlock,
	}
	
	nftScanner, err := NewNFTScanner(nftConfig)
	if err != nil {
		return holders, err
	}
	defer nftScanner.Close()
	
	// Use the NFT scanner to get holders
	result, err := nftScanner.Scan()
	if err != nil {
		return holders, err
	}
	
	// Convert to address map
	for _, holder := range result.TopHolders {
		addr := common.HexToAddress(holder.Address)
		holders[addr] = holder.Count
	}
	
	// Need to get all holders, not just top holders
	// For now, this gives us a subset
	log.Printf("Found %d EGG NFT holders (top holders only)", len(holders))
	
	return holders, nil
}

// Export exports the migration results
func (s *ZooMigrationScanner) Export(result *ZooMigrationResult) error {
	if s.config.OutputPath == "" {
		s.config.OutputPath = "zoo-migration-data.json"
	}
	
	file, err := os.Create(s.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to write JSON: %w", err)
	}
	
	log.Printf("Exported Zoo migration data to %s", s.config.OutputPath)
	return nil
}

// GenerateGenesisAllocations creates genesis allocations from migration data
func GenerateZooGenesisAllocations(result *ZooMigrationResult) map[string]string {
	allocations := make(map[string]string)
	
	for _, holder := range result.Holders {
		// Use total allocation (including burns) for genesis
		allocations[holder.Address] = holder.TotalAllocation
	}
	
	return allocations
}

// Close closes the scanner connection
func (s *ZooMigrationScanner) Close() {
	if s.client != nil {
		s.client.Close()
	}
}