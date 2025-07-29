package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Try to find any keys
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	count := 0
	fmt.Println("First 20 keys in database:")
	for iter.First(); iter.Valid() && count < 20; iter.Next() {
		key := iter.Key()
		fmt.Printf("Key %d: %x (len=%d)\n", count, key, len(key))
		if len(key) > 0 && key[0] >= 32 && key[0] <= 126 {
			fmt.Printf("  ASCII prefix: %s\n", string(key[:min(10, len(key))]))
		}
		count++
	}

	if count == 0 {
		fmt.Println("No keys found in database!")
	} else {
		fmt.Printf("\nTotal keys shown: %d\n", count)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}