package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	var src = flag.String("src", "", "source subnet database")
	flag.Parse()

	if *src == "" {
		flag.Usage()
		log.Fatal("--src is required")
	}

	db, err := pebble.Open(*src, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Analyzing Hash Structure ===")
	
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	// Sample some 'H' and 'n' keys to understand structure
	hSamples := 0
	nSamples := 0
	
	for iter.First(); iter.Valid() && (hSamples < 5 || nSamples < 5); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) < 41 {
			continue
		}
		
		logicalKey := key[33:len(key)-8]
		
		// Sample 'H' keys
		if logicalKey[0] == 'H' && hSamples < 5 {
			hSamples++
			fmt.Printf("\n'H' key sample %d:\n", hSamples)
			fmt.Printf("  Full key: %x\n", key)
			fmt.Printf("  Logical key: %x\n", logicalKey)
			fmt.Printf("  Hash part (len=%d): %x\n", len(logicalKey[1:]), logicalKey[1:])
			if len(value) == 8 {
				number := binary.BigEndian.Uint64(value)
				fmt.Printf("  Block number: %d\n", number)
			}
			fmt.Printf("  Value: %x\n", value)
		}
		
		// Sample 'n' keys
		if logicalKey[0] == 'n' && nSamples < 5 {
			nSamples++
			fmt.Printf("\n'n' key sample %d:\n", nSamples)
			fmt.Printf("  Full key: %x\n", key)
			fmt.Printf("  Logical key: %x\n", logicalKey)
			fmt.Printf("  Hash part (len=%d): %x\n", len(logicalKey[1:]), logicalKey[1:])
			fmt.Printf("  Value: %x\n", value)
			
			// The hash part might actually encode the block number differently
			// Let's check if it's a number encoded in the first 8 bytes
			if len(logicalKey) >= 9 {
				possibleNum := binary.BigEndian.Uint64(logicalKey[1:9])
				fmt.Printf("  Possible number in first 8 bytes: %d\n", possibleNum)
			}
		}
	}
	
	// Now let's analyze the pattern more systematically
	fmt.Println("\n=== Pattern Analysis ===")
	
	// Look for headers to understand block numbers
	iter2, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter2.Close()
	
	headerSamples := 0
	for iter2.First(); iter2.Valid() && headerSamples < 5; iter2.Next() {
		key := iter2.Key()
		value := iter2.Value()
		
		if len(key) < 41 {
			continue
		}
		
		logicalKey := key[33:len(key)-8]
		
		// Look for header keys
		if logicalKey[0] == 'h' && len(logicalKey) > 9 {
			headerSamples++
			fmt.Printf("\nHeader key sample %d:\n", headerSamples)
			fmt.Printf("  Logical key: %x\n", logicalKey)
			
			// Headers typically have format: h + block_number + hash
			// Try to extract block number
			blockNum := binary.BigEndian.Uint64(logicalKey[1:9])
			fmt.Printf("  Block number: %d\n", blockNum)
			fmt.Printf("  Hash part: %x\n", logicalKey[9:])
			
			// Show a bit of the header data
			if len(value) > 100 {
				fmt.Printf("  Header data (first 100 bytes): %x...\n", value[:100])
			}
		}
	}
}