package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("runtime/lux-96369-fixed/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Verifying fixed migration:")
	fmt.Println("=========================")

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

	// Check for evmh keys (headers)
	fmt.Println("\nChecking headers:")
	evmhIter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer evmhIter.Close()

	headerCount := 0
	maxHeaderBlock := uint64(0)
	for evmhIter.First(); evmhIter.Valid(); evmhIter.Next() {
		key := evmhIter.Key()
		if len(key) >= 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			if blockNum > maxHeaderBlock {
				maxHeaderBlock = blockNum
			}
		}
		headerCount++
		if headerCount > 10 {
			evmhIter.Last() // Skip to end
		}
	}
	fmt.Printf("Headers found: %d, max block: %d\n", headerCount, maxHeaderBlock)

	// Check for evmn keys (canonical)
	fmt.Println("\nChecking canonical mappings:")
	evmnIter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	defer evmnIter.Close()

	canonCount := 0
	for evmnIter.First(); evmnIter.Valid(); evmnIter.Next() {
		canonCount++
		if canonCount > 10 {
			evmnIter.Last()
		}
	}
	fmt.Printf("Canonical mappings found: %d\n", canonCount)

	// Check for specific blocks
	fmt.Println("\nChecking specific blocks:")
	targetBlocks := []uint64{0, 1, 2, 1082779, 1082780, 1082781}

	for _, blockNum := range targetBlocks {
		blockBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(blockBytes, blockNum)

		// Check header
		headerKey := append([]byte("evmh"), blockBytes...)
		iter, _ := db.NewIter(&pebble.IterOptions{
			LowerBound: headerKey,
			UpperBound: append(headerKey, 0xff, 0xff, 0xff, 0xff),
		})

		hasHeader := false
		if iter.First() && iter.Valid() {
			key := iter.Key()
			if len(key) >= 12 {
				foundBlock := binary.BigEndian.Uint64(key[4:12])
				if foundBlock == blockNum {
					hasHeader = true
				}
			}
		}
		iter.Close()

		fmt.Printf("Block %d: header=%v\n", blockNum, hasHeader)
	}

	// Check account balances
	fmt.Println("\nChecking treasury account:")
	treasuryAddr, _ := hex.DecodeString("9011e888251ab053b7bd1cdb598db4f9ded94714")

	// Try different possible key formats
	keys := [][]byte{
		treasuryAddr,
		append([]byte{0x00}, treasuryAddr...),
		append([]byte{0x26}, treasuryAddr...),
	}

	for i, key := range keys {
		if value, closer, err := db.Get(key); err == nil {
			fmt.Printf("Found treasury with key format %d: value_len=%d\n", i, len(value))
			closer.Close()
		}
	}
}
