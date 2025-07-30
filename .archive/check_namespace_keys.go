package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := flag.String("db", "", "path to PebbleDB")
	limit := flag.Int("limit", 10, "limit number of keys to show")
	flag.Parse()
	if *dbPath == "" {
		log.Fatal("--db is required")
	}

	db, err := pebble.Open(*dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	it, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("iterator: %v", err)
	}
	defer it.Close()

	// Count different key types
	keyTypes := make(map[string]int)

	// Show first few keys
	fmt.Println("First few keys in database:")
	count := 0

	for it.First(); it.Valid() && count < *limit; it.Next() {
		key := it.Key()
		val := it.Value()

		fmt.Printf("Key %d: %s (hex: %s) - Value length: %d\n",
			count+1,
			string(key[:min(10, len(key))]),
			hex.EncodeToString(key[:min(20, len(key))]),
			len(val))

		// Identify key type
		if len(key) >= 4 {
			prefix := string(key[:4])
			keyTypes[prefix]++
		}

		count++
	}

	// Count all key types
	fmt.Println("\nCounting all key types...")
	for it.First(); it.Valid(); it.Next() {
		key := it.Key()
		if len(key) >= 4 {
			prefix := string(key[:4])
			keyTypes[prefix]++
		}
	}

	fmt.Println("\nKey type summary:")
	for prefix, count := range keyTypes {
		fmt.Printf("%s (hex: %s): %d keys\n",
			prefix, hex.EncodeToString([]byte(prefix)), count)
	}

	// Check specifically for evm namespace
	evmPrefix := []byte{0x65, 0x76, 0x6d} // "evm"
	evmCount := 0
	evmnCount := 0

	for it.First(); it.Valid(); it.Next() {
		key := it.Key()
		if len(key) >= 3 && string(key[:3]) == string(evmPrefix) {
			evmCount++
			if len(key) >= 4 && key[3] == 'n' {
				evmnCount++
			}
		}
	}

	fmt.Printf("\nEVM namespace keys: %d\n", evmCount)
	fmt.Printf("EVMN keys: %d\n", evmnCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
