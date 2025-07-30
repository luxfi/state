package scanner

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// NFTHolder represents an NFT holder with their token count
type NFTHolder struct {
	Address    string   `json:"address"`
	TokenCount int      `json:"tokenCount"`
	TokenIDs   []string `json:"tokenIds,omitempty"`
}

// NFTHolderScanner scans for NFT holders
type NFTHolderScanner struct {
	client          *ethclient.Client
	contractAddress common.Address
	config          *NFTHolderScanConfig
}

// NFTHolderScanConfig configures the NFT holder scanner
type NFTHolderScanConfig struct {
	RPC             string `json:"rpc"`
	ContractAddress string `json:"contractAddress"`
	FromBlock       uint64 `json:"fromBlock"`
	ToBlock         uint64 `json:"toBlock"`
	ChunkSize       uint64 `json:"chunkSize"`
	IncludeTokenIDs bool   `json:"includeTokenIds"`
}

// NewNFTHolderScanner creates a new NFT holder scanner
func NewNFTHolderScanner(config *NFTHolderScanConfig) (*NFTHolderScanner, error) {
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	if config.ChunkSize == 0 {
		config.ChunkSize = 5000
	}

	scanner := &NFTHolderScanner{
		client:          client,
		contractAddress: common.HexToAddress(config.ContractAddress),
		config:          config,
	}

	return scanner, nil
}

// ScanHolders scans for all current NFT holders
func (s *NFTHolderScanner) ScanHolders() ([]NFTHolder, error) {
	// First, get all Transfer events to build current ownership
	ownership, err := s.buildOwnershipMap()
	if err != nil {
		return nil, err
	}

	// Convert to holder list
	holders := []NFTHolder{}
	for addr, tokenIDs := range ownership {
		if len(tokenIDs) > 0 { // Only include addresses that currently hold tokens
			holder := NFTHolder{
				Address:    addr,
				TokenCount: len(tokenIDs),
			}
			if s.config.IncludeTokenIDs {
				holder.TokenIDs = tokenIDs
			}
			holders = append(holders, holder)
		}
	}

	return holders, nil
}

// GetHoldersByCount returns holders grouped by token count
func (s *NFTHolderScanner) GetHoldersByCount() (map[int][]string, error) {
	holders, err := s.ScanHolders()
	if err != nil {
		return nil, err
	}

	byCount := make(map[int][]string)
	for _, holder := range holders {
		byCount[holder.TokenCount] = append(byCount[holder.TokenCount], holder.Address)
	}

	return byCount, nil
}

// GetTopHolders returns the top N holders by token count
func (s *NFTHolderScanner) GetTopHolders(limit int) ([]NFTHolder, error) {
	holders, err := s.ScanHolders()
	if err != nil {
		return nil, err
	}

	// Sort by token count (descending)
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].TokenCount > holders[i].TokenCount {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}

	if limit > len(holders) {
		limit = len(holders)
	}

	return holders[:limit], nil
}

// buildOwnershipMap builds current ownership by processing all Transfer events
func (s *NFTHolderScanner) buildOwnershipMap() (map[string][]string, error) {
	ctx := context.Background()

	// Parse ERC721 ABI for Transfer events
	contractABI, err := abi.JSON(strings.NewReader(ERC721TransferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	ownership := make(map[string][]string)
	transferEventSig := contractABI.Events["Transfer"].ID

	// Get latest block if not specified
	if s.config.ToBlock == 0 {
		header, err := s.client.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block: %w", err)
		}
		s.config.ToBlock = header.Number.Uint64()
	}

	log.Printf("Scanning NFT transfers from block %d to %d", s.config.FromBlock, s.config.ToBlock)

	// Scan in chunks
	totalTransfers := 0
	for startBlock := s.config.FromBlock; startBlock <= s.config.ToBlock; startBlock += s.config.ChunkSize {
		endBlock := startBlock + s.config.ChunkSize - 1
		if endBlock > s.config.ToBlock {
			endBlock = s.config.ToBlock
		}

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{s.contractAddress},
			Topics: [][]common.Hash{
				{transferEventSig},
			},
		}

		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}

		// Process transfers
		for _, vLog := range logs {
			from, to, tokenID, err := s.parseTransferLog(vLog)
			if err != nil {
				log.Printf("Warning: failed to parse log: %v", err)
				continue
			}

			// Remove from previous owner
			if from != ZeroAddress {
				s.removeTokenFromOwner(ownership, from, tokenID)
			}

			// Add to new owner (unless it's a burn)
			if to != ZeroAddress {
				s.addTokenToOwner(ownership, to, tokenID)
			}

			totalTransfers++
		}

		if len(logs) > 0 {
			log.Printf("Processed %d transfers in blocks %d-%d", len(logs), startBlock, endBlock)
		}

		// Progress update
		if (endBlock-s.config.FromBlock) > 0 && (endBlock-s.config.FromBlock)%50000 == 0 {
			progress := float64(endBlock-s.config.FromBlock) / float64(s.config.ToBlock-s.config.FromBlock) * 100
			log.Printf("Scan progress: %.1f%% (block %d/%d)", progress, endBlock, s.config.ToBlock)
		}
	}

	log.Printf("Total transfers processed: %d", totalTransfers)

	// Count total NFTs currently held
	totalNFTs := 0
	for _, tokenIDs := range ownership {
		totalNFTs += len(tokenIDs)
	}
	log.Printf("Total NFTs currently held: %d by %d unique holders", totalNFTs, len(ownership))

	return ownership, nil
}

