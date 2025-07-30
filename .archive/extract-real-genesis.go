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
    Balance string `json:"balance"`
    Nonce   uint64 `json:"nonce,omitempty"`
    Code    string `json:"code,omitempty"`
    Storage map[string]string `json:"storage,omitempty"`
}

type Genesis struct {
    Config     *ChainConfig      `json:"config"`
    Nonce      string                   `json:"nonce"`
    Timestamp  string                   `json:"timestamp"`
    ExtraData  string                   `json:"extraData"`
    GasLimit   string                   `json:"gasLimit"`
    Difficulty string                   `json:"difficulty"`
    Mixhash    string                   `json:"mixHash"`
    Coinbase   string                   `json:"coinbase"`
    Alloc      map[string]GenesisAccount `json:"alloc"`
}

type ChainConfig struct {
    ChainID                *big.Int `json:"chainId"`
    HomesteadBlock         *big.Int `json:"homesteadBlock"`
    EIP150Block            *big.Int `json:"eip150Block"`
    EIP155Block            *big.Int `json:"eip155Block"`
    EIP158Block            *big.Int `json:"eip158Block"`
    ByzantiumBlock         *big.Int `json:"byzantiumBlock"`
    ConstantinopleBlock    *big.Int `json:"constantinopleBlock"`
    PetersburgBlock        *big.Int `json:"petersburgBlock"`
    IstanbulBlock          *big.Int `json:"istanbulBlock"`
    BerlinBlock            *big.Int `json:"berlinBlock"`
    LondonBlock            *big.Int `json:"londonBlock"`
}

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: extract-real-genesis <subnet-db>")
        fmt.Println("This extracts genesis accounts from subnet database")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open database
    db, err := pebble.Open(dbPath, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    fmt.Printf("=== Extracting genesis from %s ===\n", dbPath)
    
    // Create genesis structure
    genesis := &Genesis{
        Config: &ChainConfig{
            ChainID:                big.NewInt(96369),
            HomesteadBlock:         big.NewInt(0),
            EIP150Block:            big.NewInt(0),
            EIP155Block:            big.NewInt(0),
            EIP158Block:            big.NewInt(0),
            ByzantiumBlock:         big.NewInt(0),
            ConstantinopleBlock:    big.NewInt(0),
            PetersburgBlock:        big.NewInt(0),
            IstanbulBlock:          big.NewInt(0),
            BerlinBlock:            big.NewInt(0),
            LondonBlock:            big.NewInt(0),
        },
        Nonce:      "0x0",
        Timestamp:  "0x0",
        ExtraData:  "0x00",
        GasLimit:   "0x7a1200",
        Difficulty: "0x1",
        Mixhash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
        Coinbase:   "0x0000000000000000000000000000000000000000",
        Alloc:      make(map[string]GenesisAccount),
    }
    
    // Look for accounts in the database
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    accountCount := 0
    totalBalance := big.NewInt(0)
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        // Look for account data
        if len(key) == 40 && len(val) > 0 {
            // Try to decode as account
            var acc types.StateAccount
            if err := rlp.DecodeBytes(val, &acc); err == nil && acc.Balance != nil && acc.Balance.Sign() > 0 {
                // Extract address from key
                addr := common.BytesToAddress(key[8:28])
                
                genesis.Alloc[addr.Hex()] = GenesisAccount{
                    Balance: "0x" + acc.Balance.Text(16),
                    Nonce:   acc.Nonce,
                }
                
                accountCount++
                totalBalance.Add(totalBalance, acc.Balance.ToBig())
                
                if accountCount <= 5 {
                    fmt.Printf("Account %s: balance=%s\n", addr.Hex(), acc.Balance.String())
                }
            }
        }
    }
    
    fmt.Printf("\nFound %d accounts with total balance: %s\n", accountCount, totalBalance.String())
    
    // Write genesis.json
    genesisData, err := json.MarshalIndent(genesis, "", "  ")
    if err != nil {
        log.Fatalf("Failed to marshal genesis: %v", err)
    }
    
    outputFile := "extracted-genesis.json"
    if err := os.WriteFile(outputFile, genesisData, 0644); err != nil {
        log.Fatalf("Failed to write genesis: %v", err)
    }
    
    fmt.Printf("\nWrote genesis to %s\n", outputFile)
    
    // Also look for the first few blocks to understand structure
    fmt.Println("\n=== Looking for block data ===")
    
    // Look for headers with specific patterns
    headerCount := 0
    for iter.First(); iter.Valid() && headerCount < 10; iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        if len(val) > 100 && len(val) < 1000 {
            // Try to decode as header
            var header types.Header
            if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
                fmt.Printf("Found header at key %x: block=%d, hash=%x\n", key, header.Number.Uint64(), header.Hash())
                headerCount++
            }
        }
    }
}