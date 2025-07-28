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
		fmt.Println("Usage: analyze-evmn-keys <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Look for evmn keys
	prefix := []byte("evmn")
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	fmt.Println("Analyzing evmn keys structure...")
	count := 0
	lengths := make(map[int]int)
	
	for iter.First(); iter.Valid() && count < 20; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		lengths[len(key)]++
		
		fmt.Printf("\nKey %d:\n", count)
		fmt.Printf("  Key hex: %s\n", hex.EncodeToString(key))
		fmt.Printf("  Key len: %d\n", len(key))
		fmt.Printf("  Value hex: %s\n", hex.EncodeToString(value))
		fmt.Printf("  Value len: %d\n", len(value))
		
		// Try to interpret the key structure
		if len(key) > 4 {
			fmt.Printf("  After 'evmn': %s\n", hex.EncodeToString(key[4:]))
			
			// Check if there's a uint64 at different positions
			if len(key) >= 12 {
				// Try reading uint64 from position 4
				num1 := binary.BigEndian.Uint64(key[4:12])
				fmt.Printf("  As uint64 at [4:12]: %d\n", num1)
			}
			
			if len(key) >= len(key)-8 {
				// Try reading uint64 from the end
				num2 := binary.BigEndian.Uint64(key[len(key)-8:])
				fmt.Printf("  As uint64 at end: %d\n", num2)
			}
		}
		
		count++
	}
	
	fmt.Printf("\n\nKey length distribution:\n")
	for l, c := range lengths {
		fmt.Printf("  Length %d: %d keys\n", l, c)
	}
}