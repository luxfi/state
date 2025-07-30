package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	var dbPath = flag.String("db", "", "path to pebbledb")
	flag.Parse()

	if *dbPath == "" {
		flag.Usage()
		log.Fatal("--db is required")
	}

	db, err := pebble.Open(*dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Look at hash->number mappings more carefully
	fmt.Println("Analyzing hash->number mappings...")
	prefix := []byte("evmH")

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	var maxHeight uint64
	count := 0
	samples := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Hash->number values should be 8-byte block numbers
		if len(value) == 8 {
			num := binary.BigEndian.Uint64(value)

			if samples < 10 {
				fmt.Printf("Sample %d:\n", samples)
				fmt.Printf("  Key (hash): %s\n", hex.EncodeToString(key[4:])) // Skip "evmH" prefix
				fmt.Printf("  Value (block): %d\n", num)
				samples++
			}

			if num > maxHeight {
				maxHeight = num
			}
		}
		count++
	}

	fmt.Printf("\nTotal hash->number mappings: %d\n", count)
	fmt.Printf("Maximum block number: %d\n", maxHeight)

	// Now let's check if we can find headers for these blocks
	fmt.Printf("\nChecking if we have headers for high blocks...\n")

	// Try to find header for max block
	hash := findHashForNumber(db, maxHeight)
	if hash != nil {
		fmt.Printf("Found hash for block %d: %s\n", maxHeight, hex.EncodeToString(hash))

		// Try to get the header
		headerKey := append([]byte("evmh"), hash...)
		value, closer, err := db.Get(headerKey)
		if err == nil {
			fmt.Printf("Header exists! Length: %d bytes\n", len(value))
			closer.Close()
		} else {
			fmt.Printf("No header found for this hash\n")
		}
	}
}

func findHashForNumber(db *pebble.DB, num uint64) []byte {
	// Look through evmn keys to find the hash for this number
	prefix := []byte("evmn")

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return nil
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		value := iter.Value()

		// Look for the block number in the value
		if len(value) >= 8 {
			for i := 0; i <= len(value)-8; i++ {
				if binary.BigEndian.Uint64(value[i:i+8]) == num {
					// Found it! The key contains the hash
					key := iter.Key()
					if len(key) > 4 {
						return key[4:] // Skip "evmn" prefix
					}
				}
			}
		}
	}

	return nil
}
