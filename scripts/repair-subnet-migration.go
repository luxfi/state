package main

import (
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
		fmt.Println("Usage: repair-subnet-migration <migrated_db> <original_subnet_db>")
		fmt.Println("This tool repairs the migrated database by:")
		fmt.Println("  1. Rebuilding number->hash mappings from headers")
		fmt.Println("  2. Verifying hash->number mappings")
		fmt.Println("  3. Updating head pointers")
		fmt.Println("  4. Migrating state trie data")
		os.Exit(1)
	}

	migratedPath := os.Args[1]
	originalPath := os.Args[2]

	fmt.Println("=== Repairing Subnet Migration ===")
	fmt.Printf("Migrated DB: %s\n", migratedPath)
	fmt.Printf("Original DB: %s\n\n", originalPath)

	// Open databases
	migratedDB, err := pebble.Open(migratedPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open migrated database:", err)
	}
	defer migratedDB.Close()

	originalDB, err := pebble.Open(originalPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal("Failed to open original database:", err)
	}
	defer originalDB.Close()

	// The subnet prefix from our analysis
	subnetPrefix, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")

	// Step 1: Analyze current state
	fmt.Println("Step 1: Analyzing current database state...")
	analyzeDatabase(migratedDB)

	// Step 2: Rebuild number->hash mappings
	fmt.Println("\nStep 2: Rebuilding number->hash mappings...")
	rebuildNumberToHashMappings(migratedDB)

	// Step 3: Verify and fix hash->number mappings
	fmt.Println("\nStep 3: Verifying hash->number mappings...")
	verifyHashToNumberMappings(migratedDB)

	// Step 4: Migrate state trie data
	fmt.Println("\nStep 4: Migrating state trie data...")
	migrateStateTrie(originalDB, migratedDB, subnetPrefix)

	// Step 5: Update head pointers
	fmt.Println("\nStep 5: Updating head pointers...")
	updateHeadPointers(migratedDB)

	// Step 6: Migrate additional data
	fmt.Println("\nStep 6: Migrating additional data...")
	migrateAdditionalData(originalDB, migratedDB, subnetPrefix)

	// Step 7: Final verification
	fmt.Println("\nStep 7: Final verification...")
	verifyMigration(migratedDB)

	fmt.Println("\n=== Repair Complete ===")
}

func analyzeDatabase(db *pebble.DB) {
	// Count different key types
	counts := map[string]int{
		"headers":       0,
		"bodies":        0,
		"receipts":      0,
		"hash->number":  0,
		"number->hash":  0,
		"state-trie":    0,
		"tx-lookup":     0,
		"other":         0,
	}

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 0 {
			continue
		}

		switch key[0] {
		case 0x68: // 'h' - header
			counts["headers"]++
		case 0x62: // 'b' - body
			counts["bodies"]++
		case 0x72: // 'r' - receipt
			counts["receipts"]++
		case 0x48: // 'H' - hash->number
			counts["hash->number"]++
		case 0x6e: // 'n' - number->hash
			counts["number->hash"]++
		case 0x53: // 'S' - secure trie
			counts["state-trie"]++
		case 0x6c: // 'l' - tx lookup
			counts["tx-lookup"]++
		default:
			counts["other"]++
		}
	}

	fmt.Println("Current database contents:")
	for k, v := range counts {
		if v > 0 {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}

	// Find highest block in number->hash mappings
	maxNum := findHighestNumberMapping(db)
	fmt.Printf("\nHighest number->hash mapping: %d\n", maxNum)
}

func findHighestNumberMapping(db *pebble.DB) uint64 {
	maxNum := uint64(0)
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x6e}, // 'n'
		UpperBound: []byte{0x6f},
	})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 9 {
			num := binary.BigEndian.Uint64(key[1:9])
			if num > maxNum && num < 10000000 { // Reasonable limit
				maxNum = num
			}
		}
	}

	return maxNum
}

