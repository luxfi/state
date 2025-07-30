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

	// Look for header keys which should contain block numbers
	prefix := []byte("evmh") // headers

	var maxHeight uint64
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		value := iter.Value()

		// Header structure starts with parent hash, then uncle hash, etc.
		// The block number is typically at a specific offset in the RLP-encoded header
		// Let's look for reasonable block numbers in the value
		if len(value) >= 8 {
			for i := 0; i <= len(value)-8 && i < 200; i++ {
				num := binary.BigEndian.Uint64(value[i : i+8])
				// Block numbers should be reasonable
				if num > 0 && num < 100000000 {
					if num > maxHeight {
						maxHeight = num
						if count < 5 {
							fmt.Printf("Found potential block %d at offset %d\n", num, i)
							fmt.Printf("  Value prefix: %s...\n", hex.EncodeToString(value[:min(64, len(value))]))
						}
					}
				}
			}
		}
		count++
	}

	// Also check hash->number mappings
	fmt.Println("\nChecking hash->number mappings...")
	prefix2 := []byte("evmH")
	iter2, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix2,
		UpperBound: append(prefix2, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter2.Close()

	count2 := 0
	for iter2.First(); iter2.Valid() && count2 < 1000; iter2.Next() {
		value := iter2.Value()

		// Hash->number values should be simple uint64
		if len(value) == 8 {
			num := binary.BigEndian.Uint64(value)
			if num > maxHeight && num < 100000000 {
				maxHeight = num
				if count2 < 5 {
					fmt.Printf("Found block number %d in hash->number mapping\n", num)
				}
			}
		}
		count2++
	}

	fmt.Printf("\nMaximum block height found: %d\n", maxHeight)
	fmt.Printf("(scanned %d headers, %d hash->number mappings)\n", count, count2)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
