package main

import (
	"bytes"
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
		src     = flag.String("src", "", "source subnet database path")
		dst     = flag.String("dst", "", "destination directory for C-Chain data")
		verbose = flag.Bool("v", false, "verbose output")
	)
	flag.Parse()

	if *src == "" || *dst == "" {
		flag.Usage()
		log.Fatal("Both --src and --dst are required")
	}

	// Create output directories
	evmDB := filepath.Join(*dst, "evm", "pebbledb")
	stateDB := filepath.Join(*dst, "state", "pebbledb")

	if err := os.MkdirAll(evmDB, 0755); err != nil {
		log.Fatalf("Failed to create EVM directory: %v", err)
	}
	if err := os.MkdirAll(stateDB, 0755); err != nil {
		log.Fatalf("Failed to create state directory: %v", err)
	}

	fmt.Println("=== Subnet to C-Chain Migration (Final) ===")
	fmt.Printf("Source: %s\n", *src)
	fmt.Printf("EVM DB: %s\n", evmDB)
	fmt.Printf("State DB: %s\n", stateDB)

	// Step 1: Migrate keys with proper handling of truncated hashes
	if err := migrateKeys(*src, evmDB, *verbose); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Step 2: Find max height
	maxHeight, err := findMaxHeight(evmDB)
	if err != nil {
		log.Fatalf("Failed to find max height: %v", err)
	}
	fmt.Printf("\n=== Maximum block height: %d ===\n", maxHeight)

	// Step 3: Create consensus state marker
	if err := createConsensusState(evmDB, stateDB, maxHeight); err != nil {
		log.Fatalf("Failed to create consensus state: %v", err)
	}

	fmt.Println("\n=== Migration Complete ===")
	fmt.Printf("Next steps:\n")
	fmt.Printf("1. Create consensus state:\n")
	fmt.Printf("   ./replay-consensus-pebble --evm %s --state %s --tip %d\n", evmDB, stateDB, maxHeight)
	fmt.Printf("2. Launch luxd:\n")
	fmt.Printf("   ./luxd --db-dir=%s --network-id=96369 --staking-enabled=false\n", *dst)
}

