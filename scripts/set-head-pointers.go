package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: set-head-pointers <db-path> <block-height>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	var blockHeight uint64
	fmt.Sscanf(os.Args[2], "%d", &blockHeight)

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Get the hash for this block number
	hashKey := append([]byte("evmn"), encodeBlockNumber(blockHeight)...)
	hashValue, closer, err := db.Get(hashKey)
	if err != nil {
		log.Fatal("Failed to get hash for block:", err)
	}
	blockHash := common.BytesToHash(hashValue)
	closer.Close()

	fmt.Printf("Found block %d with hash: %s\n", blockHeight, blockHash.Hex())

	// Set LastBlock
	if err := db.Set([]byte("LastBlock"), blockHash.Bytes(), pebble.Sync); err != nil {
		log.Fatal("Failed to set LastBlock:", err)
	}

	// Set LastHeader (same as LastBlock for simplicity)
	if err := db.Set([]byte("LastHeader"), blockHash.Bytes(), pebble.Sync); err != nil {
		log.Fatal("Failed to set LastHeader:", err)
	}

	// Set lastAccepted (for consensus)
	if err := db.Set([]byte("lastAccepted"), blockHash.Bytes(), pebble.Sync); err != nil {
		log.Fatal("Failed to set lastAccepted:", err)
	}

	// Set Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, blockHeight)
	if err := db.Set([]byte("Height"), heightBytes, pebble.Sync); err != nil {
		log.Fatal("Failed to set Height:", err)
	}

	fmt.Println("✅ Head pointers set successfully!")
	fmt.Printf("   LastBlock: %s\n", blockHash.Hex())
	fmt.Printf("   LastHeader: %s\n", blockHash.Hex())
	fmt.Printf("   lastAccepted: %s\n", blockHash.Hex())
	fmt.Printf("   Height: %d\n", blockHeight)
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
