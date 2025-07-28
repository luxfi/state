package main

import (
    "fmt"
    "log"
    "os"
    "sort"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/core/types"
    "github.com/luxfi/geth/rlp"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: analyze-subnet-blocks <db-path>")
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
    
    fmt.Printf("=== Analyzing %s ===\n\n", dbPath)
    
    // Track different key patterns
    keyPatterns := make(map[string]int)
    headers := make(map[uint64]*types.Header)
    bodies := make(map[uint64]*types.Body)
    receipts := make(map[uint64][]*types.Receipt)
    
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        // Categorize by key length and pattern
        pattern := fmt.Sprintf("len=%d", len(key))
        if len(key) > 0 {
            pattern += fmt.Sprintf(" prefix=%02x", key[0])
            if len(key) > 9 {
                pattern += fmt.Sprintf(" byte9=%02x", key[9])
            }
        }
        keyPatterns[pattern]++
        
        // Try to decode as different types
        if len(val) > 100 && len(val) < 2000 {
            // Try as header
            var header types.Header
            if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
                blockNum := header.Number.Uint64()
                headers[blockNum] = &header
                if blockNum < 10 || blockNum%1000 == 0 {
                    fmt.Printf("Found header: block %d, hash %x, parent %x\n", 
                        blockNum, header.Hash(), header.ParentHash)
                }
            }
            
            // Try as body
            var body types.Body
            if err := rlp.DecodeBytes(val, &body); err == nil && len(body.Transactions) > 0 {
                // Try to find which block this belongs to
                for num, hdr := range headers {
                    if hdr != nil {
                        bodies[num] = &body
                        break
                    }
                }
            }
        }
    }
    
    fmt.Println("\n=== Key Pattern Summary ===")
    patterns := make([]string, 0, len(keyPatterns))
    for p := range keyPatterns {
        patterns = append(patterns, p)
    }
    sort.Strings(patterns)
    
    for _, p := range patterns {
        fmt.Printf("%s: %d keys\n", p, keyPatterns[p])
    }
    
    fmt.Printf("\n=== Block Summary ===\n")
    fmt.Printf("Headers found: %d\n", len(headers))
    fmt.Printf("Bodies found: %d\n", len(bodies))
    fmt.Printf("Receipts found: %d\n", len(receipts))
    
    if len(headers) > 0 {
        // Find block range
        blockNums := make([]uint64, 0, len(headers))
        for num := range headers {
            blockNums = append(blockNums, num)
        }
        sort.Slice(blockNums, func(i, j int) bool { return blockNums[i] < blockNums[j] })
        
        fmt.Printf("Block range: %d to %d\n", blockNums[0], blockNums[len(blockNums)-1])
        
        // Show first few blocks
        fmt.Println("\nFirst few blocks:")
        for i := 0; i < 5 && i < len(blockNums); i++ {
            num := blockNums[i]
            hdr := headers[num]
            fmt.Printf("  Block %d: hash=%x, stateRoot=%x\n", num, hdr.Hash(), hdr.Root)
        }
    }
    
    // Now let's look for specific keys that might help us understand the structure
    fmt.Println("\n=== Looking for specific patterns ===")
    
    // Check for canonical hash keys (h + number + n)
    canonicalCount := 0
    for i := uint64(0); i <= 100; i++ {
        // Build canonical key
        key := make([]byte, 10)
        key[0] = 0x68 // 'h'
        for j := 0; j < 8; j++ {
            key[1+j] = byte(i >> uint(8*(7-j)))
        }
        key[9] = 0x6e // 'n'
        
        if val, closer, err := db.Get(key); err == nil {
            canonicalCount++
            if i < 5 {
                fmt.Printf("Canonical hash for block %d: %x\n", i, val)
            }
            closer.Close()
        }
    }
    fmt.Printf("Found %d canonical hash entries\n", canonicalCount)
}