// parseTransferLog parses an ERC721 Transfer event log
func (s *NFTHolderScanner) parseTransferLog(vLog types.Log) (from, to, tokenID string, err error) {
	// ERC721 Transfer event has 3 indexed topics: event signature, from, to
	// And tokenId in the data field
	if len(vLog.Topics) < 3 {
		return "", "", "", fmt.Errorf("invalid number of topics")
	}

	from = common.HexToAddress(vLog.Topics[1].Hex()).Hex()
	to = common.HexToAddress(vLog.Topics[2].Hex()).Hex()

	// Parse tokenId from data (if present) or from 4th topic (if indexed)
	if len(vLog.Topics) >= 4 {
		// TokenId is indexed (in topics)
		tokenID = new(big.Int).SetBytes(vLog.Topics[3].Bytes()).String()
	} else if len(vLog.Data) >= 32 {
		// TokenId is in data
		tokenID = new(big.Int).SetBytes(vLog.Data).String()
	} else {
		return "", "", "", fmt.Errorf("could not parse tokenId")
	}

	return strings.ToLower(from), strings.ToLower(to), tokenID, nil
}

// addTokenToOwner adds a token to an owner's list
func (s *NFTHolderScanner) addTokenToOwner(ownership map[string][]string, owner, tokenID string) {
	owner = strings.ToLower(owner)
	tokens := ownership[owner]

	// Check if already owned (shouldn't happen but be safe)
	for _, id := range tokens {
		if id == tokenID {
			return
		}
	}

	ownership[owner] = append(tokens, tokenID)
}

// removeTokenFromOwner removes a token from an owner's list
func (s *NFTHolderScanner) removeTokenFromOwner(ownership map[string][]string, owner, tokenID string) {
	owner = strings.ToLower(owner)
	tokens := ownership[owner]

	newTokens := []string{}
	for _, id := range tokens {
		if id != tokenID {
			newTokens = append(newTokens, id)
		}
	}

	if len(newTokens) > 0 {
		ownership[owner] = newTokens
	} else {
		delete(ownership, owner)
	}
}

// FilterHoldersByMinTokens filters holders by minimum token count
func FilterHoldersByMinTokens(holders []NFTHolder, minTokens int) []NFTHolder {
	filtered := []NFTHolder{}
	for _, holder := range holders {
		if holder.TokenCount >= minTokens {
			filtered = append(filtered, holder)
		}
	}
	return filtered
}

// GetHolderDistribution returns distribution of token holdings
func GetHolderDistribution(holders []NFTHolder) map[string]int {
	distribution := map[string]int{
		"1 token":       0,
		"2-5 tokens":    0,
		"6-10 tokens":   0,
		"11-20 tokens":  0,
		"21-50 tokens":  0,
		"51-100 tokens": 0,
		"100+ tokens":   0,
	}

	for _, holder := range holders {
		switch {
		case holder.TokenCount == 1:
			distribution["1 token"]++
		case holder.TokenCount >= 2 && holder.TokenCount <= 5:
			distribution["2-5 tokens"]++
		case holder.TokenCount >= 6 && holder.TokenCount <= 10:
			distribution["6-10 tokens"]++
		case holder.TokenCount >= 11 && holder.TokenCount <= 20:
			distribution["11-20 tokens"]++
		case holder.TokenCount >= 21 && holder.TokenCount <= 50:
			distribution["21-50 tokens"]++
		case holder.TokenCount >= 51 && holder.TokenCount <= 100:
			distribution["51-100 tokens"]++
		case holder.TokenCount > 100:
			distribution["100+ tokens"]++
		}
	}

	return distribution
}

// Close closes the scanner
func (s *NFTHolderScanner) Close() error {
	s.client.Close()
	return nil
}

// ERC721TransferABI is the minimal ABI for Transfer events
const ERC721TransferABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":true,"name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"}]`
