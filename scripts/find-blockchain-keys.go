package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("runtime/lux-96369-stripped/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Looking for blockchain keys:")
	fmt.Println("============================")

	// Look for evmh (headers)
	fmt.Println("\nevmh (headers):")
	evmhIter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer evmhIter.Close()

	count := 0
	for evmhIter.First(); evmhIter.Valid() && count < 5; evmhIter.Next() {
		key := evmhIter.Key()
		// evmh + 8 bytes block number + 32 bytes hash
		if len(key) >= 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			fmt.Printf("Header at block %d: key=%s\n", blockNum, hex.EncodeToString(key[:min(len(key), 20)]))
			count++
		}
	}
	if count == 0 {
		fmt.Println("No headers found")
	}

	// Look for evmn (canonical)
	fmt.Println("\nevmn (canonical number->hash):")
	evmnIter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmn"),
		UpperBound: []byte("evmo"),
	})
	defer evmnIter.Close()

	count = 0
	maxBlock := uint64(0)
	for evmnIter.First(); evmnIter.Valid(); evmnIter.Next() {
		key := evmnIter.Key()
		// evmn + 8 bytes block number
		if len(key) == 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
			if count < 5 {
				fmt.Printf("Canonical at block %d: key=%s\n", blockNum, hex.EncodeToString(key))
				count++
			}
		}
	}
	if count == 0 {
		fmt.Println("No canonical mappings found")
	} else {
		fmt.Printf("Max block number found: %d (0x%x)\n", maxBlock, maxBlock)
	}

	// Look for evmH (hash->number)
	fmt.Println("\nevmH (hash->number):")
	evmHIter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmH"),
		UpperBound: []byte("evmI"),
	})
	defer evmHIter.Close()

	count = 0
	for evmHIter.First(); evmHIter.Valid() && count < 5; evmHIter.Next() {
		value := evmHIter.Value()
		if len(value) == 32 {
			// Value is the hash
			fmt.Printf("Hash mapping: value=%s\n", hex.EncodeToString(value[:10]))
			count++
		}
	}
	if count == 0 {
		fmt.Println("No hash mappings found")
	}

	// Look for Height key
	fmt.Println("\nConsensus keys:")
	heightValue, closer, err := db.Get([]byte("Height"))
	if err == nil {
		fmt.Printf("Height: %s", hex.EncodeToString(heightValue))
		if len(heightValue) == 8 {
			h := binary.BigEndian.Uint64(heightValue)
			fmt.Printf(" = %d (0x%x)", h, h)
		}
		fmt.Println()
		closer.Close()
	} else {
		fmt.Println("Height: not found")
	}

	// Check account data
	fmt.Println("\nChecking account data:")
	accountIter, _ := db.NewIter(&pebble.IterOptions{
		UpperBound: []byte{0x30}, // Accounts are usually in lower range
	})
	defer accountIter.Close()

	count = 0
	for accountIter.First(); accountIter.Valid() && count < 5; accountIter.Next() {
		key := accountIter.Key()
		if len(key) == 20 || len(key) == 31 { // Address keys
			fmt.Printf("Account key: %s (len=%d)\n", hex.EncodeToString(key[:min(len(key), 20)]), len(key))
			count++
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
