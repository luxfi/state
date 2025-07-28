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
		fmt.Println("Usage: check-block-format <db-path>")
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

	fmt.Println("=== Checking Block Format ===")
	
	// Look for number->hash entries (evm + 'n' prefix)
	prefix := []byte("evmn")
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		fmt.Printf("\nKey %d:\n", count+1)
		fmt.Printf("  Raw key: %s\n", hex.EncodeToString(key))
		fmt.Printf("  Key length: %d\n", len(key))
		fmt.Printf("  Value (hash): %s\n", hex.EncodeToString(value))
		fmt.Printf("  Value length: %d\n", len(value))
		
		// Try different interpretations
		if len(key) >= 12 {
			// Standard: evm(3) + n(1) + number(8)
			num1 := binary.BigEndian.Uint64(key[4:12])
			fmt.Printf("  Block number (8 bytes from offset 4): %d\n", num1)
		}
		
		if len(key) >= 13 {
			// Maybe: evm(3) + n(1) + number(8) + suffix
			num2 := binary.BigEndian.Uint64(key[4:12])
			fmt.Printf("  Block number (8 bytes from offset 4) + suffix: %d, suffix: %x\n", num2, key[12:])
		}
		
		// Check for headers too
		headerKey := append([]byte("evmh"), key[4:]...)
		if header, closer, err := db.Get(headerKey); err == nil {
			fmt.Printf("  Has header: yes (size: %d)\n", len(header))
			closer.Close()
		}
		
		count++
	}
}