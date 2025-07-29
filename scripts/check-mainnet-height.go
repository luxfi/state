package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: check-mainnet-height <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Check for height 1082780 with different prefixes
	height := uint64(1082780)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	// Check standard geth format (0x68)
	canonicalKey := append([]byte{0x68}, heightBytes...)
	if val, closer, err := db.Get(canonicalKey); err == nil {
		defer closer.Close()
		fmt.Printf("✓ Found with standard prefix (0x68): hash=0x%x\n", val)
	} else {
		fmt.Printf("✗ Not found with standard prefix (0x68)\n")
	}

	// Check EVM format (evmn)
	evmCanonicalKey := append([]byte("evmn"), heightBytes...)
	if val, closer, err := db.Get(evmCanonicalKey); err == nil {
		defer closer.Close()
		fmt.Printf("✓ Found with EVM prefix (evmn): hash=0x%x\n", val)
	} else {
		fmt.Printf("✗ Not found with EVM prefix (evmn)\n")
	}

	// Check namespaced format (might have additional prefix)
	// Try with 0x01 namespace prefix
	nsCanonicalKey := append([]byte{0x01, 0x68}, heightBytes...)
	if val, closer, err := db.Get(nsCanonicalKey); err == nil {
		defer closer.Close()
		fmt.Printf("✓ Found with namespaced prefix (0x01,0x68): hash=0x%x\n", val)
	} else {
		fmt.Printf("✗ Not found with namespaced prefix (0x01,0x68)\n")
	}

	// List some keys to understand the structure
	fmt.Println("\nSample keys in database:")
	iter, _ := db.NewIter(nil)
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		fmt.Printf("  Key[%d]: %x (length=%d)\n", count, key, len(key))
		if len(key) > 0 {
			fmt.Printf("    First byte: 0x%02x", key[0])
			if key[0] >= 32 && key[0] <= 126 {
				fmt.Printf(" ('%c')", key[0])
			}
			fmt.Println()
		}
		count++
	}
}