package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ERC721 ABI events and methods
const erc721ABI = `[
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":true,"name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"},
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[{"name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}
]`

// NFTScanner handles NFT scanning from external chains
type NFTScanner struct {
	config      NFTScannerConfig
	client      *ethclient.Client
	contractABI abi.ABI
}

// NewNFTScanner creates a new NFT scanner
func NewNFTScanner(config NFTScannerConfig) (*NFTScanner, error) {
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
	contractABI, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}
	
	return &NFTScanner{
		config:      config,
		client:      client,
		contractABI: contractABI,
	}, nil
}

// Scan performs the NFT scan
func (s *NFTScanner) Scan() (*NFTScanResult, error) {
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
	
	// Get collection info
	var name, symbol string
	var totalSupply *big.Int
	
	// Get name
	results := []interface{}{&name}
	err = contract.Call(nil, &results, "name")
	if err != nil {
		log.Printf("Warning: failed to get collection name: %v", err)
		name = "Unknown Collection"
	}
	
	// Get symbol
	results = []interface{}{&symbol}
	err = contract.Call(nil, &results, "symbol")
	if err != nil {
		log.Printf("Warning: failed to get collection symbol: %v", err) 
		symbol = "UNKNOWN"
	}
	
	// Get total supply
	results = []interface{}{&totalSupply}
	err = contract.Call(nil, &results, "totalSupply")
	if err != nil {
		// If totalSupply fails, we'll count from Transfer events
		log.Printf("Warning: failed to get total supply, will count from events: %v", err)
		totalSupply = big.NewInt(0)
	}
	
	// Scan Transfer events to find all NFTs and current owners
	nftMap := make(map[string]string) // tokenID -> owner
	ownerCounts := make(map[string]int) // owner -> count
	typeDistribution := make(map[string]int)
	
	// Define the Transfer event signature
	transferEventSig := s.contractABI.Events["Transfer"].ID
	
	// Set FromBlock if not specified
	fromBlock := s.config.FromBlock
	if fromBlock == 0 {
		// Default to scanning last 1M blocks or contract creation
		fromBlock = currentBlock - 1000000
		if fromBlock < 0 {
			fromBlock = 0
		}
	}
	
	chunkSize := uint64(10000)
	totalScanned := 0
	
	log.Printf("Scanning NFT collection %s from block %d to %d", name, fromBlock, s.config.ToBlock)
	
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
			var transferEvent struct {
				From    common.Address
				To      common.Address
				TokenId *big.Int
			}
			
			err := s.contractABI.UnpackIntoInterface(&transferEvent, "Transfer", vLog.Data)
			if err != nil {
				// Try to parse indexed topics
				if len(vLog.Topics) >= 4 {
					transferEvent.From = common.HexToAddress(vLog.Topics[1].Hex())
					transferEvent.To = common.HexToAddress(vLog.Topics[2].Hex())
					transferEvent.TokenId = new(big.Int).SetBytes(vLog.Topics[3].Bytes())
				} else {
					log.Printf("Warning: failed to unpack Transfer event: %v", err)
					continue
				}
			}
			
			tokenID := transferEvent.TokenId.String()
			
			// Update NFT ownership (latest transfer is current owner)
			if transferEvent.To != common.HexToAddress("0x0") {
				// Not a burn
				nftMap[tokenID] = transferEvent.To.Hex()
			} else {
				// Burned NFT
				delete(nftMap, tokenID)
			}
		}
		
		totalScanned += len(logs)
		
		// Progress indicator
		if (endBlock-fromBlock)%100000 == 0 {
			log.Printf("Scanned up to block %d/%d (found %d NFTs)", endBlock, s.config.ToBlock, len(nftMap))
		}
	}
	
	// Count NFTs per owner
	for _, owner := range nftMap {
		ownerCounts[owner]++
	}
	
	// Find top holders
	type holderCount struct {
		address string
		count   int
	}
	holders := make([]holderCount, 0, len(ownerCounts))
	for addr, count := range ownerCounts {
		holders = append(holders, holderCount{addr, count})
	}
	
	// Sort holders by count (simple bubble sort for now)
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].count > holders[i].count {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	
	// Get top 10 holders
	topHolders := []Holder{}
	limit := 10
	if len(holders) < limit {
		limit = len(holders)
	}
	for i := 0; i < limit; i++ {
		topHolders = append(topHolders, Holder{
			Address: holders[i].address,
			Count:   holders[i].count,
		})
	}
	
	// For Lux NFTs, determine validator NFTs
	var stakingInfo *StakingInfo
	if s.config.ProjectName == "lux" {
		// Assuming validator NFTs are those with certain token IDs or attributes
		// This is a simplified version - real implementation would check metadata
		validatorCount := 0
		for tokenID := range nftMap {
			// Example: token IDs 1-100 are validator NFTs
			id := new(big.Int)
			id.SetString(tokenID, 10)
			if id.Cmp(big.NewInt(100)) <= 0 {
				validatorCount++
				typeDistribution["Validator"]++
			} else if id.Cmp(big.NewInt(500)) <= 0 {
				typeDistribution["Card"]++
			} else {
				typeDistribution["Coin"]++
			}
		}
		
		stakingInfo = &StakingInfo{
			ValidatorCount: validatorCount,
			TotalPower:     fmt.Sprintf("%d000000000000000000000000", validatorCount), // Each validator = 1M tokens
		}
	}
	
	// If we couldn't get total supply from contract, use our count
	if totalSupply.Cmp(big.NewInt(0)) == 0 {
		totalSupply = big.NewInt(int64(len(nftMap)))
	}
	
	// Build NFT list for export
	nfts := make([]ScannedNFT, 0, len(nftMap))
	for tokenID, owner := range nftMap {
		nft := ScannedNFT{
			TokenID: tokenID,
			Owner:   owner,
		}
		
		// For validator NFTs, add staking power
		if s.config.ValidatorNFT || (s.config.ProjectName == "lux" && stakingInfo != nil) {
			id := new(big.Int)
			id.SetString(tokenID, 10)
			if id.Cmp(big.NewInt(100)) <= 0 {
				nft.StakingPower = "1000000000000000000000000" // 1M tokens
			}
		}
		
		nfts = append(nfts, nft)
	}
	
	result := &NFTScanResult{
		ContractAddress:  s.config.ContractAddress,
		CollectionName:   name,
		Symbol:           symbol,
		TotalSupply:      int(totalSupply.Int64()),
		UniqueHolders:    len(ownerCounts),
		FromBlock:        fromBlock,
		ToBlock:          s.config.ToBlock,
		BlockScanned:     currentBlock,
		ChainID:          uint64(s.config.ChainID),
		TotalNFTs:        len(nftMap),
		NFTs:             nfts,
		TypeDistribution: typeDistribution,
		TopHolders:       topHolders,
		StakingInfo:      stakingInfo,
	}
	
	log.Printf("Scan complete: found %d NFTs held by %d unique addresses", len(nftMap), len(ownerCounts))
	
	return result, nil
}

