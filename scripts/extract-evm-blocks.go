package main

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/core/types"
    "github.com/luxfi/geth/rlp"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: extract-evm-blocks <source-db> <dest-db>")
        os.Exit(1)
    }
    
    sourceDB := os.Args[1]
    destDB := os.Args[2]
    
    // Open source PebbleDB
    srcDB, err := pebble.Open(sourceDB, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open source database: %v", err)
    }
    defer srcDB.Close()
    
    // Create destination database
    dstDB, err := pebble.Open(destDB, &pebble.Options{})
    if err != nil {
        log.Fatalf("Failed to create destination database: %v", err)
    }
    defer dstDB.Close()
    
    fmt.Println("=== Extracting EVM Blocks ===")
    
    // The subnet prefix we found
    subnetPrefix := []byte{0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1}
    
    // Scan for blocks
    iter, _ := srcDB.NewIter(&pebble.IterOptions{
        LowerBound: subnetPrefix,
    })
    defer iter.Close()
    
    blockCount := 0
    highestBlock := uint64(0)
    
    keysScanned := 0
    for iter.First(); iter.Valid() && keysScanned < 100000; iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        keysScanned++
        if keysScanned % 10000 == 0 {
            fmt.Printf("Scanned %d keys...\n", keysScanned)
        }
        
        // Check if key starts with our prefix
        if !bytes.HasPrefix(key, subnetPrefix) {
            // Continue scanning instead of breaking
            continue
        }
        
        fmt.Printf("Found key with prefix! Key length: %d\n", len(key))
        
        // Extract block number from key
        if len(key) > len(subnetPrefix) + 8 {
            numBytes := key[len(subnetPrefix):len(subnetPrefix)+8]
            blockNum := binary.BigEndian.Uint64(numBytes)
            
            // Try to decode as block
            var block types.Block
            if err := rlp.DecodeBytes(val, &block); err == nil {
                fmt.Printf("Found block %d: hash=%s\n", blockNum, block.Hash().Hex())
                
                // Convert to C-Chain format
                // Block header key: h + num (8 bytes) + hash
                headerKey := append([]byte{0x68}, numBytes...)
                headerKey = append(headerKey, block.Hash().Bytes()...)
                
                // Encode header
                headerData, _ := rlp.EncodeToBytes(block.Header())
                if err := dstDB.Set(headerKey, headerData, pebble.Sync); err != nil {
                    log.Printf("Failed to write header: %v", err)
                }
                
                // Block body key: b + num (8 bytes) + hash
                bodyKey := append([]byte{0x62}, numBytes...)
                bodyKey = append(bodyKey, block.Hash().Bytes()...)
                
                // Create body
                body := &types.Body{
                    Transactions: block.Transactions(),
                    Uncles:       block.Uncles(),
                }
                bodyData, _ := rlp.EncodeToBytes(body)
                if err := dstDB.Set(bodyKey, bodyData, pebble.Sync); err != nil {
                    log.Printf("Failed to write body: %v", err)
                }
                
                // Canonical hash key: h + num (8 bytes) + n
                canonicalKey := append([]byte{0x68}, numBytes...)
                canonicalKey = append(canonicalKey, 0x6e)
                if err := dstDB.Set(canonicalKey, block.Hash().Bytes(), pebble.Sync); err != nil {
                    log.Printf("Failed to write canonical: %v", err)
                }
                
                blockCount++
                if blockNum > highestBlock {
                    highestBlock = blockNum
                }
            }
        }
    }
    
    // Write head block hash
    if highestBlock > 0 {
        // LastHeader key
        if val, closer, err := srcDB.Get(append(subnetPrefix, []byte("lastAccepted")...)); err == nil {
            dstDB.Set([]byte("LastHeader"), val, pebble.Sync)
            closer.Close()
        }
    }
    
    fmt.Printf("\n=== Extraction Complete ===\n")
    fmt.Printf("Blocks extracted: %d\n", blockCount)
    fmt.Printf("Highest block: %d\n", highestBlock)
    fmt.Printf("Destination: %s\n", destDB)
}