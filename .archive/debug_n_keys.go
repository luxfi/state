package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	var src = flag.String("src", "", "source subnet database")
	flag.Parse()

	if *src == "" {
		flag.Usage()
		log.Fatal("--src is required")
	}

	db, err := pebble.Open(*src, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Debugging 'n' keys ===")

	// Count different types of keys
	var nKeys, hKeys, totalKeys int

	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Map to track hash->number mappings
	hashToNumber := make(map[string]uint64)

	// First pass: count keys and collect H mappings
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		if len(key) < 41 {
			continue
		}

		logicalKey := key[33 : len(key)-8]
		if len(logicalKey) == 0 {
			continue
		}

		totalKeys++

		// Count 'n' keys
		if logicalKey[0] == 'n' {
			nKeys++
		}

		// Count and collect 'H' keys
		if logicalKey[0] == 'H' && len(logicalKey) > 1 {
			hKeys++
			if len(value) == 8 {
				hash := string(logicalKey[1:])
				number := binary.BigEndian.Uint64(value)
				hashToNumber[hash] = number
			}
		}

		// Sample some keys
		if totalKeys < 10 {
			fmt.Printf("Sample key %d:\n", totalKeys)
			fmt.Printf("  Full key: %x\n", key[:min(80, len(key))])
			fmt.Printf("  Logical key type: %c (0x%x)\n", logicalKey[0], logicalKey[0])
			if logicalKey[0] == 'n' && len(logicalKey) > 1 {
				fmt.Printf("  'n' key content: %x\n", logicalKey[1:])
			}
		}
	}

	fmt.Printf("\nTotal keys processed: %d\n", totalKeys)
	fmt.Printf("'n' keys found: %d\n", nKeys)
	fmt.Printf("'H' keys found: %d\n", hKeys)
	fmt.Printf("Hash->Number mappings: %d\n", len(hashToNumber))

	// Now check how many 'n' keys we can match
	iter2, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter2.Close()

	matchedNKeys := 0
	unmatchedSamples := 0

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()

		if len(key) < 41 {
			continue
		}

		logicalKey := key[33 : len(key)-8]

		if len(logicalKey) > 1 && logicalKey[0] == 'n' {
			hashPart := string(logicalKey[1:])
			if _, found := hashToNumber[hashPart]; found {
				matchedNKeys++
			} else if unmatchedSamples < 5 {
				unmatchedSamples++
				fmt.Printf("\nUnmatched 'n' key sample %d:\n", unmatchedSamples)
				fmt.Printf("  Hash part: %x\n", hashPart)
				fmt.Printf("  Length: %d bytes\n", len(hashPart))
			}
		}
	}

	fmt.Printf("\n'n' keys that can be matched: %d / %d (%.2f%%)\n",
		matchedNKeys, nKeys, float64(matchedNKeys)*100/float64(nKeys))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
