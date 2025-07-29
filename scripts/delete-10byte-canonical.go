package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <db-path>\n", os.Args[0])
		os.Exit(1)
	}

	dbPath := os.Args[1]
	log.Printf("Opening database at %s to delete 10-byte canonical keys", dbPath)

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Find and delete all 10-byte canonical keys (0x68 prefix, ending with 0x6e)
	batch := db.NewBatch()
	deleteCount := 0

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// Check if this is a 10-byte key ending with 0x6e
		if len(key) == 10 && key[0] == 0x68 && key[9] == 0x6e {
			// Delete this key
			if err := batch.Delete(key, nil); err != nil {
				log.Printf("Failed to delete key %x: %v", key, err)
				continue
			}
			deleteCount++
			if deleteCount%1000 == 0 {
				log.Printf("Marked %d keys for deletion...", deleteCount)
			}
		}
	}

	if deleteCount > 0 {
		log.Printf("Deleting %d 10-byte canonical keys...", deleteCount)
		if err := batch.Commit(nil); err != nil {
			log.Fatalf("Failed to commit deletions: %v", err)
		}
		log.Printf("Successfully deleted %d keys", deleteCount)
	} else {
		log.Printf("No 10-byte canonical keys found to delete")
		batch.Close()
	}

	// Verify the deletions
	iter2, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	if err != nil {
		log.Fatalf("Failed to create verification iterator: %v", err)
	}
	defer iter2.Close()

	remaining10Byte := 0
	total9Byte := 0
	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		if len(key) == 10 && key[9] == 0x6e {
			remaining10Byte++
		} else if len(key) == 9 {
			total9Byte++
		}
	}

	log.Printf("Verification: %d 10-byte keys remaining, %d 9-byte keys present", remaining10Byte, total9Byte)
}