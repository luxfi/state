package scanner

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// TokenBurn represents a token burn transaction
type TokenBurn struct {
	TxHash      string    `json:"txHash"`
	BlockNumber uint64    `json:"blockNumber"`
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      string    `json:"amount"`
	TokenAddr   string    `json:"tokenAddress"`
	LogIndex    uint      `json:"logIndex"`
}

// TokenBurnScanner scans for token burns to specific addresses
type TokenBurnScanner struct {
	client       *ethclient.Client
	tokenAddress common.Address
	burnAddress  common.Address
	config       *TokenBurnScanConfig
}

// TokenBurnScanConfig configures the burn scanner
type TokenBurnScanConfig struct {
	RPC           string   `json:"rpc"`
	TokenAddress  string   `json:"tokenAddress"`
	BurnAddress   string   `json:"burnAddress"`
	FromBlock     uint64   `json:"fromBlock"`
	ToBlock       uint64   `json:"toBlock"`
	ChunkSize     uint64   `json:"chunkSize"`
	BurnAddresses []string `json:"burnAddresses,omitempty"` // Optional: multiple burn addresses
}

// Common burn addresses
const (
	DeadAddress = "0x000000000000000000000000000000000000dEaD"
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

// NewTokenBurnScanner creates a new burn scanner
func NewTokenBurnScanner(config *TokenBurnScanConfig) (*TokenBurnScanner, error) {
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	if config.ChunkSize == 0 {
		config.ChunkSize = 5000
	}

	scanner := &TokenBurnScanner{
		client:       client,
		tokenAddress: common.HexToAddress(config.TokenAddress),
		burnAddress:  common.HexToAddress(config.BurnAddress),
		config:       config,
	}

	return scanner, nil
}

// ScanBurns scans for all burns to the configured burn address
func (s *TokenBurnScanner) ScanBurns() ([]TokenBurn, error) {
	ctx := context.Background()

	// Parse ERC20 ABI for Transfer events
	contractABI, err := abi.JSON(strings.NewReader(ERC20TransferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	burns := []TokenBurn{}
	transferEventSig := contractABI.Events["Transfer"].ID

	// Get latest block if not specified
	if s.config.ToBlock == 0 {
		header, err := s.client.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block: %w", err)
		}
		s.config.ToBlock = header.Number.Uint64()
	}

	// Scan in chunks
	for startBlock := s.config.FromBlock; startBlock <= s.config.ToBlock; startBlock += s.config.ChunkSize {
		endBlock := startBlock + s.config.ChunkSize - 1
		if endBlock > s.config.ToBlock {
			endBlock = s.config.ToBlock
		}

		// Build topics for burn addresses
		burnAddresses := []common.Address{s.burnAddress}
		if len(s.config.BurnAddresses) > 0 {
			for _, addr := range s.config.BurnAddresses {
				burnAddresses = append(burnAddresses, common.HexToAddress(addr))
			}
		}

		// Convert addresses to hashes for topic filtering
		burnTopics := []common.Hash{}
		for _, addr := range burnAddresses {
			burnTopics = append(burnTopics, common.BytesToHash(addr.Bytes()))
		}

		// Filter for transfers TO burn addresses
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{s.tokenAddress},
			Topics: [][]common.Hash{
				{transferEventSig},
				nil,        // from (any)
				burnTopics, // to (burn addresses)
			},
		}

		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}

		// Process burns
		for _, vLog := range logs {
			burn, err := s.parseTransferLog(vLog)
			if err != nil {
				log.Printf("Warning: failed to parse log: %v", err)
				continue
			}
			burns = append(burns, *burn)
		}

		if len(logs) > 0 {
			log.Printf("Found %d burns in blocks %d-%d", len(logs), startBlock, endBlock)
		}

		// Progress update
		if (endBlock-s.config.FromBlock) > 0 && (endBlock-s.config.FromBlock)%50000 == 0 {
			progress := float64(endBlock-s.config.FromBlock) / float64(s.config.ToBlock-s.config.FromBlock) * 100
			log.Printf("Scan progress: %.1f%% (block %d/%d)", progress, endBlock, s.config.ToBlock)
		}
	}

	return burns, nil
}

// ScanBurnsByAddress scans and groups burns by sender address
func (s *TokenBurnScanner) ScanBurnsByAddress() (map[string]*big.Int, error) {
	burns, err := s.ScanBurns()
	if err != nil {
		return nil, err
	}

	// Aggregate by address
	burnsByAddress := make(map[string]*big.Int)
	for _, burn := range burns {
		addr := strings.ToLower(burn.From)
		amount := new(big.Int)
		amount.SetString(burn.Amount, 10)

		if existing, ok := burnsByAddress[addr]; ok {
			existing.Add(existing, amount)
		} else {
			burnsByAddress[addr] = amount
		}
	}

	return burnsByAddress, nil
}

// parseTransferLog parses a Transfer event log
func (s *TokenBurnScanner) parseTransferLog(vLog types.Log) (*TokenBurn, error) {
	var from, to common.Address
	var value *big.Int

	// Parse indexed topics
	if len(vLog.Topics) >= 3 {
		from = common.HexToAddress(vLog.Topics[1].Hex())
		to = common.HexToAddress(vLog.Topics[2].Hex())
	} else {
		return nil, fmt.Errorf("invalid log topics")
	}

	// Parse value from data
	if len(vLog.Data) >= 32 {
		value = new(big.Int).SetBytes(vLog.Data)
	} else {
		return nil, fmt.Errorf("invalid log data")
	}

	// Get block details for timestamp
	ctx := context.Background()
	block, err := s.client.BlockByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
	if err != nil {
		log.Printf("Warning: failed to get block %d: %v", vLog.BlockNumber, err)
	}

	burn := &TokenBurn{
		TxHash:      vLog.TxHash.Hex(),
		BlockNumber: vLog.BlockNumber,
		Timestamp:   time.Unix(int64(block.Time()), 0),
		From:        from.Hex(),
		To:          to.Hex(),
		Amount:      value.String(),
		TokenAddr:   s.tokenAddress.Hex(),
		LogIndex:    vLog.Index,
	}

	return burn, nil
}

// FilterBurnsByAmount filters burns by minimum amount
func FilterBurnsByAmount(burns []TokenBurn, minAmount *big.Int) []TokenBurn {
	filtered := []TokenBurn{}
	for _, burn := range burns {
		amount := new(big.Int)
		amount.SetString(burn.Amount, 10)
		if amount.Cmp(minAmount) >= 0 {
			filtered = append(filtered, burn)
		}
	}
	return filtered
}

// GetUniqueBurners returns unique addresses that have burned tokens
func GetUniqueBurners(burns []TokenBurn) []string {
	seen := make(map[string]bool)
	unique := []string{}

	for _, burn := range burns {
		addr := strings.ToLower(burn.From)
		if !seen[addr] {
			seen[addr] = true
			unique = append(unique, addr)
		}
	}

	return unique
}

// Close closes the scanner
func (s *TokenBurnScanner) Close() error {
	s.client.Close()
	return nil
}

// ERC20TransferABI is the minimal ABI for Transfer events
const ERC20TransferABI = `[{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
