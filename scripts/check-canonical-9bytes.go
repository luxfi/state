package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-canonical-9bytes <db-path>")
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

	// Check for the canonical key at height 1082780 (9 bytes)
	key9 := make([]byte, 9)
	key9[0] = 0x68
	binary.BigEndian.PutUint64(key9[1:], 1082780)
	
	// Check for the old 10-byte key
	key10, _ := hex.DecodeString("68000000000010859c6e")
	
	fmt.Println("Checking for canonical keys:")
	
	// Check 9-byte key
	value9, closer9, err9 := db.Get(key9)
	if err9 == nil {
		defer closer9.Close()
		fmt.Printf("✓ Found 9-byte canonical key: %x -> %x\n", key9, value9)
	} else {
		fmt.Printf("✗ 9-byte key not found: %x (error: %v)\n", key9, err9)
	}
	
	// Check 10-byte key
	value10, closer10, err10 := db.Get(key10)
	if err10 == nil {
		defer closer10.Close()
		fmt.Printf("⚠ Found old 10-byte canonical key: %x -> %x\n", key10, value10)
	} else {
		fmt.Printf("✓ Old 10-byte key not found (good)\n")
	}
	
	// List all keys starting with 0x68
	fmt.Println("\nAll keys starting with 0x68:")
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()
	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		fmt.Printf("  Key: %x (length: %d)\n", key, len(key))
		if len(key) >= 9 && key[0] == 0x68 {
			height := binary.BigEndian.Uint64(key[1:9])
			fmt.Printf("    Height: %d\n", height)
		}
		count++
	}
}