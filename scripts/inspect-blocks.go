package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := "/home/z/.luxd/extracted-subnet-96369"
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	// Look specifically at 0x48 keys (number to hash mapping)
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x48},
		UpperBound: []byte{0x49},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer iter.Close()
	
	fmt.Println("Examining number->hash mappings (0x48 prefix):")
	count := 0
	var maxBlock uint64
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if count < 10 {
			fmt.Printf("\nKey: %s\n", hex.EncodeToString(key))
			fmt.Printf("Value: %s\n", hex.EncodeToString(value))
			
			// Try different interpretations
			if len(key) >= 9 {
				// Standard big-endian uint64
				num1 := binary.BigEndian.Uint64(key[1:9])
				fmt.Printf("  Block number (BE uint64): %d\n", num1)
			}
			
			if len(key) >= 5 {
				// Maybe it's uint32?
				num2 := binary.BigEndian.Uint32(key[1:5])
				fmt.Printf("  Block number (BE uint32): %d\n", num2)
			}
			
			// What if the actual block number is at the end?
			if len(key) >= 8 {
				num3 := binary.BigEndian.Uint64(key[len(key)-8:])
				fmt.Printf("  Block number (last 8 bytes): %d\n", num3)
			}
		}
		
		// Let's try to find max using the last 8 bytes approach
		if len(key) >= 8 {
			blockNum := binary.BigEndian.Uint64(key[len(key)-8:])
			if blockNum < 1000000 && blockNum > maxBlock { // Sanity check
				maxBlock = blockNum
			}
		}
		
		count++
		if count > 20 {
			break
		}
	}
	
	fmt.Printf("\nTotal 0x48 keys examined: %d\n", count)
	fmt.Printf("Reasonable max block found: %d\n", maxBlock)
	
	// Also check 0x68 keys (headers)
	fmt.Println("\n\nExamining headers (0x68 prefix):")
	iter2, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer iter2.Close()
	
	headerCount := 0
	for iter2.First(); iter2.Valid(); iter2.Next() {
		if headerCount < 5 {
			key := iter2.Key()
			fmt.Printf("\nHeader key: %s\n", hex.EncodeToString(key))
			
			// Header keys often have block number encoded
			if len(key) >= 9 {
				num := binary.BigEndian.Uint64(key[1:9])
				fmt.Printf("  Possible block number: %d\n", num)
			}
		}
		headerCount++
	}
	
	fmt.Printf("\nTotal headers: %d\n", headerCount)
}