func migrateKeys(src, dst string, verbose bool) error {
	fmt.Println("\n=== Step 1: Migrating Keys ===")
	start := time.Now()

	srcDB, err := pebble.Open(src, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcDB.Close()

	dstDB, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstDB.Close()

	// First pass: Build full hash->height map from H keys
	fmt.Println("Pass 1: Building full hash->height map from H keys...")
	hashToHeight := make(map[string]uint64)
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		if len(key) < 41 {
			continue
		}

		// Extract logical key: strip 33-byte prefix and 8-byte suffix
		logicalKey := key[33 : len(key)-8]

		// Look for 'H' keys (hash->number)
		if len(logicalKey) > 1 && logicalKey[0] == 'H' {
			if len(value) == 8 {
				fullHash := logicalKey[1:] // 32 bytes
				height := binary.BigEndian.Uint64(value)
				hashToHeight[string(fullHash)] = height
			}
		}
	}
	iter.Close()

	fmt.Printf("Found %d full hash->height mappings\n", len(hashToHeight))

	// Find max height
	var maxHeight uint64
	for _, num := range hashToHeight {
		if num > maxHeight {
			maxHeight = num
		}
	}
	fmt.Printf("Maximum block height from H keys: %d\n", maxHeight)

	// Second pass: Migrate all keys and fix truncated 'n' keys
	fmt.Println("Pass 2: Migrating keys and fixing truncated 'n' keys...")
	iter, err = srcDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	evmPrefix := []byte("evm")
	batch := dstDB.NewBatch()
	count := 0
	stats := make(map[byte]int)
	fixedNKeys := 0
	unmatchedNKeys := 0

	// Key to remove (ChainConfigConfigKey)
	chainConfigKey := []byte("ChainConfigConfigKey")

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		if len(key) < 41 {
			continue
		}

		logicalKey := key[33 : len(key)-8]

		if len(logicalKey) == 0 {
			continue
		}

		// Skip ChainConfigConfigKey
		if string(logicalKey) == string(chainConfigKey) {
			fmt.Println("Skipping ChainConfigConfigKey")
			continue
		}

		// Handle number->hash keys specially
		if logicalKey[0] == 'n' {
			// Extract truncated hash
			truncated := logicalKey[1:]

			// Find full hash that starts with this truncated prefix
			var matchedHeight uint64
			var found bool
			var matchCount int

			for fullHashStr, height := range hashToHeight {
				fullHash := []byte(fullHashStr)
				if bytes.HasPrefix(fullHash, truncated) {
					matchedHeight = height
					found = true
					matchCount++
				}
			}

			if matchCount > 1 {
				log.Printf("WARNING: Multiple matches for truncated hash %x (%d matches)", truncated, matchCount)
			}

			if found {
				// Create proper canonical key: evmn + 8-byte number
				newKey := make([]byte, 12) // "evmn" + 8 bytes
				copy(newKey, []byte("evmn"))
				binary.BigEndian.PutUint64(newKey[4:], matchedHeight)

				if err := batch.Set(newKey, value, nil); err != nil {
					return fmt.Errorf("failed to set key: %w", err)
				}

				fixedNKeys++
				stats['n']++

				if verbose && fixedNKeys <= 5 {
					fmt.Printf("Fixed 'n' key: truncated=%x -> height=%d\n", truncated, matchedHeight)
				}
			} else {
				unmatchedNKeys++
				if verbose && unmatchedNKeys <= 5 {
					fmt.Printf("No match for truncated 'n' key: %x\n", truncated)
				}
			}
		} else {
			// For all other keys, just add evm prefix
			newKey := make([]byte, len(evmPrefix)+len(logicalKey))
			copy(newKey, evmPrefix)
			copy(newKey[len(evmPrefix):], logicalKey)

			if err := batch.Set(newKey, value, nil); err != nil {
				return fmt.Errorf("failed to set key: %w", err)
			}

			// Track key types
			if len(logicalKey) > 0 {
				stats[logicalKey[0]]++
			}
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

	fmt.Printf("\nMigrated %d keys in %s\n", count, time.Since(start))
	fmt.Printf("Fixed %d truncated 'n' keys to proper canonical format\n", fixedNKeys)
	fmt.Printf("Unmatched 'n' keys: %d\n", unmatchedNKeys)
	fmt.Printf("\nKey type statistics:\n")
	fmt.Printf("  Headers (h/0x68): %d\n", stats['h']+stats[0x68])
	fmt.Printf("  Bodies (b/0x62): %d\n", stats['b']+stats[0x62])
	fmt.Printf("  Receipts (r/0x72): %d\n", stats['r']+stats[0x72])
	fmt.Printf("  Number->Hash (n/0x6e): %d\n", stats['n']+stats[0x6e])
	fmt.Printf("  Hash->Number (H/0x48): %d\n", stats['H']+stats[0x48])
	fmt.Printf("  Accounts (0x26): %d\n", stats[0x26])
	fmt.Printf("  Storage (0xa3): %d\n", stats[0xa3])
	fmt.Printf("  State (s/0x73): %d\n", stats['s']+stats[0x73])

	return nil
}

func findMaxHeight(dbPath string) (uint64, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Look for evmn keys (number->hash mappings)
	prefix := []byte("evmn")
	var maxHeight uint64

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		// evmn key format: "evm" + "n" + 8-byte number
		if len(key) == 12 { // 4 ("evmn") + 8 (number)
			height := binary.BigEndian.Uint64(key[4:])
			if height > maxHeight {
				maxHeight = height
			}
			count++
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("no canonical number->hash mappings found")
	}

	fmt.Printf("Found %d canonical mappings\n", count)
	return maxHeight, nil
}

func createConsensusState(evmDB, stateDB string, maxHeight uint64) error {
	fmt.Printf("\n=== Step 2: Creating Consensus State Marker ===\n")

	// Create a marker file
	markerFile := filepath.Join(stateDB, "CONSENSUS_MARKER")
	data := fmt.Sprintf("max_height=%d\ncreated=%s\n", maxHeight, time.Now())

	if err := os.WriteFile(markerFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write marker: %w", err)
	}

	fmt.Printf("Created consensus state marker for height %d\n", maxHeight)

	return nil
}
