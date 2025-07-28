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
		fmt.Println("Usage: find-headers <db-path>")
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

	fmt.Println("=== Finding Headers ===")
	
	// Look for headers (evm + 'h' prefix)
	prefix := []byte("evmh")
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	var highestBlock uint64
	count := 0

	for iter.First(); iter.Valid() && count < 20; iter.Next() {
		key := iter.Key()
		
		fmt.Printf("\nHeader %d:\n", count+1)
		fmt.Printf("  Key: %s\n", hex.EncodeToString(key))
		fmt.Printf("  Key length: %d\n", len(key))
		
		// Headers are: evm(3) + h(1) + number(8) + hash(32)
		if len(key) >= 44 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			blockHash := key[12:44]
			fmt.Printf("  Block number: %d\n", blockNum)
			fmt.Printf("  Block hash: %s\n", hex.EncodeToString(blockHash))
			
			if blockNum > highestBlock && blockNum < 100000000 {
				highestBlock = blockNum
			}
		}
		
		count++
	}

	// Also check from the end
	fmt.Println("\n=== Checking last headers ===")
	if iter.Last() {
		for i := 0; i < 10 && iter.Valid(); i++ {
			key := iter.Key()
			if len(key) >= 44 {
				blockNum := binary.BigEndian.Uint64(key[4:12])
				if blockNum < 100000000 {
					fmt.Printf("  Block %d\n", blockNum)
					if blockNum > highestBlock {
						highestBlock = blockNum
					}
				}
			}
			iter.Prev()
		}
	}

	fmt.Printf("\nHighest block with header: %d\n", highestBlock)
	fmt.Printf("Total headers shown: %d\n", count)
}