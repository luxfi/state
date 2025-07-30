package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := "runtime/luxd-migrated/db/chains/C/v1.0.0/evm"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Database state analysis:")
	fmt.Println("=======================")

	// Check Height key
	if val, closer, err := db.Get([]byte("Height")); err == nil {
		height := binary.BigEndian.Uint64(val)
		fmt.Printf("Height: %d (0x%x)\n", height, height)
		closer.Close()
	}

	// Check LastBlock
	if val, closer, err := db.Get([]byte("LastBlock")); err == nil {
		fmt.Printf("LastBlock: %x\n", val)
		closer.Close()
	}

	// Check for canonical at 0
	canonicalKey0 := append([]byte{0x68}, make([]byte, 8)...)
	canonicalKey0 = append(canonicalKey0, 0x6e)
	if val, closer, err := db.Get(canonicalKey0); err == nil {
		fmt.Printf("\nCanonical at block 0: %x\n", val)
		closer.Close()
	}

	// Check for canonical at 1
	block1Bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(block1Bytes, 1)
	canonicalKey1 := append([]byte{0x68}, block1Bytes...)
	canonicalKey1 = append(canonicalKey1, 0x6e)
	if val, closer, err := db.Get(canonicalKey1); err == nil {
		fmt.Printf("Canonical at block 1: %x\n", val)
		closer.Close()
	}

	// Check highest block
	highestBlock := uint64(1082780)
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, highestBlock)
	canonicalKeyHigh := append([]byte{0x68}, blockBytes...)
	canonicalKeyHigh = append(canonicalKeyHigh, 0x6e)
	if val, closer, err := db.Get(canonicalKeyHigh); err == nil {
		fmt.Printf("Canonical at block %d: %x\n", highestBlock, val)
		closer.Close()
	}

	// Try to find blocks in the 0x68 prefix range
	fmt.Println("\nScanning for blocks with 0x68 prefix:")
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	count := 0
	minBlock := uint64(^uint64(0))
	maxBlock := uint64(0)

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 10 && key[9] == 0x6e { // canonical mapping
			blockNum := binary.BigEndian.Uint64(key[1:9])
			if blockNum < minBlock {
				minBlock = blockNum
			}
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
			count++

			if count <= 5 || blockNum == 0 || blockNum == 1 {
				fmt.Printf("  Block %d: %x\n", blockNum, iter.Value())
			}
		}
	}

	fmt.Printf("\nFound %d canonical mappings\n", count)
	fmt.Printf("Block range: %d to %d\n", minBlock, maxBlock)
}
