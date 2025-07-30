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
		fmt.Println("Usage: fix-consensus-keys <evm-db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}
	defer db.Close()

	// Get the canonical hash for the highest block
	highestBlock := uint64(1082780)
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, highestBlock)

	canonicalKey := []byte{0x68}
	canonicalKey = append(canonicalKey, blockBytes...)
	canonicalKey = append(canonicalKey, 0x6e)

	var headHash []byte
	if val, closer, err := db.Get(canonicalKey); err == nil {
		headHash = make([]byte, len(val))
		copy(headHash, val)
		closer.Close()
		fmt.Printf("Found head block hash: %x\n", headHash)
	} else {
		log.Fatal("Could not find canonical hash for highest block")
	}

	// The VM expects these keys to have specific values
	// For luxd/Avalanche, the block ID should be the hash encoded in a specific format

	// Update Height (already correct)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, highestBlock)
	if err := db.Set([]byte("Height"), heightBytes, nil); err != nil {
		log.Fatal("Failed to set Height:", err)
	}
	fmt.Printf("Set Height to %d\n", highestBlock)

	// Update LastBlock with the actual block hash
	// In Coreth/geth, this should be the hash of the last block
	if err := db.Set([]byte("LastBlock"), headHash, nil); err != nil {
		log.Fatal("Failed to set LastBlock:", err)
	}
	fmt.Printf("Set LastBlock to %x\n", headHash)

	// Update lastAccepted with the same hash
	if err := db.Set([]byte("lastAccepted"), headHash, nil); err != nil {
		log.Fatal("Failed to set lastAccepted:", err)
	}
	fmt.Printf("Set lastAccepted to %x\n", headHash)

	// Also set the head hash key that geth might look for
	headHashKey := []byte("LastHash")
	if err := db.Set(headHashKey, headHash, nil); err != nil {
		log.Fatal("Failed to set LastHash:", err)
	}

	// Set the head block number key
	headNumberKey := []byte("LastNumber")
	if err := db.Set(headNumberKey, heightBytes, nil); err != nil {
		log.Fatal("Failed to set LastNumber:", err)
	}

	fmt.Println("\nConsensus keys fixed!")
}
