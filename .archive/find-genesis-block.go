package main

import (
    "encoding/hex"
    "encoding/json"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: find-genesis-block <original-subnet-db>")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open PebbleDB
    db, err := pebble.Open(dbPath, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    fmt.Println("=== Finding Genesis Block in Original Subnet Data ===")
    
    // Look for specific keys that might contain genesis info
    keysToCheck := []string{
        "genesis",
        "genesisHash",
        "lastAccepted",
        "vm/lastAcceptedKey",
        "finalized/0",
        "accepted/0",
        "block/0",
    }
    
    for _, key := range keysToCheck {
        if val, closer, err := db.Get([]byte(key)); err == nil {
            fmt.Printf("\nFound key '%s':\n", key)
            fmt.Printf("  Value (hex): %s\n", hex.EncodeToString(val))
            fmt.Printf("  Value length: %d\n", len(val))
            closer.Close()
        }
    }
    
    // Also look for keys with specific prefixes
    fmt.Println("\n=== Scanning Key Patterns ===")
    
    patterns := []struct{
        name string
        prefix []byte
    }{
        {"finalized", []byte("finalized")},
        {"accepted", []byte("accepted")},
        {"block", []byte("block")},
        {"height", []byte("height")},
        {"vm/", []byte("vm/")},
    }
    
    for _, p := range patterns {
        fmt.Printf("\nPattern '%s':\n", p.name)
        
        iter, _ := db.NewIter(&pebble.IterOptions{
            LowerBound: p.prefix,
            UpperBound: append(p.prefix, 0xff),
        })
        defer iter.Close()
        
        count := 0
        for iter.First(); iter.Valid() && count < 5; iter.Next() {
            key := iter.Key()
            val := iter.Value()
            
            fmt.Printf("  Key: %s\n", string(key))
            fmt.Printf("    Value len: %d", len(val))
            if len(val) <= 64 {
                fmt.Printf(", hex: %s", hex.EncodeToString(val))
            }
            fmt.Println()
            count++
        }
    }
    
    // For subnet 96369, we need to find the actual chain config
    // Let's create a minimal genesis that matches what we know
    fmt.Println("\n=== Creating Genesis for Chain 96369 ===")
    
    genesis := map[string]interface{}{
        "config": map[string]interface{}{
            "chainId": 96369,
            "homesteadBlock": 0,
            "eip150Block": 0,
            "eip155Block": 0,
            "eip158Block": 0,
            "byzantiumBlock": 0,
            "constantinopleBlock": 0,
            "petersburgBlock": 0,
            "istanbulBlock": 0,
            "berlinBlock": 0,
            "londonBlock": 0,
            "shanghaiBlock": 0,
            "terminalTotalDifficulty": "0x0",
            "terminalTotalDifficultyPassed": true,
        },
        "difficulty": "0x1",
        "gasLimit": "0x7a1200",
        "timestamp": "0x0",
        "extraData": "0x00",
        "alloc": map[string]interface{}{
            // Treasury account with initial balance
            "0x9011e888251ab053b7bd1cdb598db4f9ded94714": map[string]interface{}{
                "balance": "0x6c6b935b8bbd400000", // 2T in wei
            },
        },
    }
    
    genesisJSON, _ := json.MarshalIndent(genesis, "", "  ")
    fmt.Printf("\nGenesis JSON:\n%s\n", string(genesisJSON))
    
    // Save to file
    err = os.WriteFile("cchain-genesis.json", genesisJSON, 0644)
    if err == nil {
        fmt.Println("\nSaved to cchain-genesis.json")
    }
}