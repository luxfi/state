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
		src = flag.String("src", "", "source subnet database path")
		dst = flag.String("dst", "", "destination directory for C-Chain data")
		verbose = flag.Bool("v", false, "verbose output")
		cacheDir = flag.String("cache", ".tmp/migration-cache", "cache directory")
	)
	flag.Parse()

	if *src == "" || *dst == "" {
		flag.Usage()
		log.Fatal("Both --src and --dst are required")
	}

	// Check if we have cached results
	cacheFile := filepath.Join(*cacheDir, "blockchain-migration.done")
	if _, err := os.Stat(cacheFile); err == nil {
		fmt.Println("=== Using cached migration results ===")
		fmt.Printf("Cache found at: %s\n", cacheFile)
		
		// Read cache info
		data, _ := os.ReadFile(cacheFile)
		fmt.Print(string(data))
		
		// Copy cached results to destination
		cachedEvm := filepath.Join(*cacheDir, "evm")
		dstEvm := filepath.Join(*dst, "evm")
		fmt.Printf("Copying cached data from %s to %s\n", cachedEvm, dstEvm)
		
		if err := os.RemoveAll(dstEvm); err != nil {
			log.Fatalf("Failed to remove existing destination: %v", err)
		}
		
		if err := copyDir(cachedEvm, dstEvm); err != nil {
			log.Fatalf("Failed to copy cached data: %v", err)
		}
		
		fmt.Println("Cache copied successfully!")
		return
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

	fmt.Println("=== Fast Blockchain Migration ===")
	fmt.Printf("Source: %s\n", *src)
	fmt.Printf("EVM DB: %s\n", evmDB)
	fmt.Printf("State DB: %s\n", stateDB)

	// Migrate only blockchain keys
	maxHeight, err := migrateBlockchainOnly(*src, evmDB, *verbose)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Printf("\n=== Maximum block height: %d ===\n", maxHeight)

	// Create consensus state marker
	if err := createConsensusState(evmDB, stateDB, maxHeight); err != nil {
		log.Fatalf("Failed to create consensus state: %v", err)
	}

	// Save to cache
	fmt.Printf("\nSaving results to cache: %s\n", *cacheDir)
	if err := os.MkdirAll(*cacheDir, 0755); err != nil {
		log.Printf("Failed to create cache dir: %v", err)
	} else {
		// Copy migrated data to cache
		cachedEvm := filepath.Join(*cacheDir, "evm")
		if err := copyDir(filepath.Join(*dst, "evm"), cachedEvm); err != nil {
			log.Printf("Failed to cache evm data: %v", err)
		} else {
			// Write cache marker
			cacheInfo := fmt.Sprintf("Migration completed at: %s\nMax height: %d\nSource: %s\n", 
				time.Now().Format(time.RFC3339), maxHeight, *src)
			os.WriteFile(cacheFile, []byte(cacheInfo), 0644)
		}
	}

	fmt.Println("\n=== Migration Complete ===")
	fmt.Printf("Next steps:\n")
	fmt.Printf("1. Create consensus state:\n")
	fmt.Printf("   ./replay-consensus-pebble --evm %s --state %s --tip %d\n", evmDB, stateDB, maxHeight)
	fmt.Printf("2. Launch luxd:\n")
	fmt.Printf("   ./luxd --db-dir=%s --network-id=96369 --staking-enabled=false\n", *dst)
}

