package main

import (
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: analyze-evm-prefixed <db-path>")
        os.Exit(1)
    }
    
    dbPath := os.Args[1]
    
    // Open database
    db, err := pebble.Open(dbPath, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    fmt.Printf("=== Analyzing evm-prefixed data in: %s ===\n\n", dbPath)
    
    // Create iterator
    iter, err := db.NewIter(&pebble.IterOptions{})
    if err != nil {
        log.Fatalf("Failed to create iterator: %v", err)
    }
    defer iter.Close()
    
    // Track patterns after "evm" prefix
    patterns := make(map[byte]int)
    examples := make(map[byte][]byte)
    
    count := 0
    for iter.First(); iter.Valid() && count < 100000; iter.Next() {
        key := iter.Key()
        count++
        
        // Check if it has "evm" prefix (65766d)
        if len(key) >= 3 && key[0] == 0x65 && key[1] == 0x76 && key[2] == 0x6d {
            if len(key) > 3 {
                nextByte := key[3]
                patterns[nextByte]++
                
                // Save example
                if patterns[nextByte] <= 3 {
                    if examples[nextByte] == nil {
                        examples[nextByte] = make([]byte, 0)
                    }
                    fmt.Printf("Found pattern evm+%02x: %s\n", nextByte, hex.EncodeToString(key[:min(64, len(key))]))
                }
            }
        }
        
        if count%10000 == 0 {
            fmt.Printf("Scanned %d keys...\n", count)
        }
    }
    
    fmt.Printf("\n=== Pattern Summary ===\n")
    fmt.Printf("Byte after 'evm'  Count   Possible meaning\n")
    fmt.Printf("----------------  ------  ----------------\n")
    
    for b, c := range patterns {
        meaning := identifyByte(b)
        fmt.Printf("0x%02x              %-6d  %s\n", b, c, meaning)
    }
    
    // Check for specific text patterns
    fmt.Println("\n=== Checking for text-based keys ===")
    checkTextPattern(db, []byte("evm/T/"), "Total Difficulty (evm/T/)")
    checkTextPattern(db, []byte("evmLastAccepted"), "Last Accepted")
    checkTextPattern(db, []byte("evmHeight"), "Height")
    checkTextPattern(db, []byte("evmvm/"), "VM prefix (evmvm/)")
}

func identifyByte(b byte) string {
    switch b {
    case 0x00:
        return "accounts/storage?"
    case 0x01:
        return "unknown-01"
    case 0x02:
        return "unknown-02"
    case 0x03:
        return "unknown-03"
    case 0x04:
        return "unknown-04"
    case 0x05:
        return "unknown-05"
    case 0x06:
        return "unknown-06"
    case 0x07:
        return "unknown-07"
    case 0x08:
        return "unknown-08"
    case 0x09:
        return "unknown-09"
    case 0x2f: // '/'
        return "slash - text key?"
    case 0x48: // 'H'
        return "Hash->Number mapping?"
    case 0x4c: // 'L'
        return "LastAccepted/LastBlock?"
    case 0x54: // 'T'
        return "Total difficulty?"
    case 0x62: // 'b'
        return "bodies?"
    case 0x68: // 'h'
        return "headers?"
    case 0x6c: // 'l'
        return "tx lookups?"
    case 0x6e: // 'n'
        return "number->hash?"
    case 0x72: // 'r'
        return "receipts?"
    case 0x73: // 's'
        return "state/storage?"
    case 0x76: // 'v'
        return "vm prefix?"
    default:
        return fmt.Sprintf("unknown-%02x", b)
    }
}

func checkTextPattern(db *pebble.DB, prefix []byte, name string) {
    val, closer, err := db.Get(prefix)
    if err == nil {
        fmt.Printf("\n%s: FOUND!\n", name)
        fmt.Printf("  Key: %s\n", hex.EncodeToString(prefix))
        fmt.Printf("  Value: %s\n", hex.EncodeToString(val[:min(64, len(val))]))
        closer.Close()
    } else {
        // Try iterator approach
        iter, err := db.NewIter(&pebble.IterOptions{
            LowerBound: prefix,
            UpperBound: append(prefix, 0xff),
        })
        if err == nil {
            defer iter.Close()
            if iter.First() {
                fmt.Printf("\n%s: FOUND (via iterator)!\n", name)
                fmt.Printf("  Key: %s\n", hex.EncodeToString(iter.Key()[:min(64, len(iter.Key()))]))
            }
        }
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}