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
    
    // Key prefixes to check
    prefixes := map[string]string{
        "LastBlock": "LastBlock",
        "LastHash": "LastHash",
        "LastHeader": "LastHeader",
        "HeadBlock": "48", // 'H' in hex
        "HeadHeader": "68", // 'h' in hex
        "Height": "Height",
        "lastAccepted": "lastAccepted",
    }
    
    fmt.Println("Checking head pointers in migrated database:")
    fmt.Println("==========================================")
    
    for name, prefix := range prefixes {
        var key []byte
        if len(prefix) == 2 {
            // Hex prefix
            hexBytes, _ := hex.DecodeString(prefix)
            key = hexBytes
        } else {
            // String key
            key = []byte(prefix)
        }
        
        value, closer, err := db.Get(key)
        if err == pebble.ErrNotFound {
            fmt.Printf("%s: NOT FOUND\n", name)
            continue
        } else if err != nil {
            fmt.Printf("%s: ERROR - %v\n", name, err)
            continue
        }
        
        fmt.Printf("%s: %s\n", name, hex.EncodeToString(value))
        closer.Close()
    }
    
    // Also check for any keys that start with these patterns
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    fmt.Println("\nFirst 10 keys in database:")
    count := 0
    for iter.First(); iter.Valid() && count < 10; iter.Next() {
        key := iter.Key()
        fmt.Printf("Key: %s (hex: %s)\n", string(key), hex.EncodeToString(key))
        count++
    }
}