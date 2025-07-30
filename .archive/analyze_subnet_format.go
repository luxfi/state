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

	fmt.Println("=== Analyzing Subnet Key Format ===")

	// Look for keys that might contain block numbers
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	samples := 0
	blockNumbersFound := make(map[uint64]bool)

	for iter.First(); iter.Valid() && samples < 100000; iter.Next() {
		key := iter.Key()

		if len(key) < 41 {
			continue
		}

		// Extract logical key
		logicalKey := key[33 : len(key)-8]

		// Look for patterns that might contain block numbers
		if len(logicalKey) > 0 && logicalKey[0] == 'n' {
			// This is a number->hash key, but in wrong format
			// The key after 'n' should be a hash, and the block number might be in the middle

			if samples < 10 {
				fmt.Printf("\nSample 'n' key %d:\n", samples)
				fmt.Printf("  Full key: %x\n", key)
				fmt.Printf("  Logical key: %x\n", logicalKey)
				fmt.Printf("  Value: %x\n", iter.Value())

				// Check if there's a number embedded in the key
				// Looking at the pattern, there seems to be a number at position 8-16
				if len(key) >= 41 {
					// Try extracting number from different positions
					for i := 33; i < len(key)-16; i += 4 {
						if i+8 <= len(key) {
							num := binary.BigEndian.Uint64(key[i : i+8])
							if num > 0 && num < 10000000 { // Reasonable block number
								fmt.Printf("  Possible block number at offset %d: %d\n", i, num)
								blockNumbersFound[num] = true
							}
						}
					}
				}
			}
			samples++
		}

		// Also look for H keys (hash->number) which should give us the proper mapping
		if len(logicalKey) > 0 && logicalKey[0] == 'H' {
			value := iter.Value()
			if len(value) == 8 {
				num := binary.BigEndian.Uint64(value)
				if num > 0 && num < 10000000 {
					blockNumbersFound[num] = true
					if samples < 20 {
						fmt.Printf("\nFound H key with block number %d\n", num)
						fmt.Printf("  Hash: %x\n", logicalKey[1:])
					}
				}
			}
		}
	}

	// Find max block number
	var maxBlock uint64
	for num := range blockNumbersFound {
		if num > maxBlock {
			maxBlock = num
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Found %d unique block numbers\n", len(blockNumbersFound))
	fmt.Printf("Maximum block number: %d\n", maxBlock)

	// Now let's look at the exact key structure more carefully
	fmt.Println("\n=== Detailed Key Analysis ===")

	// Find a specific 'n' key and analyze its structure
	iter2, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter2.Close()

	found := 0
	for iter2.First(); iter2.Valid() && found < 5; iter2.Next() {
		key := iter2.Key()

		if len(key) >= 64 {
			// Check if this might be a number key
			// Position 41 in the full key (33 + 8) should be 'n'
			if key[33] == 'n' {
				found++
				fmt.Printf("\nDetailed analysis of 'n' key %d:\n", found)
				fmt.Printf("Full key (%d bytes): %x\n", len(key), key)

				// Break down the structure
				fmt.Printf("Prefix (33 bytes): %x\n", key[:33])
				fmt.Printf("Logical part: %x\n", key[33:len(key)-8])
				fmt.Printf("Suffix (8 bytes): %x\n", key[len(key)-8:])

				// The pattern seems to be:
				// Bytes 0-32: namespace/chain prefix
				// Byte 33: 'n' (0x6e)
				// Bytes 34-41: block number (8 bytes)
				// Bytes 42-: hash (variable length)
				// Last 8 bytes: revision

				if len(key) >= 42 {
					blockNum := binary.BigEndian.Uint64(key[34:42])
					fmt.Printf("Block number (bytes 34-41): %d (0x%x)\n", blockNum, blockNum)

					if len(key) > 42 {
						hashPart := key[42 : len(key)-8]
						fmt.Printf("Hash part: %x\n", hashPart)
					}
				}
			}
		}
	}
}
