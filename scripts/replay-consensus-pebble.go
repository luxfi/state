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
		stateDB   = flag.String("state", "", "path to chain state pebbledb") 
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
	fmt.Printf("=== Consensus State Replay (PebbleDB) ===\n")
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

	// 2. Remove old state DB and create new one
	fmt.Println("Creating new state database...")
	os.RemoveAll(stateDBPath)
	
	// 3. Open state DB (pebbledb) - we'll simulate versiondb behavior
	stateDB, err := pebble.Open(stateDBPath, &pebble.Options{
		MaxOpenFiles:          512,
		MemTableSize:          64 * 1024 * 1024, // 64MB
		MemTableStopWritesThreshold: 4,
		L0CompactionThreshold: 2,
		L0StopWritesThreshold: 8,
		LBaseMaxBytes:         64 * 1024 * 1024, // 64MB
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

	batch := stateDB.NewBatch()
	defer batch.Close()
	
	for height := uint64(0); height <= tipHeight; height++ {
		// Get canonical hash for this height
		// Key format: "evmhn" + uint64 big endian
		numKey := append([]byte("evmhn"), make([]byte, 8)...)
		binary.BigEndian.PutUint64(numKey[5:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			log.Printf("Warning: No canonical hash for height %d", height)
			continue
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
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch at height %d: %w", height, err)
			}
			batch = stateDB.NewBatch()
		}
	}

	// Write final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	// Write additional consensus keys
	fmt.Println("\nWriting additional consensus keys...")
	
	// Get the last block's hash
	numKey := append([]byte("evmhn"), make([]byte, 8)...)
	binary.BigEndian.PutUint64(numKey[5:], tipHeight)
	
	lastHash, closer, err := evmDB.Get(numKey)
	if err == nil {
		defer closer.Close()
		lastSnowmanID := generateSnowmanID(lastHash, tipHeight)
		
		// Create final batch for additional keys
		finalBatch := stateDB.NewBatch()
		defer finalBatch.Close()
		
		// Write lastAcceptedID with version suffix
		statePrefix := []byte("state")
		lastAcceptedIDKey := append(statePrefix, []byte("lastAcceptedID")...)
		lastAcceptedIDKey = append(lastAcceptedIDKey, makeRevisionSuffix(currentRevision)...)
		if err := finalBatch.Set(lastAcceptedIDKey, lastSnowmanID[:], nil); err != nil {
			return fmt.Errorf("failed to set lastAcceptedID: %w", err)
		}
		
		// Write lastAcceptedHeight
		heightBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(heightBytes, tipHeight)
		lastAcceptedHeightKey := append(statePrefix, []byte("lastAcceptedHeight")...)
		lastAcceptedHeightKey = append(lastAcceptedHeightKey, makeRevisionSuffix(currentRevision)...)
		if err := finalBatch.Set(lastAcceptedHeightKey, heightBytes, nil); err != nil {
			return fmt.Errorf("failed to set lastAcceptedHeight: %w", err)
		}
		
		// Write initialized flag
		initializedKey := append(statePrefix, []byte("initialized")...)
		initializedKey = append(initializedKey, makeRevisionSuffix(currentRevision)...)
		if err := finalBatch.Set(initializedKey, []byte{0x01}, nil); err != nil {
			return fmt.Errorf("failed to set initialized: %w", err)
		}
		
		if err := finalBatch.Commit(pebble.Sync); err != nil {
			return fmt.Errorf("failed to commit additional keys: %w", err)
		}
		
		fmt.Printf("Set lastAcceptedID: %x\n", lastSnowmanID[:16])
		fmt.Printf("Set lastAcceptedHeight: %d\n", tipHeight)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Replay Complete ===\n")
	fmt.Printf("Total blocks: %d\n", tipHeight+1)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(tipHeight+1)/elapsed.Seconds())
	fmt.Printf("State DB location: %s\n", stateDBPath)

	return nil
}

// Initialize version database metadata
func initializeVersionDB(db *pebble.DB, revision uint64) error {
	batch := db.NewBatch()
	defer batch.Close()

	// Set current revision
	revisionBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(revisionBytes, revision)
	
	// Write metadata with "state" prefix
	statePrefix := []byte("state")
	if err := batch.Set(append(statePrefix, metadataKey...), revisionBytes, nil); err != nil {
		return err
	}
	if err := batch.Set(append(statePrefix, currentRevisionKey...), revisionBytes, nil); err != nil {
		return err
	}

	return batch.Commit(pebble.Sync)
}

// Add a block to the batch with proper versiondb formatting
func addBlockToBatch(batch *pebble.Batch, height uint64, ethHash []byte, revision uint64) error {
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
	if err := batch.Set(bytesKey, blockBytes, nil); err != nil {
		return err
	}

	// 2. Mark block as accepted with version suffix
	statusKey := append(statePrefix, blkStatusPrefix...)
	statusKey = append(statusKey, snowmanID[:]...)
	statusKey = append(statusKey, revisionSuffix...)
	if err := batch.Set(statusKey, []byte{statusAccepted}, nil); err != nil {
		return err
	}

	// 3. Store height -> ID mapping with version suffix
	heightKey := append(statePrefix, blkIDIndexPrefix...)
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	heightKey = append(heightKey, heightBytes...)
	heightKey = append(heightKey, revisionSuffix...)
	if err := batch.Set(heightKey, snowmanID[:], nil); err != nil {
		return err
	}

	// 4. Update last accepted with version suffix
	lastKey := append(statePrefix, lastAcceptedKey...)
	lastKey = append(lastKey, revisionSuffix...)
	if err := batch.Set(lastKey, snowmanID[:], nil); err != nil {
		return err
	}

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