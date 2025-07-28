package main

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/node/database"
	"github.com/luxfi/node/database/leveldb"
	"github.com/luxfi/node/database/prefixdb"
	"github.com/luxfi/node/database/versiondb"
	"github.com/luxfi/node/utils/logging"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Mock logger
type mockLogger struct{}

func (m mockLogger) Debug(msg string, args ...interface{}) {}
func (m mockLogger) Info(msg string, args ...interface{})  {}
func (m mockLogger) Warn(msg string, args ...interface{})  {}
func (m mockLogger) Error(msg string, args ...interface{}) {}
func (m mockLogger) Fatal(msg string, args ...interface{}) { log.Fatal(msg) }
func (m mockLogger) Trace(msg string, args ...interface{}) {}
func (m mockLogger) Verbo(msg string, args ...interface{}) {}

// Key prefixes used by Snowman consensus
var (
	blkBytesPrefix   = []byte{0x00}
	blkStatusPrefix  = []byte{0x01}
	blkIDIndexPrefix = []byte{0x02}
	lastAcceptedKey  = []byte("last_accepted")
	statusAccepted   = byte(0x02)
)

func main() {
	var (
		evmPath   = flag.String("evm", "", "path to evm database (with blocks)")
		stateDB   = flag.String("state", "", "path to chain state database") 
		tipHeight = flag.Uint64("tip", 0, "highest canonical height")
		batchSize = flag.Int("batch", 10000, "commit batch size")
	)
	flag.Parse()

	if *evmPath == "" || *stateDB == "" || *tipHeight == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err := replayConsensus(*evmPath, *stateDB, *tipHeight, *batchSize); err != nil {
		log.Fatalf("Replay failed: %v", err)
	}
}

func replayConsensus(evmPath, stateDBPath string, tipHeight uint64, batchSize int) error {
	fmt.Printf("=== Consensus State Replay ===\n")
	fmt.Printf("EVM DB: %s\n", evmPath)
	fmt.Printf("State DB: %s\n", stateDBPath)
	fmt.Printf("Tip Height: %d\n", tipHeight)
	fmt.Printf("Batch Size: %d\n\n", batchSize)

	// 1. Open EVM database using geth's rawdb
	fmt.Println("Opening EVM database...")
	evmDB, err := rawdb.NewLevelDBDatabase(evmPath, 0, 0, "", false)
	if err != nil {
		return fmt.Errorf("failed to open EVM DB: %w", err)
	}
	defer evmDB.Close()

	// 2. Open state DB via versiondb (same as AvalancheGo)
	fmt.Println("Opening state database with versiondb...")
	
	// Ensure directory exists
	if err := os.MkdirAll(stateDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create state DB directory: %w", err)
	}

	// Open using leveldb wrapper
	logger := mockLogger{}
	base, err := leveldb.New(stateDBPath, nil, logger, 0)
	if err != nil {
		return fmt.Errorf("failed to open state DB: %w", err)
	}
	defer base.Close()

	// Add prefix layer (state prefix)
	prefixed := prefixdb.New([]byte("state"), base)
	
	// Add version layer - this is critical!
	vdb := versiondb.New(prefixed)

	// 3. Replay blocks
	fmt.Printf("\nReplaying blocks 0-%d...\n", tipHeight)
	startTime := time.Now()

	for height := uint64(0); height <= tipHeight; height++ {
		// Get canonical hash for this height
		hash := rawdb.ReadCanonicalHash(evmDB, height)
		if hash == (common.Hash{}) {
			log.Printf("Warning: No canonical hash for height %d", height)
			continue
		}

		// Process this block
		if err := acceptBlock(vdb, height, hash.Bytes()); err != nil {
			return fmt.Errorf("failed to accept block %d: %w", height, err)
		}

		// Progress reporting
		if height%1000 == 0 && height > 0 {
			elapsed := time.Since(startTime)
			rate := float64(height) / elapsed.Seconds()
			eta := time.Duration(float64(tipHeight-height) / rate * float64(time.Second))
			fmt.Printf("  Height %d (%.0f blocks/sec, ETA: %v)\n", height, rate, eta)
		}

		// Commit periodically - IMPORTANT for versiondb!
		if height%uint64(batchSize) == 0 && height > 0 {
			if err := vdb.Commit(); err != nil {
				return fmt.Errorf("failed to commit at height %d: %w", height, err)
			}
		}
	}

	// Final commit - this writes metadata and currentRevision
	fmt.Println("\nFinal commit...")
	if err := vdb.Commit(); err != nil {
		return fmt.Errorf("failed to final commit: %w", err)
	}

	// Also write some additional keys that consensus might expect
	fmt.Println("Writing additional consensus keys...")
	
	// Get the last block's Snowman ID
	lastHash := rawdb.ReadCanonicalHash(evmDB, tipHeight)
	if lastHash != (common.Hash{}) {
		lastSnowmanID := generateSnowmanID(lastHash.Bytes(), tipHeight)
		
		// Write lastAcceptedID key (without version suffix, vdb will add it)
		if err := vdb.Put([]byte("lastAcceptedID"), lastSnowmanID[:]); err != nil {
			log.Printf("Warning: failed to write lastAcceptedID: %v", err)
		}
		
		// Write lastAcceptedHeight
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, tipHeight)
		if err := vdb.Put([]byte("lastAcceptedHeight"), heightBytes); err != nil {
			log.Printf("Warning: failed to write lastAcceptedHeight: %v", err)
		}
		
		// Write initialized flag
		if err := vdb.Put([]byte("initialized"), []byte{0x01}); err != nil {
			log.Printf("Warning: failed to write initialized: %v", err)
		}
		
		// Commit these additional keys
		if err := vdb.Commit(); err != nil {
			return fmt.Errorf("failed to commit additional keys: %w", err)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Replay Complete ===\n")
	fmt.Printf("Total blocks: %d\n", tipHeight+1)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(tipHeight+1)/elapsed.Seconds())

	return nil
}

