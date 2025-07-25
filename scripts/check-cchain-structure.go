package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	// Check the current C-Chain database structure
	dbPath := "/home/z/.luxd/chainData/2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC/db/pebbledb"
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	fmt.Println("C-Chain database structure:")
	fmt.Println("===========================")
	
	// Count keys by prefix
	prefixCounts := make(map[string]int)
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Show first 10 keys
		if count < 10 {
			fmt.Printf("Key[%d]: %s (len: %d)\n", count, hex.EncodeToString(key), len(key))
			
			// Analyze key structure
			if len(key) >= 32 {
				blockchainID := hex.EncodeToString(key[:32])
				remainder := hex.EncodeToString(key[32:])
				fmt.Printf("  Blockchain ID: %s\n", blockchainID)
				fmt.Printf("  Key suffix: %s\n", remainder)
			}
		}
		
		// Count by prefix length
		if len(key) >= 32 {
			prefix := hex.EncodeToString(key[:32])
			prefixCounts[prefix]++
		} else {
			prefixCounts["short_key"]++
		}
		
		count++
	}
	
	fmt.Printf("\nTotal keys: %d\n", count)
	fmt.Printf("\nKeys by blockchain ID prefix:\n")
	for prefix, cnt := range prefixCounts {
		if prefix != "short_key" {
			fmt.Printf("  %s: %d keys\n", prefix, cnt)
		} else {
			fmt.Printf("  Keys shorter than 32 bytes: %d\n", cnt)
		}
	}
}