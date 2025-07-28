package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: find-canonical-mappings <db-path>")
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

	fmt.Println("=== Finding Canonical Mappings ===")
	fmt.Println("Looking for keys with 32-byte hash values...")

	// Create iterator
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	count := 0
	hashValueCount := 0
	numberKeyCount := 0
	
	// Common prefixes in Ethereum databases
	knownPrefixes := map[string]string{
		"h": "headerPrefix",
		"H": "headerHashSuffix", 
		"n": "headerNumber",
		"b": "blockBodyPrefix",
		"r": "blockReceiptsPrefix",
		"l": "txLookupPrefix",
		"B": "bloomBitsPrefix",
	}

	for iter.First(); iter.Valid() && count < 10000; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Check if value is exactly 32 bytes (hash size)
		if len(value) == 32 {
			hashValueCount++
			if hashValueCount <= 10 {
				fmt.Printf("\nKey with 32-byte value:\n")
				fmt.Printf("  Key: %x\n", key[:min(64, len(key))])
				fmt.Printf("  Value (hash): %x\n", value)
				
				// Try to interpret key
				if len(key) > 0 {
					// Check for known prefix
					firstByte := string(key[0])
					if prefix, ok := knownPrefixes[firstByte]; ok {
						fmt.Printf("  Possible type: %s\n", prefix)
					}
					
					// Check if key contains a number (common for canonical mappings)
					if len(key) >= 9 {
						// Try parsing last 8 bytes as number
						possibleNum := binary.BigEndian.Uint64(key[len(key)-8:])
						if possibleNum < 10000000 { // reasonable block number
							fmt.Printf("  Possible block number in key: %d\n", possibleNum)
							numberKeyCount++
						}
					}
				}
			}
		}
		
		count++
	}
	
	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Scanned %d keys\n", count)
	fmt.Printf("Found %d keys with 32-byte hash values\n", hashValueCount)
	fmt.Printf("Found %d keys that might contain block numbers\n", numberKeyCount)
	
	// Now specifically look for number->hash mappings
	fmt.Println("\n=== Looking for number->hash patterns ===")
	
	// In geth, canonical hash key is: headerNumber + num (uint64)
	// Try to find keys that are exactly 9 bytes (1 byte prefix + 8 byte number)
	iter2, _ := db.NewIter(nil)
	defer iter2.Close()
	
	count = 0
	canonicalCount := 0
	for iter2.First(); iter2.Valid() && count < 50000; iter2.Next() {
		key := iter2.Key()
		value := iter2.Value()
		
		// Look for keys that could be canonical mappings
		// These are typically short keys (prefix + number) with 32-byte values
		if len(value) == 32 && len(key) >= 9 && len(key) <= 16 {
			// Try to parse potential block number from key
			foundNumber := false
			var blockNum uint64
			
			// Try different positions for the number
			if len(key) >= 9 {
				// Last 8 bytes as number
				blockNum = binary.BigEndian.Uint64(key[len(key)-8:])
				if blockNum < 10000000 {
					foundNumber = true
				}
			}
			
			if foundNumber {
				canonicalCount++
				if canonicalCount <= 5 {
					fmt.Printf("\nPotential canonical mapping:\n")
					fmt.Printf("  Key: %x (len=%d)\n", key, len(key))
					fmt.Printf("  Block number: %d\n", blockNum)
					fmt.Printf("  Hash: %x\n", value)
				}
			}
		}
		count++
	}
	
	fmt.Printf("\nFound %d potential canonical mappings\n", canonicalCount)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}