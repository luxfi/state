package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

// Snowman block state prefixes from avalanchego/vms/components/avax/state
var (
	blkBytesPrefix   = []byte{0x00}
	blkStatusPrefix  = []byte{0x01}
	blkIDIndexPrefix = []byte{0x02}
	
	// Status values
	statusAccepted = byte(0x02)
	
	// Special keys
	lastAcceptedKey = []byte("last_accepted")
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: fill-snowman-state <evm-db-path> <snowman-db-path>")
		fmt.Println()
		fmt.Println("Fills Snowman consensus state from migrated EVM database")
		os.Exit(1)
	}

	evmDBPath := os.Args[1]
	snowmanDBPath := os.Args[2]

	// Open EVM database
	evmDB, err := pebble.Open(evmDBPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open EVM database: %v", err)
	}
	defer evmDB.Close()

	// Open Snowman database
	snowmanDB, err := pebble.Open(snowmanDBPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open Snowman database: %v", err)
	}
	defer snowmanDB.Close()

	fmt.Println("=== Filling Snowman State ===")

	// Find highest block in EVM database
	var highestNum uint64
	var highestHash []byte

	fmt.Println("Finding highest block...")
	iter, err := evmDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 12 && string(key[:4]) == "evmn" {
			num := binary.BigEndian.Uint64(key[4:])
			if num > highestNum {
				highestNum = num
				highestHash = make([]byte, len(iter.Value()))
				copy(highestHash, iter.Value())
			}
		}
	}

	fmt.Printf("Highest block: %d (hash: %s)\n", highestNum, hex.EncodeToString(highestHash))

	// Create batch for Snowman state
	batch := snowmanDB.NewBatch()
	count := 0

	fmt.Println("\nFilling block state...")
	// Process each block from 0 to highest
	for height := uint64(0); height <= highestNum; height++ {
		// Get canonical hash for this height
		numKey := make([]byte, 12)
		copy(numKey[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(numKey[4:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			continue // Skip missing blocks
		}
		hashCopy := make([]byte, len(hash))
		copy(hashCopy, hash)
		closer.Close()

		// For Snowman, we need the Avalanche block ID which is typically the hash
		blkID := hashCopy

		// 1. Mark block as accepted
		statusKey := append(blkStatusPrefix, blkID...)
		if err := batch.Set(statusKey, []byte{statusAccepted}, pebble.Sync); err != nil {
			log.Fatalf("Failed to set status: %v", err)
		}

		// 2. Store height -> block ID mapping
		heightKey := make([]byte, 9)
		copy(heightKey, blkIDIndexPrefix)
		binary.BigEndian.PutUint64(heightKey[1:], height)
		if err := batch.Set(heightKey, blkID, pebble.Sync); err != nil {
			log.Fatalf("Failed to set height index: %v", err)
		}

		// Note: We're not storing block bytes (blkBytesPrefix) as the VM can reconstruct from EVM DB

		count++
		if count%10000 == 0 {
			fmt.Printf("  Processed %d blocks...\n", count)
			// Commit batch periodically
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = snowmanDB.NewBatch()
		}
	}

	// Set last accepted block
	if highestHash != nil {
		if err := batch.Set(lastAcceptedKey, highestHash, pebble.Sync); err != nil {
			log.Fatalf("Failed to set last accepted: %v", err)
		}
		fmt.Printf("\nSet last accepted to block %d\n", highestNum)
	}

	// Final commit
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	fmt.Printf("\n=== Snowman State Filled ===\n")
	fmt.Printf("Total blocks processed: %d\n", count)
	fmt.Printf("Database path: %s\n", snowmanDBPath)
}