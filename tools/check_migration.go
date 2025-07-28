package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	var db = flag.String("db", "", "database path")
	flag.Parse()

	if *db == "" {
		log.Fatal("--db is required")
	}

	database, err := pebble.Open(*db, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Check evmn keys
	iter, err := database.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	fmt.Println("=== Checking Canonical Mappings ===")
	count := 0
	maxHeight := uint64(0)

	prefix := []byte("evmn")
	for iter.SeekGE(prefix); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		if len(key) >= 4 && string(key[:4]) == "evmn" {
			if len(key) == 12 { // proper format
				height := binary.BigEndian.Uint64(key[4:])
				fmt.Printf("  Block %d -> hash %x\n", height, iter.Value())
				if height > maxHeight {
					maxHeight = height
				}
				count++
			} else {
				fmt.Printf("  Malformed evmn key: %x\n", key)
			}
		}
	}

	if count == 0 {
		fmt.Println("  No canonical mappings found!")
	} else {
		fmt.Printf("\nMax height with canonical mapping: %d\n", maxHeight)
	}

	// Check if we have headers
	hCount := 0
	hPrefix := []byte("evmh")
	for iter.SeekGE(hPrefix); iter.Valid() && hCount < 100; iter.Next() {
		key := iter.Key()
		if len(key) >= 4 && string(key[:4]) == "evmh" {
			hCount++
		}
	}
	fmt.Printf("Headers found: %d\n", hCount)

	// Check bodies
	bCount := 0
	bPrefix := []byte("evmb")
	for iter.SeekGE(bPrefix); iter.Valid() && bCount < 100; iter.Next() {
		key := iter.Key()
		if len(key) >= 4 && string(key[:4]) == "evmb" {
			bCount++
		}
	}
	fmt.Printf("Bodies found: %d\n", bCount)
}