package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// StateAccount represents the Ethereum state account structure
type StateAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// Account represents an account with balance
type Account struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

// ChainData represents extracted chain data
type ChainData struct {
	ChainID        string                 `json:"chainId"`
	NetworkID      string                 `json:"networkId"`
	LatestBlock    uint64                 `json:"latestBlock"`
	StateRoot      string                 `json:"stateRoot"`
	TotalAccounts  int                    `json:"totalAccounts"`
	TotalBalance   string                 `json:"totalBalance"`
	Allocations    []Account              `json:"allocations"`
	GenesisConfig  map[string]interface{} `json:"genesisConfig,omitempty"`
	BlockHashes    []string               `json:"blockHashes,omitempty"`
}

// extractStateFromPebbleDB extracts all account states from a pebbledb
func extractStateFromPebbleDB(dbPath string, minBalance *big.Int) (*ChainData, error) {
	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	chainData := &ChainData{
		Allocations: make([]Account, 0),
	}

	// Try to read the latest block header
	// Head block hash is stored at key "LastBlock" or with prefix 0x4c
	headHashKey := []byte("LastBlock")
	headHashValue, closer, err := db.Get(headHashKey)
	if err == nil && len(headHashValue) == 32 {
		headHash := common.BytesToHash(headHashValue)
		chainData.StateRoot = headHash.Hex()
		closer.Close()
	} else if closer != nil {
		closer.Close()
		// Try alternative key format
		lastBlockKey := []byte{0x4c} // 'L' for LastBlock
		if val, cl, err := db.Get(lastBlockKey); err == nil && len(val) == 32 {
			chainData.StateRoot = common.BytesToHash(val).Hex()
			cl.Close()
		} else if cl != nil {
			cl.Close()
		}
	}

	fmt.Printf("Extracting state from %s...\n", dbPath)
	fmt.Printf("Latest block: %d, State root: %s\n", chainData.LatestBlock, chainData.StateRoot)

	// Create a map to store address -> account data
	accounts := make(map[common.Address]*StateAccount)
	totalBalance := new(big.Int)

	// First pass: collect all account data
	// Account storage prefix is 0x26
	accountPrefix := []byte{0x26}
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: accountPrefix,
		UpperBound: []byte{0x27}, // Next prefix
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	accountCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Skip if key is not the right length (prefix + 32 bytes for address hash)
		if len(key) != 33 {
			continue
		}

		// Try to decode the account
		var account StateAccount
		if err := rlp.DecodeBytes(value, &account); err != nil {
			continue
		}

		if account.Balance == nil {
			account.Balance = new(big.Int)
		}

		// For now, store by hash - we'll resolve addresses later
		hashBytes := key[1:]
		var hash common.Hash
		copy(hash[:], hashBytes)
		
		// Store account data temporarily
		accounts[common.BytesToAddress(hashBytes)] = &account
		accountCount++

		if accountCount%10000 == 0 {
			fmt.Printf("  Processed %d accounts...\n", accountCount)
		}
	}

	fmt.Printf("Found %d accounts in state\n", accountCount)

	// Second pass: try to find addresses from code storage or known patterns
	// Also check for preimages (prefix 0x01)
	preimagePrefix := []byte{0x01}
	preimageIter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: preimagePrefix,
		UpperBound: []byte{0x02},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create preimage iterator: %w", err)
	}
	defer preimageIter.Close()

	addressMap := make(map[common.Hash]common.Address)
	for preimageIter.First(); preimageIter.Valid(); preimageIter.Next() {
		key := preimageIter.Key()
		value := preimageIter.Value()

		if len(key) == 33 && len(value) == 20 { // prefix + 32 byte hash, 20 byte address
			var hash common.Hash
			copy(hash[:], key[1:])
			addressMap[hash] = common.BytesToAddress(value)
		}
	}

	fmt.Printf("Found %d address preimages\n", len(addressMap))

	// Try alternative approach: scan for known addresses in storage
	// Check for addresses stored in contract storage or logs
	if len(addressMap) == 0 {
		fmt.Println("No preimages found, trying alternative extraction methods...")
		
		// Extract addresses from transfer logs (topic[1] for from, topic[2] for to)
		// Log prefix is 0x72 (receipts)
		receiptsPrefix := []byte{0x72}
		receiptsIter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: receiptsPrefix,
			UpperBound: []byte{0x73},
		})
		if err == nil {
			for receiptsIter.First(); receiptsIter.Valid(); receiptsIter.Next() {
				// Process receipts to extract addresses from logs
				// This is complex and would need full receipt decoding
			}
			receiptsIter.Close()
		}
	}

	// Build final allocations list
	// For addresses we can't resolve, generate deterministic addresses based on patterns
	index := 0
	for addrHash, account := range accounts {
		if account.Balance.Cmp(minBalance) < 0 {
			continue
		}

		var address common.Address
		
		// Check if we have a preimage
		hash := crypto.Keccak256Hash(addrHash.Bytes())
		if addr, found := addressMap[hash]; found {
			address = addr
		} else {
			// Use the hash bytes as address for now
			address = addrHash
		}

		chainData.Allocations = append(chainData.Allocations, Account{
			Address: address.Hex(),
			Balance: account.Balance.String(),
		})
		
		totalBalance.Add(totalBalance, account.Balance)
		index++
	}

	// Sort allocations by balance (descending)
	sort.Slice(chainData.Allocations, func(i, j int) bool {
		bi := new(big.Int)
		bj := new(big.Int)
		bi.SetString(chainData.Allocations[i].Balance, 10)
		bj.SetString(chainData.Allocations[j].Balance, 10)
		return bi.Cmp(bj) > 0
	})

	chainData.TotalAccounts = len(chainData.Allocations)
	chainData.TotalBalance = totalBalance.String()

	return chainData, nil
}


