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
		fmt.Println("Usage: find-highest-block-v2 <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Look for header keys with namespace prefix
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	var highestBlock uint64
	foundHeaders := false

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// Look for header keys: namespace(33) + 'h'(0x68) + number(8) + hash(32)
		if len(key) >= 34 && key[33] == 0x68 && len(key) == 73 {
			// Extract block number
			blockNum := binary.BigEndian.Uint64(key[34:42])
			if blockNum > highestBlock && blockNum < 10000000 { // Sanity check
				highestBlock = blockNum
				foundHeaders = true
			}
		}
	}

	if foundHeaders {
		fmt.Printf("Highest block found: %d (0x%x)\n", highestBlock, highestBlock)
	} else {
		fmt.Println("No header keys found")
	}
}