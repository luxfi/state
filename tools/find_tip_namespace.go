package main

import (
    "encoding/binary"
    "encoding/hex"
    "flag"
    "fmt"
    "log"

    "github.com/cockroachdb/pebble"
)

func main() {
    dbPath := flag.String("db", "", "path to PebbleDB with evm namespace")
    flag.Parse()
    if *dbPath == "" {
        log.Fatal("--db is required")
    }

    db, err := pebble.Open(*dbPath, &pebble.Options{})
    if err != nil {
        log.Fatalf("open: %v", err)
    }
    defer db.Close()

    it, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("iterator: %v", err)
    }
    defer it.Close()

    // evmn prefix in hex is 65766d6e
    evmnPrefix := []byte{0x65, 0x76, 0x6d, 0x6e} // "evmn"
    
    var tip uint64
    var count int
    var sampleKeys []string

    // Seek to evmn keys
    for it.SeekGE(evmnPrefix); it.Valid(); it.Next() {
        key := it.Key()
        
        // Check if key starts with evmn
        if len(key) >= 4 && string(key[:4]) == string(evmnPrefix) {
            // After evmn prefix, we should have the block number
            if len(key) == 12 { // evmn(4) + number(8)
                blockNum := binary.BigEndian.Uint64(key[4:12])
                if blockNum > tip {
                    tip = blockNum
                }
                count++
                
                // Save first few keys as examples
                if len(sampleKeys) < 5 {
                    sampleKeys = append(sampleKeys, fmt.Sprintf("Key: %s, Block: %d (0x%x)", 
                        hex.EncodeToString(key), blockNum, blockNum))
                }
            } else if len(key) > 4 {
                // Check if this might be hash format instead
                if count < 5 {
                    fmt.Printf("Non-standard evmn key length %d: %s\n", 
                        len(key), hex.EncodeToString(key[:min(40, len(key))]))
                }
            }
        } else {
            // We've moved past evmn keys
            break
        }
    }

    if count == 0 {
        fmt.Println("No standard format evmn keys found")
        fmt.Println("Checking for non-standard format evmn keys...")
        
        // Try again looking for any evmn prefix
        it.SeekGE(evmnPrefix)
        nonStandardCount := 0
        for it.Valid() && nonStandardCount < 10 {
            key := it.Key()
            if len(key) >= 4 && string(key[:4]) == string(evmnPrefix) {
                fmt.Printf("Found evmn key (length %d): %s\n", 
                    len(key), hex.EncodeToString(key[:min(40, len(key))]))
                nonStandardCount++
            }
            it.Next()
        }
    } else {
        fmt.Printf("Found %d evmn keys in standard format\n", count)
        fmt.Printf("Maximum block number: %d (0x%x)\n", tip, tip)
        
        fmt.Println("\nSample keys:")
        for _, sample := range sampleKeys {
            fmt.Println(sample)
        }
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}