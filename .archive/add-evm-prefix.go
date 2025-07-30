package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: add-evm-prefix <source-db> <dest-db>")
		os.Exit(1)
	}

	sourceDB := os.Args[1]
	destDB := os.Args[2]

	// Open source database
	src, err := pebble.Open(sourceDB, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer src.Close()

	// Create destination database
	dst, err := pebble.Open(destDB, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create destination database: %v", err)
	}
	defer dst.Close()

	fmt.Println("=== Adding 'evm' prefix to all keys ===")

	// Define the prefix
	evmPrefix := []byte("evm")

	// Copy all keys with prefix
	iter, _ := src.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	batch := dst.NewBatch()
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		// Add evm prefix to the key
		newKey := append(evmPrefix, key...)

		if err := batch.Set(newKey, val, nil); err != nil {
			log.Printf("Error setting key: %v", err)
		}

		count++

		// Commit batch periodically
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Printf("Error committing batch: %v", err)
			}
			batch = dst.NewBatch()
			fmt.Printf("Processed %d keys...\n", count)
		}
	}

	// Commit final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Printf("Error committing final batch: %v", err)
	}

	fmt.Printf("\n✅ Complete! Added 'evm' prefix to %d keys\n", count)
	fmt.Printf("Destination: %s\n", destDB)
}
