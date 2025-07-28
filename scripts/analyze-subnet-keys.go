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
	if len(os.Args) < 2 {
		fmt.Println("Usage: analyze-subnet-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Look specifically for number->hash mappings
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	fmt.Println("Looking for number->hash mappings in subnet database...")
	count := 0
	found := 0
	
	for iter.First(); iter.Valid() && count < 50000; iter.Next() {
		key := iter.Key()
		
		// Skip short keys
		if len(key) < 34 {
			count++
			continue
		}
		
		// Get the actual key after namespace
		actualKey := key[33:]
		
		// Look for 'n' prefix (number->hash)
		if len(actualKey) > 0 && actualKey[0] == 'n' {
			found++
			
			if found <= 10 {
				fmt.Printf("\nFound number->hash key %d:\n", found)
				fmt.Printf("  Full key: %s\n", hex.EncodeToString(key))
				fmt.Printf("  Namespace: %s\n", hex.EncodeToString(key[:33]))
				fmt.Printf("  Actual key: %s\n", hex.EncodeToString(actualKey))
				fmt.Printf("  Actual key len: %d\n", len(actualKey))
				
				// The key format appears to be 'n' + data
				// Let's check if there's a block number encoded
				if len(actualKey) >= 9 {
					// Try reading uint64 after 'n'
					blockNum := binary.BigEndian.Uint64(actualKey[1:9])
					fmt.Printf("  Block number (if at [1:9]): %d\n", blockNum)
				}
				
				// Also show the value (should be a hash)
				value := iter.Value()
				fmt.Printf("  Value (hash): %s\n", hex.EncodeToString(value))
				fmt.Printf("  Value len: %d\n", len(value))
			}
		}
		
		count++
	}
	
	fmt.Printf("\n\nScanned %d keys, found %d number->hash mappings\n", count, found)
	
	// Now find the maximum block number
	if found > 0 {
		fmt.Println("\nFinding maximum block number...")
		
		iter2, err := db.NewIter(nil)
		if err != nil {
			log.Fatalf("Failed to create iterator: %v", err)
		}
		defer iter2.Close()
		
		var maxBlock uint64
		for iter2.First(); iter2.Valid(); iter2.Next() {
			key := iter2.Key()
			if len(key) < 34 {
				continue
			}
			
			actualKey := key[33:]
			if len(actualKey) >= 9 && actualKey[0] == 'n' {
				blockNum := binary.BigEndian.Uint64(actualKey[1:9])
				if blockNum > maxBlock && blockNum < 10000000 { // Sanity check
					maxBlock = blockNum
				}
			}
		}
		
		fmt.Printf("Maximum block number found: %d\n", maxBlock)
	}
}