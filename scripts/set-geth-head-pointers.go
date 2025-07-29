package main

import (
    "encoding/binary"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: set-geth-head-pointers <evm-db-path>")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open database
    db, err := pebble.Open(dbPath, &pebble.Options{})
    if err != nil {
        log.Fatal("Failed to open DB:", err)
    }
    defer db.Close()
    
    // Get the canonical hash for the highest block
    highestBlock := uint64(1082780)
    blockBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(blockBytes, highestBlock)
    
    canonicalKey := []byte{0x68}
    canonicalKey = append(canonicalKey, blockBytes...)
    canonicalKey = append(canonicalKey, 0x6e)
    
    var headHash []byte
    if val, closer, err := db.Get(canonicalKey); err == nil {
        headHash = make([]byte, len(val))
        copy(headHash, val)
        closer.Close()
        fmt.Printf("Head block hash: %x\n", headHash)
    } else {
        log.Fatal("Could not find canonical hash for highest block")
    }
    
    // Set standard geth head pointers
    // These are the keys that geth uses to track the head of the chain
    
    // LastHeader - points to the latest known header
    if err := db.Set([]byte("LastHeader"), headHash, nil); err != nil {
        fmt.Printf("Failed to set LastHeader: %v\n", err)
    } else {
        fmt.Println("Set LastHeader")
    }
    
    // LastBlock - points to the latest known full block
    if err := db.Set([]byte("LastBlock"), headHash, nil); err != nil {
        fmt.Printf("Failed to set LastBlock: %v\n", err)
    } else {
        fmt.Println("Set LastBlock")
    }
    
    // LastFast - points to the latest fast-sync'd block
    if err := db.Set([]byte("LastFast"), headHash, nil); err != nil {
        fmt.Printf("Failed to set LastFast: %v\n", err)
    } else {
        fmt.Println("Set LastFast")
    }
    
    // HeadHeaderHash
    if err := db.Set([]byte("LastHeader"), headHash, nil); err != nil {
        fmt.Printf("Failed to set HeadHeaderHash: %v\n", err)
    } else {
        fmt.Println("Set HeadHeaderHash")
    }
    
    // HeadBlockHash
    if err := db.Set([]byte("LastBlock"), headHash, nil); err != nil {
        fmt.Printf("Failed to set HeadBlockHash: %v\n", err)
    } else {
        fmt.Println("Set HeadBlockHash")
    }
    
    // HeadFastBlockHash
    if err := db.Set([]byte("LastFast"), headHash, nil); err != nil {
        fmt.Printf("Failed to set HeadFastBlockHash: %v\n", err)
    } else {
        fmt.Println("Set HeadFastBlockHash")
    }
    
    fmt.Println("\nGeth head pointers set!")
}