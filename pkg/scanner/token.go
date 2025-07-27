package scanner

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	// TODO: Replace with github.com/luxfi/geth when available
	ethereum "github.com/luxfi/geth"
	"github.com/luxfi/geth/accounts/abi"
	"github.com/luxfi/geth/common"
)

func (s *Scanner) scanTokenHolders(contractAddr common.Address, currentBlock uint64) (map[string]*AssetHolder, error) {
	holders := make(map[string]*AssetHolder)

	// Load ABI
	tokenABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse token ABI: %w", err)
	}

	// Calculate block range
	fromBlock := currentBlock - uint64(s.config.BlockRange)
	if fromBlock < 0 {
		fromBlock = 0
	}

	// Scan in chunks to avoid timeout
	chunkSize := uint64(10000)

	for start := fromBlock; start < currentBlock; start += chunkSize {
		end := start + chunkSize - 1
		if end > currentBlock {
			end = currentBlock
		}

		log.Printf("Scanning blocks %d to %d...", start, end)

		// Create filter query for Transfer events
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(start)),
			ToBlock:   big.NewInt(int64(end)),
			Addresses: []common.Address{contractAddr},
			Topics:    [][]common.Hash{{tokenABI.Events["Transfer"].ID}},
		}

		// Get logs
		logs, err := s.client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Printf("Warning: Failed to get logs for blocks %d-%d: %v", start, end, err)
			continue
		}

		// Process each transfer
		for _, vLog := range logs {
			// Extract from and to addresses from topics
			if len(vLog.Topics) >= 3 {
				// from := common.HexToAddress(vLog.Topics[1].Hex()) // Not used yet
				to := common.HexToAddress(vLog.Topics[2].Hex())

				// Skip zero addresses
				if to != (common.Address{}) {
					if _, exists := holders[to.Hex()]; !exists {
						holders[to.Hex()] = &AssetHolder{
							Address:         to,
							Balance:         big.NewInt(0),
							AssetType:       "Token",
							CollectionType:  "Token",
							StakingPower:    s.project.StakingPowers["Token"],
							ChainName:       s.config.Chain,
							ContractAddress: contractAddr.Hex(),
							ProjectName:     s.config.ProjectName,
							LastActivity:    vLog.BlockNumber,
						}
					}
					// Update last activity
					if vLog.BlockNumber > holders[to.Hex()].LastActivity {
						holders[to.Hex()].LastActivity = vLog.BlockNumber
					}
				}
			}
		}

		time.Sleep(100 * time.Millisecond) // Rate limiting
	}

	// Now get current balances for all holders
	log.Printf("\nFetching current balances for %d holders...", len(holders))
	count := 0

	for addr, holder := range holders {
		balance, err := s.getTokenBalance(contractAddr, holder.Address, tokenABI)
		if err != nil {
			log.Printf("Warning: Could not get balance for %s: %v", addr, err)
			continue
		}

		holder.Balance = balance

		count++
		if count%100 == 0 {
			log.Printf("Fetched %d balances...", count)
		}
	}

	// Remove holders with zero balance
	for addr, holder := range holders {
		if holder.Balance.Cmp(big.NewInt(0)) == 0 {
			delete(holders, addr)
		}
	}

	return holders, nil
}

func (s *Scanner) getTokenBalance(contractAddr common.Address, holder common.Address, abi abi.ABI) (*big.Int, error) {
	data, err := abi.Pack("balanceOf", holder)
	if err != nil {
		return nil, err
	}

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	result, err := s.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, err
	}

	var balance *big.Int
	err = abi.UnpackIntoInterface(&balance, "balanceOf", result)
	if err != nil {
		return nil, err
	}

	return balance, nil
}