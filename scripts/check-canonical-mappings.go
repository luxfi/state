package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("runtime/luxd-migrated/db/chains/C/v1.0.0/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Checking canonical mappings...")

	// Check specific blocks
	blocks := []uint64{0, 1, 2, 1082779, 1082780}
	for _, blockNum := range blocks {
		blockBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(blockBytes, blockNum)
		canonicalKey := append([]byte{0x68}, blockBytes...)
		canonicalKey = append(canonicalKey, 0x6e)

		if val, closer, err := db.Get(canonicalKey); err == nil {
			fmt.Printf("Block %d: %x\n", blockNum, val)
			closer.Close()
		} else {
			fmt.Printf("Block %d: not found\n", blockNum)
		}
	}

	// Count total canonical mappings
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 10 && key[9] == 0x6e {
			count++
		}
	}

	fmt.Printf("\nTotal canonical mappings: %d\n", count)

	// Check header at block 0
	headerKey := append([]byte{0x68}, make([]byte, 8)...)
	headerKey = append(headerKey, []byte{0x61, 0x24, 0xe7, 0x10, 0x01, 0xa6, 0xa5, 0x9f, 0xb5, 0x28, 0x34, 0xb2, 0xb4, 0xe9, 0x05, 0xf0, 0x8d, 0x15, 0x98, 0xa7, 0xda, 0x81, 0x94, 0x67, 0xeb, 0xb8, 0xd9, 0xda, 0x41, 0x29, 0xf3, 0x7c, 0xe0}...)

	if val, closer, err := db.Get(headerKey); err == nil {
		fmt.Printf("\nGenesis header found: %d bytes\n", len(val))
		closer.Close()
	} else {
		fmt.Printf("\nGenesis header not found\n")
	}
}
