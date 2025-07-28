package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: find-blockchain-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Look for different key patterns
	patterns := map[string][]byte{
		"headers (evmh)":      []byte("evmh"),
		"bodies (evmb)":       []byte("evmb"),
		"receipts (evmr)":     []byte("evmr"),
		"number->hash (evmn)": []byte("evmn"),
		"hash->number (evmH)": []byte("evmH"),
		"state (evms)":        []byte("evms"),
		"state (evm0x73)":     append([]byte("evm"), 0x73),
		"accounts (evm0x26)":  append([]byte("evm"), 0x26),
	}

	for name, prefix := range patterns {
		fmt.Printf("\nSearching for %s keys...\n", name)
		
		iter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: append(prefix, 0xff),
		})
		if err != nil {
			log.Printf("Failed to create iterator for %s: %v", name, err)
			continue
		}
		
		count := 0
		var maxHeight uint64
		for iter.First(); iter.Valid() && count < 5; iter.Next() {
			key := iter.Key()
			fmt.Printf("  Found: %s", hex.EncodeToString(key))
			
			// For number->hash keys, extract height
			if bytes.HasPrefix(key, []byte("evmn")) && len(key) == 12 {
				height := binary.BigEndian.Uint64(key[4:])
				fmt.Printf(" (height=%d)", height)
				if height > maxHeight {
					maxHeight = height
				}
			}
			fmt.Println()
			count++
		}
		
		if count == 0 {
			fmt.Printf("  No %s keys found\n", name)
		} else {
			// Continue counting
			for iter.Next(); iter.Valid(); iter.Next() {
				count++
				if bytes.HasPrefix(iter.Key(), []byte("evmn")) && len(iter.Key()) == 12 {
					height := binary.BigEndian.Uint64(iter.Key()[4:])
					if height > maxHeight {
						maxHeight = height
					}
				}
			}
			fmt.Printf("  Total %s keys: %d", name, count)
			if maxHeight > 0 {
				fmt.Printf(" (max height: %d)", maxHeight)
			}
			fmt.Println()
		}
		
		iter.Close()
	}
}