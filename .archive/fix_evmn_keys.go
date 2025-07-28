package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
)

func main() {
	var dbPath = flag.String("db", "", "path to pebbledb to fix")
	flag.Parse()

	if *dbPath == "" {
		flag.Usage()
		log.Fatal("--db is required")
	}

	fmt.Println("=== Fixing evmn Keys ===")
	fmt.Printf("Database: %s\n", *dbPath)

	start := time.Now()

	// Open database
	db, err := pebble.Open(*dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// First, collect all hash->number mappings
	fmt.Println("\nStep 1: Reading hash->number mappings...")
	hashToNumber := make(map[string]uint64)
	
	prefix := []byte("evmH")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) > 4 && len(value) == 8 {
			hash := key[4:] // Skip "evmH" prefix
			number := binary.BigEndian.Uint64(value)
			hashToNumber[string(hash)] = number
		}
	}
	iter.Close()
	
	fmt.Printf("Found %d hash->number mappings\n", len(hashToNumber))

	// Now create proper evmn keys
	fmt.Println("\nStep 2: Creating canonical number->hash keys...")
	batch := db.NewBatch()
	count := 0
	
	for hash, number := range hashToNumber {
		// Create key: evmn + 8-byte number
		key := make([]byte, 12)
		copy(key, []byte("evmn"))
		binary.BigEndian.PutUint64(key[4:], number)
		
		// Value is the hash
		if err := batch.Set(key, []byte(hash), nil); err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}
		
		count++
		if count%100 == 0 {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = db.NewBatch()
			fmt.Printf("Created %d canonical keys...\n", count)
		}
	}
	
	// Commit final batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}
	
	// Remove old evmn keys with wrong format
	fmt.Println("\nStep 3: Removing old evmn keys...")
	oldCount := 0
	
	prefix = []byte("evmn")
	iter2, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	
	batch = db.NewBatch()
	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		// Old keys are longer than 12 bytes (evmn + hash instead of evmn + number)
		if len(key) > 12 {
			if err := batch.Delete(key, nil); err != nil {
				log.Fatalf("Failed to delete key: %v", err)
			}
			oldCount++
			
			if oldCount%100 == 0 {
				if err := batch.Commit(nil); err != nil {
					log.Fatalf("Failed to commit batch: %v", err)
				}
				batch = db.NewBatch()
			}
		}
	}
	iter2.Close()
	
	// Commit final batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}
	
	fmt.Printf("Removed %d old evmn keys\n", oldCount)
	
	// Verify the fix
	fmt.Println("\nStep 4: Verifying fix...")
	
	// Check a few canonical keys
	samples := 0
	for i := uint64(0); i <= 10 && samples < 5; i++ {
		key := make([]byte, 12)
		copy(key, []byte("evmn"))
		binary.BigEndian.PutUint64(key[4:], i)
		
		value, closer, err := db.Get(key)
		if err == nil {
			fmt.Printf("  Block %d -> hash %s\n", i, hex.EncodeToString(value))
			closer.Close()
			samples++
		}
	}
	
	fmt.Printf("\n=== Fix Complete in %s ===\n", time.Since(start))
	fmt.Printf("Created %d canonical number->hash mappings\n", count)
	fmt.Printf("Removed %d old format keys\n", oldCount)
}