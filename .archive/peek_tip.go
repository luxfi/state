package main

import (
	"encoding/binary"
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

	// Look for evmn keys (number->hash mappings)
	// Key format: "evm" + "n" + data
	// The actual number is at the end of the key based on the sample data
	prefix := []byte("evmn")
	
	var maxHeight uint64
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 12 { // "evm" + "n" + 8 bytes
			height := binary.BigEndian.Uint64(key[4:])
			if height > maxHeight {
				maxHeight = height
			}
			count++
		}
	}

	if count == 0 {
		fmt.Println("No evmn keys found in database")
	} else {
		fmt.Printf("maxHeight = %d\n", maxHeight)
		fmt.Printf("(found %d canonical number mappings)\n", count)
	}
}