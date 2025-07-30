package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate blockchain data between formats",
		Long:  `Tools for migrating blockchain data between different storage formats`,
	}

	cmd.AddCommand(
		newMigrateNamespaceCmd(),
		newMigrateCanonicalCmd(),
		newMigrateConsensusCmd(),
		newMigrateFullCmd(),
	)

	return cmd
}

func newMigrateNamespaceCmd() *cobra.Command {
	var strip bool
	var prefix string

	cmd := &cobra.Command{
		Use:   "namespace <src-db> <dst-db>",
		Short: "Migrate data between namespaced and non-namespaced formats",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateNamespace(args[0], args[1], strip, prefix)
		},
	}

	cmd.Flags().BoolVar(&strip, "strip", false, "Strip namespace prefixes")
	cmd.Flags().StringVar(&prefix, "prefix", "evm:", "Namespace prefix to add/strip")

	return cmd
}

func newMigrateCanonicalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "canonical <src-db> <dst-db>",
		Short: "Migrate canonical mappings to proper format",
		Args:  cobra.ExactArgs(2),
		RunE:  runMigrateCanonical,
	}
}

func newMigrateConsensusCmd() *cobra.Command {
	var height uint64
	var hash string

	cmd := &cobra.Command{
		Use:   "consensus <db-path>",
		Short: "Migrate consensus state to proper format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateConsensus(args[0], height, hash)
		},
	}

	cmd.Flags().Uint64Var(&height, "height", 0, "Consensus height")
	cmd.Flags().StringVar(&hash, "hash", "", "Last accepted hash")

	return cmd
}

func newMigrateFullCmd() *cobra.Command {
	var skipNamespace bool
	var skipCanonical bool
	var skipConsensus bool

	cmd := &cobra.Command{
		Use:   "full <src-db> <dst-db>",
		Short: "Full migration pipeline for subnet to C-Chain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrateFull(args[0], args[1], skipNamespace, skipCanonical, skipConsensus)
		},
	}

	cmd.Flags().BoolVar(&skipNamespace, "skip-namespace", false, "Skip namespace stripping")
	cmd.Flags().BoolVar(&skipCanonical, "skip-canonical", false, "Skip canonical key migration")
	cmd.Flags().BoolVar(&skipConsensus, "skip-consensus", false, "Skip consensus state setup")

	return cmd
}

func runMigrateNamespace(srcPath, dstPath string, strip bool, prefix string) error {
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()

	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()

	if strip {
		fmt.Printf("Stripping namespace prefix '%s'...\n", prefix)
	} else {
		fmt.Printf("Adding namespace prefix '%s'...\n", prefix)
	}

	iter, _ := srcDB.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	batch := dstDB.NewBatch()
	count := 0
	skipped := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		var newKey []byte
		if strip {
			// Strip prefix if present
			if strings.HasPrefix(string(key), prefix) {
				newKey = key[len(prefix):]
			} else {
				newKey = key
				skipped++
			}
		} else {
			// Add prefix if not present
			if !strings.HasPrefix(string(key), prefix) {
				newKey = append([]byte(prefix), key...)
			} else {
				newKey = key
				skipped++
			}
		}

		if err := batch.Set(newKey, val, nil); err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Migrated %d keys...\n", count)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	fmt.Printf("✅ Migration complete: %d keys migrated, %d skipped\n", count, skipped)
	return nil
}

func runMigrateCanonical(cmd *cobra.Command, args []string) error {
	srcPath, dstPath := args[0], args[1]

	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()

	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()

	fmt.Println("Migrating canonical mappings...")

	// First, copy all non-canonical keys
	iter, _ := srcDB.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	batch := dstDB.NewBatch()
	count := 0
	canonicalCount := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		// Check if this is a canonical key needing migration
		if len(key) == 10 && key[0] == 0x68 && key[9] == 0x6e {
			// Convert 10-byte to 9-byte canonical key
			newKey := key[:9]
			if err := batch.Set(newKey, val, nil); err != nil {
				return fmt.Errorf("failed to set canonical key: %w", err)
			}
			canonicalCount++
		} else {
			// Copy as-is
			if err := batch.Set(key, val, nil); err != nil {
				return fmt.Errorf("failed to set key: %w", err)
			}
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Processed %d keys...\n", count)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	// Now handle evmn to canonical mappings
	fmt.Println("\nConverting evmn mappings to canonical format...")
	evmnCount := 0

	iter2, _ := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	defer iter2.Close()

	batch2 := dstDB.NewBatch()

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		hash := iter2.Value()

		if len(key) == 12 { // evmn + 8 bytes
			blockNum := binary.BigEndian.Uint64(key[4:])

			// Create 9-byte canonical key
			canonicalKey := make([]byte, 9)
			canonicalKey[0] = 0x68
			binary.BigEndian.PutUint64(canonicalKey[1:], blockNum)

			if err := batch2.Set(canonicalKey, hash, nil); err != nil {
				return fmt.Errorf("failed to set canonical mapping: %w", err)
			}

			evmnCount++
			if evmnCount%1000 == 0 {
				if err := batch2.Commit(pebble.Sync); err != nil {
					return fmt.Errorf("failed to commit evmn batch: %w", err)
				}
				batch2 = dstDB.NewBatch()
				fmt.Printf("Converted %d evmn mappings...\n", evmnCount)
			}
		}
	}

	if err := batch2.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final evmn batch: %w", err)
	}

	fmt.Printf("✅ Migration complete:\n")
	fmt.Printf("   Total keys: %d\n", count)
	fmt.Printf("   Canonical keys migrated: %d\n", canonicalCount)
	fmt.Printf("   EVMN mappings converted: %d\n", evmnCount)

	return nil
}

