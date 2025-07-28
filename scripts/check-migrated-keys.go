package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-migrated-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Checking Migrated Database Keys ===")

	// Look for canonical hash keys
	fmt.Println("\nLooking for canonical hash keys (evmn prefix):")
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		fmt.Printf("  Key: %s (hex: %s)\n", key[:4], hex.EncodeToString(key))
		fmt.Printf("  Value: %s\n", hex.EncodeToString(value))
		count++
	}

	// Check all unique prefixes
	fmt.Println("\nAll key prefixes:")
	prefixCounts := make(map[string]int)
	
	allIter, _ := db.NewIter(nil)
	defer allIter.Close()
	
	totalKeys := 0
	for allIter.First(); allIter.Valid(); allIter.Next() {
		key := allIter.Key()
		if len(key) >= 4 {
			prefix := string(key[:4])
			prefixCounts[prefix]++
		}
		totalKeys++
		
		// Show first few keys
		if totalKeys <= 5 {
			fmt.Printf("  Sample key: %s (hex: %s)\n", string(key[:min(8, len(key))]), hex.EncodeToString(key[:min(16, len(key))]))
		}
	}

	fmt.Printf("\nTotal keys: %d\n", totalKeys)
	fmt.Println("\nKey prefix counts:")
	for prefix, count := range prefixCounts {
		fmt.Printf("  %s: %d\n", prefix, count)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}