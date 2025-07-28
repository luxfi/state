package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: find-highest-block <db-path>")
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

	fmt.Println("=== Finding Highest Block ===")
	
	// Look for number->hash entries (evm + 'n' prefix)
	prefix := []byte("evmn")
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	var highestBlock uint64
	var count int

	// Iterate through all number->hash entries
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// Format is: evm(3) + n(1) + number(8) + hash(32)
		// But some keys might be just evm(3) + n(1) + number(8)
		if len(key) >= 12 { // Minimum size for number->hash
			// Extract block number
			blockNum := binary.BigEndian.Uint64(key[4:12])
			
			// Sanity check - block numbers should be reasonable
			if blockNum < 100000000 { // Less than 100M
				if blockNum > highestBlock {
					highestBlock = blockNum
				}
				count++
				
				// Show first few and last few
				if count <= 5 || blockNum > 10000 {
					fmt.Printf("  Found block %d\n", blockNum)
				}
			}
		}
	}

	fmt.Printf("\nHighest block found: %d\n", highestBlock)
	fmt.Printf("Total blocks checked: %d\n", count)
}