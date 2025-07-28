package main

import (
    "encoding/json"
    "fmt"
    "log"
    "math/big"
    "os"
    "path/filepath"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/common"
    "github.com/luxfi/geth/core/types"
    "github.com/luxfi/geth/params"
    "github.com/luxfi/geth/rlp"
)

// GenesisAlloc represents the genesis allocation
type GenesisAlloc map[common.Address]types.Account

// Genesis represents the genesis block configuration
type Genesis struct {
    Config    *params.ChainConfig `json:"config"`
    Nonce     uint64             `json:"nonce"`
    Timestamp uint64             `json:"timestamp"`
    ExtraData []byte             `json:"extraData"`
    GasLimit  uint64             `json:"gasLimit"`
    Difficulty *big.Int          `json:"difficulty"`
    Mixhash   common.Hash        `json:"mixHash"`
    Coinbase  common.Address     `json:"coinbase"`
    Alloc     GenesisAlloc       `json:"alloc"`
}

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: export-subnet-to-geth <subnet-db> <output-dir>")
        fmt.Println("This will create:")
        fmt.Println("  <output-dir>/genesis.json - Genesis configuration")
        fmt.Println("  <output-dir>/chaindata/   - Geth-compatible database")
        os.Exit(1)
    }
    
    subnetDB := os.Args[1]
    outputDir := os.Args[2]
    
    // Create output directories
    chainDataDir := filepath.Join(outputDir, "chaindata")
    if err := os.MkdirAll(chainDataDir, 0755); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }
    
    // Open subnet database
    db, err := pebble.Open(subnetDB, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open subnet database: %v", err)
    }
    defer db.Close()
    
    fmt.Printf("=== Exporting subnet data from %s ===\n", subnetDB)
    
    // First, let's analyze what we have
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    keyTypes := make(map[string]int)
    accounts := make(map[common.Address]*big.Int)
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        if len(key) < 10 {
            continue
        }
        
        // Categorize keys
        switch key[9] {
        case 0x01: // Account trie
            if len(val) > 0 {
                // Try to decode account
                var acc types.StateAccount
                if err := rlp.DecodeBytes(val, &acc); err == nil {
                    // Extract address from key (this is simplified, real extraction is more complex)
                    if len(key) >= 42 {
                        addr := common.BytesToAddress(key[10:30])
                        accounts[addr] = acc.Balance.ToBig()
                        if len(accounts) < 5 {
                            fmt.Printf("Found account %s with balance %s\n", addr.Hex(), acc.Balance.String())
                        }
                    }
                }
            }
            keyTypes["accounts"]++
            
        case 0x48: // Headers
            keyTypes["headers"]++
            
        case 0x62: // Bodies
            keyTypes["bodies"]++
            
        case 0x72: // Receipts
            keyTypes["receipts"]++
            
        case 0xa3: // Storage
            keyTypes["storage"]++
            
        default:
            keyTypes["other"]++
        }
    }
    
    fmt.Println("\nKey type distribution:")
    for typ, count := range keyTypes {
        fmt.Printf("  %s: %d\n", typ, count)
    }
    
    fmt.Printf("\nTotal accounts found: %d\n", len(accounts))
    
    // Create genesis.json with the accounts
    genesis := &Genesis{
        Config: &params.ChainConfig{
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
            TerminalTotalDifficulty:       big.NewInt(0),
        },
        Nonce:      0,
        Timestamp:  1640995200, // 2022-01-01
        ExtraData:  []byte{},
        GasLimit:   8000000,
        Difficulty: big.NewInt(0),
        Mixhash:    common.Hash{},
        Coinbase:   common.Address{},
        Alloc:      make(GenesisAlloc),
    }
    
    // Add accounts to genesis
    for addr, balance := range accounts {
        genesis.Alloc[addr] = types.Account{
            Balance: balance,
        }
    }
    
    // Write genesis.json
    genesisPath := filepath.Join(outputDir, "genesis.json")
    genesisData, err := json.MarshalIndent(genesis, "", "  ")
    if err != nil {
        log.Fatalf("Failed to marshal genesis: %v", err)
    }
    
    if err := os.WriteFile(genesisPath, genesisData, 0644); err != nil {
        log.Fatalf("Failed to write genesis.json: %v", err)
    }
    
    fmt.Printf("\nCreated %s with %d accounts\n", genesisPath, len(accounts))
    
    // Now the important part: we need to use geth to import the blocks
    fmt.Println("\nTo complete the migration:")
    fmt.Println("1. Initialize geth with the genesis:")
    fmt.Printf("   geth --datadir %s init %s\n", outputDir, genesisPath)
    fmt.Println("\n2. Import the blocks (we need to create an RLP export first)")
    fmt.Println("3. The geth import will create the canonical tables")
}