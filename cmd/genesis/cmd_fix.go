package main

import (
	"encoding/binary"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newFixCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Fix various data issues in blockchain databases", 
		Long:  `Commands to fix and clean up blockchain data issues`,
	}

	cmd.AddCommand(
		newFixCanonicalCmd(),
		newFixConsensusCmd(),
	)

	return cmd
}

func newFixCanonicalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "canonical <db-path>",
		Short: "Fix canonical key format (convert 10-byte to 9-byte)",
		Args:  cobra.ExactArgs(1),
		RunE:  runFixCanonical,
	}
}

func newFixConsensusCmd() *cobra.Command {
	var height uint64
	var hash string
	
	cmd := &cobra.Command{
		Use:   "consensus <db-path>",
		Short: "Write consensus state markers (Height and LastAccepted)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFixConsensus(args[0], height, hash)
		},
	}
	
	cmd.Flags().Uint64Var(&height, "height", 0, "Block height to set")
	cmd.Flags().StringVar(&hash, "hash", "", "Block hash to set as LastAccepted")
	cmd.MarkFlagRequired("height")
	cmd.MarkFlagRequired("hash")
	
	return cmd
}

func runFixCanonical(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Fixing canonical key format...")
	
	// Find all 10-byte canonical keys and convert to 9-byte
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()
	
	batch := db.NewBatch()
	fixed := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()
		
		// Check if this is a 10-byte canonical key (0x68 + 8 bytes + 0x6e)
		if len(key) == 10 && key[9] == 0x6e {
			// Create 9-byte key (remove the 0x6e suffix)
			newKey := make([]byte, 9)
			copy(newKey, key[:9])
			
			// Write new key
			batch.Set(newKey, val, nil)
			
			// Delete old key
			batch.Delete(key, nil)
			
			fixed++
			
			if fixed % 1000 == 0 {
				if err := batch.Commit(pebble.Sync); err != nil {
					return fmt.Errorf("failed to commit batch: %w", err)
				}
				batch = db.NewBatch()
				fmt.Printf("Fixed %d keys...\n", fixed)
			}
		}
	}
	
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}
	
	fmt.Printf("✅ Fixed %d canonical keys to 9-byte format\n", fixed)
	
	return nil
}

func runFixConsensus(dbPath string, height uint64, hashStr string) error {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Writing consensus state markers...")
	
	// Parse hash
	if len(hashStr) < 2 {
		return fmt.Errorf("invalid hash: %s", hashStr)
	}
	if hashStr[:2] == "0x" {
		hashStr = hashStr[2:]
	}
	
	hash := make([]byte, 32)
	n, err := fmt.Sscanf(hashStr, "%64x", &hash)
	if err != nil || n != 1 {
		return fmt.Errorf("invalid hash format: %s", hashStr)
	}
	
	batch := db.NewBatch()
	
	// Write Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	
	if err := batch.Set([]byte("Height"), heightBytes, nil); err != nil {
		return fmt.Errorf("failed to set Height: %w", err)
	}
	
	// Write LastAccepted
	if err := batch.Set([]byte("LastAccepted"), hash, nil); err != nil {
		return fmt.Errorf("failed to set LastAccepted: %w", err)
	}
	
	// Also write other consensus keys that might be needed
	if err := batch.Set([]byte("lastAccepted"), hash, nil); err != nil {
		return fmt.Errorf("failed to set lastAccepted: %w", err)
	}
	
	if err := batch.Set([]byte("consensus/accepted"), hash, nil); err != nil {
		return fmt.Errorf("failed to set consensus/accepted: %w", err)
	}
	
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit batch: %w", err)
	}
	
	fmt.Printf("✅ Wrote consensus state:\n")
	fmt.Printf("   Height: %d (0x%x)\n", height, height)
	fmt.Printf("   LastAccepted: 0x%x\n", hash)
	
	return nil
}