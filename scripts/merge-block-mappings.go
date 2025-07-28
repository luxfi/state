package main

import (
	"fmt"
	"log"
	"os"
	
	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: merge-block-mappings <source-db> <target-db>")
		fmt.Println()
		fmt.Println("Merges block mappings from source into target database")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]

	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Open target database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open target database: %v", err)
	}
	defer dstDB.Close()

	fmt.Println("=== Merging Block Mappings ===")

	// Create iterator
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	batch := dstDB.NewBatch()
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Copy key-value pair
		if err := batch.Set(key, value, pebble.Sync); err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}

		count++

		// Show progress
		if count%100000 == 0 {
			fmt.Printf("  Merged %d entries...\n", count)
		}

		// Commit batch periodically
		if count%1000000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
		}
	}

	// Final batch commit
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	fmt.Printf("\n=== Merge Complete ===\n")
	fmt.Printf("Total entries merged: %d\n", count)
}