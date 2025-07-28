package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-head-pointers <db-path>")
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

	fmt.Println("=== Checking Head Pointers ===")

	// All possible head pointer keys
	headKeys := []string{
		"evmLastBlock",
		"evmLastHeader",
		"evmLastFast",
		"evmlastheader",
		"evmlastfast",
		"evmlastblock",
		"evmLastPivot",
		"evmlastpivot",
		"evmSnapshotRoot",
		"evmsnapshotroot",
		"evmh", // Single letter head header
		"evmH", // Single letter head block  
		"evmF", // Single letter head fast
		"evmS", // Single letter snapshot
	}

	fmt.Println("\nChecking all head pointers:")
	foundAny := false
	for _, key := range headKeys {
		value, closer, err := db.Get([]byte(key))
		if err == nil {
			fmt.Printf("  %s: %s\n", key, hex.EncodeToString(value))
			foundAny = true
			closer.Close()
		}
	}

	if !foundAny {
		fmt.Println("  No head pointers found!")
	}

	// Scan for highest block number
	fmt.Println("\nScanning for highest block...")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	var highestNum uint64
	var highestHash []byte
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 12 && string(key[:4]) == "evmn" {
			num := binary.BigEndian.Uint64(key[4:])
			if num > highestNum {
				highestNum = num
				highestHash = make([]byte, len(iter.Value()))
				copy(highestHash, iter.Value())
			}
			count++
		}
	}

	fmt.Printf("\nTotal blocks found: %d\n", count)
	fmt.Printf("Highest block number: %d\n", highestNum)
	if highestHash != nil {
		fmt.Printf("Highest block hash: %s\n", hex.EncodeToString(highestHash))
		
		// Check if we need to set head pointers
		fmt.Println("\nRecommendation: If no head pointers found, set the following:")
		fmt.Printf("  evmLastBlock = %s\n", hex.EncodeToString(highestHash))
		fmt.Printf("  evmLastHeader = %s\n", hex.EncodeToString(highestHash))
		fmt.Printf("  evmLastFast = %s\n", hex.EncodeToString(highestHash))
	}
}