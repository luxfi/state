package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

// Snowman block state prefixes
var (
	blkBytesPrefix   = []byte{0x00}
	blkStatusPrefix  = []byte{0x01}
	blkIDIndexPrefix = []byte{0x02}
	
	// Status values
	statusAccepted = byte(0x02)
	
	// Special keys
	lastAcceptedKey = []byte("last_accepted")
)

// For C-Chain, we'll generate deterministic Snowman IDs based on Ethereum hashes
// This is a simplified approach that creates valid Snowman block structure
func generateSnowmanID(ethHash []byte, height uint64) [32]byte {
	// Create a deterministic ID by combining height and hash
	data := make([]byte, 8+len(ethHash))
	binary.BigEndian.PutUint64(data[:8], height)
	copy(data[8:], ethHash)
	return sha256.Sum256(data)
}

// Create minimal Snowman block bytes that will be accepted
func createSnowmanBlockBytes(snowmanID [32]byte, parentID [32]byte, height uint64, ethHash []byte) []byte {
	// Simplified Snowman block structure
	// Format: [parentID(32)] [height(8)] [timestamp(8)] [ethHash(32)] [id(32)]
	blockBytes := make([]byte, 32+8+8+32+32)
	
	offset := 0
	// Parent ID
	copy(blockBytes[offset:offset+32], parentID[:])
	offset += 32
	
	// Height
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height)
	offset += 8
	
	// Timestamp (use height as timestamp for simplicity)
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height*12) // ~12 seconds per block
	offset += 8
	
	// Ethereum hash
	copy(blockBytes[offset:offset+32], ethHash)
	offset += 32
	
	// Snowman ID
	copy(blockBytes[offset:offset+32], snowmanID[:])
	
	return blockBytes
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: fix-snowman-simple <evm-db-path> <snowman-db-path>")
		fmt.Println()
		fmt.Println("Creates Snowman consensus state with deterministic IDs")
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

	// Clear and recreate Snowman database
	os.RemoveAll(snowmanDBPath)
	snowmanDB, err := pebble.Open(snowmanDBPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open Snowman database: %v", err)
	}
	defer snowmanDB.Close()

	fmt.Println("=== Creating Snowman State with Deterministic IDs ===")

	// Find highest block
	var highestNum uint64
	var highestHash []byte
	
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

	// Process blocks and build Snowman state
	batch := snowmanDB.NewBatch()
	count := 0
	
	// Keep track of Snowman IDs
	blockIDs := make(map[uint64][32]byte)
	
	// Genesis parent ID
	var genesisParentID [32]byte // All zeros
	
	fmt.Println("\nProcessing blocks...")
	for height := uint64(0); height <= highestNum; height++ {
		// Get canonical hash for this height
		numKey := make([]byte, 12)
		copy(numKey[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(numKey[4:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			continue // Skip missing blocks
		}
		ethHash := make([]byte, len(hash))
		copy(ethHash, hash)
		closer.Close()
		
		// Generate deterministic Snowman ID
		snowmanID := generateSnowmanID(ethHash, height)
		blockIDs[height] = snowmanID
		
		// Determine parent ID
		var parentID [32]byte
		if height == 0 {
			parentID = genesisParentID
		} else if prevID, ok := blockIDs[height-1]; ok {
			parentID = prevID
		} else {
			// Generate parent ID from previous hash
			parentNumKey := make([]byte, 12)
			copy(parentNumKey[:4], []byte("evmn"))
			binary.BigEndian.PutUint64(parentNumKey[4:], height-1)
			
			if parentHash, closer, err := evmDB.Get(parentNumKey); err == nil {
				parentID = generateSnowmanID(parentHash, height-1)
				closer.Close()
			}
		}
		
		// Create Snowman block bytes
		snowmanBytes := createSnowmanBlockBytes(snowmanID, parentID, height, ethHash)
		
		// 1. Store block bytes
		bytesKey := append(blkBytesPrefix, snowmanID[:]...)
		if err := batch.Set(bytesKey, snowmanBytes, pebble.Sync); err != nil {
			log.Fatalf("Failed to set block bytes: %v", err)
		}
		
		// 2. Mark block as accepted
		statusKey := append(blkStatusPrefix, snowmanID[:]...)
		if err := batch.Set(statusKey, []byte{statusAccepted}, pebble.Sync); err != nil {
			log.Fatalf("Failed to set status: %v", err)
		}
		
		// 3. Store height -> ID mapping
		heightKey := make([]byte, 9)
		copy(heightKey, blkIDIndexPrefix)
		binary.BigEndian.PutUint64(heightKey[1:], height)
		if err := batch.Set(heightKey, snowmanID[:], pebble.Sync); err != nil {
			log.Fatalf("Failed to set height index: %v", err)
		}
		
		count++
		if count%10000 == 0 {
			fmt.Printf("  Processed %d blocks...\n", count)
			// Commit batch periodically
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = snowmanDB.NewBatch()
		}
		
		// Show first few IDs
		if height < 3 || height == highestNum {
			fmt.Printf("  Block %d: ETH hash=%s, Snowman ID=%s\n", 
				height, 
				hex.EncodeToString(ethHash[:16])+"...",
				hex.EncodeToString(snowmanID[:16])+"...")
		}
	}
	
	// Set last accepted block
	if tipID, ok := blockIDs[highestNum]; ok {
		if err := batch.Set(lastAcceptedKey, tipID[:], pebble.Sync); err != nil {
			log.Fatalf("Failed to set last accepted: %v", err)
		}
		fmt.Printf("\nSet last accepted to block %d\n", highestNum)
		fmt.Printf("  Snowman ID: %s\n", hex.EncodeToString(tipID[:]))
		
		// Also try the key format that consensus might expect
		if err := batch.Set([]byte("lastAcceptedID"), tipID[:], pebble.Sync); err != nil {
			log.Printf("Warning: Failed to set lastAcceptedID: %v", err)
		}
	}
	
	// Final commit
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("\n=== Snowman State Created ===\n")
	fmt.Printf("Total blocks processed: %d\n", count)
	fmt.Printf("Database path: %s\n", snowmanDBPath)
	
	// Verify by reading back some entries
	fmt.Println("\nVerifying database...")
	if val, closer, err := snowmanDB.Get(lastAcceptedKey); err == nil {
		fmt.Printf("last_accepted: %s\n", hex.EncodeToString(val))
		closer.Close()
	}
	
	// Check height 0 mapping
	heightKey := make([]byte, 9)
	copy(heightKey, blkIDIndexPrefix)
	binary.BigEndian.PutUint64(heightKey[1:], 0)
	if val, closer, err := snowmanDB.Get(heightKey); err == nil {
		fmt.Printf("Height 0 -> ID: %s\n", hex.EncodeToString(val))
		closer.Close()
	}
}