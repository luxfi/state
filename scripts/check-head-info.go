package main

import (
    "encoding/binary"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    // Check EVM database
    fmt.Println("Checking EVM database:")
    fmt.Println("=====================")
    
    evmDB, err := pebble.Open("runtime/lux-96369-vm-ready/evm", &pebble.Options{})
    if err != nil {
        log.Fatal("Failed to open EVM DB:", err)
    }
    defer evmDB.Close()
    
    // Check Height key
    if val, closer, err := evmDB.Get([]byte("Height")); err == nil {
        height := binary.BigEndian.Uint64(val)
        fmt.Printf("Height: %d (0x%x)\n", height, height)
        closer.Close()
    }
    
    // Check LastBlock
    if val, closer, err := evmDB.Get([]byte("LastBlock")); err == nil {
        fmt.Printf("LastBlock: %s\n", string(val))
        closer.Close()
    }
    
    // Check lastAccepted
    if val, closer, err := evmDB.Get([]byte("lastAccepted")); err == nil {
        fmt.Printf("lastAccepted: %s\n", string(val))
        closer.Close()
    }
    
    // Check for head hash key
    headHashKey := []byte("LastBlockHash")
    if val, closer, err := evmDB.Get(headHashKey); err == nil {
        fmt.Printf("LastBlockHash: %x\n", val)
        closer.Close()
    } else {
        fmt.Printf("LastBlockHash: not found\n")
    }
    
    // Check for the head block hash in geth format
    fmt.Println("\nChecking for head block in geth format:")
    
    // Try to find the head block hash
    headKey := append([]byte("h"), []byte("n")...)
    if val, closer, err := evmDB.Get(headKey); err == nil {
        fmt.Printf("Head block hash (hn): %x\n", val)
        closer.Close()
    }
    
    // Check if we can find the highest block's canonical mapping
    highestBlock := uint64(1082780)
    blockBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(blockBytes, highestBlock)
    
    canonicalKey := []byte{0x68}
    canonicalKey = append(canonicalKey, blockBytes...)
    canonicalKey = append(canonicalKey, 0x6e)
    
    fmt.Printf("\nChecking canonical hash at block %d:\n", highestBlock)
    fmt.Printf("Key: %x\n", canonicalKey)
    
    if val, closer, err := evmDB.Get(canonicalKey); err == nil {
        fmt.Printf("Canonical hash: %x\n", val)
        closer.Close()
        
        // Check if we have the header for this block
        headerKey := []byte{0x68}
        headerKey = append(headerKey, blockBytes...)
        headerKey = append(headerKey, val...)
        
        if _, closer2, err2 := evmDB.Get(headerKey); err2 == nil {
            fmt.Printf("Header exists for block %d\n", highestBlock)
            closer2.Close()
        } else {
            fmt.Printf("Header NOT found for block %d\n", highestBlock)
        }
    } else {
        fmt.Printf("No canonical mapping for block %d\n", highestBlock)
    }
    
    // Check state database
    fmt.Println("\n\nChecking state database:")
    fmt.Println("=======================")
    
    stateDB, err := pebble.Open("runtime/lux-96369-fixed/state", &pebble.Options{})
    if err != nil {
        log.Fatal("Failed to open state DB:", err)
    }
    defer stateDB.Close()
    
    // List first few keys
    iter, _ := stateDB.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    count := 0
    for iter.First(); iter.Valid() && count < 10; iter.Next() {
        key := iter.Key()
        fmt.Printf("State key: %s (hex: %x)\n", string(key), key)
        count++
    }
}