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

// TokenTransfer represents a token transfer
type TokenTransfer struct {
	TxHash      string    `json:"txHash"`
	BlockNumber uint64    `json:"blockNumber"`
	Timestamp   time.Time `json:"timestamp"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      string    `json:"amount"`
	TokenAddr   string    `json:"tokenAddress"`
	LogIndex    uint      `json:"logIndex"`
}

// TokenTransferScanner scans for token transfers to/from specific addresses
type TokenTransferScanner struct {
	client       *ethclient.Client
	tokenAddress common.Address
	config       *TokenTransferScanConfig
}

// TokenTransferScanConfig configures the transfer scanner
type TokenTransferScanConfig struct {
	RPC             string   `json:"rpc"`
	TokenAddress    string   `json:"tokenAddress"`
	TargetAddresses []string `json:"targetAddresses,omitempty"` // Filter by to/from addresses
	FromBlock       uint64   `json:"fromBlock"`
	ToBlock         uint64   `json:"toBlock"`
	ChunkSize       uint64   `json:"chunkSize"`
	Direction       string   `json:"direction"` // "to", "from", or "both"
}

// NewTokenTransferScanner creates a new transfer scanner
func NewTokenTransferScanner(config *TokenTransferScanConfig) (*TokenTransferScanner, error) {
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	if config.ChunkSize == 0 {
		config.ChunkSize = 5000
	}

	if config.Direction == "" {
		config.Direction = "both"
	}

	scanner := &TokenTransferScanner{
		client:       client,
		tokenAddress: common.HexToAddress(config.TokenAddress),
		config:       config,
	}

	return scanner, nil
}

// ScanTransfers scans for transfers based on configuration
func (s *TokenTransferScanner) ScanTransfers() ([]TokenTransfer, error) {
	if len(s.config.TargetAddresses) == 0 {
		return s.scanAllTransfers()
	}
	return s.scanTargetedTransfers()
}

// scanAllTransfers scans all transfers of the token
func (s *TokenTransferScanner) scanAllTransfers() ([]TokenTransfer, error) {
	ctx := context.Background()

	contractABI, err := abi.JSON(strings.NewReader(ERC20TransferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	transfers := []TokenTransfer{}
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

		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{s.tokenAddress},
			Topics: [][]common.Hash{
				{transferEventSig},
			},
		}

		logs, err := s.client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}

		for _, vLog := range logs {
			transfer, err := s.parseTransferLog(vLog)
			if err != nil {
				log.Printf("Warning: failed to parse log: %v", err)
				continue
			}
			transfers = append(transfers, *transfer)
		}

		if len(logs) > 0 {
			log.Printf("Found %d transfers in blocks %d-%d", len(logs), startBlock, endBlock)
		}
	}

	return transfers, nil
}

// scanTargetedTransfers scans transfers to/from specific addresses
func (s *TokenTransferScanner) scanTargetedTransfers() ([]TokenTransfer, error) {
	ctx := context.Background()

	contractABI, err := abi.JSON(strings.NewReader(ERC20TransferABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	transfers := []TokenTransfer{}
	transferEventSig := contractABI.Events["Transfer"].ID

	// Convert target addresses
	targetAddrs := []common.Address{}
	targetHashes := []common.Hash{}
	for _, addr := range s.config.TargetAddresses {
		a := common.HexToAddress(addr)
		targetAddrs = append(targetAddrs, a)
		targetHashes = append(targetHashes, common.BytesToHash(a.Bytes()))
	}

	// Get latest block if not specified
	if s.config.ToBlock == 0 {
		header, err := s.client.HeaderByNumber(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get latest block: %w", err)
		}
		s.config.ToBlock = header.Number.Uint64()
	}

	// Scan based on direction
	scanTypes := []string{}
	switch s.config.Direction {
	case "to":
		scanTypes = []string{"to"}
	case "from":
		scanTypes = []string{"from"}
	case "both":
		scanTypes = []string{"to", "from"}
	}

	for _, scanType := range scanTypes {
		for startBlock := s.config.FromBlock; startBlock <= s.config.ToBlock; startBlock += s.config.ChunkSize {
			endBlock := startBlock + s.config.ChunkSize - 1
			if endBlock > s.config.ToBlock {
				endBlock = s.config.ToBlock
			}

			var query ethereum.FilterQuery
			if scanType == "to" {
				query = ethereum.FilterQuery{
					FromBlock: big.NewInt(int64(startBlock)),
					ToBlock:   big.NewInt(int64(endBlock)),
					Addresses: []common.Address{s.tokenAddress},
					Topics: [][]common.Hash{
						{transferEventSig},
						nil, // from (any)
						targetHashes, // to (target addresses)
					},
				}
			} else { // from
				query = ethereum.FilterQuery{
					FromBlock: big.NewInt(int64(startBlock)),
					ToBlock:   big.NewInt(int64(endBlock)),
					Addresses: []common.Address{s.tokenAddress},
					Topics: [][]common.Hash{
						{transferEventSig},
						targetHashes, // from (target addresses)
						nil, // to (any)
					},
				}
			}

			logs, err := s.client.FilterLogs(ctx, query)
			if err != nil {
				log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
				continue
			}

			for _, vLog := range logs {
				transfer, err := s.parseTransferLog(vLog)
				if err != nil {
					log.Printf("Warning: failed to parse log: %v", err)
					continue
				}
				transfers = append(transfers, *transfer)
			}

			if len(logs) > 0 {
				log.Printf("Found %d %s transfers in blocks %d-%d", len(logs), scanType, startBlock, endBlock)
			}
		}
	}

	// Remove duplicates if scanning both directions
	if s.config.Direction == "both" {
		transfers = deduplicateTransfers(transfers)
	}

	return transfers, nil
}

// GetTransfersByAddress groups transfers by address (as sender or receiver)
func (s *TokenTransferScanner) GetTransfersByAddress() (map[string][]TokenTransfer, error) {
	transfers, err := s.ScanTransfers()
	if err != nil {
		return nil, err
	}

	byAddress := make(map[string][]TokenTransfer)
	for _, transfer := range transfers {
		fromAddr := strings.ToLower(transfer.From)
		toAddr := strings.ToLower(transfer.To)

		byAddress[fromAddr] = append(byAddress[fromAddr], transfer)
		if fromAddr != toAddr {
			byAddress[toAddr] = append(byAddress[toAddr], transfer)
		}
	}

	return byAddress, nil
}

// GetBalanceChanges calculates net balance changes from transfers
func GetBalanceChanges(transfers []TokenTransfer) map[string]*big.Int {
	balances := make(map[string]*big.Int)

	for _, transfer := range transfers {
		from := strings.ToLower(transfer.From)
		to := strings.ToLower(transfer.To)
		amount := new(big.Int)
		amount.SetString(transfer.Amount, 10)

		// Subtract from sender
		if balance, ok := balances[from]; ok {
			balance.Sub(balance, amount)
		} else {
			balances[from] = new(big.Int).Neg(amount)
		}

		// Add to receiver
		if balance, ok := balances[to]; ok {
			balance.Add(balance, amount)
		} else {
			balances[to] = new(big.Int).Set(amount)
		}
	}

	return balances
}

// parseTransferLog parses a Transfer event log
func (s *TokenTransferScanner) parseTransferLog(vLog types.Log) (*TokenTransfer, error) {
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

	transfer := &TokenTransfer{
		TxHash:      vLog.TxHash.Hex(),
		BlockNumber: vLog.BlockNumber,
		Timestamp:   time.Unix(int64(block.Time()), 0),
		From:        from.Hex(),
		To:          to.Hex(),
		Amount:      value.String(),
		TokenAddr:   s.tokenAddress.Hex(),
		LogIndex:    vLog.Index,
	}

	return transfer, nil
}

// deduplicateTransfers removes duplicate transfers (same tx hash and log index)
func deduplicateTransfers(transfers []TokenTransfer) []TokenTransfer {
	seen := make(map[string]bool)
	unique := []TokenTransfer{}

	for _, transfer := range transfers {
		key := fmt.Sprintf("%s-%d", transfer.TxHash, transfer.LogIndex)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, transfer)
		}
	}

	return unique
}

// Close closes the scanner
func (s *TokenTransferScanner) Close() error {
	s.client.Close()
	return nil
}