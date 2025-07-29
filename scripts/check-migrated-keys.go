package main

import (
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("runtime/lux-96369-cchain-fixed/db/pebbledb", &pebble.Options{})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    fmt.Println("Checking keys in migrated database:")
    fmt.Println("====================================")
    
    // Count different key types
    counts := map[string]int{
        "evmh": 0,
        "evmH": 0,
        "evmb": 0,
        "evmr": 0,
        "evmn": 0,
        "other": 0,
    }
    
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    total := 0
    for iter.First(); iter.Valid() && total < 100; iter.Next() {
        key := iter.Key()
        keyStr := string(key)
        
        switch {
        case len(key) >= 4 && keyStr[:4] == "evmh":
            counts["evmh"]++
        case len(key) >= 4 && keyStr[:4] == "evmH":
            counts["evmH"]++
        case len(key) >= 4 && keyStr[:4] == "evmb":
            counts["evmb"]++
        case len(key) >= 4 && keyStr[:4] == "evmr":
            counts["evmr"]++
        case len(key) >= 4 && keyStr[:4] == "evmn":
            counts["evmn"]++
        default:
            counts["other"]++
            if total < 10 {
                fmt.Printf("Other key: %s (hex: %s)\n", keyStr[:min(len(keyStr), 20)], hex.EncodeToString(key[:min(len(key), 20)]))
            }
        }
        
        total++
    }
    
    fmt.Printf("\nKey type counts (first %d keys):\n", total)
    for typ, count := range counts {
        if count > 0 {
            fmt.Printf("%s: %d\n", typ, count)
        }
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}