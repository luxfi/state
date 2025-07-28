package main

import (
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Key prefixes used by Snowman consensus
var (
	blkBytesPrefix   = []byte{0x00}
	blkStatusPrefix  = []byte{0x01}
	blkIDIndexPrefix = []byte{0x02}
	lastAcceptedKey  = []byte("last_accepted")
	statusAccepted   = byte(0x02)
	
	// VersionDB metadata keys
	metadataKey        = []byte("metadata")
	currentRevisionKey = []byte("currentRevision")
)

func main() {
	var (
		evmPath   = flag.String("evm", "", "path to evm pebbledb (with blocks)")
		stateDB   = flag.String("state", "", "path to chain state leveldb") 
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
	fmt.Printf("=== Consensus State Replay (Simple) ===\n")
	fmt.Printf("EVM DB: %s\n", evmPath)
	fmt.Printf("State DB: %s\n", stateDBPath)
	fmt.Printf("Tip Height: %d\n", tipHeight)
	fmt.Printf("Batch Size: %d\n\n", batchSize)

	// 1. Open EVM database (pebbledb)
	fmt.Println("Opening EVM database...")
	evmDB, err := pebble.Open(evmPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open EVM DB: %w", err)
	}
	defer evmDB.Close()

	// 2. Create state DB directory
	if err := os.MkdirAll(stateDBPath, 0755); err != nil {
		return fmt.Errorf("failed to create state DB directory: %w", err)
	}

	// 3. Open state DB (leveldb) - we'll simulate versiondb behavior
	fmt.Println("Opening state database...")
	stateDB, err := leveldb.OpenFile(stateDBPath, &opt.Options{
		OpenFilesCacheCapacity: 256,
		BlockCacheCapacity:     256 * opt.MiB,
		WriteBuffer:            16 * opt.MiB,
	})
	if err != nil {
		return fmt.Errorf("failed to open state DB: %w", err)
	}
	defer stateDB.Close()

	// Initialize versiondb metadata
	currentRevision := uint64(1)
	if err := initializeVersionDB(stateDB, currentRevision); err != nil {
		return fmt.Errorf("failed to initialize version DB: %w", err)
	}

	// 4. Replay blocks
	fmt.Printf("\nReplaying blocks 0-%d...\n", tipHeight)
	startTime := time.Now()

	batch := new(leveldb.Batch)
	for height := uint64(0); height <= tipHeight; height++ {
		// Get canonical hash for this height
		// Key format: "evmn" + uint64 big endian
		numKey := make([]byte, 12)
		copy(numKey[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(numKey[4:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			// Try without prefix for older format
			numKey = numKey[3:] // Remove "evm" prefix
			hash, closer, err = evmDB.Get(numKey)
			if err != nil {
				log.Printf("Warning: No canonical hash for height %d", height)
				continue
			}
		}
		ethHash := make([]byte, len(hash))
		copy(ethHash, hash)
		closer.Close()

		// Process this block
		if err := addBlockToBatch(batch, height, ethHash, currentRevision); err != nil {
			return fmt.Errorf("failed to add block %d: %w", height, err)
		}

		// Progress reporting
		if height%1000 == 0 && height > 0 {
			elapsed := time.Since(startTime)
			rate := float64(height) / elapsed.Seconds()
			eta := time.Duration(float64(tipHeight-height) / rate * float64(time.Second))
			fmt.Printf("  Height %d (%.0f blocks/sec, ETA: %v)\n", height, rate, eta)
		}

		// Commit batch periodically
		if height%uint64(batchSize) == 0 && height > 0 {
			if err := stateDB.Write(batch, nil); err != nil {
				return fmt.Errorf("failed to write batch at height %d: %w", height, err)
			}
			batch.Reset()
		}
	}

	// Write final batch
	if batch.Len() > 0 {
		if err := stateDB.Write(batch, nil); err != nil {
			return fmt.Errorf("failed to write final batch: %w", err)
		}
	}

	// Write additional consensus keys
	fmt.Println("\nWriting additional consensus keys...")
	
	// Get the last block's hash
	numKey := make([]byte, 12)
	copy(numKey[:4], []byte("evmn"))
	binary.BigEndian.PutUint64(numKey[4:], tipHeight)
	
	lastHash, closer, err := evmDB.Get(numKey)
	if err == nil {
		defer closer.Close()
		lastSnowmanID := generateSnowmanID(lastHash, tipHeight)
		
		// Create final batch for additional keys
		finalBatch := new(leveldb.Batch)
		
		// Write lastAcceptedID with version suffix
		statePrefix := []byte("state")
		lastAcceptedIDKey := append(statePrefix, []byte("lastAcceptedID")...)
		lastAcceptedIDKey = append(lastAcceptedIDKey, makeRevisionSuffix(currentRevision)...)
		finalBatch.Put(lastAcceptedIDKey, lastSnowmanID[:])
		
		// Write lastAcceptedHeight
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, tipHeight)
		lastAcceptedHeightKey := append(statePrefix, []byte("lastAcceptedHeight")...)
		lastAcceptedHeightKey = append(lastAcceptedHeightKey, makeRevisionSuffix(currentRevision)...)
		finalBatch.Put(lastAcceptedHeightKey, heightBytes)
		
		// Write initialized flag
		initializedKey := append(statePrefix, []byte("initialized")...)
		initializedKey = append(initializedKey, makeRevisionSuffix(currentRevision)...)
		finalBatch.Put(initializedKey, []byte{0x01})
		
		if err := stateDB.Write(finalBatch, nil); err != nil {
			return fmt.Errorf("failed to write additional keys: %w", err)
		}
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Replay Complete ===\n")
	fmt.Printf("Total blocks: %d\n", tipHeight+1)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(tipHeight+1)/elapsed.Seconds())

	return nil
}

// Initialize version database metadata
func initializeVersionDB(db *leveldb.DB, revision uint64) error {
	batch := new(leveldb.Batch)

	// Set current revision
	revisionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(revisionBytes, revision)
	
	// Write metadata with "state" prefix
	statePrefix := []byte("state")
	batch.Put(append(statePrefix, metadataKey...), revisionBytes)
	batch.Put(append(statePrefix, currentRevisionKey...), revisionBytes)

	return db.Write(batch, nil)
}

// Add a block to the batch with proper versiondb formatting
func addBlockToBatch(batch *leveldb.Batch, height uint64, ethHash []byte, revision uint64) error {
	// Generate deterministic Snowman ID
	snowmanID := generateSnowmanID(ethHash, height)

	// Create simple block bytes
	blockBytes := createMinimalBlockBytes(snowmanID, height, ethHash)

	// State prefix for all keys
	statePrefix := []byte("state")
	
	// Revision suffix
	revisionSuffix := makeRevisionSuffix(revision)

	// 1. Store block bytes with version suffix
	bytesKey := append(statePrefix, blkBytesPrefix...)
	bytesKey = append(bytesKey, snowmanID[:]...)
	bytesKey = append(bytesKey, revisionSuffix...)
	batch.Put(bytesKey, blockBytes)

	// 2. Mark block as accepted with version suffix
	statusKey := append(statePrefix, blkStatusPrefix...)
	statusKey = append(statusKey, snowmanID[:]...)
	statusKey = append(statusKey, revisionSuffix...)
	batch.Put(statusKey, []byte{statusAccepted})

	// 3. Store height -> ID mapping with version suffix
	heightKey := append(statePrefix, blkIDIndexPrefix...)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	heightKey = append(heightKey, heightBytes...)
	heightKey = append(heightKey, revisionSuffix...)
	batch.Put(heightKey, snowmanID[:])

	// 4. Update last accepted with version suffix
	lastKey := append(statePrefix, lastAcceptedKey...)
	lastKey = append(lastKey, revisionSuffix...)
	batch.Put(lastKey, snowmanID[:])

	return nil
}

// Make revision suffix (8 bytes big endian)
func makeRevisionSuffix(revision uint64) []byte {
	suffix := make([]byte, 8)
	binary.BigEndian.PutUint64(suffix, revision)
	return suffix
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