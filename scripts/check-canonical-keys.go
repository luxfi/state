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
		fmt.Println("Usage: check-canonical-keys <db-path>")
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

	fmt.Println("=== Checking for Canonical Hash Keys ===")

	// Try different key formats
	testFormats := []struct {
		name   string
		makeKey func(height uint64) []byte
	}{
		{
			name: "Format 1: evmn + height",
			makeKey: func(height uint64) []byte {
				key := make([]byte, 12)
				copy(key[:4], []byte("evmn"))
				binary.BigEndian.PutUint64(key[4:], height)
				return key
			},
		},
		{
			name: "Format 2: column(4) + evmn + height",
			makeKey: func(height uint64) []byte {
				key := make([]byte, 16)
				binary.BigEndian.PutUint32(key[:4], 1)
				copy(key[4:8], []byte("evmn"))
				binary.BigEndian.PutUint64(key[8:], height)
				return key
			},
		},
		{
			name: "Format 3: column(4) + evmn + height + revision(8)",
			makeKey: func(height uint64) []byte {
				key := make([]byte, 24)
				binary.BigEndian.PutUint32(key[:4], 1)
				copy(key[4:8], []byte("evmn"))
				binary.BigEndian.PutUint64(key[8:16], height)
				// Try with revision 0
				binary.BigEndian.PutUint64(key[16:], 0)
				return key
			},
		},
		{
			name: "Format 4: evmn + height + revision(8)",
			makeKey: func(height uint64) []byte {
				key := make([]byte, 20)
				copy(key[:4], []byte("evmn"))
				binary.BigEndian.PutUint64(key[4:12], height)
				// Try with revision 0
				binary.BigEndian.PutUint64(key[12:], 0)
				return key
			},
		},
	}

	// Test each format with a few heights
	testHeights := []uint64{0, 1, 100, 1000, 10000, 100000, 1082780}

	for _, format := range testFormats {
		fmt.Printf("\n%s:\n", format.name)
		found := 0
		
		for _, height := range testHeights {
			key := format.makeKey(height)
			val, closer, err := db.Get(key)
			if err == nil {
				fmt.Printf("  Height %d: Found! Key=%x Value(len=%d)=%x...\n", 
					height, key, len(val), val[:min(32, len(val))])
				found++
				closer.Close()
			}
		}
		
		if found == 0 {
			fmt.Printf("  No keys found with this format\n")
		}
	}

	// Also scan for keys that look like canonical hashes
	fmt.Println("\n=== Scanning for evmn pattern ===")
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	count := 0
	evmnCount := 0
	for iter.First(); iter.Valid() && count < 100000; iter.Next() {
		key := iter.Key()
		
		// Check if key contains "evmn" pattern
		for i := 0; i <= len(key)-4; i++ {
			if string(key[i:i+4]) == "evmn" {
				evmnCount++
				if evmnCount <= 5 {
					fmt.Printf("Found evmn at offset %d: key=%x\n", i, key[:min(32, len(key))])
					
					// Try to parse height if it follows evmn
					if i+12 <= len(key) {
						height := binary.BigEndian.Uint64(key[i+4:i+12])
						fmt.Printf("  Possible height: %d\n", height)
					}
				}
				break
			}
		}
		count++
	}
	
	fmt.Printf("\nScanned %d keys, found %d with 'evmn' pattern\n", count, evmnCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}