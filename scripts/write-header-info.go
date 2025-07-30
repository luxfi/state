package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <db-path> <hash>\n", os.Args[0])
		os.Exit(1)
	}

	dbPath := os.Args[1]
	hashStr := os.Args[2]
	height := uint64(1082780)

	// Decode hash
	hash, err := hex.DecodeString(hashStr)
	if err != nil {
		log.Fatalf("Failed to decode hash: %v", err)
	}

	log.Printf("Writing header information to database")
	log.Printf("Database: %s", dbPath)
	log.Printf("Height: %d", height)
	log.Printf("Hash: %x", hash)

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()

	// Write various head pointers that geth expects
	// These are standard geth database keys

	// headBlockKey = []byte("LastBlock")
	if err := batch.Set([]byte("LastBlock"), hash, nil); err != nil {
		log.Printf("Failed to set LastBlock: %v", err)
	}

	// headHeaderKey = []byte("LastHeader")
	if err := batch.Set([]byte("LastHeader"), hash, nil); err != nil {
		log.Printf("Failed to set LastHeader: %v", err)
	}

	// headFastBlockKey = []byte("LastFast")
	if err := batch.Set([]byte("LastFast"), hash, nil); err != nil {
		log.Printf("Failed to set LastFast: %v", err)
	}

	// Write with 0x prefix versions too
	if err := batch.Set([]byte{0x48}, hash, nil); err != nil { // 'H' for head block
		log.Printf("Failed to set head block: %v", err)
	}

	// Commit batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}

	log.Printf("Successfully wrote header pointers")

	// Verify the canonical hash is still there
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	canonicalKey := append([]byte{0x68}, heightBytes...)

	val, closer, err := db.Get(canonicalKey)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Verified canonical hash at height %d: %x", height, val)

		// Check if they match
		if bytes.Equal(val, hash) {
			log.Printf("✓ Canonical hash matches expected hash")
		} else {
			log.Printf("✗ WARNING: Canonical hash does not match! Expected: %x, Got: %x", hash, val)
		}
	} else {
		log.Printf("✗ Could not find canonical hash at height %d", height)
	}
}