func runMigrateConsensus(dbPath string, height uint64, hashStr string) error {
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Setting up consensus state...")

	// If height not provided, find it
	if height == 0 {
		height = findHighestBlockInDB(db)
		fmt.Printf("Found highest block: %d\n", height)
	}

	// If hash not provided, find it
	var hash []byte
	if hashStr == "" {
		hash = findBlockHash(db, height)
		if hash == nil {
			return fmt.Errorf("could not find hash for block %d", height)
		}
		fmt.Printf("Found block hash: 0x%s\n", hex.EncodeToString(hash))
	} else {
		// Parse provided hash
		if strings.HasPrefix(hashStr, "0x") {
			hashStr = hashStr[2:]
		}
		hash = make([]byte, 32)
		if _, err := hex.Decode(hash, []byte(hashStr)); err != nil {
			return fmt.Errorf("invalid hash format: %w", err)
		}
	}

	// Write all consensus keys
	batch := db.NewBatch()

	// Height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	consensusKeys := map[string][]byte{
		"Height":             heightBytes,
		"LastAccepted":       hash,
		"lastAccepted":       hash,
		"consensus/accepted": hash,
		"LastBlock":          hash,
		"LastHeader":         hash,
	}

	for key, value := range consensusKeys {
		if err := batch.Set([]byte(key), value, nil); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit consensus state: %w", err)
	}

	fmt.Printf("✅ Consensus state migrated:\n")
	fmt.Printf("   Height: %d\n", height)
	fmt.Printf("   Hash: 0x%s\n", hex.EncodeToString(hash))

	return nil
}

func runMigrateFull(srcPath, dstPath string, skipNamespace, skipCanonical, skipConsensus bool) error {
	fmt.Println("Running full migration pipeline...")
	fmt.Printf("Source: %s\n", srcPath)
	fmt.Printf("Destination: %s\n", dstPath)

	// Step 1: Copy and strip namespace
	if !skipNamespace {
		fmt.Println("\n=== Step 1: Namespace Migration ===")
		tempDB := filepath.Join(filepath.Dir(dstPath), "temp-namespace")
		if err := runMigrateNamespace(srcPath, tempDB, true, "evm:"); err != nil {
			return fmt.Errorf("namespace migration failed: %w", err)
		}
		srcPath = tempDB
	}

	// Step 2: Migrate canonical keys
	if !skipCanonical {
		fmt.Println("\n=== Step 2: Canonical Key Migration ===")
		if err := runMigrateCanonical(&cobra.Command{}, []string{srcPath, dstPath}); err != nil {
			return fmt.Errorf("canonical migration failed: %w", err)
		}
	} else {
		// Just copy if skipping canonical migration
		if err := copyDatabase(srcPath, dstPath); err != nil {
			return fmt.Errorf("database copy failed: %w", err)
		}
	}

	// Step 3: Setup consensus state
	if !skipConsensus {
		fmt.Println("\n=== Step 3: Consensus State Setup ===")
		if err := runMigrateConsensus(dstPath, 0, ""); err != nil {
			return fmt.Errorf("consensus migration failed: %w", err)
		}
	}

	fmt.Println("\n✅ Full migration complete!")
	return nil
}

// Helper functions
func findHighestBlockInDB(db *pebble.DB) uint64 {
	maxBlock := uint64(0)

	// Check evmh prefix
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
		}
	}

	// Check Height key
	if heightBytes, closer, err := db.Get([]byte("Height")); err == nil {
		height := binary.BigEndian.Uint64(heightBytes)
		if height > maxBlock {
			maxBlock = height
		}
		closer.Close()
	}

	return maxBlock
}

func findBlockHash(db *pebble.DB, blockNum uint64) []byte {
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)

	// Try evmn key first
	evmnKey := append([]byte("evmn"), blockBytes...)
	if hash, closer, err := db.Get(evmnKey); err == nil {
		result := make([]byte, len(hash))
		copy(result, hash)
		closer.Close()
		return result
	}

	// Try canonical key (9-byte)
	canonicalKey := append([]byte{0x68}, blockBytes...)
	if hash, closer, err := db.Get(canonicalKey); err == nil {
		result := make([]byte, len(hash))
		copy(result, hash)
		closer.Close()
		return result
	}

	// Try 10-byte canonical key
	canonicalKey10 := append(canonicalKey, 0x6e)
	if hash, closer, err := db.Get(canonicalKey10); err == nil {
		result := make([]byte, len(hash))
		copy(result, hash)
		closer.Close()
		return result
	}

	return nil
}

func copyDatabase(src, dst string) error {
	srcDB, err := pebble.Open(src, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcDB.Close()

	dstDB, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination: %w", err)
	}
	defer dstDB.Close()

	iter, _ := srcDB.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	batch := dstDB.NewBatch()
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		if err := batch.Set(iter.Key(), iter.Value(), nil); err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	return nil
}
