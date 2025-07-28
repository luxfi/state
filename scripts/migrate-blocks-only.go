package main

import (
	"encoding/binary"
	"encoding/hex"
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

// Block-related prefixes we want to migrate
var blockPrefixes = []byte{
	0x68, // 'h' - headers
	0x62, // 'b' - bodies
	0x6e, // 'n' - number->hash
	0x48, // 'H' - hash->number
	0x54, // 'T' - total difficulty
	0x72, // 'r' - receipts
	0x6c, // 'l' - tx lookups
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: migrate-blocks-only <source-db> <target-db>")
		fmt.Println()
		fmt.Println("This tool migrates only block data from subnet to C-Chain format")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]
	
	fmt.Printf("=== Block Data Migration ===\n")
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
	if err := migrateBlockData(srcDB, dstDB); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	
	fmt.Printf("\n=== Migration Complete in %s ===\n", time.Since(start))
}

func migrateBlockData(srcDB, dstDB *pebble.DB) error {
	keyTypes := make(map[string]int)
	totalMigrated := 0
	
	// First, find the latest block to understand the range
	fmt.Println("Finding block range...")
	highestBlock, err := findHighestBlock(srcDB)
	if err != nil {
		fmt.Printf("Warning: Could not find highest block: %v\n", err)
		highestBlock = 100000 // Default to check first 100k blocks
	} else {
		fmt.Printf("Highest block found: %d\n", highestBlock)
	}
	
	// Migrate each block prefix type
	for _, prefix := range blockPrefixes {
		prefixName := getPrefixName(prefix)
		fmt.Printf("\nMigrating %s...\n", prefixName)
		
		count, err := migratePrefix(srcDB, dstDB, prefix, prefixName, keyTypes)
		if err != nil {
			return fmt.Errorf("failed to migrate %s: %w", prefixName, err)
		}
		
		totalMigrated += count
		fmt.Printf("  Migrated %d %s entries\n", count, prefixName)
	}
	
	// Also migrate special keys
	fmt.Println("\nMigrating special keys...")
	if err := migrateSpecialKeys(srcDB, dstDB); err != nil {
		return fmt.Errorf("failed to migrate special keys: %w", err)
	}
	
	// Print summary
	fmt.Printf("\n=== Migration Summary ===\n")
	fmt.Printf("Total keys migrated: %d\n", totalMigrated)
	fmt.Printf("\nKey type breakdown:\n")
	
	for keyType, count := range keyTypes {
		fmt.Printf("  %s: %d\n", keyType, count)
	}
	
	return nil
}

func migratePrefix(srcDB, dstDB *pebble.DB, prefix byte, prefixName string, keyTypes map[string]int) (int, error) {
	// Create batch for efficient writes
	batch := dstDB.NewBatch()
	defer batch.Close()
	
	// Create iterator for this prefix
	lowerBound := append(subnetPrefix, prefix)
	upperBound := append(subnetPrefix, prefix+1)
	
	iter, err := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	count := 0
	batchSize := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Remove subnet prefix
		newKey := key[len(subnetPrefix):]
		
		// Copy to new database
		if err := batch.Set(newKey, value, pebble.Sync); err != nil {
			return count, fmt.Errorf("failed to set key: %w", err)
		}
		
		count++
		batchSize++
		keyTypes[prefixName]++
		
		// Show examples
		if count <= 3 {
			fmt.Printf("  Example: %s\n", formatKeyValue(newKey, value))
		}
		
		// Commit batch periodically
		if batchSize >= 10000 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return count, fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			batchSize = 0
			
			if count%100000 == 0 {
				fmt.Printf("  Progress: %d %s entries...\n", count, prefixName)
			}
		}
	}
	
	// Final batch commit
	if batchSize > 0 {
		if err := batch.Commit(pebble.Sync); err != nil {
			return count, fmt.Errorf("failed to commit final batch: %w", err)
		}
	}
	
	return count, nil
}

