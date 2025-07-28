package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cockroachdb/pebble"
)

func main() {
	// Command line flags
	srcPath := flag.String("src", "", "Source PebbleDB path")
	dstPath := flag.String("dst", "", "Destination PebbleDB path")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	if *srcPath == "" || *dstPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s --src <source-db> --dst <destination-db> [--verbose]\n", os.Args[0])
		os.Exit(1)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(*dstPath, 0755); err != nil {
		log.Fatalf("Failed to create destination directory: %v", err)
	}

	// Open source database
	srcDB, err := pebble.Open(*srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Create destination database
	dstDB, err := pebble.Open(*dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create destination database: %v", err)
	}
	defer dstDB.Close()

	fmt.Println("=== EVM Key Migration v2 ===")
	fmt.Printf("Source: %s\n", *srcPath)
	fmt.Printf("Destination: %s\n", *dstPath)

	start := time.Now()
	if err := migrateKeys(srcDB, dstDB, *verbose); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	fmt.Printf("\n=== Migration Complete in %s ===\n", time.Since(start))
}

func migrateKeys(src, dst *pebble.DB, verbose bool) error {
	// First pass: Build full hash->height map from H keys
	fmt.Println("Pass 1: Building hash->height map...")
	hashToHeight := make(map[string]uint64)
	
	iter, err := src.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Skip too short keys
		if len(key) < 41 { // 33 prefix + 1 type + 8 suffix minimum
			continue
		}
		
		// Extract logical key: strip 33-byte prefix and 8-byte suffix
		logicalKey := key[33:len(key)-8]
		
		// Look for 'H' keys (hash->number)
		if len(logicalKey) > 1 && logicalKey[0] == 'H' {
			if len(value) == 8 {
				fullHash := logicalKey[1:] // 32 bytes typically
				height := binary.BigEndian.Uint64(value)
				hashToHeight[string(fullHash)] = height
			}
		}
	}
	iter.Close()
	
	fmt.Printf("Found %d hash->height mappings\n", len(hashToHeight))

	// Second pass: Migrate all keys and fix truncated 'n' keys
	fmt.Println("Pass 2: Migrating keys...")
	
	// EVM prefix for C-Chain
	evmPrefix := []byte("evm")
	
	// Statistics
	stats := make(map[string]int)
	fixedNKeys := 0
	unmatchedNKeys := 0
	
	// Create iterator
	iter, err = src.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()

	// Create batch
	batch := dst.NewBatch()
	defer batch.Close()

	// Process all keys
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Skip too short keys
		if len(key) < 41 {
			continue
		}
		
		// Extract logical key: strip 33-byte prefix and 8-byte suffix
		logicalKey := key[33:len(key)-8]
		
		// Skip empty keys
		if len(logicalKey) == 0 {
			continue
		}

		// Track key type
		keyType := fmt.Sprintf("%c", logicalKey[0])
		stats[keyType]++
		
		// Handle number->hash keys specially
		if logicalKey[0] == 'n' && len(logicalKey) > 1 {
			// Extract truncated hash
			truncated := logicalKey[1:]
			
			// Find full hash that starts with this truncated prefix
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
				// Create proper canonical key: evmn + 8-byte number
				newKey := make([]byte, 12) // "evmn" + 8 bytes
				copy(newKey, []byte("evmn"))
				binary.BigEndian.PutUint64(newKey[4:], matchedHeight)
				
				if err := batch.Set(newKey, value, nil); err != nil {
					return fmt.Errorf("failed to set key: %w", err)
				}
				
				fixedNKeys++
				
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
		}

		count++
		if count%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dst.NewBatch()
			fmt.Printf("Migrated %d keys...\n", count)
		}
		
		// Log first few keys in verbose mode
		if verbose && count < 10 {
			fmt.Printf("Key %d: %x -> %c (logical len=%d)\n", count, key[:40], logicalKey[0], len(logicalKey))
		}
	}

	// Commit final batch
	if err := batch.Commit(nil); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}

	// Print statistics
	fmt.Printf("\nMigrated %d keys total\n", count)
	fmt.Printf("Fixed %d truncated 'n' keys\n", fixedNKeys)
	fmt.Printf("Unmatched 'n' keys: %d\n", unmatchedNKeys)
	fmt.Println("\nKey type statistics:")
	fmt.Printf("  Headers (h): %d\n", stats["h"])
	fmt.Printf("  Bodies (b): %d\n", stats["b"])
	fmt.Printf("  Receipts (r): %d\n", stats["r"])
	fmt.Printf("  Number->Hash (n): %d (fixed: %d)\n", stats["n"], fixedNKeys)
	fmt.Printf("  Hash->Number (H): %d\n", stats["H"])
	fmt.Printf("  State (s): %d\n", stats["s"])
	fmt.Printf("  Accounts (0x26): %d\n", stats["\x26"])
	
	// Check for any errors
	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}

	return nil
}