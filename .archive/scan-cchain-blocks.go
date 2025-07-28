package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/opt"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: scan-cchain-blocks <leveldb-path>")
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
    
    fmt.Printf("=== Scanning C-Chain database: %s ===\n\n", dbPath)
    
    // Check for "evm" prefixed keys first
    fmt.Println("Checking for 'evm' prefixed keys...")
    checkEvmPrefix(db)
    
    // Check for standard rawdb prefixes
    fmt.Println("\nChecking for standard rawdb prefixes...")
    checkRawdbPrefixes(db)
    
    // Check for simple height keys (like avalanchego test)
    fmt.Println("\nChecking for simple height keys...")
    checkSimpleHeightKeys(db)
    
    // Scan for any patterns
    fmt.Println("\nScanning first 1000 keys for patterns...")
    scanPatterns(db)
}

func checkEvmPrefix(db *leveldb.DB) {
    // Check various evm/ patterns
    patterns := []string{
        "evm/h/",      // headers
        "evm/b/",      // bodies
        "evm/n/",      // number->hash
        "evm/H/",      // hash->number
        "evm/T/",      // total difficulty
        "evm/r/",      // receipts
        "evm/l/",      // tx lookups
        "evm/secure/", // state trie
        "evm/head_header",
        "evm/head_block",
        "evm/chain_config",
    }
    
    for _, pattern := range patterns {
        iter := db.NewIterator(nil, nil)
        defer iter.Release()
        
        count := 0
        for iter.Seek([]byte(pattern)); iter.Valid(); iter.Next() {
            key := string(iter.Key())
            if len(key) >= len(pattern) && key[:len(pattern)] == pattern {
                if count == 0 {
                    fmt.Printf("  Found %s data!\n", pattern)
                    fmt.Printf("    Example key: %s\n", key[:min(64, len(key))])
                    fmt.Printf("    Value size: %d bytes\n", len(iter.Value()))
                }
                count++
            } else {
                break
            }
        }
        if count > 0 {
            fmt.Printf("    Total %s entries: %d\n", pattern, count)
        }
    }
}

func checkRawdbPrefixes(db *leveldb.DB) {
    // Check for standard single-byte prefixes
    prefixes := map[byte]string{
        0x68: "headers (h)",
        0x62: "bodies (b)", 
        0x6e: "number->hash (n)",
        0x48: "hash->number (H)",
        0x54: "total difficulty (T)",
        0x72: "receipts (r)",
        0x6c: "tx lookups (l)",
        0x53: "secure trie (S)",
    }
    
    for prefix, name := range prefixes {
        iter := db.NewIterator(nil, nil)
        defer iter.Release()
        
        count := 0
        for iter.Seek([]byte{prefix}); iter.Valid(); iter.Next() {
            key := iter.Key()
            if len(key) > 0 && key[0] == prefix {
                if count == 0 {
                    fmt.Printf("  Found %s data!\n", name)
                    fmt.Printf("    Example key: %s\n", hex.EncodeToString(key[:min(32, len(key))]))
                    fmt.Printf("    Value size: %d bytes\n", len(iter.Value()))
                }
                count++
            } else if len(key) > 0 && key[0] > prefix {
                break
            }
        }
        if count > 0 {
            fmt.Printf("    Total entries: %d\n", count)
        }
    }
}

func checkSimpleHeightKeys(db *leveldb.DB) {
    // Check for blocks stored as simple 8-byte height keys
    for height := uint64(0); height < 10; height++ {
        key := make([]byte, 8)
        binary.BigEndian.PutUint64(key, height)
        
        val, err := db.Get(key, nil)
        if err == nil {
            fmt.Printf("  Found block at height %d! Size: %d bytes\n", height, len(val))
            fmt.Printf("    Key: %s\n", hex.EncodeToString(key))
            fmt.Printf("    First 64 bytes: %s\n", hex.EncodeToString(val[:min(64, len(val))]))
        }
    }
}

func scanPatterns(db *leveldb.DB) {
    iter := db.NewIterator(nil, nil)
    defer iter.Release()
    
    patterns := make(map[string]int)
    count := 0
    
    for iter.Next() && count < 1000 {
        key := iter.Key()
        count++
        
        // Identify pattern
        var pattern string
        if len(key) > 0 {
            if key[0] >= 32 && key[0] <= 126 { // Printable ASCII
                // Look for string prefix
                end := 0
                for end < len(key) && end < 10 && key[end] >= 32 && key[end] <= 126 {
                    end++
                }
                pattern = string(key[:end])
            } else {
                pattern = fmt.Sprintf("0x%02x", key[0])
            }
        }
        
        patterns[pattern]++
    }
    
    fmt.Println("\nTop patterns found:")
    for p, c := range patterns {
        if c > 5 {
            fmt.Printf("  %s: %d occurrences\n", p, c)
        }
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}