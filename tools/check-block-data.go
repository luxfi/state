package main

import (
    "encoding/hex"
    "fmt"
    "log"
    "os"

    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) != 2 {
        log.Fatalf("Usage: %s <path/to/db>", os.Args[0])
    }
    dbPath := os.Args[1]

    db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatalf("pebble.Open: %v", err)
    }
    defer db.Close()

    // Define the prefixes we're looking for
    prefixes := map[string]byte{
        "headers":     0x68, // 'h'
        "bodies":      0x62, // 'b' 
        "canonical":   0x6e, // 'n'
        "hash->num":   0x48, // 'H'
        "difficulty":  0x74, // 't'
        "receipts":    0x72, // 'r'
        "tx-lookup":   0x6c, // 'l'
    }

    // Check for each prefix
    for name, prefix := range prefixes {
        count := 0
        iter, err := db.NewIter(&pebble.IterOptions{})
        if err != nil {
            log.Fatalf("NewIter: %v", err)
        }
        
        for iter.First(); iter.Valid() && count < 1000000; iter.Next() {
            key := iter.Key()
            if len(key) > 0 && key[0] == prefix {
                count++
                if count <= 3 {
                    fmt.Printf("%s key found: %s (len=%d)\n", name, hex.EncodeToString(key[:min(len(key), 32)]), len(key))
                }
            }
        }
        iter.Close()
        
        if count > 0 {
            fmt.Printf("✓ Found %d %s entries\n", count, name)
        }
    }
    
    // Also check for evm-prefixed data
    fmt.Println("\nChecking for evm-prefixed data...")
    evmCount := 0
    iter, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("NewIter: %v", err)
    }
    defer iter.Close()
    
    for iter.First(); iter.Valid() && evmCount < 100; iter.Next() {
        key := iter.Key()
        if len(key) >= 3 && string(key[:3]) == "evm" {
            evmCount++
            if evmCount <= 3 {
                fmt.Printf("evm-prefixed key: %s (len=%d)\n", hex.EncodeToString(key[:min(len(key), 32)]), len(key))
            }
        }
    }
    
    if evmCount > 0 {
        fmt.Printf("✓ Found %d evm-prefixed entries\n", evmCount)
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}