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
		fmt.Println("Usage: find-highest-block <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Look for keys with 0x336e prefix (namespace + 'n' for canonical)
	prefix := []byte{0x33}
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: []byte{0x34}, // Next byte after 0x33
	})
	defer iter.Close()

	var highestBlock uint64
	foundCanonical := false

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 34 && key[33] == 'n' { // 33-byte namespace + 'n'
			// This is a canonical hash key
			if len(key) >= 42 { // 33 + 1 + 8
				blockNum := binary.BigEndian.Uint64(key[34:42])
				if blockNum > highestBlock {
					highestBlock = blockNum
				}
				foundCanonical = true
			}
		}
	}

	if foundCanonical {
		fmt.Printf("Highest block found: %d (0x%x)\n", highestBlock, highestBlock)
	} else {
		fmt.Println("No canonical mappings found")
	}
}