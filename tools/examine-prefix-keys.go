package main

import (
    "bytes"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: examine-prefix-keys <db-path>")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open PebbleDB
    db, err := pebble.Open(dbPath, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    fmt.Println("=== Examining Keys with Subnet Prefix ===")
    
    // The subnet prefix we found
    subnetPrefix := []byte{0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1}
    
    // Scan for keys with this prefix
    iter, _ := db.NewIter(&pebble.IterOptions{
        LowerBound: subnetPrefix,
    })
    defer iter.Close()
    
    examined := 0
    for iter.First(); iter.Valid() && examined < 10; iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        // Check if key starts with our prefix
        if !bytes.HasPrefix(key, subnetPrefix) {
            break
        }
        
        fmt.Printf("\n=== Key %d ===\n", examined+1)
        fmt.Printf("Full key: %s\n", hex.EncodeToString(key))
        fmt.Printf("Key length: %d\n", len(key))
        
        // Extract parts after prefix
        if len(key) > len(subnetPrefix) {
            suffix := key[len(subnetPrefix):]
            fmt.Printf("Suffix: %s\n", hex.EncodeToString(suffix))
            fmt.Printf("Suffix length: %d\n", len(suffix))
        }
        
        fmt.Printf("Value length: %d\n", len(val))
        if len(val) <= 64 {
            fmt.Printf("Value: %s\n", hex.EncodeToString(val))
        } else {
            fmt.Printf("Value (first 64 bytes): %s...\n", hex.EncodeToString(val[:64]))
        }
        
        // Check if value looks like RLP
        if len(val) > 0 {
            firstByte := val[0]
            fmt.Printf("First byte: 0x%02x\n", firstByte)
            
            // RLP encoding checks
            if firstByte >= 0xf7 && firstByte <= 0xff {
                fmt.Println("Looks like RLP list with long length")
            } else if firstByte >= 0xc0 && firstByte <= 0xf6 {
                fmt.Println("Looks like RLP list")
            } else if firstByte >= 0xb8 && firstByte <= 0xbf {
                fmt.Println("Looks like RLP string with long length")
            } else if firstByte >= 0x80 && firstByte <= 0xb7 {
                fmt.Println("Looks like RLP string")
            }
        }
        
        examined++
    }
    
    fmt.Printf("\n=== Examined %d keys ===\n", examined)
}