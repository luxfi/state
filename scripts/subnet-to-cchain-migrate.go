package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cockroachdb/pebble"
)

// The subnet prefix for chain 96369
var subnetPrefix = []byte{
	0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c,
	0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e,
	0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a,
	0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1,
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: subnet-to-cchain-migrate <source-db> <target-db>")
		fmt.Println()
		fmt.Println("This tool migrates subnet EVM data to C-Chain format by:")
		fmt.Println("  1. Removing the 32-byte subnet prefix from all keys")
		fmt.Println("  2. Preserving rawdb prefixes (h, b, n, H, T, r, l, S)")
		fmt.Println("  3. Copying special keys for head pointers and chain config")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]
	
	fmt.Printf("=== Subnet to C-Chain Migration ===\n")
	fmt.Printf("Source: %s\n", srcPath)
	fmt.Printf("Target: %s\n\n", dstPath)
	
	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()
	
	// Create target database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create target database: %v", err)
	}
	defer dstDB.Close()
	
	// Start migration
	start := time.Now()
	if err := migrateData(srcDB, dstDB); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	
	fmt.Printf("\n=== Migration Complete in %s ===\n", time.Since(start))
}

func migrateData(srcDB, dstDB *pebble.DB) error {
	// Create batch for efficient writes
	batch := dstDB.NewBatch()
	defer batch.Close()
	
	// Create iterator
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	var (
		totalKeys    int
		migratedKeys int
		batchSize    int
		skippedKeys  int
	)
	
	// Track key types
	keyTypes := make(map[string]int)
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		totalKeys++
		
		// Check if key has subnet prefix
		if bytes.HasPrefix(key, subnetPrefix) {
			// Remove subnet prefix
			newKey := key[len(subnetPrefix):]
			
			// Track key type
			keyType := identifyKeyType(newKey)
			keyTypes[keyType]++
			
			// Copy to new database
			if err := batch.Set(newKey, value, pebble.Sync); err != nil {
				return fmt.Errorf("failed to set key: %w", err)
			}
			
			migratedKeys++
			batchSize++
			
			// Show progress for first few keys
			if migratedKeys <= 5 {
				fmt.Printf("  Migrated: %s -> %s (%s)\n", 
					formatKey(key), formatKey(newKey), keyType)
			}
		} else {
			// Key doesn't have subnet prefix - copy as-is if it's a special key
			if isSpecialKey(key) {
				if err := batch.Set(key, value, pebble.Sync); err != nil {
					return fmt.Errorf("failed to set special key: %w", err)
				}
				migratedKeys++
				batchSize++
				fmt.Printf("  Copied special key: %s\n", formatKey(key))
			} else {
				skippedKeys++
			}
		}
		
		// Commit batch periodically
		if batchSize >= 10000 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			batchSize = 0
			
			if totalKeys%100000 == 0 {
				fmt.Printf("  Progress: %d keys processed, %d migrated...\n", totalKeys, migratedKeys)
			}
		}
	}
	
	// Final batch commit
	if batchSize > 0 {
		if err := batch.Commit(pebble.Sync); err != nil {
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
	}
	
	// Also check and copy special keys
	specialKeys := [][]byte{
		[]byte("LastBlock"),
		[]byte("LastHeader"), 
		[]byte("ethereum-config-"),
		[]byte("secure-key-"),
	}
	
	for _, sk := range specialKeys {
		val, closer, err := srcDB.Get(sk)
		if err == nil {
			newVal := make([]byte, len(val))
			copy(newVal, val)
			closer.Close()
			
			if err := dstDB.Set(sk, newVal, pebble.Sync); err != nil {
				return fmt.Errorf("failed to set special key %s: %w", string(sk), err)
			}
			fmt.Printf("  Migrated special key: %s\n", string(sk))
			migratedKeys++
		}
	}
	
	// Print summary
	fmt.Printf("\n=== Migration Summary ===\n")
	fmt.Printf("Total keys processed: %d\n", totalKeys)
	fmt.Printf("Keys migrated: %d\n", migratedKeys)
	fmt.Printf("Keys skipped: %d\n", skippedKeys)
	fmt.Printf("\nKey type breakdown:\n")
	
	for keyType, count := range keyTypes {
		fmt.Printf("  %s: %d\n", keyType, count)
	}
	
	return nil
}

func identifyKeyType(key []byte) string {
	if len(key) == 0 {
		return "empty"
	}
	
	firstByte := key[0]
	switch firstByte {
	case 0x68: // 'h'
		return "header"
	case 0x62: // 'b'
		return "body"
	case 0x6e: // 'n'
		return "number->hash"
	case 0x48: // 'H'
		return "hash->number"
	case 0x54: // 'T'
		return "total-difficulty"
	case 0x72: // 'r'
		return "receipt"
	case 0x6c: // 'l'
		return "tx-lookup"
	case 0x53: // 'S'
		return "secure-trie"
	default:
		if firstByte >= 0x00 && firstByte <= 0x0f {
			return fmt.Sprintf("trie-node-%02x", firstByte)
		}
		return fmt.Sprintf("unknown-%02x", firstByte)
	}
}

func isSpecialKey(key []byte) bool {
	specialPrefixes := [][]byte{
		[]byte("LastBlock"),
		[]byte("LastHeader"),
		[]byte("ethereum-config-"),
		[]byte("secure-key-"),
	}
	
	for _, prefix := range specialPrefixes {
		if bytes.HasPrefix(key, prefix) {
			return true
		}
	}
	
	return false
}

func formatKey(key []byte) string {
	if len(key) > 32 {
		return fmt.Sprintf("%x...%x", key[:16], key[len(key)-16:])
	}
	return fmt.Sprintf("%x", key)
}