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

// Key prefixes for geth/coreth database
var (
	// EVM prefix for C-Chain
	evmPrefix = []byte("evm")
	
	// Ethereum database schema prefixes
	headerPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	headerHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	blockBodyPrefix    = []byte("b") // blockBodyPrefix + num (uint64 big endian) + hash -> block body
	blockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	
	// Head tracking keys (with evm prefix)
	headHeaderKey = append(evmPrefix, []byte("LastHeader")...)
	headBlockKey  = append(evmPrefix, []byte("LastBlock")...)
	headFastKey   = append(evmPrefix, []byte("LastFast")...)
)

func main() {
	var (
		statePath  = flag.String("state", "", "path to state database (with trie nodes)")
		outputPath = flag.String("output", "", "path to output blockchain database")
		endBlock   = flag.Uint64("blocks", 1082780, "number of blocks to create")
		chainID    = flag.Uint64("chainid", 96369, "chain ID")
	)
	flag.Parse()

	if *statePath == "" || *outputPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := createSyntheticBlockchain(*statePath, *outputPath, *endBlock, *chainID); err != nil {
		log.Fatalf("Failed to create blockchain: %v", err)
	}
}

func createSyntheticBlockchain(statePath, outputPath string, numBlocks, chainID uint64) error {
	fmt.Println("=== Creating Synthetic Blockchain ===")
	fmt.Printf("State DB: %s\n", statePath)
	fmt.Printf("Output DB: %s\n", outputPath)
	fmt.Printf("Blocks to create: %d\n", numBlocks)
	fmt.Printf("Chain ID: %d\n", chainID)

	// Open state database (read-only)
	stateDB, err := pebble.Open(statePath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open state DB: %w", err)
	}
	defer stateDB.Close()

	// Create output database
	os.RemoveAll(outputPath)
	outDB, err := pebble.Open(outputPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to create output DB: %w", err)
	}
	defer outDB.Close()

	// First, copy all state data
	fmt.Println("\nCopying state data...")
	if err := copyStateData(stateDB, outDB); err != nil {
		return fmt.Errorf("failed to copy state: %w", err)
	}

	// Now create synthetic blockchain
	fmt.Println("\nCreating synthetic blockchain...")
	startTime := time.Now()

	// Create genesis block (block 0)
	genesisHash := createGenesisBlock(outDB, chainID)
	fmt.Printf("Genesis hash: %x\n", genesisHash)

	// Create blocks 1 through numBlocks
	prevHash := genesisHash
	batch := outDB.NewBatch()
	
	for blockNum := uint64(1); blockNum <= numBlocks; blockNum++ {
		// Create synthetic block
		blockHash := createSyntheticBlock(batch, blockNum, prevHash)
		prevHash = blockHash

		// Commit batch periodically
		if blockNum%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch at block %d: %w", blockNum, err)
			}
			batch = outDB.NewBatch()

			// Progress report
			elapsed := time.Since(startTime)
			rate := float64(blockNum) / elapsed.Seconds()
			eta := time.Duration(float64(numBlocks-blockNum) / rate * float64(time.Second))
			fmt.Printf("  Block %d (%.0f blocks/sec, ETA: %v)\n", blockNum, rate, eta)
		}
	}

	// Final batch commit
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	// Write head pointers
	fmt.Println("\nWriting head pointers...")
	if err := writeHeadPointers(outDB, prevHash, numBlocks); err != nil {
		return fmt.Errorf("failed to write head pointers: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Complete ===\n")
	fmt.Printf("Created %d blocks in %v\n", numBlocks, elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(numBlocks)/elapsed.Seconds())

	return nil
}

func copyStateData(src, dst *pebble.DB) error {
	iter, err := src.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	batch := dst.NewBatch()
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		// Copy key-value pair
		if err := batch.Set(iter.Key(), iter.Value(), nil); err != nil {
			return err
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return err
			}
			batch = dst.NewBatch()
			fmt.Printf("  Copied %d state entries...\n", count)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return err
	}

	fmt.Printf("  Total state entries copied: %d\n", count)
	return nil
}

func createGenesisBlock(db *pebble.DB, chainID uint64) []byte {
	// Create a simple genesis block
	blockNum := uint64(0)
	
	// Create genesis hash (deterministic based on chain ID)
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, chainID)
	hash := sha256.Sum256(data)

	// Create minimal header data (RLP-encoded in production)
	header := createMinimalHeader(blockNum, []byte{}, hash[:])
	
	// Write header
	headerKey := append(evmPrefix, headerPrefix...)
	headerKey = append(headerKey, encodeBlockNumber(blockNum)...)
	headerKey = append(headerKey, hash[:]...)
	db.Set(headerKey, header, nil)

	// Write header hash mapping (canonical chain)
	// Format: "evmhn" + block number -> hash
	canonicalKey := append(evmPrefix, []byte("hn")...)
	canonicalKey = append(canonicalKey, encodeBlockNumber(blockNum)...)
	db.Set(canonicalKey, hash[:], nil)

	// Write empty body
	bodyKey := append(evmPrefix, blockBodyPrefix...)
	bodyKey = append(bodyKey, encodeBlockNumber(blockNum)...)
	bodyKey = append(bodyKey, hash[:]...)
	db.Set(bodyKey, []byte{0xc0}, nil) // Empty RLP list

	// Write empty receipts
	receiptsKey := append(evmPrefix, blockReceiptsPrefix...)
	receiptsKey = append(receiptsKey, encodeBlockNumber(blockNum)...)
	receiptsKey = append(receiptsKey, hash[:]...)
	db.Set(receiptsKey, []byte{0xc0}, nil) // Empty RLP list

	return hash[:]
}

