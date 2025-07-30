package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chain-id>\n", os.Args[0])
		os.Exit(1)
	}

	chainID := os.Args[1]
	dbPath := fmt.Sprintf("/home/z/lux-node-data/chainData/%s/db/pebbledb", chainID)

	log.Printf("Checking C-Chain database: %s", dbPath)

	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Look for canonical hash at height 1082780
	height := uint64(1082780)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	// Try standard geth canonical key
	canonicalKey := append([]byte{0x68}, heightBytes...)
	val, closer, err := db.Get(canonicalKey)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Found canonical hash with standard geth key (0x68): %x", val)
		return
	} else {
		log.Printf("✗ No canonical hash with standard geth key")
	}

	// Try to find any canonical-like keys
	log.Printf("\nSearching for canonical-like keys around height %d:", height)

	// Create bounds around the height
	startHeight := height - 10
	endHeight := height + 10

	for h := startHeight; h <= endHeight; h++ {
		hBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(hBytes, h)

		// Try 0x68 prefix
		key := append([]byte{0x68}, hBytes...)
		if val, closer, err := db.Get(key); err == nil {
			log.Printf("  Height %d: found hash %x", h, val)
			closer.Close()
		}
	}

	// Check what keys exist in the database
	log.Printf("\nFirst 20 keys in database:")
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 20; iter.Next() {
		key := iter.Key()
		log.Printf("  Key[%d]: %x (len=%d)", count, key, len(key))
		if len(key) > 0 {
			log.Printf("    First byte: 0x%02x", key[0])
		}
		count++
	}
}