func rebuildNumberToHashMappings(db *pebble.DB) {
	// First, collect all headers with their numbers and hashes
	headers := make(map[uint64][]byte) // number -> hash

	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68}, // 'h'
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	fmt.Println("Scanning headers...")
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 41 { // h + 8 bytes number + 32 bytes hash
			num := binary.BigEndian.Uint64(key[1:9])
			hash := key[9:41]
			
			// Only process reasonable block numbers
			if num < 10000000 {
				headers[num] = hash
				count++
				if count%100000 == 0 {
					fmt.Printf("  Scanned %d headers...\n", count)
				}
			}
		}
	}

	fmt.Printf("Found %d unique headers\n", len(headers))

	// Now write number->hash mappings
	batch := db.NewBatch()
	batchCount := 0
	written := 0

	// Sort block numbers
	var nums []uint64
	for num := range headers {
		nums = append(nums, num)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

	fmt.Println("Writing number->hash mappings...")
	for _, num := range nums {
		hash := headers[num]
		
		// Create number->hash key
		nKey := make([]byte, 9)
		nKey[0] = 0x6e // 'n'
		binary.BigEndian.PutUint64(nKey[1:], num)
		
		// Write mapping
		if err := batch.Set(nKey, hash, nil); err != nil {
			log.Printf("Failed to set n key for block %d: %v", num, err)
			continue
		}
		
		batchCount++
		written++
		
		// Commit batch periodically
		if batchCount >= 1000 {
			if err := batch.Commit(nil); err != nil {
				log.Printf("Failed to commit batch: %v", err)
			}
			batch = db.NewBatch()
			batchCount = 0
			
			if written%100000 == 0 {
				fmt.Printf("  Written %d mappings...\n", written)
			}
		}
	}

	// Commit final batch
	if batchCount > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Printf("Failed to commit final batch: %v", err)
		}
	}

	fmt.Printf("Wrote %d number->hash mappings\n", written)
}

func verifyHashToNumberMappings(db *pebble.DB) {
	// Verify that hash->number mappings exist for all headers
	missing := 0
	verified := 0

	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68}, // 'h'
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	batch := db.NewBatch()
	batchCount := 0

	fmt.Println("Verifying hash->number mappings...")
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 41 {
			num := binary.BigEndian.Uint64(key[1:9])
			hash := key[9:41]
			
			if num >= 10000000 {
				continue // Skip invalid numbers
			}
			
			// Check if hash->number mapping exists
			HKey := append([]byte{0x48}, hash...) // 'H' + hash
			if _, closer, err := db.Get(HKey); err != nil {
				// Missing, create it
				numBytes := make([]byte, 8)
				binary.BigEndian.PutUint64(numBytes, num)
				
				if err := batch.Set(HKey, numBytes, nil); err == nil {
					missing++
					batchCount++
				}
			} else {
				closer.Close()
				verified++
			}
			
			// Commit batch periodically
			if batchCount >= 1000 {
				if err := batch.Commit(nil); err != nil {
					log.Printf("Failed to commit batch: %v", err)
				}
				batch = db.NewBatch()
				batchCount = 0
			}
		}
	}

	// Commit final batch
	if batchCount > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Printf("Failed to commit final batch: %v", err)
		}
	}

	fmt.Printf("Verified %d existing mappings, created %d missing ones\n", verified, missing)
}

func migrateStateTrie(srcDB, dstDB *pebble.DB, subnetPrefix []byte) {
	// Migrate secure trie nodes (state data)
	count := 0
	batch := dstDB.NewBatch()
	batchCount := 0

	// Secure trie prefix with subnet
	secureTriePrefix := append(subnetPrefix, 0x53) // 'S'
	
	iter, _ := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: secureTriePrefix,
		UpperBound: incrementBytes(secureTriePrefix),
	})
	defer iter.Close()

	fmt.Println("Migrating state trie nodes...")
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Strip subnet prefix, keep the secure trie data
		if len(key) > 33 {
			newKey := append([]byte{0x53}, key[33:]...) // 'S' + rest
			
			if err := batch.Set(newKey, value, nil); err != nil {
				log.Printf("Failed to set secure trie key: %v", err)
				continue
			}
			
			count++
			batchCount++
			
			if batchCount >= 1000 {
				if err := batch.Commit(nil); err != nil {
					log.Printf("Failed to commit batch: %v", err)
				}
				batch = dstDB.NewBatch()
				batchCount = 0
				
				if count%100000 == 0 {
					fmt.Printf("  Migrated %d state nodes...\n", count)
				}
			}
		}
	}

	// Commit final batch
	if batchCount > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Printf("Failed to commit final batch: %v", err)
		}
	}

	fmt.Printf("Migrated %d state trie nodes\n", count)
}

