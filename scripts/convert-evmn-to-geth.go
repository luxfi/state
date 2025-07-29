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
	log.Printf("Converting EVM canonical keys to Geth format in-place")
	log.Printf("Database: %s", dbPath)

	// Open database for read-write
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Find all evmn keys and convert them
	evmnPrefix := []byte("evmn")
	
	// First, collect all evmn keys
	type keyValue struct {
		oldKey []byte
		value  []byte
	}
	var toConvert []keyValue
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: evmnPrefix,
		UpperBound: append(evmnPrefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if bytes.HasPrefix(key, evmnPrefix) {
			kv := keyValue{
				oldKey: make([]byte, len(key)),
				value:  make([]byte, len(iter.Value())),
			}
			copy(kv.oldKey, key)
			copy(kv.value, iter.Value())
			toConvert = append(toConvert, kv)
		}
	}

	log.Printf("Found %d evmn keys to convert", len(toConvert))

	// Convert all keys in a batch
	batch := db.NewBatch()
	for _, kv := range toConvert {
		// Delete old evmn key
		if err := batch.Delete(kv.oldKey, nil); err != nil {
			log.Printf("Failed to delete old key %x: %v", kv.oldKey, err)
			continue
		}
		
		// Create new geth key (0x68 + height bytes)
		// evmn key format: "evmn" + 8 bytes height
		heightBytes := kv.oldKey[4:] // Skip "evmn" prefix
		newKey := append([]byte{0x68}, heightBytes...)
		
		// Write with geth prefix
		if err := batch.Set(newKey, kv.value, nil); err != nil {
			log.Printf("Failed to set new key %x: %v", newKey, err)
			continue
		}
	}

	log.Printf("Committing %d key conversions...", len(toConvert))
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit batch: %v", err)
	}

	log.Printf("Successfully converted %d keys", len(toConvert))

	// Verify canonical hash at height 1082780
	height := uint64(1082780)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	canonicalKey := append([]byte{0x68}, heightBytes...)
	
	val, closer, err := db.Get(canonicalKey)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Verified canonical hash at height %d: %x", height, val)
	} else {
		log.Printf("✗ Could not find canonical hash at height %d", height)
	}
}