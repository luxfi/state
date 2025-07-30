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

	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Count prefixes
	prefixCounts := make(map[string]int)
	totalKeys := 0

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid() && totalKeys < 100000; iter.Next() {
		key := iter.Key()
		totalKeys++

		// Get prefix (first 4 bytes or less)
		prefixLen := 4
		if len(key) < 4 {
			prefixLen = len(key)
		}

		prefix := string(key[:prefixLen])
		prefixCounts[prefix]++

		// Also check single byte prefix
		if len(key) > 0 {
			singleByte := fmt.Sprintf("0x%02x", key[0])
			prefixCounts[singleByte]++
		}
	}

	log.Printf("Database: %s", dbPath)
	log.Printf("Analyzed %d keys", totalKeys)
	log.Printf("\nTop prefixes:")

	// Print most common prefixes
	for prefix, count := range prefixCounts {
		if count > 100 {
			if len(prefix) == 4 {
				log.Printf("  %q (%x): %d", prefix, []byte(prefix), count)
			} else {
				log.Printf("  %s: %d", prefix, count)
			}
		}
	}
}
