package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := flag.String("db", "", "Path to EVM PebbleDB")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("--db required")
	}

	// Open the database
	opts := &pebble.Options{
		ReadOnly: false,
	}
	db, err := pebble.Open(*dbPath, opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Step 1: Find the tip by scanning headers
	log.Println("Step 1: Finding tip by scanning headers...")
	tipNum := uint64(0)
	var tipHash [32]byte

	// Create iterator for headers (evmh prefix)
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"), // next prefix
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	headerCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 44 && string(key[:4]) == "evmh" {
			// Key format: evmh + 8-byte number + 32-byte hash
			num := binary.BigEndian.Uint64(key[4:12])
			copy(tipHash[:], key[12:44])
			
			if num > tipNum {
				tipNum = num
			}
			headerCount++
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	log.Printf("Found %d headers, tip at height %d, hash %s", headerCount, tipNum, hex.EncodeToString(tipHash[:]))

	if tipNum == 0 {
		log.Fatal("No headers found in database")
	}

	// Step 2: Walk back and build canonical chain map
	log.Println("Step 2: Building canonical chain by walking back from tip...")
	canon := make(map[uint64][32]byte)
	hash := tipHash
	num := tipNum

	for {
		canon[num] = hash
		if num == 0 {
			break
		}

		// Read the header
		key := append([]byte("evmh"), make([]byte, 40)...)
		binary.BigEndian.PutUint64(key[4:12], num)
		copy(key[12:44], hash[:])
		
		val, closer, err := db.Get(key)
		if err != nil {
			log.Fatalf("Missing header at height %d, hash %s", num, hex.EncodeToString(hash[:]))
		}
		
		// Extract parent hash from RLP-encoded header
		// The header is RLP encoded, but we can extract parent hash directly
		// Parent hash is the first field in the header, typically after RLP prefix
		if len(val) < 35 {
			closer.Close()
			log.Fatalf("Header too short at height %d", num)
		}
		
		// Simple extraction: for most headers, parent hash starts at offset 3
		// This works for headers where the RLP list prefix is 2-3 bytes
		parentHashOffset := 3
		if val[0] == 0xf9 { // Long list prefix (2 bytes)
			parentHashOffset = 3
		} else if val[0] == 0xfa { // Even longer list
			parentHashOffset = 3
		}
		
		copy(hash[:], val[parentHashOffset:parentHashOffset+32])
		closer.Close()

		num--

		if num%10000 == 0 {
			log.Printf("Progress: at height %d", num)
		}
	}

	log.Printf("Built canonical chain with %d blocks", len(canon))

	// Step 3: Write evmn (canonical number -> hash) mappings
	log.Println("Step 3: Writing canonical number->hash mappings...")
	batch := db.NewBatch()
	written := 0

	for num, hash := range canon {
		// Key format: evmn + 8-byte big-endian number
		key := make([]byte, 12)
		copy(key[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(key[4:], num)
		
		if err := batch.Set(key, hash[:], nil); err != nil {
			log.Fatalf("Failed to write mapping for height %d: %v", num, err)
		}
		
		written++
		if written%10000 == 0 {
			// Flush batch periodically
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to write batch: %v", err)
			}
			batch = db.NewBatch()
			log.Printf("Written %d mappings...", written)
		}
	}

	// Write final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to write final batch: %v", err)
	}

	log.Printf("Successfully wrote %d canonical mappings", written)
	log.Printf("Canonical chain tip: height=%d, hash=%s", tipNum, hex.EncodeToString(tipHash[:]))
	
	// Verify by reading back a sample
	testNum := tipNum
	testKey := make([]byte, 12)
	copy(testKey[:4], []byte("evmn"))
	binary.BigEndian.PutUint64(testKey[4:], testNum)
	
	if val, closer, err := db.Get(testKey); err == nil {
		defer closer.Close()
		log.Printf("Verification: height %d -> hash %s", testNum, hex.EncodeToString(val))
	}
}