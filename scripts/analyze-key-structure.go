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
		fmt.Println("Usage: analyze-key-structure <db-path>")
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

	fmt.Println("=== Analyzing Key Structure ===")

	// Create iterator
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 100; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) >= 8 {
			// Analyze key structure
			fmt.Printf("\nKey %d:\n", count)
			fmt.Printf("  Raw key (hex): %s\n", hex.EncodeToString(key[:min(32, len(key))]))
			fmt.Printf("  Key length: %d\n", len(key))
			
			// Check if it looks like number-prefixed key
			if len(key) >= 12 {
				// Try to parse as evmn + uint64
				prefix := string(key[:4])
				if len(key) == 12 {
					num := binary.BigEndian.Uint64(key[4:12])
					fmt.Printf("  Possible format: prefix='%s' num=%d\n", prefix, num)
				}
			}
			
			// Check value
			fmt.Printf("  Value length: %d\n", len(value))
			if len(value) == 32 {
				fmt.Printf("  Value (hash): %s\n", hex.EncodeToString(value))
			}
		}
		
		count++
	}
	
	// Now specifically look for keys with patterns
	fmt.Println("\n=== Looking for specific patterns ===")
	
	// Look for keys starting with ASCII characters
	iter2, _ := db.NewIter(nil)
	defer iter2.Close()
	
	asciiPrefixes := make(map[string]int)
	totalScanned := 0
	
	for iter2.First(); iter2.Valid() && totalScanned < 10000; iter2.Next() {
		key := iter2.Key()
		if len(key) >= 4 {
			// Check if first 4 bytes are printable ASCII
			prefix := key[:4]
			if isPrintableASCII(prefix) {
				asciiPrefixes[string(prefix)]++
			}
		}
		totalScanned++
	}
	
	fmt.Printf("\nScanned %d keys, found ASCII prefixes:\n", totalScanned)
	for prefix, count := range asciiPrefixes {
		fmt.Printf("  '%s': %d\n", prefix, count)
	}
}

func isPrintableASCII(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}