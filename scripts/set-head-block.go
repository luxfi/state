package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

// Key prefixes for head pointers
var (
	// Head block keys (these should have "evm" prefix already)
	headHeaderKey = []byte("evmLastHeader")
	headBlockKey  = []byte("evmLastBlock")
	headFastKey   = []byte("evmLastFast")
	
	// Alternative keys used by geth
	headHeaderHashKey = []byte("evmh")
	headBlockHashKey  = []byte("evmH")
	headFastHashKey   = []byte("evmF")
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: set-head-block <db-path> <block-number>")
		fmt.Println()
		fmt.Println("This tool sets the head block pointers in the database")
		fmt.Println("Example: set-head-block ./chaindata 14644")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	blockNum, err := parseBlockNumber(os.Args[2])
	if err != nil {
		log.Fatalf("Invalid block number: %v", err)
	}

	fmt.Printf("=== Setting Head Block Pointers ===\n")
	fmt.Printf("Database: %s\n", dbPath)
	fmt.Printf("Block Number: %d\n", blockNum)
	fmt.Println()

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Find the block hash for this number
	blockHash, err := findBlockHash(db, blockNum)
	if err != nil {
		log.Fatalf("Failed to find block hash: %v", err)
	}

	fmt.Printf("Found block hash: %s\n", hex.EncodeToString(blockHash))

	// Set head pointers
	if err := setHeadPointers(db, blockNum, blockHash); err != nil {
		log.Fatalf("Failed to set head pointers: %v", err)
	}

	fmt.Println("\n=== Head Block Pointers Set Successfully ===")
}

func parseBlockNumber(s string) (uint64, error) {
	var blockNum uint64
	_, err := fmt.Sscanf(s, "%d", &blockNum)
	return blockNum, err
}

func findBlockHash(db *pebble.DB, blockNum uint64) ([]byte, error) {
	// Construct the number->hash key
	// Format: evm + 'n' + 8-byte big-endian block number
	key := make([]byte, 0, 3+1+8)
	key = append(key, []byte("evm")...)
	key = append(key, 'n')
	numBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(numBytes, blockNum)
	key = append(key, numBytes...)

	// Get the hash
	hash, closer, err := db.Get(key)
	if err != nil {
		return nil, fmt.Errorf("block %d not found: %w", blockNum, err)
	}
	defer closer.Close()

	// Make a copy of the hash
	result := make([]byte, len(hash))
	copy(result, hash)
	
	return result, nil
}

func setHeadPointers(db *pebble.DB, blockNum uint64, blockHash []byte) error {
	batch := db.NewBatch()
	defer batch.Close()

	// Encode block number
	numBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(numBytes, blockNum)

	// Set text-based keys
	if err := batch.Set(headHeaderKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set LastHeader: %w", err)
	}
	fmt.Printf("Set LastHeader: %s -> %s\n", string(headHeaderKey), hex.EncodeToString(blockHash))

	if err := batch.Set(headBlockKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set LastBlock: %w", err)
	}
	fmt.Printf("Set LastBlock: %s -> %s\n", string(headBlockKey), hex.EncodeToString(blockHash))

	if err := batch.Set(headFastKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set LastFast: %w", err)
	}
	fmt.Printf("Set LastFast: %s -> %s\n", string(headFastKey), hex.EncodeToString(blockHash))

	// Set single-letter keys used by rawdb
	if err := batch.Set(headHeaderHashKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set head header hash: %w", err)
	}
	fmt.Printf("Set head header hash: evmh -> %s\n", hex.EncodeToString(blockHash))

	if err := batch.Set(headBlockHashKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set head block hash: %w", err)
	}
	fmt.Printf("Set head block hash: evmH -> %s\n", hex.EncodeToString(blockHash))

	if err := batch.Set(headFastHashKey, blockHash, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set head fast hash: %w", err)
	}
	fmt.Printf("Set head fast hash: evmF -> %s\n", hex.EncodeToString(blockHash))

	// Also set the canonical hash for block 0 if not already set
	genesisKey := make([]byte, 0, 3+1+8)
	genesisKey = append(genesisKey, []byte("evm")...)
	genesisKey = append(genesisKey, 'n')
	genesisNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(genesisNumBytes, 0)
	genesisKey = append(genesisKey, genesisNumBytes...)

	if _, _, err := db.Get(genesisKey); err != nil {
		// Genesis block not found, let's check if we have a genesis header
		fmt.Println("\nNote: Genesis canonical mapping not found, chain might need reinitialization")
	}

	// Commit the batch
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}

	return nil
}