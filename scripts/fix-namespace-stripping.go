package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) != 3 {
        fmt.Println("Usage: fix-namespace-stripping <source-db> <dest-db>")
        os.Exit(1)
    }
    
    srcPath := os.Args[1]
    dstPath := os.Args[2]
    
    fmt.Printf("Properly stripping namespace from %s to %s\n", srcPath, dstPath)
    
    // Open source database
    srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatalf("Failed to open source DB: %v", err)
    }
    defer srcDB.Close()
    
    // Create destination database
    os.MkdirAll(dstPath, 0755)
    dstDB, err := pebble.Open(dstPath, &pebble.Options{})
    if err != nil {
        log.Fatalf("Failed to create destination DB: %v", err)
    }
    defer dstDB.Close()
    
    // Expected namespace
    expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
    nsBytes, _ := hex.DecodeString(expectedNamespace)
    
    iter, err := srcDB.NewIter(nil)
    if err != nil {
        log.Fatalf("Failed to create iterator: %v", err)
    }
    defer iter.Close()
    
    totalKeys := 0
    strippedKeys := 0
    batch := dstDB.NewBatch()
    batchSize := 0
    
    // Track blockchain data
    headers := 0
    bodies := 0
    receipts := 0
    canonical := 0
    hashToNum := 0
    maxBlock := uint64(0)
    
    for iter.First(); iter.Valid(); iter.Next() {
        totalKeys++
        key := iter.Key()
        value := iter.Value()
        
        // Check if key has namespace prefix
        if len(key) >= 33 {
            // Check if first 32 bytes match expected namespace
            hasNamespace := true
            for i := 0; i < 32; i++ {
                if key[i] != nsBytes[i] {
                    hasNamespace = false
                    break
                }
            }
            
            if hasNamespace {
                // Strip the 33-byte prefix (32-byte namespace + 1-byte key type)
                keyType := key[32]
                actualKey := key[33:]
                
                // Map key types to EVM prefixes
                var newKey []byte
                switch keyType {
                case 0x68: // 'h' - headers
                    newKey = append([]byte("evmh"), actualKey...)
                    headers++
                    if len(actualKey) >= 8 {
                        blockNum := binary.BigEndian.Uint64(actualKey[:8])
                        if blockNum > maxBlock && blockNum < 10000000 {
                            maxBlock = blockNum
                        }
                    }
                case 0x62: // 'b' - bodies
                    newKey = append([]byte("evmb"), actualKey...)
                    bodies++
                case 0x72: // 'r' - receipts
                    newKey = append([]byte("evmr"), actualKey...)
                    receipts++
                case 0x6e: // 'n' - canonical (number->hash)
                    // For canonical, the key after namespace might not start with block number
                    // Let's check the structure
                    if len(actualKey) == 8 {
                        // This is just a block number
                        newKey = append([]byte("evmn"), actualKey...)
                        canonical++
                        blockNum := binary.BigEndian.Uint64(actualKey)
                        if blockNum > maxBlock && blockNum < 10000000 {
                            maxBlock = blockNum
                        }
                    } else {
                        // Skip malformed canonical keys
                        continue
                    }
                case 0x48: // 'H' - hash->number
                    newKey = append([]byte("evmH"), actualKey...)
                    hashToNum++
                case 0x74: // 't' - transactions
                    newKey = append([]byte("evmt"), actualKey...)
                case 0x26: // account state
                    newKey = actualKey // No prefix for accounts
                case 0x73: // 's' - state
                    newKey = actualKey // No prefix for state
                default:
                    // For consensus keys (0x00), check if it's a special key
                    if keyType == 0x00 {
                        keyStr := string(actualKey)
                        if keyStr == "Height" || keyStr == "LastBlock" || keyStr == "lastAccepted" {
                            newKey = actualKey // Keep consensus keys without prefix
                        } else {
                            // Skip other 0x00 keys (they're state data)
                            continue
                        }
                    } else {
                        // Skip unknown key types
                        continue
                    }
                }
                
                batch.Set(newKey, value, nil)
                strippedKeys++
            } else {
                // Not our namespace, copy as-is
                batch.Set(key, value, nil)
            }
        } else {
            // Key too short to have namespace, copy as-is
            batch.Set(key, value, nil)
        }
        
        batchSize++
        if batchSize >= 1000 {
            if err := batch.Commit(nil); err != nil {
                log.Fatalf("Failed to commit batch: %v", err)
            }
            batch = dstDB.NewBatch()
            batchSize = 0
            
            if totalKeys%100000 == 0 {
                fmt.Printf("Progress: %d keys, %d stripped (h=%d, b=%d, r=%d, n=%d, H=%d)\n", 
                    totalKeys, strippedKeys, headers, bodies, receipts, canonical, hashToNum)
            }
        }
    }
    
    // Commit final batch
    if batchSize > 0 {
        if err := batch.Commit(nil); err != nil {
            log.Fatalf("Failed to commit final batch: %v", err)
        }
    }
    
    // Add consensus keys if we found blockchain data
    if headers > 0 && maxBlock > 0 {
        fmt.Printf("\nAdding consensus keys for block %d...\n", maxBlock)
        
        // Height
        heightBytes := make([]byte, 8)
        binary.BigEndian.PutUint64(heightBytes, maxBlock)
        if err := dstDB.Set([]byte("Height"), heightBytes, nil); err != nil {
            fmt.Printf("Failed to set Height: %v\n", err)
        }
        
        // LastBlock (dummy for now)
        if err := dstDB.Set([]byte("LastBlock"), []byte("dummy-block-id"), nil); err != nil {
            fmt.Printf("Failed to set LastBlock: %v\n", err)
        }
        
        // lastAccepted
        if err := dstDB.Set([]byte("lastAccepted"), []byte("dummy-block-id"), nil); err != nil {
            fmt.Printf("Failed to set lastAccepted: %v\n", err)
        }
    }
    
    fmt.Printf("\nCompleted: %d total keys, %d stripped\n", totalKeys, strippedKeys)
    fmt.Printf("Blockchain data: headers=%d, bodies=%d, receipts=%d, canonical=%d, H=%d\n",
        headers, bodies, receipts, canonical, hashToNum)
    fmt.Printf("Max block found: %d\n", maxBlock)
}