func migrateBlockchainOnly(src, dst string, verbose bool) (uint64, error) {
	fmt.Println("\n=== Migrating Blockchain Keys Only ===")
	start := time.Now()

	srcDB, err := pebble.Open(src, &pebble.Options{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("failed to open source: %w", err)
	}
	defer srcDB.Close()

	dstDB, err := pebble.Open(dst, &pebble.Options{})
	if err != nil {
		return 0, fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstDB.Close()

	// First pass: Build hash->height map from H keys only
	fmt.Println("Pass 1: Building hash->height map from H keys...")
	hashToHeight := make(map[string]uint64)
	
	// Use prefix iterator for efficiency
	hPrefix := make([]byte, 34)
	copy(hPrefix[:33], bytes.Repeat([]byte{0xff}, 33)) // Assuming namespace
	hPrefix[33] = 'H'
	
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}

	hCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) < 41 {
			continue
		}
		
		logicalKey := key[33:len(key)-8]
		
		// Only process H keys
		if len(logicalKey) > 1 && logicalKey[0] == 'H' {
			if len(value) == 8 {
				fullHash := logicalKey[1:]
				height := binary.BigEndian.Uint64(value)
				hashToHeight[string(fullHash)] = height
				hCount++
			}
		}
	}
	iter.Close()

	fmt.Printf("Found %d hash->height mappings\n", len(hashToHeight))

	// Find max height
	var maxHeight uint64
	for _, num := range hashToHeight {
		if num > maxHeight {
			maxHeight = num
		}
	}
	fmt.Printf("Maximum block height: %d\n", maxHeight)

	// Second pass: Migrate only blockchain keys
	fmt.Println("Pass 2: Migrating blockchain keys...")
	iter, err = srcDB.NewIter(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	evmPrefix := []byte("evm")
	batch := dstDB.NewBatch()
	count := 0
	stats := make(map[byte]int)
	fixedNKeys := 0
	unmatchedNKeys := 0
	skipped := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) < 41 {
			skipped++
			continue
		}
		
		logicalKey := key[33:len(key)-8]
		
		if len(logicalKey) == 0 {
			skipped++
			continue
		}

		// Only process blockchain-related keys
		isBlockchainKey := false
		switch logicalKey[0] {
		case 'h', 'b', 'r', 'n', 'H', 'l':
			isBlockchainKey = true
		}

		if !isBlockchainKey {
			skipped++
			continue
		}

		// Handle number->hash keys specially
		if logicalKey[0] == 'n' {
			truncated := logicalKey[1:]
			
			// Find full hash match
			var matchedHeight uint64
			var found bool
			
			for fullHashStr, height := range hashToHeight {
				fullHash := []byte(fullHashStr)
				if bytes.HasPrefix(fullHash, truncated) {
					matchedHeight = height
					found = true
					break
				}
			}
			
			if found {
				// Create proper evmn key
				newKey := make([]byte, 12)
				copy(newKey, []byte("evmn"))
				binary.BigEndian.PutUint64(newKey[4:], matchedHeight)
				
				if err := batch.Set(newKey, value, nil); err != nil {
					return 0, fmt.Errorf("failed to set key: %w", err)
				}
				
				fixedNKeys++
				stats['n']++
			} else {
				unmatchedNKeys++
			}
		} else {
			// For other blockchain keys, just add evm prefix
			newKey := make([]byte, len(evmPrefix)+len(logicalKey))
			copy(newKey, evmPrefix)
			copy(newKey[len(evmPrefix):], logicalKey)

			if err := batch.Set(newKey, value, nil); err != nil {
				return 0, fmt.Errorf("failed to set key: %w", err)
			}

			stats[logicalKey[0]]++
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				return 0, fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Migrated %d blockchain keys (skipped %d non-blockchain)...\n", count, skipped)
		}
	}

	if err := batch.Commit(nil); err != nil {
		return 0, fmt.Errorf("failed to commit final batch: %w", err)
	}

	fmt.Printf("\nMigrated %d blockchain keys in %s\n", count, time.Since(start))
	fmt.Printf("Skipped %d non-blockchain keys\n", skipped)
	fmt.Printf("Fixed %d truncated 'n' keys\n", fixedNKeys)
	fmt.Printf("Unmatched 'n' keys: %d\n", unmatchedNKeys)
	fmt.Printf("\nKey type statistics:\n")
	fmt.Printf("  Headers (h): %d\n", stats['h'])
	fmt.Printf("  Bodies (b): %d\n", stats['b'])
	fmt.Printf("  Receipts (r): %d\n", stats['r'])
	fmt.Printf("  Number->Hash (n): %d\n", stats['n'])
	fmt.Printf("  Hash->Number (H): %d\n", stats['H'])

	return maxHeight, nil
}

func createConsensusState(evmDB, stateDB string, maxHeight uint64) error {
	fmt.Printf("\n=== Creating Consensus State Marker ===\n")
	
	markerFile := filepath.Join(stateDB, "CONSENSUS_MARKER")
	data := fmt.Sprintf("max_height=%d\ncreated=%s\n", maxHeight, time.Now())
	
	if err := os.WriteFile(markerFile, []byte(data), 0644); err != nil {
		return fmt.Errorf("failed to write marker: %w", err)
	}
	
	fmt.Printf("Created consensus state marker for height %d\n", maxHeight)
	
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		dstFile, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		if _, err := dstFile.ReadFrom(srcFile); err != nil {
			return err
		}

		return os.Chmod(dstPath, info.Mode())
	})
}