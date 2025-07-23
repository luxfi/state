package crosschain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// EventScanner scans blockchain events
type EventScanner struct {
	client    *Client
	batchSize int64
}

// NewEventScanner creates a new event scanner
func NewEventScanner(client *Client) *EventScanner {
	return &EventScanner{
		client:    client,
		batchSize: 5000, // Scan 5000 blocks at a time
	}
}

// ScanBurnEvents scans for burn events (transfers to 0x0)
func (s *EventScanner) ScanBurnEvents(ctx context.Context, tokenAddr common.Address, fromBlock, toBlock *big.Int) ([]BurnEvent, error) {
	// Transfer event signature
	transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	
	// Zero address
	zeroAddr := common.HexToAddress("0x0000000000000000000000000000000000000000")
	zeroAddrTopic := common.BytesToHash(common.LeftPadBytes(zeroAddr.Bytes(), 32))
	
	burns := []BurnEvent{}
	
	// Process in batches
	current := new(big.Int).Set(fromBlock)
	for current.Cmp(toBlock) <= 0 {
		end := new(big.Int).Add(current, big.NewInt(s.batchSize))
		if end.Cmp(toBlock) > 0 {
			end = toBlock
		}
		
		// Create filter query
		query := ethereum.FilterQuery{
			FromBlock: current,
			ToBlock:   end,
			Addresses: []common.Address{tokenAddr},
			Topics: [][]common.Hash{
				{transferSig},         // Transfer event
				nil,                   // from (any)
				{zeroAddrTopic},       // to (0x0)
			},
		}
		
		// Get logs
		logs, err := s.client.client.FilterLogs(ctx, query)
		if err != nil {
			// Try smaller batch on error
			if s.batchSize > 100 {
				s.batchSize = s.batchSize / 2
				continue
			}
			return nil, fmt.Errorf("failed to get logs: %w", err)
		}
		
		// Process logs
		for _, log := range logs {
			if len(log.Topics) < 3 {
				continue
			}
			
			// Extract from address
			from := common.BytesToAddress(log.Topics[1].Bytes()[12:])
			
			// Extract amount
			amount := new(big.Int).SetBytes(log.Data)
			
			// Get block timestamp
			block, err := s.client.client.BlockByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
			if err != nil {
				continue
			}
			
			burns = append(burns, BurnEvent{
				From:            from,
				Amount:          amount,
				BlockNumber:     log.BlockNumber,
				TransactionHash: log.TxHash,
				Timestamp:       block.Time(),
			})
		}
		
		// Progress
		fmt.Printf("Scanned blocks %s to %s, found %d burns\n", current, end, len(burns))
		
		// Next batch
		current = new(big.Int).Add(end, big.NewInt(1))
		
		// Rate limit
		time.Sleep(100 * time.Millisecond)
	}
	
	return burns, nil
}

// ScanTransferEvents scans for all transfer events
func (s *EventScanner) ScanTransferEvents(ctx context.Context, tokenAddr common.Address, fromBlock, toBlock *big.Int) ([]TransferEvent, error) {
	// Transfer event signature
	transferSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	
	transfers := []TransferEvent{}
	
	// Process in batches
	current := new(big.Int).Set(fromBlock)
	for current.Cmp(toBlock) <= 0 {
		end := new(big.Int).Add(current, big.NewInt(s.batchSize))
		if end.Cmp(toBlock) > 0 {
			end = toBlock
		}
		
		// Create filter query
		query := ethereum.FilterQuery{
			FromBlock: current,
			ToBlock:   end,
			Addresses: []common.Address{tokenAddr},
			Topics: [][]common.Hash{
				{transferSig}, // Transfer event
			},
		}
		
		// Get logs
		logs, err := s.client.client.FilterLogs(ctx, query)
		if err != nil {
			// Try smaller batch on error
			if s.batchSize > 100 {
				s.batchSize = s.batchSize / 2
				continue
			}
			return nil, fmt.Errorf("failed to get logs: %w", err)
		}
		
		// Process logs
		for _, log := range logs {
			transfer := s.parseTransferLog(log)
			if transfer != nil {
				transfers = append(transfers, *transfer)
			}
		}
		
		// Progress
		fmt.Printf("Scanned blocks %s to %s, found %d transfers\n", current, end, len(transfers))
		
		// Next batch
		current = new(big.Int).Add(end, big.NewInt(1))
		
		// Rate limit
		time.Sleep(100 * time.Millisecond)
	}
	
	return transfers, nil
}

// parseTransferLog parses a transfer event log
func (s *EventScanner) parseTransferLog(log types.Log) *TransferEvent {
	// Check topics length
	if len(log.Topics) < 3 {
		return nil
	}
	
	// Extract addresses
	from := common.BytesToAddress(log.Topics[1].Bytes()[12:])
	to := common.BytesToAddress(log.Topics[2].Bytes()[12:])
	
	// Extract value or token ID
	var value *big.Int
	var tokenID *big.Int
	
	if len(log.Data) > 0 {
		// ERC20 - value in data
		value = new(big.Int).SetBytes(log.Data)
	} else if len(log.Topics) > 3 {
		// ERC721 - token ID in topics[3]
		tokenID = new(big.Int).SetBytes(log.Topics[3].Bytes())
	}
	
	return &TransferEvent{
		From:            from,
		To:              to,
		Value:           value,
		TokenID:         tokenID,
		BlockNumber:     log.BlockNumber,
		TransactionHash: log.TxHash,
	}
}

// BuildTokenHolderSnapshot builds current holder snapshot from transfer events
func BuildTokenHolderSnapshot(transfers []TransferEvent) []TokenHolder {
	// Track balances
	balances := make(map[common.Address]*big.Int)
	
	// Process transfers chronologically
	for _, transfer := range transfers {
		// Subtract from sender (unless minting from 0x0)
		if transfer.From != (common.Address{}) {
			if balance, ok := balances[transfer.From]; ok {
				balances[transfer.From] = new(big.Int).Sub(balance, transfer.Value)
			} else {
				balances[transfer.From] = new(big.Int).Neg(transfer.Value)
			}
		}
		
		// Add to recipient (unless burning to 0x0)
		if transfer.To != (common.Address{}) {
			if balance, ok := balances[transfer.To]; ok {
				balances[transfer.To] = new(big.Int).Add(balance, transfer.Value)
			} else {
				balances[transfer.To] = new(big.Int).Set(transfer.Value)
			}
		}
	}
	
	// Build holder list
	holders := []TokenHolder{}
	for addr, balance := range balances {
		if balance.Sign() > 0 {
			holders = append(holders, TokenHolder{
				Address: addr,
				Balance: balance,
			})
		}
	}
	
	return holders
}