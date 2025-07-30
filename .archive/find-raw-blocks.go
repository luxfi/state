package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <path/to/db>", os.Args[0])
	}
	dbPath := os.Args[1]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("pebble.Open: %v", err)
	}
	defer db.Close()

	fmt.Println("Searching for block data...")

	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("NewIter: %v", err)
	}
	defer iter.Close()

	possibleHeaders := 0
	keysChecked := 0

	for iter.First(); iter.Valid() && keysChecked < 100000; iter.Next() {
		key := iter.Key()
		val := iter.Value()
		keysChecked++

		// Show first 10 keys for debugging
		if keysChecked <= 10 {
			fmt.Printf("Key %d: %s (len=%d, val_len=%d)\n", keysChecked, hex.EncodeToString(key[:min(len(key), 32)]), len(key), len(val))
		}

		// Try to decode value as a block header
		if len(val) > 100 && len(val) < 2000 {
			var header types.Header
			if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
				possibleHeaders++
				blockNum := header.Number.Uint64()

				if possibleHeaders <= 5 {
					fmt.Printf("\nPossible header found at key: %x\n", key[:min(len(key), 32)])
					fmt.Printf("  Block number: %d\n", blockNum)
					fmt.Printf("  Parent hash: %x\n", header.ParentHash)
					fmt.Printf("  State root: %x\n", header.Root)
					fmt.Printf("  Time: %d\n", header.Time)
				}
			}
		}

		// Check for canonical number keys (just number encoded)
		if len(key) == 8 {
			num := binary.BigEndian.Uint64(key)
			if num < 100000 { // reasonable block number
				fmt.Printf("Possible block number key: %d -> %x\n", num, val[:min(len(val), 32)])
			}
		}

		if keysChecked%10000 == 0 {
			fmt.Printf("Checked %d keys...\n", keysChecked)
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Keys checked: %d\n", keysChecked)
	fmt.Printf("Possible headers found: %d\n", possibleHeaders)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
