package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

type GenesisAccount struct {
	Balance string            `json:"balance"`
	Nonce   uint64            `json:"nonce,omitempty"`
	Code    string            `json:"code,omitempty"`
	Storage map[string]string `json:"storage,omitempty"`
}

type GenesisAlloc map[string]GenesisAccount

type Genesis struct {
	Config     *ChainConfig `json:"config"`
	Nonce      string       `json:"nonce"`
	Timestamp  string       `json:"timestamp"`
	ExtraData  string       `json:"extraData"`
	GasLimit   string       `json:"gasLimit"`
	Difficulty string       `json:"difficulty"`
	Mixhash    string       `json:"mixhash"`
	Coinbase   string       `json:"coinbase"`
	Alloc      GenesisAlloc `json:"alloc"`
	GasUsed    string       `json:"gasUsed"`
	Number     string       `json:"number"`
	ParentHash string       `json:"parentHash"`
}

type ChainConfig struct {
	ChainID                       *big.Int `json:"chainId"`
	HomesteadBlock                *big.Int `json:"homesteadBlock"`
	EIP150Block                   *big.Int `json:"eip150Block"`
	EIP155Block                   *big.Int `json:"eip155Block"`
	EIP158Block                   *big.Int `json:"eip158Block"`
	ByzantiumBlock                *big.Int `json:"byzantiumBlock"`
	ConstantinopleBlock           *big.Int `json:"constantinopleBlock"`
	PetersburgBlock               *big.Int `json:"petersburgBlock"`
	IstanbulBlock                 *big.Int `json:"istanbulBlock"`
	BerlinBlock                   *big.Int `json:"berlinBlock"`
	LondonBlock                   *big.Int `json:"londonBlock"`
	ShanghaiBlock                 *big.Int `json:"shanghaiBlock,omitempty"`
	TerminalTotalDifficulty       string   `json:"terminalTotalDifficulty"`
	TerminalTotalDifficultyPassed bool     `json:"terminalTotalDifficultyPassed"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: export-state-to-genesis <db-path> <output-genesis.json>")
		fmt.Println("This exports all state from a database to a genesis.json file")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	outputFile := os.Args[2]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("=== Exporting state from %s ===\n", dbPath)

	// Create genesis structure
	genesis := &Genesis{
		Config: &ChainConfig{
			ChainID:                       big.NewInt(96369),
			HomesteadBlock:                big.NewInt(0),
			EIP150Block:                   big.NewInt(0),
			EIP155Block:                   big.NewInt(0),
			EIP158Block:                   big.NewInt(0),
			ByzantiumBlock:                big.NewInt(0),
			ConstantinopleBlock:           big.NewInt(0),
			PetersburgBlock:               big.NewInt(0),
			IstanbulBlock:                 big.NewInt(0),
			BerlinBlock:                   big.NewInt(0),
			LondonBlock:                   big.NewInt(0),
			ShanghaiBlock:                 big.NewInt(0),
			TerminalTotalDifficulty:       "0x0",
			TerminalTotalDifficultyPassed: true,
		},
		Nonce:      "0x0",
		Timestamp:  "0x0",
		ExtraData:  "0x00",
		GasLimit:   "0x7a1200",
		Difficulty: "0x0",
		Mixhash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		GasUsed:    "0x0",
		Number:     "0x0",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
		Alloc:      make(GenesisAlloc),
	}

	// Track accounts and their data
	accounts := make(map[common.Address]*types.StateAccount)
	contractCode := make(map[common.Address][]byte)
	contractStorage := make(map[common.Address]map[common.Hash]common.Hash)

	// Iterate through database
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	accountCount := 0
	codeCount := 0
	storageCount := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		// Skip if not prefixed with "evm"
		if len(key) < 3 || string(key[:3]) != "evm" {
			continue
		}

		// Remove "evm" prefix for analysis
		key = key[3:]

		// Identify key type based on patterns
		if len(key) == 40 { // Account key
			// Try to decode as account
			var acc types.StateAccount
			if err := rlp.DecodeBytes(val, &acc); err == nil {
				// Extract address (simplified - in reality needs proper decoding)
				addr := common.BytesToAddress(key[8:28])
				accounts[addr] = &acc
				accountCount++

				if accountCount%1000 == 0 {
					fmt.Printf("Processed %d accounts...\n", accountCount)
				}
			}
		} else if len(key) == 41 && key[40] == 0x63 { // Code key (ends with 'c')
			// Extract address and code
			addr := common.BytesToAddress(key[8:28])
			contractCode[addr] = val
			codeCount++
		} else if len(key) == 72 { // Storage key (address + slot)
			// Extract address and storage slot
			addr := common.BytesToAddress(key[8:28])
			slot := common.BytesToHash(key[40:72])
			value := common.BytesToHash(val)

			if contractStorage[addr] == nil {
				contractStorage[addr] = make(map[common.Hash]common.Hash)
			}
			contractStorage[addr][slot] = value
			storageCount++
		}
	}

	fmt.Printf("\nFound:\n")
	fmt.Printf("  Accounts: %d\n", accountCount)
	fmt.Printf("  Contract codes: %d\n", codeCount)
	fmt.Printf("  Storage entries: %d\n", storageCount)

	// Build genesis alloc
	totalBalance := big.NewInt(0)

	for addr, acc := range accounts {
		if acc.Balance == nil || acc.Balance.Sign() == 0 {
			continue // Skip zero-balance accounts
		}

		genesisAcc := GenesisAccount{
			Balance: "0x" + acc.Balance.Hex(),
			Nonce:   acc.Nonce,
		}

		// Add code if this is a contract
		if code, hasCode := contractCode[addr]; hasCode && len(code) > 0 {
			genesisAcc.Code = "0x" + common.Bytes2Hex(code)

			// Add storage
			if storage, hasStorage := contractStorage[addr]; hasStorage && len(storage) > 0 {
				genesisAcc.Storage = make(map[string]string)
				for slot, value := range storage {
					if value != (common.Hash{}) { // Skip zero values
						genesisAcc.Storage["0x"+slot.Hex()[2:]] = "0x" + value.Hex()[2:]
					}
				}
			}
		}

		genesis.Alloc[addr.Hex()] = genesisAcc
		totalBalance.Add(totalBalance, acc.Balance.ToBig())
	}

	fmt.Printf("\nTotal balance across all accounts: %s\n", totalBalance.String())
	fmt.Printf("Genesis alloc contains %d accounts\n", len(genesis.Alloc))

	// Write genesis file
	genesisData, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal genesis: %v", err)
	}

	if err := os.WriteFile(outputFile, genesisData, 0644); err != nil {
		log.Fatalf("Failed to write genesis file: %v", err)
	}

	fmt.Printf("\nWrote genesis to %s\n", outputFile)
	fmt.Println("\nNext steps:")
	fmt.Println("1. Review the genesis file to ensure it's correct")
	fmt.Println("2. Initialize geth with: geth init <genesis-file>")
	fmt.Println("3. Start your new network with this genesis")
}
