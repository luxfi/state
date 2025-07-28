package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/rlp"
    "github.com/luxfi/geth/core/types"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: find-blocks-raw <db-path>")
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
    
    fmt.Printf("=== Finding Blocks in: %s ===\n\n", dbPath)
    
    // Look for RLP encoded blocks or headers
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    blocksFound := 0
    headersFound := 0
    bodiesFound := 0
    receiptsFound := 0
    
    for iter.First(); iter.Valid() && blocksFound < 10; iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        if len(val) < 100 {
            continue
        }
        
        // Try to decode as header
        var header types.Header
        if err := rlp.DecodeBytes(val, &header); err == nil {
            // Verify it looks like a valid header
            if header.Number != nil && header.Time > 0 {
                fmt.Printf("Found header at key %x:\n", key)
                fmt.Printf("  Block: %d\n", header.Number.Uint64())
                fmt.Printf("  Hash: %x\n", header.Hash())
                fmt.Printf("  Parent: %x\n", header.ParentHash)
                fmt.Printf("  Time: %d\n", header.Time)
                fmt.Printf("  Key length: %d\n", len(key))
                fmt.Printf("  Val length: %d\n", len(val))
                headersFound++
                blocksFound++
                
                // Now look for the body with the same hash pattern
                if len(key) >= 10 {
                    // Try to find body key
                    bodyKey := make([]byte, len(key))
                    copy(bodyKey, key)
                    // Change prefix byte if it exists
                    bodyKey[9] = 0x62 // 'b'
                    
                    if bodyVal, closer, err := db.Get(bodyKey); err == nil {
                        fmt.Printf("  Found matching body (len=%d)\n", len(bodyVal))
                        bodiesFound++
                        closer.Close()
                    }
                }
                fmt.Println()
            }
        }
        
        // Try to decode as body
        var body types.Body
        if err := rlp.DecodeBytes(val, &body); err == nil && len(body.Transactions) > 0 {
            fmt.Printf("Found body with %d transactions at key %x\n", len(body.Transactions), key)
            bodiesFound++
        }
    }
    
    fmt.Printf("\nSummary:\n")
    fmt.Printf("Headers found: %d\n", headersFound)
    fmt.Printf("Bodies found: %d\n", bodiesFound)
    fmt.Printf("Receipts found: %d\n", receiptsFound)
    
    // Let's look for specific key patterns that might contain blocks
    fmt.Println("\n=== Checking Known Patterns ===")
    
    // Pattern 1: Headers might be at 0x00 + ... + 0x48 ('H')
    // Pattern 2: Bodies might be at 0x00 + ... + 0x62 ('b')
    // Pattern 3: Receipts might be at 0x00 + ... + 0x72 ('r')
    
    patterns := []struct{
        name string
        check func([]byte) bool
    }{
        {"header-like", func(k []byte) bool { return len(k) > 9 && k[9] == 0x48 }},
        {"body-like", func(k []byte) bool { return len(k) > 9 && k[9] == 0x62 }},
        {"receipt-like", func(k []byte) bool { return len(k) > 9 && k[9] == 0x72 }},
        {"canonical-like", func(k []byte) bool { return len(k) == 10 && k[9] == 0x6e }},
    }
    
    for _, p := range patterns {
        count := 0
        iter, _ := db.NewIter(&pebble.IterOptions{})
        for iter.First(); iter.Valid() && count < 5; iter.Next() {
            if p.check(iter.Key()) {
                fmt.Printf("%s key found: %x (val len: %d)\n", p.name, iter.Key(), len(iter.Value()))
                count++
            }
        }
        iter.Close()
        if count == 0 {
            fmt.Printf("No %s keys found\n", p.name)
        }
    }
}