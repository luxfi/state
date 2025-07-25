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
	
	// Count different key types
	prefixCounts := make(map[byte]int)
	var maxBlockNum uint64
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal(err)
	}
	defer iter.Close()
	
	fmt.Println("Inspecting first 20 keys:")
	count := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) > 0 {
			prefixCounts[key[0]]++
			
			// Show first 20 keys
			if count < 20 {
				fmt.Printf("Key[%d]: %s (prefix: 0x%02x, len: %d)\n", count, hex.EncodeToString(key), key[0], len(key))
				if key[0] == 0x48 && len(key) >= 9 {
					blockNum := binary.BigEndian.Uint64(key[1:9])
					fmt.Printf("  -> Block number: %d\n", blockNum)
				}
				count++
			}
			
			// Track highest block number from 0x48 keys
			if key[0] == 0x48 && len(key) >= 9 {
				blockNum := binary.BigEndian.Uint64(key[1:9])
				if blockNum > maxBlockNum {
					maxBlockNum = blockNum
				}
			}
		}
	}
	
	fmt.Printf("\nPrefix counts:\n")
	for prefix, count := range prefixCounts {
		fmt.Printf("  0x%02x: %d keys\n", prefix, count)
	}
	
	fmt.Printf("\nHighest block number found: %d\n", maxBlockNum)
}