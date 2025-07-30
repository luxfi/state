package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: fix-canonical-9bytes <db-path> <height> <hash>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	height := os.Args[2]
	blockHash := os.Args[3]

	// Convert height to uint64
	var blockHeight uint64
	if _, err := fmt.Sscanf(height, "%d", &blockHeight); err != nil {
		fmt.Printf("Invalid height: %v\n", err)
		os.Exit(1)
	}

	// Convert hash to bytes
	hashBytes, err := hex.DecodeString(blockHash)
	if err != nil {
		fmt.Printf("Invalid hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fixing canonical hash mapping:\n")
	fmt.Printf("  Height: %d\n", blockHeight)
	fmt.Printf("  Hash: 0x%x\n", hashBytes)

	// Open database
	opts := &pebble.Options{}
	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create the canonical hash key - 9 bytes total
	// Format: 0x68 + 8-byte height (big endian)
	key := make([]byte, 9)
	key[0] = 0x68 // 'h' prefix
	binary.BigEndian.PutUint64(key[1:], blockHeight)

	// Write the canonical hash mapping
	if err := db.Set(key, hashBytes, pebble.Sync); err != nil {
		fmt.Printf("Failed to write canonical hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote canonical hash mapping:\n")
	fmt.Printf("  Key: %x (length: %d)\n", key, len(key))
	fmt.Printf("  Value: %x\n", hashBytes)

	// Verify the write
	value, closer, err := db.Get(key)
	if err != nil {
		fmt.Printf("Failed to verify write: %v\n", err)
		os.Exit(1)
	}
	defer closer.Close()

	fmt.Printf("\nVerification:\n")
	fmt.Printf("  Read back: %x\n", value)
	if hex.EncodeToString(value) == hex.EncodeToString(hashBytes) {
		fmt.Printf("  ✓ Canonical hash successfully written\n")
	} else {
		fmt.Printf("  ✗ Verification failed\n")
	}
}
