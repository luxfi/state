package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <db-path>\n", os.Args[0])
		os.Exit(1)
	}

	dbPath := os.Args[1]
	log.Printf("Checking EVM canonical keys in database: %s", dbPath)

	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check for evmn prefix (canonical keys in EVM format)
	evmnPrefix := []byte("evmn")
	
	// Count keys with evmn prefix
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: evmnPrefix,
		UpperBound: append(evmnPrefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	count10byte := 0
	count9byte := 0
	var firstKey, lastKey []byte
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, evmnPrefix) {
			count++
			keyLen := len(key) - len(evmnPrefix)
			if keyLen == 10 && key[len(key)-1] == 0x6e {
				count10byte++
			} else if keyLen == 9 {
				count9byte++
			}
			
			if count == 1 {
				firstKey = make([]byte, len(key))
				copy(firstKey, key)
			}
			lastKey = make([]byte, len(key))
			copy(lastKey, key)
		}
	}

	log.Printf("Found %d evmn keys total", count)
	log.Printf("  - %d keys with 10-byte suffix (ending with 0x6e)", count10byte)
	log.Printf("  - %d keys with 9-byte suffix", count9byte)
	
	if count > 0 {
		log.Printf("First key: %x", firstKey)
		log.Printf("Last key: %x", lastKey)
	}

	// Try to read specific height
	height := uint64(1082780)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	
	// Try 9-byte format
	key9 := append(evmnPrefix, heightBytes...)
	val, closer, err := db.Get(key9)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Found canonical hash at height %d with 9-byte key: %x", height, val)
	} else {
		log.Printf("✗ No canonical hash at height %d with 9-byte key", height)
	}
	
	// Try 10-byte format
	key10 := append(key9, 0x6e)
	val2, closer2, err := db.Get(key10)
	if err == nil {
		defer closer2.Close()
		log.Printf("✓ Found canonical hash at height %d with 10-byte key: %x", height, val2)
	} else {
		log.Printf("✗ No canonical hash at height %d with 10-byte key", height)
	}
}