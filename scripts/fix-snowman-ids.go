package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
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

// Simplified Snowman block structure for C-Chain
type snowmanBlock struct {
	ParentID     [32]byte
	Height       uint64
	Timestamp    uint64
	EthBlock     []byte // RLP encoded Ethereum block
}

func computeSnowmanID(ethBlock *types.Block, parentID [32]byte) ([32]byte, []byte, error) {
	// Create Snowman block wrapper
	smBlock := snowmanBlock{
		ParentID:  parentID,
		Height:    ethBlock.NumberU64(),
		Timestamp: ethBlock.Time(),
	}
	
	// RLP encode the Ethereum block
	ethRLP, err := rlp.EncodeToBytes(ethBlock)
	if err != nil {
		return [32]byte{}, nil, err
	}
	smBlock.EthBlock = ethRLP
	
	// Serialize the Snowman block
	snowmanBytes, err := rlp.EncodeToBytes(smBlock)
	if err != nil {
		return [32]byte{}, nil, err
	}
	
	// Compute SHA-256 hash as the Snowman ID
	id := sha256.Sum256(snowmanBytes)
	
	return id, snowmanBytes, nil
}

func getEthereumBlock(evmDB *pebble.DB, height uint64) (*types.Block, error) {
	// Get canonical hash for this height
	numKey := make([]byte, 12)
	copy(numKey[:4], []byte("evmn"))
	binary.BigEndian.PutUint64(numKey[4:], height)
	
	hash, closer, err := evmDB.Get(numKey)
	if err != nil {
		return nil, fmt.Errorf("no canonical hash for height %d: %v", height, err)
	}
	hashCopy := make([]byte, len(hash))
	copy(hashCopy, hash)
	closer.Close()
	
	// Get header
	headerKey := append([]byte("evmh"), append(hashCopy, make([]byte, 8)...)...)
	binary.BigEndian.PutUint64(headerKey[len(headerKey)-8:], height)
	
	headerRLP, closer, err := evmDB.Get(headerKey)
	if err != nil {
		return nil, fmt.Errorf("no header for height %d: %v", height, err)
	}
	headerBytes := make([]byte, len(headerRLP))
	copy(headerBytes, headerRLP)
	closer.Close()
	
	var header types.Header
	if err := rlp.DecodeBytes(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("failed to decode header: %v", err)
	}
	
	// Get body
	bodyKey := append([]byte("evmb"), append(hashCopy, make([]byte, 8)...)...)
	binary.BigEndian.PutUint64(bodyKey[len(bodyKey)-8:], height)
	
	bodyRLP, closer, err := evmDB.Get(bodyKey)
	if err != nil {
		// Genesis block might not have body
		if height == 0 {
			return types.NewBlockWithHeader(&header), nil
		}
		return nil, fmt.Errorf("no body for height %d: %v", height, err)
	}
	bodyBytes := make([]byte, len(bodyRLP))
	copy(bodyBytes, bodyRLP)
	closer.Close()
	
	var body types.Body
	if err := rlp.DecodeBytes(bodyBytes, &body); err != nil {
		return nil, fmt.Errorf("failed to decode body: %v", err)
	}
	
	return types.NewBlockWithHeader(&header).WithBody(body), nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: fix-snowman-ids <evm-db-path> <snowman-db-path>")
		fmt.Println()
		fmt.Println("Rewrites Snowman consensus state with correct block IDs")
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

	fmt.Println("=== Rewriting Snowman State with Correct IDs ===")

	// Find highest block
	var highestNum uint64
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
			}
		}
	}

	fmt.Printf("Highest block: %d\n", highestNum)

	// Process blocks and build Snowman state
	batch := snowmanDB.NewBatch()
	count := 0
	
	// Keep track of parent IDs
	blockIDs := make(map[uint64][32]byte)
	
	// Genesis block has special parent ID (all zeros)
	var genesisParentID [32]byte
	
	fmt.Println("\nProcessing blocks...")
	for height := uint64(0); height <= highestNum; height++ {
		// Get Ethereum block
		ethBlock, err := getEthereumBlock(evmDB, height)
		if err != nil {
			// Skip missing blocks
			continue
		}
		
		// Determine parent ID
		var parentID [32]byte
		if height == 0 {
			parentID = genesisParentID
		} else if prevID, ok := blockIDs[height-1]; ok {
			parentID = prevID
		} else {
			// If parent is missing, use a deterministic ID based on parent hash
			parentID = sha256.Sum256(ethBlock.ParentHash().Bytes())
		}
		
		// Compute Snowman block ID and bytes
		snowmanID, snowmanBytes, err := computeSnowmanID(ethBlock, parentID)
		if err != nil {
			log.Printf("Failed to compute Snowman ID for height %d: %v", height, err)
			continue
		}
		
		// Store the ID for next iteration
		blockIDs[height] = snowmanID
		
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
		if height < 3 {
			fmt.Printf("  Block %d: Snowman ID = %s\n", height, hex.EncodeToString(snowmanID[:]))
		}
	}
	
	// Set last accepted block
	if tipID, ok := blockIDs[highestNum]; ok {
		if err := batch.Set(lastAcceptedKey, tipID[:], pebble.Sync); err != nil {
			log.Fatalf("Failed to set last accepted: %v", err)
		}
		fmt.Printf("\nSet last accepted to block %d (ID: %s)\n", highestNum, hex.EncodeToString(tipID[:]))
	}
	
	// Final commit
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("\n=== Snowman State Rewritten ===\n")
	fmt.Printf("Total blocks processed: %d\n", count)
	fmt.Printf("Database path: %s\n", snowmanDBPath)
}