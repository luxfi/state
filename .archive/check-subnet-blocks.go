package main

import (
    "bytes"
    "encoding/binary"
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

    // The subnet prefix we found
    subnetPrefix := []byte{0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1}

    fmt.Printf("Looking for block data with subnet prefix: %x\n\n", subnetPrefix)

    // Block prefixes to look for after the subnet prefix
    blockPrefixes := map[string]byte{
        "headers":     0x68, // 'h'
        "bodies":      0x62, // 'b' 
        "canonical":   0x6e, // 'n'
        "hash->num":   0x48, // 'H'
        "difficulty":  0x74, // 't'
        "receipts":    0x72, // 'r'
        "tx-lookup":   0x6c, // 'l'
    }

    counts := make(map[string]int)
    
    iter, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("NewIter: %v", err)
    }
    defer iter.Close()

    totalKeys := 0
    for iter.First(); iter.Valid() && totalKeys < 1000000; iter.Next() {
        key := iter.Key()
        totalKeys++
        
        // Check if key starts with subnet prefix
        if len(key) > 32 && bytes.HasPrefix(key, subnetPrefix) {
            keyAfterPrefix := key[32:]
            
            // Check what type of key follows the prefix
            if len(keyAfterPrefix) > 0 {
                firstByte := keyAfterPrefix[0]
                
                for name, prefix := range blockPrefixes {
                    if firstByte == prefix {
                        counts[name]++
                        
                        // Show first few examples
                        if counts[name] <= 3 {
                            fmt.Printf("Found %s key:\n", name)
                            fmt.Printf("  Full key: %x\n", key[:min(len(key), 64)])
                            fmt.Printf("  After prefix: %x\n", keyAfterPrefix[:min(len(keyAfterPrefix), 32)])
                            
                            // Try to decode block number for canonical keys
                            if name == "canonical" && len(keyAfterPrefix) >= 9 {
                                blockNum := binary.BigEndian.Uint64(keyAfterPrefix[1:9])
                                fmt.Printf("  Block number: %d\n", blockNum)
                            }
                            
                            // For header/body keys, try to extract block number
                            if (name == "headers" || name == "bodies") && len(keyAfterPrefix) >= 41 {
                                blockNum := binary.BigEndian.Uint64(keyAfterPrefix[1:9])
                                hash := keyAfterPrefix[9:41]
                                fmt.Printf("  Block number: %d\n", blockNum)
                                fmt.Printf("  Block hash: %x\n", hash)
                            }
                            
                            fmt.Println()
                        }
                    }
                }
            }
        }
        
        if totalKeys%100000 == 0 {
            fmt.Printf("Scanned %d keys...\n", totalKeys)
        }
    }
    
    fmt.Println("\n=== Summary ===")
    fmt.Printf("Total keys scanned: %d\n", totalKeys)
    
    for name, count := range counts {
        fmt.Printf("%s: %d\n", name, count)
    }
    
    if len(counts) == 0 {
        fmt.Println("No block data found with subnet prefix!")
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}