func updateHeadPointers(db *pebble.DB) {
	// Find the highest block with complete data
	var bestNum uint64
	var bestHash []byte

	// Check headers from high to low
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x6e}, // 'n'
		UpperBound: []byte{0x6f},
	})
	defer iter.Close()

	// Find highest block
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := iter.Key()
		if len(key) >= 9 {
			num := binary.BigEndian.Uint64(key[1:9])
			if num < 10000000 {
				bestNum = num
				bestHash = iter.Value()
				break
			}
		}
	}

	if bestHash == nil {
		fmt.Println("WARNING: No valid blocks found!")
		return
	}

	fmt.Printf("Setting head to block %d (hash: %x)\n", bestNum, bestHash)

	// Update all head pointers
	headKeys := [][]byte{
		[]byte("LastBlock"),    // rawdb.HeadBlockKey
		[]byte("LastHeader"),   // rawdb.HeadHeaderKey
		[]byte("LastFast"),     // rawdb.HeadFastBlockKey
	}

	for _, key := range headKeys {
		if err := db.Set(key, bestHash, pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("  Updated %s\n", key)
		}
	}

	// Also store the last pivot block (for fast sync)
	pivotKey := []byte("LastPivot")
	db.Set(pivotKey, bestHash, nil)
}

func migrateAdditionalData(srcDB, dstDB *pebble.DB, subnetPrefix []byte) {
	// Migrate any remaining important data types
	
	// 1. Code (contract bytecode)
	codePrefix := append(subnetPrefix, 0x74) // 't'
	count := migratePrefix(srcDB, dstDB, codePrefix, 0x74, "code")
	fmt.Printf("  Migrated %d code entries\n", count)
	
	// 2. Preimages
	preimagePrefix := append(subnetPrefix, 0x00)
	count = migratePrefix(srcDB, dstDB, preimagePrefix, 0x00, "preimages")
	fmt.Printf("  Migrated %d preimages\n", count)
	
	// 3. Account snapshot data
	snapshotPrefix := append(subnetPrefix, []byte("snapshot")...)
	count = migratePrefixRaw(srcDB, dstDB, snapshotPrefix, []byte("snapshot"))
	fmt.Printf("  Migrated %d snapshot entries\n", count)
}

func migratePrefix(srcDB, dstDB *pebble.DB, srcPrefix []byte, dstType byte, name string) int {
	count := 0
	batch := dstDB.NewBatch()
	batchCount := 0
	
	iter, _ := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: srcPrefix,
		UpperBound: incrementBytes(srcPrefix),
	})
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) > len(srcPrefix) {
			newKey := append([]byte{dstType}, key[len(srcPrefix):]...)
			
			if err := batch.Set(newKey, value, nil); err == nil {
				count++
				batchCount++
			}
			
			if batchCount >= 1000 {
				batch.Commit(nil)
				batch = dstDB.NewBatch()
				batchCount = 0
			}
		}
	}
	
	if batchCount > 0 {
		batch.Commit(nil)
	}
	
	return count
}

func migratePrefixRaw(srcDB, dstDB *pebble.DB, srcPrefix, dstPrefix []byte) int {
	count := 0
	batch := dstDB.NewBatch()
	
	iter, _ := srcDB.NewIter(&pebble.IterOptions{
		LowerBound: srcPrefix,
		UpperBound: incrementBytes(srcPrefix),
	})
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		newKey := append(dstPrefix, key[len(srcPrefix):]...)
		if err := batch.Set(newKey, value, nil); err == nil {
			count++
		}
		
		if count%1000 == 0 {
			batch.Commit(nil)
			batch = dstDB.NewBatch()
		}
	}
	
	batch.Commit(nil)
	return count
}

func verifyMigration(db *pebble.DB) {
	// Final verification
	maxNum := findHighestNumberMapping(db)
	fmt.Printf("\nFinal highest block: %d\n", maxNum)
	
	// Verify we can read block 0 and the highest block
	for _, num := range []uint64{0, maxNum} {
		// Check n mapping
		nKey := make([]byte, 9)
		nKey[0] = 0x6e
		binary.BigEndian.PutUint64(nKey[1:], num)
		
		if hash, closer, err := db.Get(nKey); err == nil {
			fmt.Printf("Block %d -> hash %x\n", num, hash)
			closer.Close()
			
			// Check if we have the header
			hKey := append([]byte{0x68}, nKey[1:]...) // h + number
			hKey = append(hKey, hash...)               // + hash
			
			if _, closer2, err := db.Get(hKey); err == nil {
				fmt.Printf("  Header exists ✓\n")
				closer2.Close()
			} else {
				fmt.Printf("  Header missing ✗\n")
			}
		} else {
			fmt.Printf("Block %d mapping missing\n", num)
		}
	}
	
	// Count final keys
	analyzeDatabase(db)
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