package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/cockroachdb/pebble"
)

func main() {
	var (
		src          = flag.String("src", "", "source subnet database path")
		dst          = flag.String("dst", "", "destination directory for migrated data")
		chainID      = flag.Uint64("chain-id", 96369, "chain ID (96369 for LUX mainnet)")
		cacheDir     = flag.String("cache", ".tmp/migration-cache", "cache directory")
		skipIfExists = flag.Bool("cache-skip", true, "skip if cached result exists")
	)
	flag.Parse()

	if *src == "" || *dst == "" {
		flag.Usage()
		log.Fatal("Both --src and --dst are required")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(*cacheDir, 0755); err != nil {
		log.Fatalf("Failed to create cache directory: %v", err)
	}

	// Step 1: Migrate with EVM prefix (cached)
	migratedDB := filepath.Join(*cacheDir, fmt.Sprintf("migrated-%d", *chainID), "pebbledb")
	if *skipIfExists {
		if _, err := os.Stat(filepath.Join(migratedDB, "CURRENT")); err == nil {
			fmt.Printf("Using cached migrated database at %s\n", migratedDB)
		} else {
			if err := migrateWithEVMPrefix(*src, migratedDB); err != nil {
				log.Fatalf("Migration failed: %v", err)
			}
		}
	} else {
		if err := migrateWithEVMPrefix(*src, migratedDB); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
	}

	// Step 2: Fix evmn keys
	fixedDB := filepath.Join(*cacheDir, fmt.Sprintf("fixed-%d", *chainID), "pebbledb")
	if *skipIfExists {
		if _, err := os.Stat(filepath.Join(fixedDB, "CURRENT")); err == nil {
			fmt.Printf("Using cached fixed database at %s\n", fixedDB)
		} else {
			if err := copyAndFixDatabase(migratedDB, fixedDB); err != nil {
				log.Fatalf("Fix evmn keys failed: %v", err)
			}
		}
	} else {
		if err := copyAndFixDatabase(migratedDB, fixedDB); err != nil {
			log.Fatalf("Fix evmn keys failed: %v", err)
		}
	}

	// Step 3: Find max height
	maxHeight, err := findMaxHeight(fixedDB)
	if err != nil {
		log.Fatalf("Failed to find max height: %v", err)
	}
	fmt.Printf("Maximum block height: %d\n", maxHeight)

	// Step 4: Create final output
	finalDB := filepath.Join(*dst, "chaindata", "pebbledb")
	if err := os.MkdirAll(filepath.Dir(finalDB), 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Copy fixed database to final location
	if err := copyDatabase(fixedDB, finalDB); err != nil {
		log.Fatalf("Failed to copy to final location: %v", err)
	}

	// Step 5: Create consensus state
	consensusDB := filepath.Join(*dst, "statedata", "pebbledb")
	if err := createConsensusState(finalDB, consensusDB, maxHeight); err != nil {
		log.Fatalf("Failed to create consensus state: %v", err)
	}

	fmt.Println("\n=== Migration Complete ===")
	fmt.Printf("Chain Data: %s\n", finalDB)
	fmt.Printf("State Data: %s\n", consensusDB)
	fmt.Printf("Max Height: %d\n", maxHeight)
	fmt.Printf("\nTo test with luxd:\n")
	fmt.Printf("./luxd --chain-data-dir=%s\n", *dst)
}

func migrateWithEVMPrefix(src, dst string) error {
	fmt.Printf("\n=== Step 1: Migrating with EVM prefix ===\n")
	fmt.Printf("Source: %s\n", src)
	fmt.Printf("Destination: %s\n", dst)

	start := time.Now()

	srcDB, err := pebble.Open(src, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcDB.Close()

	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}

	dstDB, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstDB.Close()

	iter, err := srcDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	evmPrefix := []byte("evm")
	batch := dstDB.NewBatch()
	count := 0
	stats := make(map[byte]int)

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 34 {
			continue
		}

		// Strip 33-byte namespace prefix
		actualKey := key[33:]

		// Create new key with evm prefix
		newKey := make([]byte, len(evmPrefix)+len(actualKey))
		copy(newKey, evmPrefix)
		copy(newKey[len(evmPrefix):], actualKey)

		// Track statistics
		if len(actualKey) > 0 {
			stats[actualKey[0]]++
		}

		if err := batch.Set(newKey, iter.Value(), nil); err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Migrated %d keys...\n", count)
		}
	}

	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	fmt.Printf("Migrated %d keys in %s\n", count, time.Since(start))
	fmt.Printf("Key types: h=%d H=%d b=%d r=%d n=%d s=%d\n",
		stats[0x68], stats[0x48], stats[0x62], stats[0x72], stats[0x6e], stats[0x73])

	return nil
}

func copyAndFixDatabase(src, dst string) error {
	fmt.Printf("\n=== Step 2: Fixing evmn keys ===\n")

	// First copy the database
	if err := copyDatabase(src, dst); err != nil {
		return fmt.Errorf("failed to copy database: %w", err)
	}

	// Open destination for fixing
	db, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Collect hash->number mappings
	hashToNumber := make(map[string]uint64)
	prefix := []byte("evmH")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		if len(key) > 4 && len(value) == 8 {
			hash := key[4:]
			number := binary.BigEndian.Uint64(value)
			hashToNumber[string(hash)] = number
		}
	}
	iter.Close()

	fmt.Printf("Found %d hash->number mappings\n", len(hashToNumber))

	// Create proper evmn keys
	batch := db.NewBatch()
	count := 0
	for hash, number := range hashToNumber {
		key := make([]byte, 12)
		copy(key, []byte("evmn"))
		binary.BigEndian.PutUint64(key[4:], number)

		if err := batch.Set(key, []byte(hash), nil); err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}

		count++
		if count%100 == 0 {
			if err := batch.Commit(nil); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = db.NewBatch()
		}
	}

	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	// Remove old format evmn keys
	oldCount := 0
	prefix = []byte("evmn")
	iter2, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	batch = db.NewBatch()
	keysToDelete := [][]byte{}
	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		if len(key) > 12 {
			keysToDelete = append(keysToDelete, append([]byte{}, key...))
		}
	}
	iter2.Close()

	for _, key := range keysToDelete {
		if err := batch.Delete(key, nil); err != nil {
			return fmt.Errorf("failed to delete key: %w", err)
		}
		oldCount++

		if oldCount%100 == 0 {
			if err := batch.Commit(nil); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = db.NewBatch()
		}
	}

	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	fmt.Printf("Created %d canonical keys, removed %d old format keys\n", count, oldCount)
	return nil
}

func findMaxHeight(dbPath string) (uint64, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var maxHeight uint64
	prefix := []byte("evmH")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		value := iter.Value()
		if len(value) == 8 {
			height := binary.BigEndian.Uint64(value)
			if height > maxHeight {
				maxHeight = height
			}
		}
	}

	return maxHeight, nil
}

func copyDatabase(src, dst string) error {
	// Simple file-by-file copy
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDatabase(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

func createConsensusState(evmDB, stateDB string, maxHeight uint64) error {
	fmt.Printf("\n=== Step 3: Creating consensus state ===\n")

	// This is a simplified version - in production, use replay-consensus-pebble
	if err := os.MkdirAll(stateDB, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create a marker file to indicate consensus state exists
	marker := filepath.Join(stateDB, "CONSENSUS_STATE")
	data := fmt.Sprintf("max_height=%d\ncreated=%s\n", maxHeight, time.Now())
	if err := os.WriteFile(marker, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write marker: %w", err)
	}

	fmt.Printf("Created consensus state marker for height %d\n", maxHeight)
	return nil
}
