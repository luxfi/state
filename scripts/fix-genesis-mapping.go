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
	if len(os.Args) < 2 {
		fmt.Println("Usage: fix-genesis-mapping <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}
	defer db.Close()

	// The VM's expected genesis hash
	vmGenesisHash, _ := hex.DecodeString("a24e71001a6a59fb52834b2b4e905f08d1598a7da819467ebb8d9da4129f37ce")

	// Update canonical mapping for block 0
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, 0)

	canonicalKey := []byte{0x68}
	canonicalKey = append(canonicalKey, blockBytes...)
	canonicalKey = append(canonicalKey, 0x6e)

	fmt.Printf("Updating canonical mapping for block 0...\n")
	fmt.Printf("Key: %x\n", canonicalKey)
	fmt.Printf("New value: %x\n", vmGenesisHash)

	if err := db.Set(canonicalKey, vmGenesisHash, nil); err != nil {
		log.Fatal("Failed to update canonical mapping:", err)
	}

	// Verify the update
	if val, closer, err := db.Get(canonicalKey); err == nil {
		fmt.Printf("Verified canonical hash at block 0: %x\n", val)
		closer.Close()
	}

	fmt.Println("Genesis mapping fixed!")
}
