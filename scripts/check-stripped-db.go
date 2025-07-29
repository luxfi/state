package main

import (
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("runtime/lux-96369-stripped/evm", &pebble.Options{})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    fmt.Println("Checking keys in stripped database:")
    fmt.Println("==================================")
    
    // Count different key types
    evmKeys := 0
    namespacedKeys := 0
    plainKeys := 0
    
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    for iter.First(); iter.Valid() && evmKeys+namespacedKeys+plainKeys < 100; iter.Next() {
        key := iter.Key()
        keyHex := hex.EncodeToString(key)
        
        if len(key) >= 3 && string(key[:3]) == "evm" {
            evmKeys++
            if evmKeys <= 5 {
                fmt.Printf("EVM key: %s (len=%d)\n", keyHex[:min(len(keyHex), 40)], len(key))
            }
        } else if len(key) >= 33 && keyHex[:2] == "33" {
            namespacedKeys++
            if namespacedKeys <= 5 {
                fmt.Printf("NAMESPACED key: %s (len=%d)\n", keyHex[:min(len(keyHex), 40)], len(key))
            }
        } else {
            plainKeys++
            if plainKeys <= 5 {
                fmt.Printf("Plain key: %s (len=%d)\n", keyHex[:min(len(keyHex), 40)], len(key))
            }
        }
    }
    
    fmt.Printf("\nKey counts (first 100): evm=%d, namespaced=%d, plain=%d\n", evmKeys, namespacedKeys, plainKeys)
    
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
    
    // Check for block at expected height
    blockKey := []byte{0x48, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x85, 0x9d} // hash->number at 1082781
    value, closer, err := db.Get(append([]byte("evmH"), blockKey[1:]...))
    if err == nil {
        fmt.Printf("Block hash for 1082781: %s\n", hex.EncodeToString(value))
        closer.Close()
    } else {
        fmt.Printf("Block hash for 1082781 not found (checking evmH key)\n")
    }
    
    // Check Height key
    heightValue, closer, err := db.Get([]byte("Height"))
    if err == nil {
        fmt.Printf("\nHeight key found: %s\n", hex.EncodeToString(heightValue))
        closer.Close()
    }
    
    // Look for any evmH keys
    fmt.Println("\nLooking for evmH keys:")
    evmHIter, _ := db.NewIter(&pebble.IterOptions{
        LowerBound: []byte("evmH"),
        UpperBound: []byte("evmI"),
    })
    defer evmHIter.Close()
    
    count := 0
    for evmHIter.First(); evmHIter.Valid() && count < 5; evmHIter.Next() {
        key := evmHIter.Key()
        fmt.Printf("evmH key: %s (len=%d)\n", hex.EncodeToString(key[:min(len(key), 20)]), len(key))
        count++
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}