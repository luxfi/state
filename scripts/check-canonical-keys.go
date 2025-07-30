package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-canonical-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Checking canonical keys in database...")

	// Check all 0x68 prefixed keys
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	if err != nil {
		fmt.Printf("Failed to create iterator: %v\n", err)
		os.Exit(1)
	}
	defer iter.Close()

	var nineByteKeys, tenByteKeys, otherKeys int
	var toDelete [][]byte

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		switch len(key) {
		case 9:
			nineByteKeys++
			if nineByteKeys <= 5 {
				height := binary.BigEndian.Uint64(key[1:])
				fmt.Printf("  9-byte key (correct): height=%d, key=%x\n", height, key)
			}
		case 10:
			tenByteKeys++
			if tenByteKeys <= 5 {
				height := binary.BigEndian.Uint64(key[1:9])
				fmt.Printf("  10-byte key (wrong): height=%d, key=%x, last-byte=%02x\n", height, key, key[9])
			}
			// Collect keys to delete
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			toDelete = append(toDelete, keyCopy)
		default:
			otherKeys++
			fmt.Printf("  Other key length %d: %x\n", len(key), key)
		}
	}

	fmt.Printf("\nSummary:\n")
	fmt.Printf("  9-byte keys (correct): %d\n", nineByteKeys)
	fmt.Printf("  10-byte keys (wrong): %d\n", tenByteKeys)
	fmt.Printf("  Other length keys: %d\n", otherKeys)

	// Check for the specific height
	canonicalKey := make([]byte, 9)
	canonicalKey[0] = 0x68
	binary.BigEndian.PutUint64(canonicalKey[1:], 1082780)

	if value, closer, err := db.Get(canonicalKey); err == nil {
		defer closer.Close()
		fmt.Printf("\n✓ Found canonical hash at height 1082780: 0x%x\n", value)
	} else {
		fmt.Printf("\n✗ Canonical hash not found at height 1082780: %v\n", err)
	}

	// Delete the 10-byte keys if any found
	if len(toDelete) > 0 {
		fmt.Printf("\nDeleting %d wrong format keys...\n", len(toDelete))

		batch := db.NewBatch()
		for _, key := range toDelete {
			if err := batch.Delete(key, nil); err != nil {
				fmt.Printf("Failed to delete key %x: %v\n", key, err)
			}
		}

		if err := batch.Commit(pebble.Sync); err != nil {
			fmt.Printf("Failed to commit deletions: %v\n", err)
		} else {
			fmt.Printf("Successfully deleted %d wrong format keys\n", len(toDelete))
		}
		batch.Close()
	}
}
