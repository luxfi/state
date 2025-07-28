package main

import (
    "encoding/hex"
    "fmt"
    "log"
    "os"

    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) != 2 {
        log.Fatalf("Usage: %s <path/to/db>", os.Args[0])
    }
    dbPath := os.Args[1]

    db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatalf("pebble.Open: %v", err)
    }
    defer db.Close()

    iter, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("NewIter: %v", err)
    }
    defer iter.Close()

    fmt.Println("First 20 keys in database:")
    count := 0
    for iter.First(); iter.Valid() && count < 20; iter.Next() {
        key := iter.Key()
        count++
        fmt.Printf("%d. %s (len=%d)", count, hex.EncodeToString(key[:min(len(key), 32)]), len(key))
        
        // Check what type of key this might be
        if len(key) >= 3 && string(key[:3]) == "evm" {
            fmt.Printf(" [evm-prefixed]")
        } else if len(key) > 0 {
            switch key[0] {
            case 0x68:
                fmt.Printf(" [header?]")
            case 0x62:
                fmt.Printf(" [body?]")
            case 0x72:
                fmt.Printf(" [receipt?]")
            case 0x6e:
                fmt.Printf(" [canonical?]")
            case 0x48:
                fmt.Printf(" [hash->num?]")
            }
        }
        fmt.Println()
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}