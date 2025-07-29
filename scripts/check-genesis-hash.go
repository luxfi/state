package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("runtime/lux-96369-vm-ready/evm", &pebble.Options{})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    fmt.Println("Checking genesis block data:")
    fmt.Println("===========================")
    
    // Check canonical hash at block 0
    blockBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(blockBytes, 0)
    
    canonicalKey := []byte{0x68}
    canonicalKey = append(canonicalKey, blockBytes...)
    canonicalKey = append(canonicalKey, 0x6e)
    
    fmt.Printf("Canonical key for block 0: %x\n", canonicalKey)
    
    if val, closer, err := db.Get(canonicalKey); err == nil {
        fmt.Printf("Canonical hash at block 0: %x\n", val)
        closer.Close()
        
        // Now check if we have a header for this hash
        headerKey := []byte{0x68}
        headerKey = append(headerKey, blockBytes...)
        headerKey = append(headerKey, val...)
        
        fmt.Printf("\nLooking for header at key: %x\n", headerKey)
        if headerVal, closer2, err2 := db.Get(headerKey); err2 == nil {
            fmt.Printf("Found header, length: %d bytes\n", len(headerVal))
            closer2.Close()
        } else {
            fmt.Printf("Header not found: %v\n", err2)
        }
    }
    
    // Check what the VM expects
    expectedGenesis, _ := hex.DecodeString("a24e71001a6a59fb52834b2b4e905f08d1598a7da819467ebb8d9da4129f37ce")
    fmt.Printf("\nVM expects genesis hash: %x\n", expectedGenesis)
    
    // Look for header with VM's expected hash
    vmHeaderKey := []byte{0x68}
    vmHeaderKey = append(vmHeaderKey, blockBytes...)
    vmHeaderKey = append(vmHeaderKey, expectedGenesis...)
    
    if _, closer, err := db.Get(vmHeaderKey); err == nil {
        fmt.Printf("Found header for VM's expected hash!\n")
        closer.Close()
    } else {
        fmt.Printf("No header found for VM's expected hash\n")
    }
}