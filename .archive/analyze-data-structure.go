package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: analyze-data-structure <db-path>")
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
    
    fmt.Println("=== Analyzing Data Structure ===")
    
    // Count keys by prefix
    prefixCounts := make(map[byte]int)
    prefixExamples := make(map[byte][]string)
    
    iter, _ := db.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    totalKeys := 0
    for iter.First(); iter.Valid() && totalKeys < 100000; iter.Next() {
        key := iter.Key()
        if len(key) > 0 {
            prefix := key[0]
            prefixCounts[prefix]++
            
            // Store first few examples
            if len(prefixExamples[prefix]) < 3 {
                val := iter.Value()
                example := fmt.Sprintf("Key: %s (len=%d), Val len=%d", 
                    hex.EncodeToString(key), len(key), len(val))
                
                // If it's a small value, show it
                if len(val) <= 32 {
                    example += fmt.Sprintf(", Val: %s", hex.EncodeToString(val))
                }
                
                prefixExamples[prefix] = append(prefixExamples[prefix], example)
            }
        }
        totalKeys++
    }
    
    fmt.Printf("\nTotal keys scanned: %d\n", totalKeys)
    fmt.Println("\n=== Key Prefix Analysis ===")
    
    // Common Ethereum prefixes
    prefixNames := map[byte]string{
        0x00: "misc/version",
        0x26: "accounts",
        0x48: "hash->number (H)",
        0x62: "bodies (b)",
        0x63: "code (c)",
        0x68: "headers (h)",
        0x6c: "last values (l)",
        0x6e: "canonical (n)",
        0x6f: "storage snapshot (o)",
        0x72: "receipts (r)",
        0x73: "state (s)",
        0x74: "transactions (t)",
        0xa3: "storage",
        0x42: "Bodies (B)",
        0xfd: "metadata",
    }
    
    for prefix, count := range prefixCounts {
        name := prefixNames[prefix]
        if name == "" {
            name = "unknown"
        }
        fmt.Printf("\n0x%02x (%s): %d keys\n", prefix, name, count)
        
        if examples, ok := prefixExamples[prefix]; ok {
            for _, ex := range examples {
                fmt.Printf("  %s\n", ex)
            }
        }
    }
    
    // Look for specific patterns
    fmt.Println("\n=== Looking for Block Numbers ===")
    
    // Try to find the highest block number
    // Look for canonical number keys (h + 8 bytes + n)
    highestBlock := uint64(0)
    
    iter2, _ := db.NewIter(&pebble.IterOptions{
        LowerBound: []byte{0x68}, // h prefix
        UpperBound: []byte{0x69},
    })
    defer iter2.Close()
    
    for iter2.First(); iter2.Valid(); iter2.Next() {
        key := iter2.Key()
        if len(key) == 10 && key[9] == 0x6e { // canonical number key
            blockNum := binary.BigEndian.Uint64(key[1:9])
            if blockNum > highestBlock && blockNum < 1000000 { // reasonable range
                highestBlock = blockNum
            }
        }
    }
    
    fmt.Printf("Highest block number found: %d\n", highestBlock)
    
    // Check if we can find block 0
    fmt.Println("\n=== Looking for Block 0 ===")
    
    // Try different key formats for block 0
    block0Keys := [][]byte{
        // h + 00000000 + n (canonical)
        {0x68, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e},
        // Just the number as key
        {0, 0, 0, 0, 0, 0, 0, 0},
    }
    
    for _, key := range block0Keys {
        if val, closer, err := db.Get(key); err == nil {
            fmt.Printf("Found key %s: %s\n", hex.EncodeToString(key), hex.EncodeToString(val))
            closer.Close()
        }
    }
}