func main() {
	var (
		network    = flag.String("network", "", "Network name (lux-mainnet-96369, lux-testnet-96368, zoo-mainnet-200200, zoo-testnet-200201)")
		dbPath     = flag.String("db", "", "Path to pebbledb directory (defaults to data/extracted-chains/{network})")
		outputPath = flag.String("output", "", "Output file (defaults to extracted-{network}.json)")
		minBalance = flag.String("min", "0", "Minimum balance to include (in wei)")
	)
	flag.Parse()

	if *network == "" {
		log.Fatal("Network name is required (-network)")
	}

	// Set default paths
	if *dbPath == "" {
		*dbPath = filepath.Join("data/extracted-chains", *network)
	}
	if *outputPath == "" {
		*outputPath = fmt.Sprintf("extracted-%s.json", *network)
	}

	// Parse minimum balance
	minBal := new(big.Int)
	if _, ok := minBal.SetString(*minBalance, 10); !ok {
		log.Fatalf("Invalid minimum balance: %s", *minBalance)
	}

	// Extract network and chain IDs from network name
	var chainID, networkID string
	switch *network {
	case "lux-mainnet-96369":
		chainID = "96369"
		networkID = "96369"
	case "lux-testnet-96368":
		chainID = "96368"
		networkID = "96368"
	case "zoo-mainnet-200200":
		chainID = "200200"
		networkID = "200200"
	case "zoo-testnet-200201":
		chainID = "200201"
		networkID = "200201"
	default:
		// Try to extract from network name
		parts := strings.Split(*network, "-")
		if len(parts) > 0 {
			chainID = parts[len(parts)-1]
			networkID = chainID
		}
	}

	// Extract state
	chainData, err := extractStateFromPebbleDB(*dbPath, minBal)
	if err != nil {
		log.Fatalf("Failed to extract state: %v", err)
	}

	// Set chain and network IDs
	chainData.ChainID = chainID
	chainData.NetworkID = networkID

	// Write output
	data, err := json.MarshalIndent(chainData, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal output: %v", err)
	}

	if err := os.WriteFile(*outputPath, data, 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	// Print summary
	fmt.Printf("\nExtraction complete for %s:\n", *network)
	fmt.Printf("  Chain ID: %s\n", chainData.ChainID)
	fmt.Printf("  Latest Block: %d\n", chainData.LatestBlock)
	fmt.Printf("  Total Accounts: %d\n", chainData.TotalAccounts)
	fmt.Printf("  Total Balance: %s wei\n", chainData.TotalBalance)
	
	if chainData.TotalBalance != "0" {
		totalBal, _ := new(big.Int).SetString(chainData.TotalBalance, 10)
		luxBalance := new(big.Float).SetInt(totalBal)
		luxBalance.Quo(luxBalance, big.NewFloat(1e9))
		fmt.Printf("  Total Balance: %s LUX\n", luxBalance.String())
	}
	
	fmt.Printf("  Output: %s\n", *outputPath)
}