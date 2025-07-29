package main

import (
    "encoding/binary"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: fix-migration-for-vm <input-db> <output-db>")
        os.Exit(1)
    }
    
    inputDB := os.Args[1]
    outputDB := os.Args[2]
    
    // Open input database
    inDB, err := pebble.Open(inputDB, &pebble.Options{ReadOnly: true})
    if err != nil {
        log.Fatal("Failed to open input DB:", err)
    }
    defer inDB.Close()
    
    // Open output database
    outDB, err := pebble.Open(outputDB, &pebble.Options{})
    if err != nil {
        log.Fatal("Failed to open output DB:", err)
    }
    defer outDB.Close()
    
    // Create a batch for efficiency
    batch := outDB.NewBatch()
    
    // First, copy consensus keys as-is
    consensusKeys := []string{"Height", "LastBlock", "lastAccepted"}
    for _, key := range consensusKeys {
        if val, closer, err := inDB.Get([]byte(key)); err == nil {
            batch.Set([]byte(key), val, nil)
            closer.Close()
            fmt.Printf("Copied consensus key: %s\n", key)
        }
    }
    
    // Now handle the evmh, evmb, evmr, evmn keys
    iter, _ := inDB.NewIter(&pebble.IterOptions{})
    defer iter.Close()
    
    headers := 0
    bodies := 0
    receipts := 0
    canonicals := 0
    hashToNumber := 0
    accounts := 0
    other := 0
    
    for iter.First(); iter.Valid(); iter.Next() {
        key := iter.Key()
        value := iter.Value()
        
        if len(key) >= 4 {
            prefix := string(key[:4])
            
            switch prefix {
            case "evmh": // headers
                // Convert evmh{blockNum}{hash} to 0x68{blockNum}{hash}
                newKey := []byte{0x68}
                newKey = append(newKey, key[4:]...)
                batch.Set(newKey, value, nil)
                headers++
                
            case "evmb": // bodies
                // Convert evmb{blockNum}{hash} to 0x62{blockNum}{hash}
                newKey := []byte{0x62}
                newKey = append(newKey, key[4:]...)
                batch.Set(newKey, value, nil)
                bodies++
                
            case "evmr": // receipts
                // Convert evmr{blockNum}{hash} to 0x72{blockNum}{hash}
                newKey := []byte{0x72}
                newKey = append(newKey, key[4:]...)
                batch.Set(newKey, value, nil)
                receipts++
                
            case "evmn": // canonical
                // Convert evmn{blockNum} to 0x68{blockNum}0x6e
                // This is the canonical block hash mapping
                if len(key) == 12 { // evmn + 8 byte block number
                    newKey := []byte{0x68}
                    newKey = append(newKey, key[4:12]...) // 8-byte block number
                    newKey = append(newKey, 0x6e)         // canonical suffix
                    batch.Set(newKey, value, nil)
                    canonicals++
                }
                
            case "evmH": // hash to number
                // Convert evmH{hash} to 0x48{hash}
                newKey := []byte{0x48}
                newKey = append(newKey, key[4:]...)
                batch.Set(newKey, value, nil)
                hashToNumber++
                
            default:
                // Copy other keys as-is (like account data)
                batch.Set(key, value, nil)
                if len(key) == 31 || len(key) == 32 {
                    accounts++
                } else {
                    other++
                }
            }
        } else {
            // Copy short keys as-is
            batch.Set(key, value, nil)
            other++
        }
    }
    
    // Commit the batch
    if err := batch.Commit(nil); err != nil {
        log.Fatal("Failed to commit batch:", err)
    }
    
    fmt.Printf("\nMigration complete:\n")
    fmt.Printf("Headers: %d\n", headers)
    fmt.Printf("Bodies: %d\n", bodies)
    fmt.Printf("Receipts: %d\n", receipts)
    fmt.Printf("Canonical mappings: %d\n", canonicals)
    fmt.Printf("Hash to number: %d\n", hashToNumber)
    fmt.Printf("Accounts: %d\n", accounts)
    fmt.Printf("Other keys: %d\n", other)
    
    // Verify a few canonical keys
    fmt.Println("\nVerifying canonical keys:")
    for i := uint64(0); i < 5; i++ {
        blockBytes := make([]byte, 8)
        binary.BigEndian.PutUint64(blockBytes, i)
        
        checkKey := []byte{0x68}
        checkKey = append(checkKey, blockBytes...)
        checkKey = append(checkKey, 0x6e)
        
        if val, closer, err := outDB.Get(checkKey); err == nil {
            fmt.Printf("Block %d canonical: %x\n", i, val)
            closer.Close()
        }
    }
}