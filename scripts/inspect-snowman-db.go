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
		fmt.Println("Usage: inspect-snowman-db <db-path>")
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

	fmt.Println("=== Inspecting Snowman Database ===")

	// Check last accepted
	if val, closer, err := db.Get([]byte("last_accepted")); err == nil {
		fmt.Printf("\nlast_accepted: %s\n", hex.EncodeToString(val))
		closer.Close()
	} else {
		fmt.Println("\nlast_accepted: NOT FOUND")
	}

	// Count entries by prefix
	prefixCounts := make(map[byte]int)
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	fmt.Println("\nScanning database...")
	totalCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) > 0 {
			prefixCounts[key[0]]++
		}
		totalCount++
		
		// Show first few entries
		if totalCount <= 10 {
			fmt.Printf("  Key: %s", hex.EncodeToString(key))
			if len(key) > 20 {
				fmt.Printf(" (first 20 bytes)")
			}
			fmt.Printf("\n  Value: %s\n", hex.EncodeToString(iter.Value()))
		}
	}

	fmt.Printf("\nTotal entries: %d\n", totalCount)
	fmt.Println("\nEntries by prefix:")
	for prefix, count := range prefixCounts {
		fmt.Printf("  0x%02x: %d entries\n", prefix, count)
	}

	// Check specific height mappings
	fmt.Println("\nChecking height mappings (prefix 0x02)...")
	heightIter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x02},
		UpperBound: []byte{0x03},
	})
	if err != nil {
		log.Fatalf("Failed to create height iterator: %v", err)
	}
	defer heightIter.Close()

	heightCount := 0
	var maxHeight uint64
	for heightIter.First(); heightIter.Valid(); heightIter.Next() {
		key := heightIter.Key()
		if len(key) == 9 && key[0] == 0x02 {
			height := binary.BigEndian.Uint64(key[1:])
			if height > maxHeight {
				maxHeight = height
			}
			heightCount++
			
			// Show first few and last
			if heightCount <= 3 || height >= 1082778 {
				fmt.Printf("  Height %d -> Block ID: %s\n", height, hex.EncodeToString(heightIter.Value()))
			}
		}
	}
	
	fmt.Printf("\nTotal height mappings: %d\n", heightCount)
	fmt.Printf("Max height found: %d\n", maxHeight)
}