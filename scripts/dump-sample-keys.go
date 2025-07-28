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
		fmt.Println("Usage: dump-sample-keys <db-path>")
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

	fmt.Println("=== Dumping Sample Keys ===")

	// Create iterator
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	// Group keys by prefix pattern
	prefixGroups := make(map[string][]string)
	count := 0
	
	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Create a prefix identifier
		var prefix string
		if len(key) >= 4 {
			// Check if starts with column ID
			if key[0] == 0 && key[1] == 0 && key[2] == 0 {
				prefix = fmt.Sprintf("column-%d-", key[3])
				if len(key) > 4 {
					// Add next few bytes
					for i := 4; i < min(8, len(key)); i++ {
						if key[i] >= 32 && key[i] <= 126 { // printable ASCII
							prefix += string(key[i])
						} else {
							prefix += fmt.Sprintf("\\x%02x", key[i])
						}
					}
				}
			} else {
				// Regular prefix
				for i := 0; i < min(4, len(key)); i++ {
					if key[i] >= 32 && key[i] <= 126 { // printable ASCII
						prefix += string(key[i])
					} else {
						prefix += fmt.Sprintf("\\x%02x", key[i])
					}
				}
			}
		} else {
			prefix = "short"
		}
		
		// Store sample
		if len(prefixGroups[prefix]) < 3 {
			sample := fmt.Sprintf("  Key[%d]: %x, Value[%d]: %x...", 
				len(key), key[:min(32, len(key))], 
				len(value), value[:min(32, len(value))])
			prefixGroups[prefix] = append(prefixGroups[prefix], sample)
		}
		
		count++
	}
	
	fmt.Printf("\nScanned %d keys, found %d unique prefix patterns:\n\n", count, len(prefixGroups))
	
	for prefix, samples := range prefixGroups {
		fmt.Printf("Prefix: %s (%d samples shown)\n", prefix, len(samples))
		for _, sample := range samples {
			fmt.Println(sample)
		}
		fmt.Println()
	}
	
	// Look specifically for keys that might be block-related
	fmt.Println("=== Looking for block-related patterns ===")
	
	iter2, _ := db.NewIter(nil)
	defer iter2.Close()
	
	blockPatterns := 0
	for iter2.First(); iter2.Valid() && blockPatterns < 20; iter2.Next() {
		key := iter2.Key()
		value := iter2.Value()
		
		// Look for patterns that might indicate blocks
		// - Keys containing sequential numbers
		// - Values that are larger (block data)
		// - Keys with specific lengths
		
		if len(value) > 100 { // Might be block data
			fmt.Printf("\nLarge value found:\n")
			fmt.Printf("  Key[%d]: %x\n", len(key), key[:min(64, len(key))])
			fmt.Printf("  Value size: %d bytes\n", len(value))
			
			// Try to interpret key
			if len(key) >= 8 {
				// Check if last 8 bytes could be a number
				possibleNum := binary.BigEndian.Uint64(key[len(key)-8:])
				if possibleNum < 10000000 {
					fmt.Printf("  Possible number in key: %d\n", possibleNum)
				}
			}
			
			blockPatterns++
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}