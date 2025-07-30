package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <state-db-path> <height> <hash>\n", os.Args[0])
		os.Exit(1)
	}

	stateDbPath := os.Args[1]
	height := uint64(1082780) // Use hardcoded height to ensure correct
	hashStr := os.Args[3]

	// Parse hash
	hash, err := hex.DecodeString(hashStr)
	if err != nil {
		log.Fatalf("Failed to decode hash: %v", err)
	}

	log.Printf("Writing consensus markers to state database: %s", stateDbPath)
	log.Printf("Height: %d", height)
	log.Printf("Hash: %x", hash)

	// Open state database
	db, err := pebble.Open(stateDbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	batch := db.NewBatch()

	// Write Height pointer
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)
	if err := batch.Set([]byte("Height"), heightBytes, nil); err != nil {
		log.Fatalf("Failed to set Height: %v", err)
	}
	log.Printf("Set Height = %d", height)

	// Write LastAccepted pointer
	if err := batch.Set([]byte("LastAccepted"), hash, nil); err != nil {
		log.Fatalf("Failed to set LastAccepted: %v", err)
	}
	log.Printf("Set LastAccepted = %x", hash)

	// Write consensus/accepted key
	acceptedKey := []byte("consensus/accepted")
	if err := batch.Set(acceptedKey, hash, nil); err != nil {
		log.Fatalf("Failed to set consensus/accepted: %v", err)
	}
	log.Printf("Set consensus/accepted = %x", hash)

	// Commit all changes
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}

	log.Printf("Successfully wrote consensus markers")

	// Verify the writes
	val, closer, err := db.Get([]byte("Height"))
	if err == nil {
		defer closer.Close()
		readHeight := binary.BigEndian.Uint64(val)
		log.Printf("Verified Height = %d", readHeight)
	}

	val2, closer2, err := db.Get([]byte("LastAccepted"))
	if err == nil {
		defer closer2.Close()
		log.Printf("Verified LastAccepted = %x", val2)
	}

	val3, closer3, err := db.Get(acceptedKey)
	if err == nil {
		defer closer3.Close()
		log.Printf("Verified consensus/accepted = %x", val3)
	}
}
