package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: cleanup-canonical <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	opts := &pebble.Options{}
	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Cleaning up old 10-byte canonical keys (ending with 0x6e)...")

	// Find and delete all 10-byte keys starting with 0x68 and ending with 0x6e
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	batch := db.NewBatch()
	deleteCount := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// Check if it's a 10-byte key ending with 0x6e
		if len(key) == 10 && key[0] == 0x68 && key[9] == 0x6e {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)

			height := binary.BigEndian.Uint64(key[1:9])
			fmt.Printf("  Deleting 10-byte key for height %d: %x\n", height, keyCopy)

			if err := batch.Delete(keyCopy, nil); err != nil {
				fmt.Printf("    Error deleting key: %v\n", err)
			} else {
				deleteCount++
			}
		}
	}

	if deleteCount > 0 {
		fmt.Printf("\nCommitting deletion of %d keys...\n", deleteCount)
		if err := batch.Commit(pebble.Sync); err != nil {
			fmt.Printf("Failed to commit deletions: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Successfully deleted %d old 10-byte canonical keys\n", deleteCount)
	} else {
		fmt.Println("✓ No old 10-byte canonical keys found")
	}

	// Verify the cleanup
	fmt.Println("\nVerifying cleanup...")
	iter2, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter2.Close()

	count10byte := 0
	count9byte := 0

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		if len(key) == 10 && key[0] == 0x68 && key[9] == 0x6e {
			count10byte++
		} else if len(key) == 9 && key[0] == 0x68 {
			count9byte++
		}
	}

	fmt.Printf("\nRemaining keys:\n")
	fmt.Printf("  9-byte canonical keys: %d\n", count9byte)
	fmt.Printf("  10-byte canonical keys: %d\n", count10byte)

	// Check for height 1082780 specifically
	key9 := make([]byte, 9)
	key9[0] = 0x68
	binary.BigEndian.PutUint64(key9[1:], 1082780)

	if value, closer, err := db.Get(key9); err == nil {
		defer closer.Close()
		fmt.Printf("\n✓ Canonical key for height 1082780 exists: %x -> %x\n", key9, value)
	} else {
		fmt.Printf("\n✗ Canonical key for height 1082780 NOT FOUND!\n")
	}
}
