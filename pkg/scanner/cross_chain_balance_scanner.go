package scanner

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// CrossChainBalance represents a balance on a specific chain
type CrossChainBalance struct {
	Address      string `json:"address"`
	Balance      string `json:"balance"`
	ChainID      int64  `json:"chainId"`
	TokenAddress string `json:"tokenAddress"`
	BlockNumber  uint64 `json:"blockNumber"`
}

// CrossChainBalanceScanner scans balances across multiple chains
type CrossChainBalanceScanner struct {
	clients map[string]*ethclient.Client
	config  *CrossChainBalanceScanConfig
}

// CrossChainBalanceScanConfig configures the cross-chain scanner
type CrossChainBalanceScanConfig struct {
	Chains []ChainConfig `json:"chains"`
}

// ChainConfig represents a single chain configuration
type ChainConfig struct {
	Name         string `json:"name"`
	ChainID      int64  `json:"chainId"`
	RPC          string `json:"rpc"`
	TokenAddress string `json:"tokenAddress"`
}

// NewCrossChainBalanceScanner creates a new cross-chain balance scanner
func NewCrossChainBalanceScanner(config *CrossChainBalanceScanConfig) (*CrossChainBalanceScanner, error) {
	clients := make(map[string]*ethclient.Client)

	for _, chain := range config.Chains {
		client, err := ethclient.Dial(chain.RPC)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s: %w", chain.Name, err)
		}
		clients[chain.Name] = client
	}

	scanner := &CrossChainBalanceScanner{
		clients: clients,
		config:  config,
	}

	return scanner, nil
}

// ScanBalances scans balances for addresses across all configured chains
func (s *CrossChainBalanceScanner) ScanBalances(addresses []string) (map[string][]CrossChainBalance, error) {
	balanceMap := make(map[string][]CrossChainBalance)

	for _, chain := range s.config.Chains {
		client, ok := s.clients[chain.Name]
		if !ok {
			continue
		}

		log.Printf("Scanning balances on %s (chain %d)", chain.Name, chain.ChainID)

		for _, addr := range addresses {
			balance, blockNum, err := s.getTokenBalance(client, addr, chain.TokenAddress)
			if err != nil {
				log.Printf("Warning: failed to get balance for %s on %s: %v", addr, chain.Name, err)
				continue
			}

			if balance.Cmp(big.NewInt(0)) > 0 {
				balanceMap[strings.ToLower(addr)] = append(balanceMap[strings.ToLower(addr)], CrossChainBalance{
					Address:      addr,
					Balance:      balance.String(),
					ChainID:      chain.ChainID,
					TokenAddress: chain.TokenAddress,
					BlockNumber:  blockNum,
				})
			}
		}
	}

	return balanceMap, nil
}

// ScanBalancesForBurners scans mainnet balances specifically for burn addresses
func (s *CrossChainBalanceScanner) ScanBalancesForBurners(burns []TokenBurn) (map[string]CrossChainBalance, error) {
	// Extract unique burner addresses
	burners := make(map[string]bool)
	for _, burn := range burns {
		burners[strings.ToLower(burn.From)] = true
	}

	addresses := []string{}
	for addr := range burners {
		addresses = append(addresses, addr)
	}

	// Scan balances
	allBalances, err := s.ScanBalances(addresses)
	if err != nil {
		return nil, err
	}

	// For burners, we typically want just the mainnet balance
	mainnetBalances := make(map[string]CrossChainBalance)
	for addr, balances := range allBalances {
		// Find mainnet balance (you can configure which chain is "mainnet")
		for _, balance := range balances {
			// Assuming the first configured chain is mainnet
			if len(s.config.Chains) > 0 && balance.ChainID == s.config.Chains[0].ChainID {
				mainnetBalances[addr] = balance
				break
			}
		}
	}

	return mainnetBalances, nil
}

// CompareBalances compares balances between source and target chains
func (s *CrossChainBalanceScanner) CompareBalances(addresses []string) ([]BalanceComparison, error) {
	if len(s.config.Chains) < 2 {
		return nil, fmt.Errorf("need at least 2 chains for comparison")
	}

	sourceChain := s.config.Chains[0]
	targetChain := s.config.Chains[1]

	comparisons := []BalanceComparison{}

	for _, addr := range addresses {
		sourceBalance, _, err := s.getTokenBalance(s.clients[sourceChain.Name], addr, sourceChain.TokenAddress)
		if err != nil {
			log.Printf("Warning: failed to get source balance for %s: %v", addr, err)
			continue
		}

		targetBalance, _, err := s.getTokenBalance(s.clients[targetChain.Name], addr, targetChain.TokenAddress)
		if err != nil {
			log.Printf("Warning: failed to get target balance for %s: %v", addr, err)
			continue
		}

		comparison := BalanceComparison{
			Address:       addr,
			SourceBalance: sourceBalance.String(),
			TargetBalance: targetBalance.String(),
			SourceChainID: sourceChain.ChainID,
			TargetChainID: targetChain.ChainID,
		}

		// Calculate difference
		diff := new(big.Int).Sub(sourceBalance, targetBalance)
		comparison.Difference = diff.String()

		comparisons = append(comparisons, comparison)
	}

	return comparisons, nil
}

// getTokenBalance gets the ERC20 token balance for an address
func (s *CrossChainBalanceScanner) getTokenBalance(client *ethclient.Client, address, tokenAddress string) (*big.Int, uint64, error) {
	ctx := context.Background()

	// Get current block number
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, 0, err
	}
	blockNum := header.Number.Uint64()

	// Create token instance
	token, err := NewERC20(common.HexToAddress(tokenAddress), client)
	if err != nil {
		return nil, 0, err
	}

	// Get balance
	opts := &bind.CallOpts{Context: ctx}
	balance, err := token.BalanceOf(opts, common.HexToAddress(address))
	if err != nil {
		return nil, 0, err
	}

	return balance, blockNum, nil
}

// Close closes all client connections
func (s *CrossChainBalanceScanner) Close() error {
	for name, client := range s.clients {
		client.Close()
		delete(s.clients, name)
	}
	return nil
}

// BalanceComparison represents a balance comparison between chains
type BalanceComparison struct {
	Address       string `json:"address"`
	SourceBalance string `json:"sourceBalance"`
	TargetBalance string `json:"targetBalance"`
	Difference    string `json:"difference"`
	SourceChainID int64  `json:"sourceChainId"`
	TargetChainID int64  `json:"targetChainId"`
}

// Simple ERC20 interface for balance queries
type ERC20 struct {
	contract *bind.BoundContract
}

func NewERC20(address common.Address, client bind.ContractBackend) (*ERC20, error) {
	parsed, err := abi.JSON(strings.NewReader(ERC20BalanceABI))
	if err != nil {
		return nil, err
	}
	
	contract := bind.NewBoundContract(address, parsed, client, client, client)
	return &ERC20{contract: contract}, nil
}

func (e *ERC20) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var result []interface{}
	err := e.contract.Call(opts, &result, "balanceOf", account)
	if err != nil {
		return nil, err
	}
	return result[0].(*big.Int), nil
}

// Minimal ERC20 ABI for balance queries
const ERC20BalanceABI = `[{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`