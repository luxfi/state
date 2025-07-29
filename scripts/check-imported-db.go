package main

import (
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("runtime/lux-96369-imported/db/pebbledb", &pebble.Options{})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    fmt.Println("Checking keys in imported database:")
    fmt.Println("====================================")
    
    // Count different key types
    evmKeys := 0
    otherKeys := 0
    
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    for iter.First(); iter.Valid() && evmKeys+otherKeys < 100; iter.Next() {
        key := iter.Key()
        keyStr := string(key)
        
        if len(key) >= 3 && keyStr[:3] == "evm" {
            evmKeys++
            if evmKeys <= 5 {
                fmt.Printf("EVM key: %s\n", hex.EncodeToString(key[:min(len(key), 20)]))
            }
        } else {
            otherKeys++
            if otherKeys <= 5 {
                fmt.Printf("Other key: %s (hex: %s)\n", keyStr[:min(len(keyStr), 20)], hex.EncodeToString(key[:min(len(key), 20)]))
            }
        }
    }
    
    fmt.Printf("\nKey counts (first 100): evm=%d, other=%d\n", evmKeys, otherKeys)
    
    // Check specific keys
    fmt.Println("\nChecking specific keys:")
    keys := []string{"Height", "LastBlock", "lastAccepted", "LastAccepted"}
    for _, k := range keys {
        value, closer, err := db.Get([]byte(k))
        if err == nil {
            fmt.Printf("%s: %s\n", k, hex.EncodeToString(value))
            closer.Close()
        } else {
            fmt.Printf("%s: not found\n", k)
        }
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}