package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <path/to/chaindata>", os.Args[0])
	}
	dbPath := os.Args[1]

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("pebble.Open: %v", err)
	}
	defer db.Close()

	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("NewIter: %v", err)
	}
	defer iter.Close()

	headerCount := 0
	bodyCount := 0
	canonicalCount := 0
	hashToNumCount := 0
	tdCount := 0

	for ok := iter.First(); ok; ok = iter.Next() {
		key := iter.Key()

		// Geth block-related prefixes:
		//   'h' = header
		//   'b' = body
		//   'n' = number→hash
		//   'H' = hash→number
		//   'T' = total difficulty
		switch key[0] {
		case 'h':
			headerCount++
			if headerCount <= 5 {
				fmt.Printf("prefix='h'  raw hex=%s\n", hex.EncodeToString(key))
			}
		case 'b':
			bodyCount++
			if bodyCount <= 5 {
				fmt.Printf("prefix='b'  raw hex=%s\n", hex.EncodeToString(key))
			}
		case 'n':
			canonicalCount++
			if canonicalCount <= 5 {
				fmt.Printf("prefix='n'  raw hex=%s\n", hex.EncodeToString(key))
			}
		case 'H':
			hashToNumCount++
			if hashToNumCount <= 5 {
				fmt.Printf("prefix='H'  raw hex=%s\n", hex.EncodeToString(key))
			}
		case 'T':
			tdCount++
			if tdCount <= 5 {
				fmt.Printf("prefix='T'  raw hex=%s\n", hex.EncodeToString(key))
			}
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Headers (h prefix): %d\n", headerCount)
	fmt.Printf("Bodies (b prefix): %d\n", bodyCount)
	fmt.Printf("Canonical (n prefix): %d\n", canonicalCount)
	fmt.Printf("Hash->Number (H prefix): %d\n", hashToNumCount)
	fmt.Printf("Total Difficulty (T prefix): %d\n", tdCount)
}
