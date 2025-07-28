package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: fix-mappings-correctly <db_path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	fmt.Println("=== Fixing Number->Hash Mappings Correctly ===")
	
	// Step 1: Delete all incorrect number->hash mappings
	fmt.Println("\nStep 1: Cleaning up incorrect mappings...")
	deleteIncorrectMappings(db)
	
	// Step 2: Delete duplicate/malformed headers
	fmt.Println("\nStep 2: Removing malformed headers...")
	removeMalformedHeaders(db)
	
	// Step 3: Rebuild number->hash mappings from correct headers
	fmt.Println("\nStep 3: Rebuilding number->hash mappings from headers...")
	rebuildMappingsCorrectly(db)
	
	// Step 4: Update head pointers with correct hash
	fmt.Println("\nStep 4: Updating head pointers...")
	updateHeadPointersCorrectly(db)
	
	// Step 5: Verify
	fmt.Println("\nStep 5: Verifying...")
	verifyFixed(db)
	
	fmt.Println("\n=== Fix Complete ===")
}

func deleteIncorrectMappings(db *pebble.DB) {
	// Delete all existing number->hash mappings
	batch := db.NewBatch()
	count := 0
	
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x6e}, // 'n'
		UpperBound: []byte{0x6f},
	})
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		
		if err := batch.Delete(keyCopy, nil); err != nil {
			log.Printf("Failed to delete key: %v", err)
			continue
		}
		
		count++
		if count%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				log.Printf("Failed to commit batch: %v", err)
			}
			batch = db.NewBatch()
			fmt.Printf("  Deleted %d mappings...\n", count)
		}
	}
	
	if err := batch.Commit(nil); err != nil {
		log.Printf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("Deleted %d incorrect mappings\n", count)
}

func removeMalformedHeaders(db *pebble.DB) {
	// Remove headers with wrong format (10-byte keys)
	batch := db.NewBatch()
	count := 0
	
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68}, // 'h'
		UpperBound: []byte{0x69},
	})
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Delete if it's a malformed header (10 bytes: h + number + 0x6e)
		if len(key) == 10 && key[9] == 0x6e {
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			
			if err := batch.Delete(keyCopy, nil); err != nil {
				log.Printf("Failed to delete malformed header: %v", err)
				continue
			}
			
			count++
			if count%1000 == 0 {
				if err := batch.Commit(nil); err != nil {
					log.Printf("Failed to commit batch: %v", err)
				}
				batch = db.NewBatch()
			}
		}
	}
	
	if err := batch.Commit(nil); err != nil {
		log.Printf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("Removed %d malformed headers\n", count)
}

func rebuildMappingsCorrectly(db *pebble.DB) {
	// Rebuild number->hash mappings from correct headers (41-byte keys)
	batch := db.NewBatch()
	count := 0
	highest := uint64(0)
	var highestHash []byte
	
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68}, // 'h'
		UpperBound: []byte{0x69},
	})
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Only process correct headers (41 bytes: h + number + hash)
		if len(key) == 41 {
			num := binary.BigEndian.Uint64(key[1:9])
			hash := key[9:41]
			
			// Skip unreasonable block numbers
			if num >= 10000000 {
				continue
			}
			
			// Create number->hash mapping
			nKey := make([]byte, 9)
			nKey[0] = 0x6e // 'n'
			binary.BigEndian.PutUint64(nKey[1:], num)
			
			if err := batch.Set(nKey, hash, nil); err != nil {
				log.Printf("Failed to set n key for block %d: %v", num, err)
				continue
			}
			
			// Track highest block
			if num > highest {
				highest = num
				highestHash = make([]byte, len(hash))
				copy(highestHash, hash)
			}
			
			count++
			if count%10000 == 0 {
				if err := batch.Commit(nil); err != nil {
					log.Printf("Failed to commit batch: %v", err)
				}
				batch = db.NewBatch()
				fmt.Printf("  Created %d mappings...\n", count)
			}
		}
	}
	
	if err := batch.Commit(nil); err != nil {
		log.Printf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("Created %d correct number->hash mappings\n", count)
	fmt.Printf("Highest block: %d (hash: %x)\n", highest, highestHash)
}

func updateHeadPointersCorrectly(db *pebble.DB) {
	// Find the correct highest block
	var bestNum uint64
	var bestHash []byte
	
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x6e}, // 'n'
		UpperBound: []byte{0x6f},
	})
	defer iter.Close()
	
	// Find highest block by iterating backwards
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := iter.Key()
		if len(key) == 9 {
			num := binary.BigEndian.Uint64(key[1:9])
			if num < 10000000 { // Reasonable limit
				bestNum = num
				bestHash = make([]byte, len(iter.Value()))
				copy(bestHash, iter.Value())
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
		[]byte("LastBlock"),
		[]byte("LastHeader"), 
		[]byte("LastFast"),
		[]byte("LastPivot"),
	}
	
	for _, key := range headKeys {
		if err := db.Set(key, bestHash, pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("  Updated %s\n", key)
		}
	}
}

func verifyFixed(db *pebble.DB) {
	// Check a few blocks to verify the fix
	blocks := []uint64{0, 1, 2, 100, 1000, 10000}
	
	fmt.Println("\nVerifying sample blocks:")
	for _, num := range blocks {
		// Get hash from number->hash mapping
		nKey := make([]byte, 9)
		nKey[0] = 0x6e
		binary.BigEndian.PutUint64(nKey[1:], num)
		
		hash, closer, err := db.Get(nKey)
		if err != nil {
			continue
		}
		closer.Close()
		
		// Check if header exists
		hKey := make([]byte, 41)
		hKey[0] = 0x68
		copy(hKey[1:9], nKey[1:9])
		copy(hKey[9:41], hash)
		
		if _, closer2, err := db.Get(hKey); err == nil {
			fmt.Printf("  Block %d: ✓ (hash: %x...)\n", num, hash[:8])
			closer2.Close()
		} else {
			fmt.Printf("  Block %d: ✗ Header missing\n", num)
		}
	}
	
	// Check highest block
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x6e},
		UpperBound: []byte{0x6f},
	})
	
	var highest uint64
	for iter.Last(); iter.Valid(); iter.Prev() {
		key := iter.Key()
		if len(key) == 9 {
			num := binary.BigEndian.Uint64(key[1:9])
			if num < 10000000 {
				highest = num
				break
			}
		}
	}
	iter.Close()
	
	fmt.Printf("\nHighest block with mapping: %d\n", highest)
}