func createSyntheticBlock(batch *pebble.Batch, blockNum uint64, parentHash []byte) []byte {
	// Create deterministic hash for this block
	data := make([]byte, 8+32)
	binary.BigEndian.PutUint64(data[:8], blockNum)
	copy(data[8:], parentHash)
	hash := sha256.Sum256(data)

	// Create minimal header
	header := createMinimalHeader(blockNum, parentHash, hash[:])

	// Write header
	headerKey := append(evmPrefix, headerPrefix...)
	headerKey = append(headerKey, encodeBlockNumber(blockNum)...)
	headerKey = append(headerKey, hash[:]...)
	batch.Set(headerKey, header, nil)

	// Write header hash mapping (canonical chain)
	// Format: "evmhn" + block number -> hash
	canonicalKey := append(evmPrefix, []byte("hn")...)
	canonicalKey = append(canonicalKey, encodeBlockNumber(blockNum)...)
	batch.Set(canonicalKey, hash[:], nil)

	// Write empty body
	bodyKey := append(evmPrefix, blockBodyPrefix...)
	bodyKey = append(bodyKey, encodeBlockNumber(blockNum)...)
	bodyKey = append(bodyKey, hash[:]...)
	batch.Set(bodyKey, []byte{0xc0}, nil) // Empty RLP list

	// Write empty receipts
	receiptsKey := append(evmPrefix, blockReceiptsPrefix...)
	receiptsKey = append(receiptsKey, encodeBlockNumber(blockNum)...)
	receiptsKey = append(receiptsKey, hash[:]...)
	batch.Set(receiptsKey, []byte{0xc0}, nil) // Empty RLP list

	return hash[:]
}

func createMinimalHeader(blockNum uint64, parentHash, hash []byte) []byte {
	// This creates a simplified header structure
	// In production, this would be proper RLP encoding
	header := make([]byte, 0, 200)
	
	// Add parent hash (32 bytes)
	header = append(header, parentHash...)
	if len(parentHash) < 32 {
		header = append(header, make([]byte, 32-len(parentHash))...)
	}
	
	// Add uncles hash (32 bytes, empty)
	header = append(header, make([]byte, 32)...)
	
	// Add coinbase (20 bytes, zero address)
	header = append(header, make([]byte, 20)...)
	
	// Add state root (32 bytes, could use actual state root)
	header = append(header, make([]byte, 32)...)
	
	// Add tx root (32 bytes, empty)
	header = append(header, make([]byte, 32)...)
	
	// Add receipt root (32 bytes, empty)
	header = append(header, make([]byte, 32)...)
	
	// Add bloom (256 bytes, empty)
	header = append(header, make([]byte, 256)...)
	
	// Add difficulty (big.Int, 0)
	header = append(header, 0x80) // RLP encoding of 0
	
	// Add number
	numBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(numBytes, blockNum)
	header = append(header, numBytes...)
	
	// Add gas limit (8 bytes)
	gasLimit := make([]byte, 8)
	binary.BigEndian.PutUint64(gasLimit, 8000000)
	header = append(header, gasLimit...)
	
	// Add gas used (8 bytes, 0)
	header = append(header, make([]byte, 8)...)
	
	// Add timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	header = append(header, timestamp...)
	
	// Add extra data
	header = append(header, []byte("synthetic")...)
	
	// Add mix digest (32 bytes)
	header = append(header, make([]byte, 32)...)
	
	// Add nonce (8 bytes)
	header = append(header, make([]byte, 8)...)

	return header
}

func writeHeadPointers(db *pebble.DB, lastHash []byte, lastNum uint64) error {
	batch := db.NewBatch()

	// Write head block hash
	batch.Set(headBlockKey, lastHash, nil)
	batch.Set(headHeaderKey, lastHash, nil)
	batch.Set(headFastKey, lastHash, nil)

	// Also write the canonical hash for easy lookup
	// Format: "evmhn" + block number -> hash
	canonicalKey := append(evmPrefix, []byte("hn")...)
	canonicalKey = append(canonicalKey, encodeBlockNumber(lastNum)...)
	batch.Set(canonicalKey, lastHash, nil)

	return batch.Commit(pebble.Sync)
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}