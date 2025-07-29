package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-namespace <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Look for keys that might be canonical mappings
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	count := 0
	fmt.Println("Looking for canonical mapping keys (should have 'n' after namespace):")
	
	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		key := iter.Key()
		if len(key) >= 34 {
			// Check if byte 33 is 'n' (0x6e)
			if key[33] == 0x6e {
				fmt.Printf("\nFound canonical key!\n")
				fmt.Printf("Full key: %x\n", key)
				fmt.Printf("Namespace (first 33): %x\n", key[:33])
				fmt.Printf("Key type: %c (0x%x)\n", key[33], key[33])
				if len(key) >= 42 {
					fmt.Printf("Block number bytes: %x\n", key[34:42])
				}
				count++
				if count >= 5 {
					break
				}
			}
		}
	}

	if count == 0 {
		fmt.Println("No canonical keys found!")
	}
}