package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/core/types"
    "github.com/luxfi/geth/rlp"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: debug-keys <db-path>")
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
    
    fmt.Printf("=== Debugging keys in: %s ===\n\n", dbPath)
    
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    // Look at first 20 keys with val > 100 bytes
    count := 0
    for iter.First(); iter.Valid() && count < 20; iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        if len(val) < 100 {
            continue
        }
        
        fmt.Printf("Key %d:\n", count+1)
        fmt.Printf("  Key hex: %x\n", key)
        fmt.Printf("  Key len: %d\n", len(key))
        fmt.Printf("  Val len: %d\n", len(val))
        
        // Try to identify the key pattern
        if len(key) >= 10 {
            fmt.Printf("  Prefix: %x\n", key[:10])
            fmt.Printf("  Byte[9]: 0x%02x ('%c')\n", key[9], key[9])
        }
        
        // Try to decode as header
        var header types.Header
        if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
            fmt.Printf("  ✓ Valid header! Block: %d, Hash: %x\n", header.Number.Uint64(), header.Hash())
        }
        
        // Try to decode as body
        var body types.Body
        if err := rlp.DecodeBytes(val, &body); err == nil {
            fmt.Printf("  ✓ Valid body! Txs: %d, Uncles: %d\n", len(body.Transactions), len(body.Uncles))
        }
        
        // Show first 64 bytes of value
        if len(val) > 64 {
            fmt.Printf("  Val start: %x...\n", val[:64])
        } else {
            fmt.Printf("  Val: %x\n", val)
        }
        
        fmt.Println()
        count++
    }
    
    // Now specifically look for keys that might be headers/bodies
    fmt.Println("\n=== Looking for specific patterns ===")
    
    // Count different byte[9] values
    byteCounts := make(map[byte]int)
    iter2, _ := db.NewIter(&pebble.IterOptions{})
    for iter2.First(); iter2.Valid(); iter2.Next() {
        key := iter2.Key()
        if len(key) >= 10 && len(iter2.Value()) > 100 {
            byteCounts[key[9]]++
        }
    }
    iter2.Close()
    
    fmt.Println("Distribution of byte[9] for keys with val > 100:")
    for b, cnt := range byteCounts {
        fmt.Printf("  0x%02x ('%c'): %d keys\n", b, b, cnt)
    }
}