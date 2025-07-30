package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("runtime/lux-96369-vm-ready/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Verifying VM-ready database:")
	fmt.Println("===========================")

	// Check consensus keys
	fmt.Println("\nConsensus keys:")
	heightValue, closer, err := db.Get([]byte("Height"))
	if err == nil {
		height := binary.BigEndian.Uint64(heightValue)
		fmt.Printf("Height: %d (0x%x)\n", height, height)
		closer.Close()
	} else {
		fmt.Printf("Height: not found\n")
	}

	// Check canonical key for block 0
	fmt.Println("\nChecking canonical key (block 0):")
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, 0)

	canonicalKey := []byte{0x68}
	canonicalKey = append(canonicalKey, blockBytes...)
	canonicalKey = append(canonicalKey, 0x6e)

	fmt.Printf("Looking for key: %x\n", canonicalKey)

	if val, closer, err := db.Get(canonicalKey); err == nil {
		fmt.Printf("Found canonical hash: %x\n", val)
		closer.Close()
	} else {
		fmt.Printf("Not found: %v\n", err)
	}

	// Check for headers with raw format
	fmt.Println("\nChecking for headers (0x68 prefix):")
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 5; iter.Next() {
		key := iter.Key()
		fmt.Printf("Header key: %x (len=%d)\n", key, len(key))
		count++
	}

	fmt.Printf("Total headers found with 0x68 prefix: (showing first %d)\n", count)
}