func migrateSpecialKeys(srcDB, dstDB *pebble.DB) error {
	// Keys that might exist without subnet prefix
	specialKeys := [][]byte{
		[]byte("LastBlock"),
		[]byte("LastHeader"),
		[]byte("LastFastBlock"),
		[]byte("ethereum-config-"),
		[]byte("databaseVersion"),
		[]byte("headHeaderKey"),
		[]byte("headBlockKey"),
		[]byte("headFastBlockKey"),
		[]byte("fastTrieProgressKey"),
		[]byte("snapshotRootKey"),
		[]byte("snapshotJournalKey"),
		[]byte("snapshotGeneratorKey"),
		[]byte("snapshotRecoveryKey"),
		[]byte("snapshotSyncStatusKey"),
	}
	
	migrated := 0
	
	for _, key := range specialKeys {
		// Try with subnet prefix
		keyWithPrefix := append(subnetPrefix, key...)
		val, closer, err := srcDB.Get(keyWithPrefix)
		if err == nil {
			newVal := make([]byte, len(val))
			copy(newVal, val)
			closer.Close()
			
			if err := dstDB.Set(key, newVal, pebble.Sync); err != nil {
				return fmt.Errorf("failed to set special key %s: %w", string(key), err)
			}
			fmt.Printf("  Migrated special key: %s\n", string(key))
			migrated++
			continue
		}
		
		// Try without prefix
		val, closer, err = srcDB.Get(key)
		if err == nil {
			newVal := make([]byte, len(val))
			copy(newVal, val)
			closer.Close()
			
			if err := dstDB.Set(key, newVal, pebble.Sync); err != nil {
				return fmt.Errorf("failed to set special key %s: %w", string(key), err)
			}
			fmt.Printf("  Migrated special key: %s (no prefix)\n", string(key))
			migrated++
		}
	}
	
	fmt.Printf("  Total special keys migrated: %d\n", migrated)
	return nil
}

func findHighestBlock(db *pebble.DB) (uint64, error) {
	// Look for number->hash entries to find the highest block
	lowerBound := append(subnetPrefix, 0x6e) // 'n'
	upperBound := append(subnetPrefix, 0x6f)
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: lowerBound,
		UpperBound: upperBound,
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()
	
	var highestBlock uint64
	
	// Go to last entry
	if iter.Last() {
		key := iter.Key()
		// Remove prefix and 'n' byte
		key = key[len(subnetPrefix)+1:]
		if len(key) >= 8 {
			// Block number is encoded as 8-byte big endian
			highestBlock = binary.BigEndian.Uint64(key[:8])
		}
	}
	
	return highestBlock, nil
}

func getPrefixName(prefix byte) string {
	switch prefix {
	case 0x68:
		return "headers"
	case 0x62:
		return "bodies"
	case 0x6e:
		return "number->hash"
	case 0x48:
		return "hash->number"
	case 0x54:
		return "total-difficulty"
	case 0x72:
		return "receipts"
	case 0x6c:
		return "tx-lookups"
	default:
		return fmt.Sprintf("prefix-%02x", prefix)
	}
}

func formatKeyValue(key, value []byte) string {
	keyStr := hex.EncodeToString(key)
	if len(keyStr) > 64 {
		keyStr = keyStr[:32] + "..." + keyStr[len(keyStr)-32:]
	}
	
	valueInfo := fmt.Sprintf("(%d bytes)", len(value))
	
	// For headers and bodies, try to show block number
	if len(key) > 0 && (key[0] == 0x68 || key[0] == 0x62) && len(key) >= 9 {
		blockNum := binary.BigEndian.Uint64(key[1:9])
		return fmt.Sprintf("key=%s (block %d), value=%s", keyStr, blockNum, valueInfo)
	}
	
	return fmt.Sprintf("key=%s, value=%s", keyStr, valueInfo)
}