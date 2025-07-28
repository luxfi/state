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
	"github.com/luxfi/node/database"
	"github.com/luxfi/node/database/leveldb"
	"github.com/luxfi/node/database/prefixdb"
	"github.com/luxfi/node/database/versiondb"
)

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
		chainDBPath = flag.String("chain-db", "", "Path to chain database (e.g., runtime/mainnet/chainData/<ID>/db)")
		evmDBPath   = flag.String("evm-db", "", "Path to EVM database with blocks")
		tipHeight   = flag.Uint64("tip", 0, "Highest block height to import")
		batchSize   = flag.Int("batch", 10000, "Commit batch size")
	)
	flag.Parse()

	if *chainDBPath == "" || *evmDBPath == "" || *tipHeight == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err := importConsensus(*chainDBPath, *evmDBPath, *tipHeight, *batchSize); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}

func importConsensus(chainDBPath, evmDBPath string, tipHeight uint64, batchSize int) error {
	fmt.Printf("=== Consensus State Import ===\n")
	fmt.Printf("Chain DB: %s\n", chainDBPath)
	fmt.Printf("EVM DB: %s\n", evmDBPath)
	fmt.Printf("Tip Height: %d\n", tipHeight)
	fmt.Printf("Batch Size: %d\n\n", batchSize)

	// 1. Open the chain's logical DB using LevelDB (Avalanche uses this wrapper)
	fmt.Println("Opening chain database...")
	
	// Open using LevelDB wrapper which implements database.Database
	base, err := leveldb.New(chainDBPath, nil, log.New("leveldb", ""), 0)
	if err != nil {
		return fmt.Errorf("failed to open chain DB: %w", err)
	}
	defer base.Close()

	// Add prefix layer (state prefix)
	prefixed := prefixdb.New([]byte("state"), base)
	
	// Add version layer
	vdb := versiondb.New(prefixed)

	// 2. Open EVM database
	fmt.Println("Opening EVM database...")
	evmDB, err := pebble.Open(evmDBPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open EVM DB: %w", err)
	}
	defer evmDB.Close()

	// 3. Import blocks
	fmt.Printf("\nImporting blocks 0-%d...\n", tipHeight)
	startTime := time.Now()

	for height := uint64(0); height <= tipHeight; height++ {
		// Get canonical hash for this height
		numKey := make([]byte, 12)
		copy(numKey[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(numKey[4:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			log.Printf("Warning: No canonical hash for height %d: %v", height, err)
			continue
		}
		ethHash := make([]byte, len(hash))
		copy(ethHash, hash)
		closer.Close()

		// Import this block
		if err := acceptBlock(vdb, height, ethHash); err != nil {
			return fmt.Errorf("failed to accept block %d: %w", height, err)
		}

		// Progress reporting
		if height%1000 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(height) / elapsed.Seconds()
			eta := time.Duration(float64(tipHeight-height) / rate * float64(time.Second))
			fmt.Printf("  Height %d (%.0f blocks/sec, ETA: %v)\n", height, rate, eta)
		}

		// Commit periodically
		if height%uint64(batchSize) == 0 && height > 0 {
			if err := vdb.Commit(); err != nil {
				return fmt.Errorf("failed to commit at height %d: %w", height, err)
			}
		}
	}

	// Final commit
	fmt.Println("\nFinal commit...")
	if err := vdb.Commit(); err != nil {
		return fmt.Errorf("failed to final commit: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Import Complete ===\n")
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

	// 1. Store block bytes
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

	// 4. Update last accepted
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