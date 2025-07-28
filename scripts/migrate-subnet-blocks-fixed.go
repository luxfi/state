package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: migrate-subnet-blocks-fixed <source_db> <target_db>")
		os.Exit(1)
	}

	sourcePath := os.Args[1]
	targetPath := os.Args[2]

	// The actual subnet prefix from the analysis
	subnetPrefix, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")
	
	fmt.Printf("=== Migrating SubnetEVM Blocks ===\n")
	fmt.Printf("Source: %s\n", sourcePath)
	fmt.Printf("Target: %s\n", targetPath)
	fmt.Printf("Subnet prefix: %x\n\n", subnetPrefix)

	// Open source database
	srcDB, err := pebble.Open(sourcePath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal("Failed to open source database:", err)
	}
	defer srcDB.Close()

	// Open target database
	dstDB, err := pebble.Open(targetPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open target database:", err)
	}
	defer dstDB.Close()

	// First, find all valid block numbers
	fmt.Println("Phase 1: Finding all valid blocks...")
	validBlocks := findValidBlocks(srcDB, subnetPrefix)
	fmt.Printf("Found %d valid blocks\n", len(validBlocks))
	
	if len(validBlocks) > 0 {
		fmt.Printf("Block range: %d - %d\n\n", validBlocks[0], validBlocks[len(validBlocks)-1])
	}

	// Migrate each type of data
	fmt.Println("Phase 2: Migrating block data...")
	
	// Define the key types to migrate
	keyTypes := []struct {
		name       string
		prefix     byte
		hasNumber  bool
		hasHash    bool
	}{
		{"Headers", 0x68, true, true},          // h + number + hash
		{"Bodies", 0x62, true, true},           // b + number + hash
		{"Receipts", 0x72, true, true},         // r + number + hash
		{"TDs", 0x54, true, true},              // T + number + hash
		{"Hash->Number", 0x48, false, true},    // H + hash
		{"Number->Hash", 0x6e, true, false},    // n + number
		{"TxLookup", 0x6c, false, true},        // l + hash
	}

	for _, kt := range keyTypes {
		count := migrateKeyType(srcDB, dstDB, subnetPrefix, kt.prefix, kt.name, kt.hasNumber, kt.hasHash, validBlocks)
		fmt.Printf("  Migrated %d %s\n", count, kt.name)
	}

	// Also migrate other important keys
	fmt.Println("\nPhase 3: Migrating special keys...")
	migrateSpecialKeys(srcDB, dstDB, subnetPrefix)

	fmt.Println("\n=== Migration Complete ===")
}

func findValidBlocks(db *pebble.DB, subnetPrefix []byte) []uint64 {
	blockMap := make(map[uint64]bool)
	
	// Use hash->number mappings to find valid blocks
	hashNumPrefix := append(subnetPrefix, 0x48) // 'H'
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: hashNumPrefix,
		UpperBound: incrementBytes(hashNumPrefix),
	})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		value := iter.Value()
		if len(value) == 8 {
			blockNum := binary.BigEndian.Uint64(value)
			// Filter out obviously invalid block numbers
			if blockNum < 10000000 { // Reasonable upper limit
				blockMap[blockNum] = true
			}
		}
	}
	
	// Convert to sorted slice
	var blocks []uint64
	for num := range blockMap {
		blocks = append(blocks, num)
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i] < blocks[j] })
	
	return blocks
}

func migrateKeyType(srcDB, dstDB *pebble.DB, subnetPrefix []byte, typePrefix byte, 
	typeName string, hasNumber, hasHash bool, validBlocks []uint64) int {
	
	count := 0
	prefix := append(subnetPrefix, typePrefix)
	
	iter, err := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: incrementBytes(prefix),
	})
	if err != nil {
		log.Printf("Failed to create iterator for %s: %v", typeName, err)
		return 0
	}
	defer iter.Close()
	
	batch := dstDB.NewBatch()
	batchCount := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Validate and extract components
		if len(key) < 33 { // At least prefix + type
			continue
		}
		
		// Skip if it has a block number and it's not in our valid set
		if hasNumber && len(key) >= 41 {
			blockNum := binary.BigEndian.Uint64(key[33:41])
			if blockNum >= 10000000 { // Skip invalid blocks
				continue
			}
		}
		
		// Create new key without subnet prefix
		var newKey []byte
		newKey = append(newKey, typePrefix)
		newKey = append(newKey, key[33:]...) // Skip subnet prefix and type byte
		
		// Write to batch
		if err := batch.Set(newKey, value, nil); err != nil {
			log.Printf("Failed to set key: %v", err)
			continue
		}
		
		count++
		batchCount++
		
		// Commit batch periodically
		if batchCount >= 1000 {
			if err := batch.Commit(nil); err != nil {
				log.Printf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
			batchCount = 0
			
			if count%10000 == 0 {
				fmt.Printf("    %s: %d migrated...\n", typeName, count)
			}
		}
	}
	
	// Commit final batch
	if batchCount > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Printf("Failed to commit final batch: %v", err)
		}
	}
	
	return count
}

func migrateSpecialKeys(srcDB, dstDB *pebble.DB, subnetPrefix []byte) {
	// Migrate any keys that don't have the subnet prefix
	// These might include metadata keys
	
	specialKeys := [][]byte{
		[]byte("LastBlock"),
		[]byte("LastHeader"),
		[]byte("LastFast"),
		[]byte("LastPivot"),
		[]byte("SnapshotRoot"),
		[]byte("SnapshotJournal"),
		[]byte("SnapshotGenerator"),
		[]byte("SnapshotRecovery"),
		[]byte("SnapshotSyncStatus"),
		[]byte("SnapshotDisabled"),
	}
	
	// Also check for the special key with different prefix
	otherPrefix, _ := hex.DecodeString("c26713aca8e3980be9b3f6004002447ff93d83db740bc9d9f4a22d170149b3ac")
	iter, _ := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: otherPrefix,
		UpperBound: incrementBytes(otherPrefix),
	})
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// This appears to be "last_accepted_key"
		if bytes.Contains(key, []byte("last_accepted_key")) {
			newKey := []byte("LastAccepted")
			if err := dstDB.Set(newKey, value, nil); err == nil {
				fmt.Printf("  Migrated LastAccepted key\n")
				count++
			}
		}
	}
	iter.Close()
	
	// Try to migrate standard special keys
	for _, key := range specialKeys {
		// Try with subnet prefix
		prefixedKey := append(subnetPrefix, key...)
		if value, closer, err := srcDB.Get(prefixedKey); err == nil {
			if err := dstDB.Set(key, value, nil); err == nil {
				count++
			}
			closer.Close()
		}
		
		// Try without prefix
		if value, closer, err := srcDB.Get(key); err == nil {
			if err := dstDB.Set(key, value, nil); err == nil {
				count++
			}
			closer.Close()
		}
	}
	
	fmt.Printf("  Migrated %d special keys\n", count)
}

func incrementBytes(b []byte) []byte {
	result := make([]byte, len(b))
	copy(result, b)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 255 {
			result[i]++
			break
		}
		result[i] = 0
	}
	return result
}