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
    
    fmt.Println("Analyzing key patterns in database:")
    fmt.Println("==================================")
    
    // Expected namespace for chain 96369
    expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
    nsBytes, _ := hex.DecodeString(expectedNamespace)
    
    // Analyze key patterns for each type
    keyTypeExamples := make(map[byte][]string)
    
    iter, _ := db.NewIter(nil)
    defer iter.Close()
    
    totalKeys := 0
    for iter.First(); iter.Valid() && totalKeys < 100000; iter.Next() {
        key := iter.Key()
        value := iter.Value()
        totalKeys++
        
        if len(key) >= 33 {
            // Check namespace
            hasNamespace := true
            for i := 0; i < 32; i++ {
                if key[i] != nsBytes[i] {
                    hasNamespace = false
                    break
                }
            }
            
            if hasNamespace {
                keyType := key[32]
                actualKey := key[33:]
                
                // Store up to 5 examples per key type
                if len(keyTypeExamples[keyType]) < 5 {
                    example := fmt.Sprintf("Key: %s (len=%d), Value: %s (len=%d)", 
                        hex.EncodeToString(actualKey[:min(len(actualKey), 20)]), 
                        len(actualKey),
                        hex.EncodeToString(value[:min(len(value), 20)]),
                        len(value))
                    keyTypeExamples[keyType] = append(keyTypeExamples[keyType], example)
                }
            }
        }
    }
    
    fmt.Printf("Analyzed %d keys\n\n", totalKeys)
    
    // Print examples for each key type
    for keyType := byte(0); keyType <= 0x09; keyType++ {
        if examples, ok := keyTypeExamples[keyType]; ok {
            fmt.Printf("Key type 0x%02x:\n", keyType)
            for _, ex := range examples {
                fmt.Printf("  %s\n", ex)
            }
            fmt.Println()
        }
    }
    
    // Look for patterns that might be block numbers
    fmt.Println("Looking for potential block number patterns:")
    
    // Check if 0x00 keys might contain block data
    iter2, _ := db.NewIter(nil)
    defer iter2.Close()
    
    count := 0
    for iter2.First(); iter2.Valid() && count < 20; iter2.Next() {
        key := iter2.Key()
        value := iter2.Value()
        
        if len(key) >= 33 && key[32] == 0x00 {
            actualKey := key[33:]
            
            // Check if key starts with 8 bytes that could be a block number
            if len(actualKey) >= 8 {
                possibleBlockNum := binary.BigEndian.Uint64(actualKey[:8])
                if possibleBlockNum > 0 && possibleBlockNum < 2000000 { // Reasonable block range
                    fmt.Printf("Possible block at 0x00: num=%d (0x%x), key=%s, value_len=%d\n",
                        possibleBlockNum, possibleBlockNum,
                        hex.EncodeToString(actualKey[:min(len(actualKey), 16)]),
                        len(value))
                    count++
                }
            }
        }
    }
    
    // Check for the specific block 1082781 (0x10859d)
    fmt.Printf("\nLooking for block 1082781 (0x10859d):\n")
    targetBlock := make([]byte, 8)
    binary.BigEndian.PutUint64(targetBlock, 1082781)
    
    for keyType := byte(0); keyType <= 0x09; keyType++ {
        // Try to find this block number with each key type
        testKey := append(nsBytes, keyType)
        testKey = append(testKey, targetBlock...)
        
        // Try with just block number
        if value, closer, err := db.Get(testKey); err == nil {
            fmt.Printf("Found with type 0x%02x: value_len=%d\n", keyType, len(value))
            closer.Close()
        }
        
        // Try with block number + hash (like headers)
        iter3, _ := db.NewIter(&pebble.IterOptions{
            LowerBound: testKey,
            UpperBound: append(testKey, 0xff),
        })
        
        if iter3.First() && iter3.Valid() {
            key := iter3.Key()
            if len(key) > len(testKey) && string(key[:len(testKey)]) == string(testKey) {
                fmt.Printf("Found with prefix type 0x%02x: full_key_len=%d, value_len=%d\n", 
                    keyType, len(key), len(iter3.Value()))
            }
        }
        iter3.Close()
    }
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}