package main

import (
    "encoding/hex"
    "fmt"
    "log"
    "os"
    "strings"
    
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/opt"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: inspect-network-db <network-db-path>")
        fmt.Println("Example: inspect-network-db runtime/clean-cchain/db/network-96369/v1.4.5")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open LevelDB
    db, err := leveldb.OpenFile(dbPath, &opt.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    fmt.Printf("=== Network Database Analysis: %s ===\n\n", dbPath)
    
    // Look for C-Chain related keys
    iter := db.NewIterator(nil, nil)
    defer iter.Release()
    
    // Track key patterns
    patterns := make(map[string]int)
    cchainKeys := []string{}
    
    for iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        keyHex := hex.EncodeToString(key)
        
        // Look for patterns
        var pattern string
        if len(key) >= 32 {
            // Might start with blockchain ID
            pattern = "blockchain-prefix-" + keyHex[:64] + "..."
        } else if len(key) >= 2 {
            pattern = fmt.Sprintf("short-key-%02x", key[0])
        } else {
            pattern = "empty-key"
        }
        
        patterns[pattern]++
        
        // Look for C-Chain blockchain ID
        if strings.Contains(keyHex, "58364355357167674d4a667a73544239395577786a325a593568643638783431486646345a346d346843425762486a314562633333") {
            // This is the hex encoding of "X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3"
            cchainKeys = append(cchainKeys, fmt.Sprintf("C-Chain key found: %s (val len: %d)", keyHex, len(val)))
        }
        
        // Check for vm prefix after blockchain ID
        if len(key) > 32 && string(key[32:34]) == "vm" {
            if len(cchainKeys) < 10 {
                suffix := ""
                if len(key) > 34 {
                    suffix = fmt.Sprintf(" suffix: %x", key[34:])
                    if len(key) > 35 && key[34] == 'h' && len(key) == 44 {
                        suffix += " (looks like canonical hash key)"
                    }
                }
                cchainKeys = append(cchainKeys, fmt.Sprintf("VM key: %s%s", keyHex[:68], suffix))
            }
        }
    }
    
    fmt.Println("Key pattern distribution:")
    for pattern, count := range patterns {
        fmt.Printf("  %s: %d keys\n", pattern, count)
    }
    
    fmt.Printf("\nC-Chain related keys found: %d\n", len(cchainKeys))
    for i, key := range cchainKeys {
        fmt.Printf("  %d. %s\n", i+1, key)
        if i >= 9 {
            fmt.Printf("  ... and %d more\n", len(cchainKeys)-10)
            break
        }
    }
    
    // Now specifically look for the canonical hash key
    fmt.Println("\nLooking for canonical hash key (h + block 0)...")
    
    // Try different possible prefixes
    prefixes := []struct{
        name string
        prefix []byte
    }{
        {"Direct", []byte{}},
        {"VM prefix", []byte("vm")},
    }
    
    // C-Chain blockchain ID bytes (we need to decode this)
    // cchainIDStr := "X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3"
    
    for _, p := range prefixes {
        // Canonical hash key for block 0
        canonicalKey := append(p.prefix, []byte{0x68, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6e}...)
        
        val, err := db.Get(canonicalKey, nil)
        if err == nil {
            fmt.Printf("✓ Found with %s: key=%x value=%x\n", p.name, canonicalKey, val)
        } else {
            fmt.Printf("✗ Not found with %s: key=%x\n", p.name, canonicalKey)
        }
    }
}