package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    db, err := pebble.Open("chaindata/lux-mainnet-96369/db/pebbledb", &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    fmt.Println("Examining blockchain keys:")
    fmt.Println("=========================")
    
    // Expected namespace
    expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
    nsBytes, _ := hex.DecodeString(expectedNamespace)
    
    // Check blockchain key types
    blockchainTypes := map[byte]string{
        0x42: "unknown (0x42)",
        0x48: "hash->number (H)",
        0x62: "bodies (b)",
        0x68: "headers (h)",
        0x6c: "unknown (0x6c)",
        0x6e: "canonical (n)",
        0x72: "receipts (r)",
    }
    
    for keyType, name := range blockchainTypes {
        fmt.Printf("\n%s:\n", name)
        
        prefix := append(nsBytes, keyType)
        iter, _ := db.NewIter(&pebble.IterOptions{
            LowerBound: prefix,
            UpperBound: append(prefix, 0xff),
        })
        
        count := 0
        maxBlockNum := uint64(0)
        
        for iter.First(); iter.Valid() && count < 10; iter.Next() {
            key := iter.Key()
            value := iter.Value()
            actualKey := key[33:]
            
            // If key starts with 8 bytes, it might be a block number
            if len(actualKey) >= 8 {
                blockNum := binary.BigEndian.Uint64(actualKey[:8])
                if blockNum > maxBlockNum {
                    maxBlockNum = blockNum
                }
                
                if count < 5 {
                    fmt.Printf("  Block %d (0x%x): key_len=%d, value_len=%d\n", 
                        blockNum, blockNum, len(actualKey), len(value))
                    
                    // Show the rest of the key (probably hash)
                    if len(actualKey) > 8 {
                        fmt.Printf("    Rest of key: %s\n", hex.EncodeToString(actualKey[8:]))
                    }
                }
            } else {
                if count < 5 {
                    fmt.Printf("  Key: %s (len=%d), value_len=%d\n", 
                        hex.EncodeToString(actualKey), len(actualKey), len(value))
                }
            }
            count++
        }
        
        if count > 0 {
            fmt.Printf("  Total entries: %d+\n", count)
            if maxBlockNum > 0 {
                fmt.Printf("  Max block number seen: %d (0x%x)\n", maxBlockNum, maxBlockNum)
            }
        }
        
        iter.Close()
    }
    
    // Look specifically for block 1082781
    fmt.Printf("\nLooking for block 1082781 (0x10859d):\n")
    targetBlock := make([]byte, 8)
    binary.BigEndian.PutUint64(targetBlock, 1082781)
    
    for keyType, name := range blockchainTypes {
        prefix := append(nsBytes, keyType)
        prefix = append(prefix, targetBlock...)
        
        // Try to find keys starting with this block number
        iter, _ := db.NewIter(&pebble.IterOptions{
            LowerBound: prefix,
            UpperBound: append(prefix, 0xff, 0xff, 0xff, 0xff),
        })
        
        if iter.First() && iter.Valid() {
            key := iter.Key()
            value := iter.Value()
            fmt.Printf("Found in %s: key_len=%d, value_len=%d\n", name, len(key), len(value))
            
            // Show the hash part if present
            actualKey := key[33:]
            if len(actualKey) > 8 {
                fmt.Printf("  Hash: %s\n", hex.EncodeToString(actualKey[8:]))
            }
        }
        
        iter.Close()
    }
    
    // Check canonical mappings (type 0x6e)
    fmt.Println("\nChecking canonical mappings (number->hash):")
    prefix := append(nsBytes, 0x6e)
    
    iter, _ := db.NewIter(&pebble.IterOptions{
        LowerBound: prefix,
        UpperBound: append(prefix, 0xff),
    })
    defer iter.Close()
    
    highestBlock := uint64(0)
    count := 0
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        value := iter.Value()
        actualKey := key[33:]
        
        if len(actualKey) >= 8 {
            blockNum := binary.BigEndian.Uint64(actualKey[:8])
            if blockNum > highestBlock {
                highestBlock = blockNum
            }
            
            if count < 5 || blockNum > 1082700 {
                fmt.Printf("  Block %d: hash=%s\n", blockNum, hex.EncodeToString(value[:min(len(value), 32)]))
            }
        }
        count++
    }
    
    fmt.Printf("\nHighest canonical block: %d (0x%x)\n", highestBlock, highestBlock)
    fmt.Printf("Total canonical entries: %d\n", count)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}