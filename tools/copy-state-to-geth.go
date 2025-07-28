package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: copy-state-to-geth <source-db> <target-db>")
        fmt.Println("This copies state data from subnet database to geth database")
        os.Exit(1)
    }
    
    sourceDB := os.Args[1]
    targetDB := os.Args[2]
    
    // Open source database (read-only)
    src, err := pebble.Open(sourceDB, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open source database: %v", err)
    }
    defer src.Close()
    
    // Open target database
    dst, err := pebble.Open(targetDB, &pebble.Options{})
    if err != nil {
        log.Fatalf("Failed to open target database: %v", err)
    }
    defer dst.Close()
    
    fmt.Printf("=== Copying state from %s to %s ===\n", sourceDB, targetDB)
    
    // Create iterator
    iter, _ := src.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    batch := dst.NewBatch()
    count := 0
    stateCount := 0
    storageCount := 0
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        // Copy all keys - the subnet data is already in the right format
        // We just need to copy the state data
        if err := batch.Set(key, val, nil); err != nil {
            log.Printf("Error setting key: %v", err)
            continue
        }
        
        // Track what we're copying
        if len(key) >= 10 {
            switch key[9] {
            case 0x01: // Account trie
                stateCount++
            case 0xa3: // Storage trie
                storageCount++
            }
        }
        
        count++
        
        // Commit batch every 10000 keys
        if count%10000 == 0 {
            if err := batch.Commit(nil); err != nil {
                log.Fatalf("Failed to commit batch: %v", err)
            }
            batch = dst.NewBatch()
            fmt.Printf("Copied %d keys (state: %d, storage: %d)...\n", count, stateCount, storageCount)
        }
    }
    
    // Commit final batch
    if err := batch.Commit(nil); err != nil {
        log.Fatalf("Failed to commit final batch: %v", err)
    }
    
    fmt.Printf("\nCopy complete!\n")
    fmt.Printf("Total keys copied: %d\n", count)
    fmt.Printf("State entries: %d\n", stateCount)
    fmt.Printf("Storage entries: %d\n", storageCount)
    
    // Now we need to set the LastBlock key to point to block 14552
    fmt.Println("\nSetting LastBlock pointer...")
    
    // The LastBlock key should point to the hash of block 14552
    // For now, we'll need to reconstruct this from the data
    lastBlockKey := []byte("LastBlock")
    
    // We need to find the hash of block 14552
    // This is a placeholder - we'd need to reconstruct the actual hash
    blockHash := make([]byte, 32)
    
    if err := dst.Set(lastBlockKey, blockHash, nil); err != nil {
        log.Printf("Warning: Failed to set LastBlock: %v", err)
    }
    
    fmt.Println("State migration complete. Next steps:")
    fmt.Println("1. We need to reconstruct the block headers from the state data")
    fmt.Println("2. Set the proper pointer keys (LastBlock, LastHeader, LastFast)")
    fmt.Println("3. Move this database to the C-Chain VM location")
}