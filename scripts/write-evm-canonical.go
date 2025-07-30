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
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <chain-id>\n", os.Args[0])
		os.Exit(1)
	}

	chainID := os.Args[1]
	dbPath := fmt.Sprintf("/home/z/lux-node-data/chainData/%s/db/pebbledb", chainID)

	height := uint64(1082780)
	hashStr := "32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0"
	hash, err := hex.DecodeString(hashStr)
	if err != nil {
		log.Fatalf("Failed to decode hash: %v", err)
	}

	log.Printf("Writing EVM-prefixed canonical hash")
	log.Printf("Database: %s", dbPath)
	log.Printf("Height: %d", height)
	log.Printf("Hash: %x", hash)

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create height bytes
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	batch := db.NewBatch()

	// Write with "evm" prefix followed by the geth canonical key
	// The prefix database will prepend "evm" to all keys, so when the VM
	// accesses key 0x68..., it actually accesses "evm" + 0x68...
	evmCanonicalKey := append([]byte("evm"), 0x68)
	evmCanonicalKey = append(evmCanonicalKey, heightBytes...)

	if err := batch.Set(evmCanonicalKey, hash, nil); err != nil {
		log.Printf("Failed to set evm-prefixed canonical: %v", err)
	} else {
		log.Printf("Wrote evm-prefixed canonical key: %x", evmCanonicalKey)
	}

	// Also try writing "LastAccepted" and other keys the VM might look for
	if err := batch.Set(append([]byte("evm"), []byte("LastAccepted")...), hash, nil); err != nil {
		log.Printf("Failed to set evm+LastAccepted: %v", err)
	}

	if err := batch.Set(append([]byte("evm"), []byte("Height")...), heightBytes, nil); err != nil {
		log.Printf("Failed to set evm+Height: %v", err)
	}

	// Commit batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit: %v", err)
	}

	log.Printf("Successfully wrote EVM-prefixed keys")

	// Verify what we wrote
	val, closer, err := db.Get(evmCanonicalKey)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Verified EVM-prefixed canonical hash: %x", val)
	}

	// Also check if the standard key is still there
	standardKey := append([]byte{0x68}, heightBytes...)
	val2, closer2, err := db.Get(standardKey)
	if err == nil {
		defer closer2.Close()
		log.Printf("✓ Standard canonical hash also present: %x", val2)
	}
}
