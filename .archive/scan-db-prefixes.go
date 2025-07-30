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
		fmt.Println("Usage: scan-db-prefixes <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Track prefix counts
	prefixCounts := make(map[byte]int)
	
	// Sample some keys
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		key := iter.Key()
		if len(key) > 0 {
			prefixCounts[key[0]]++
			
			// Print first 10 samples
			if count < 10 {
				fmt.Printf("Sample key[%d]: hex=%s", count, hex.EncodeToString(key))
				if len(key) <= 32 {
					fmt.Printf(" (len=%d)", len(key))
				}
				fmt.Println()
			}
		}
		count++
	}

	fmt.Printf("\nScanned %d keys\n", count)
	fmt.Println("\nPrefix distribution:")
	for prefix, cnt := range prefixCounts {
		fmt.Printf("  key[0]=0x%02x: %d occurrences\n", prefix, cnt)
	}
}