package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

var fixCanonicalCmd = &cobra.Command{
	Use:   "fix-canonical [db-path] [height] [hash]",
	Short: "Fix canonical hash mapping for a specific height",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := args[0]
		height := args[1]
		blockHash := args[2]

		// Convert height to uint64
		var blockHeight uint64
		if _, err := fmt.Sscanf(height, "%d", &blockHeight); err != nil {
			return fmt.Errorf("invalid height: %w", err)
		}

		// Convert hash to bytes
		hashBytes, err := hex.DecodeString(blockHash)
		if err != nil {
			return fmt.Errorf("invalid hash: %w", err)
		}

		fmt.Printf("Fixing canonical hash mapping:\n")
		fmt.Printf("  Height: %d\n", blockHeight)
		fmt.Printf("  Hash: 0x%x\n", hashBytes)

		// Open database
		opts := &pebble.Options{}
		db, err := pebble.Open(filepath.Clean(dbPath), opts)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		defer db.Close()

		// Create the canonical hash key
		// Format: 0x68 + 8-byte height (big endian) + 0x6e
		key := make([]byte, 10)
		key[0] = 0x68 // 'h' prefix
		binary.BigEndian.PutUint64(key[1:9], blockHeight)
		key[9] = 0x6e // 'n' suffix

		// Write the canonical hash mapping
		if err := db.Set(key, hashBytes, pebble.Sync); err != nil {
			return fmt.Errorf("failed to write canonical hash: %w", err)
		}

		fmt.Printf("Successfully wrote canonical hash mapping:\n")
		fmt.Printf("  Key: %x\n", key)
		fmt.Printf("  Value: %x\n", hashBytes)

		// Verify the write
		value, closer, err := db.Get(key)
		if err != nil {
			return fmt.Errorf("failed to verify write: %w", err)
		}
		defer closer.Close()

		fmt.Printf("\nVerification:\n")
		fmt.Printf("  Read back: %x\n", value)
		if hex.EncodeToString(value) == hex.EncodeToString(hashBytes) {
			fmt.Printf("  ✓ Canonical hash successfully written\n")
		} else {
			fmt.Printf("  ✗ Verification failed\n")
		}

		// Also check if we can read the header at this height
		headerKey := append([]byte{0x48}, key[1:9]...)
		headerKey = append(headerKey, hashBytes...)

		if headerValue, closer, err := db.Get(headerKey); err == nil {
			defer closer.Close()
			fmt.Printf("\nHeader found at this height:\n")
			fmt.Printf("  Key: %x\n", headerKey)
			fmt.Printf("  Size: %d bytes\n", len(headerValue))
		}

		return nil
	},
}
