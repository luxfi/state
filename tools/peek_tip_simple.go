package main

import (
    "encoding/binary"
    "flag"
    "fmt"
    "log"

    "github.com/cockroachdb/pebble"
)

func main() {
    dbPath := flag.String("db", "", "path to PebbleDB with evm namespace")
    flag.Parse()
    if *dbPath == "" {
        log.Fatal("--db is required")
    }

    db, err := pebble.Open(*dbPath, &pebble.Options{})
    if err != nil {
        log.Fatalf("open: %v", err)
    }
    defer db.Close()

    it, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("iterator: %v", err)
    }
    defer it.Close()

    // The "evm" namespace is 3 bytes: 0x65, 0x76, 0x6d
    // After that comes the actual key type
    // evmn key format: [evm namespace (3)] + ['n' (1)] + [block number (8)]
    evmPrefix := []byte{0x65, 0x76, 0x6d} // "evm"
    evmnPrefix := append(evmPrefix, 'n')   // "evmn"

    var tip uint64
    var count int

    // Seek to the evmn prefix
    for it.SeekGE(evmnPrefix); it.Valid(); it.Next() {
        key := it.Key()
        
        // Check if key starts with evmn prefix
        if len(key) >= len(evmnPrefix) && string(key[:len(evmnPrefix)]) == string(evmnPrefix) {
            // For evmn keys, the format is: evmn<8-byte-number>
            if len(key) == len(evmnPrefix)+8 {
                blockNum := binary.BigEndian.Uint64(key[len(evmnPrefix):])
                if blockNum > tip {
                    tip = blockNum
                }
                count++
            }
        } else if len(key) < len(evmnPrefix) || string(key[:len(evmPrefix)]) != string(evmPrefix) {
            // We've moved past all evm keys
            break
        }
    }

    if count == 0 {
        log.Fatal("no evmn keys found")
    }

    fmt.Printf("Found %d evmn keys\n", count)
    fmt.Printf("Maximum block number: %d (0x%x)\n", tip, tip)
}