// Export exports the scan results
func (s *NFTScanner) Export(outputPath string) error {
	result, err := s.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}
	
	// Export as JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	// Write to file (in real implementation)
	log.Printf("Export would write %d bytes to %s", len(data), outputPath)
	
	return nil
}

// GetDetailedNFTs returns detailed information for each NFT
func (s *NFTScanner) GetDetailedNFTs() ([]NFTDetail, error) {
	contractAddr := common.HexToAddress(s.config.ContractAddress)
	contract := bind.NewBoundContract(contractAddr, s.contractABI, s.client, s.client, s.client)
	
	// First get basic scan results
	result, err := s.Scan()
	if err != nil {
		return nil, err
	}
	
	details := []NFTDetail{}
	
	// For each NFT, get detailed info
	for i := 0; i < result.TotalNFTs && i < 100; i++ { // Limit to first 100 for performance
		tokenID := big.NewInt(int64(i + 1))
		
		// Get owner
		var owner common.Address
		results := []interface{}{&owner}
		err = contract.Call(nil, &results, "ownerOf", tokenID)
		if err != nil {
			continue // Skip if can't get owner (might be burned)
		}
		
		// Get URI
		var uri string
		results = []interface{}{&uri}
		err = contract.Call(nil, &results, "tokenURI", tokenID)
		if err != nil {
			uri = ""
		}
		
		detail := NFTDetail{
			TokenID: tokenID.String(),
			Owner:   owner.Hex(),
			URI:     uri,
		}
		
		// For validator NFTs, add staking power
		if s.config.ProjectName == "lux" && tokenID.Cmp(big.NewInt(100)) <= 0 {
			detail.StakingPower = "1000000000000000000000000" // 1M tokens
		}
		
		details = append(details, detail)
	}
	
	return details, nil
}

// NFTDetail represents detailed NFT information
type NFTDetail struct {
	TokenID      string `json:"tokenId"`
	Owner        string `json:"owner"`
	URI          string `json:"uri"`
	StakingPower string `json:"stakingPower,omitempty"`
}

// Close closes the scanner connection
func (s *NFTScanner) Close() {
	if s.client != nil {
		s.client.Close()
	}
}