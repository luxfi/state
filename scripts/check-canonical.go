package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-canonical <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check for the canonical key at height 1082780
	key, _ := hex.DecodeString("68000000000010859c6e")

	value, closer, err := db.Get(key)
	if err != nil {
		fmt.Printf("Key not found: %v\n", err)
		fmt.Printf("Key hex: %x\n", key)
	} else {
		defer closer.Close()
		fmt.Printf("Found canonical hash!\n")
		fmt.Printf("Key: %x\n", key)
		fmt.Printf("Value: %x\n", value)
	}
}
