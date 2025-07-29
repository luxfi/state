package main

import (
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("runtime/lux-96369-cchain/db/pebbledb", &pebble.Options{})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    fmt.Println("Checking key structure in migrated database:")
    fmt.Println("==========================================")
    
    count := 0
    namespacedCount := 0
    
    for iter.First(); iter.Valid() && count < 100; iter.Next() {
        key := iter.Key()
        keyLen := len(key)
        
        // Check if this looks like a namespaced key (33+ bytes)
        if keyLen >= 33 {
            namespacedCount++
            fmt.Printf("NAMESPACED KEY (len=%d): %s\n", keyLen, hex.EncodeToString(key[:33]))
        } else {
            fmt.Printf("Regular key (len=%d): %s (%s)\n", keyLen, hex.EncodeToString(key), string(key))
        }
        
        count++
    }
    
    fmt.Printf("\nTotal keys checked: %d\n", count)
    fmt.Printf("Namespaced keys: %d\n", namespacedCount)
    
    // Check specific important keys
    fmt.Println("\nChecking for specific keys:")
    
    // Try to get the canonical hash key for block 1082781
    blockNumKey := append([]byte("H"), encodeBlockNumber(1082781)...)
    value, closer, err := db.Get(blockNumKey)
    if err == nil {
        fmt.Printf("Found block hash for 1082781: %s\n", hex.EncodeToString(value))
        closer.Close()
    } else {
        fmt.Printf("Block hash for 1082781 not found (key: %s)\n", hex.EncodeToString(blockNumKey))
    }
}

func encodeBlockNumber(n uint64) []byte {
    enc := make([]byte, 8)
    for i := 7; i >= 0; i-- {
        enc[i] = byte(n)
        n >>= 8
    }
    return enc
}