// acceptBlock mimics what the consensus engine does when accepting a block
func acceptBlock(vdb database.Database, height uint64, ethHash []byte) error {
	// Generate deterministic Snowman ID
	snowmanID := generateSnowmanID(ethHash, height)

	// Create simple block bytes
	blockBytes := createMinimalBlockBytes(snowmanID, height, ethHash)

	// 1. Store block bytes (versiondb will add revision suffix automatically)
	bytesKey := append(blkBytesPrefix, snowmanID[:]...)
	if err := vdb.Put(bytesKey, blockBytes); err != nil {
		return fmt.Errorf("failed to put block bytes: %w", err)
	}

	// 2. Mark block as accepted
	statusKey := append(blkStatusPrefix, snowmanID[:]...)
	if err := vdb.Put(statusKey, []byte{statusAccepted}); err != nil {
		return fmt.Errorf("failed to put status: %w", err)
	}

	// 3. Store height -> ID mapping
	heightKey := make([]byte, 9)
	copy(heightKey, blkIDIndexPrefix)
	binary.BigEndian.PutUint64(heightKey[1:], height)
	if err := vdb.Put(heightKey, snowmanID[:]); err != nil {
		return fmt.Errorf("failed to put height index: %w", err)
	}

	// 4. Update last accepted (this gets overwritten each time, which is what we want)
	if err := vdb.Put(lastAcceptedKey, snowmanID[:]); err != nil {
		return fmt.Errorf("failed to put last accepted: %w", err)
	}

	return nil
}

// Generate deterministic Snowman ID
func generateSnowmanID(ethHash []byte, height uint64) [32]byte {
	data := make([]byte, 8+len(ethHash))
	binary.BigEndian.PutUint64(data[:8], height)
	copy(data[8:], ethHash)
	return sha256.Sum256(data)
}

// Create minimal block bytes that consensus can parse
func createMinimalBlockBytes(snowmanID [32]byte, height uint64, ethHash []byte) []byte {
	// Simple format: [height(8)] [timestamp(8)] [ethHash(32)] [id(32)]
	blockBytes := make([]byte, 8+8+32+32)
	
	offset := 0
	// Height
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height)
	offset += 8
	
	// Timestamp (use height*12 for ~12 second blocks)
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height*12)
	offset += 8
	
	// Ethereum hash
	copy(blockBytes[offset:offset+32], ethHash)
	offset += 32
	
	// Snowman ID
	copy(blockBytes[offset:offset+32], snowmanID[:])
	
	return blockBytes
}