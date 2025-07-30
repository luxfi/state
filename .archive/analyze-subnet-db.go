package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: analyze-subnet-db <db-path>")
		fmt.Println("Example: analyze-subnet-db output/mainnet/C/chaindata-namespaced")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("=== Analyzing Subnet Database: %s ===\n\n", dbPath)

	// Check key patterns
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	patterns := make(map[string]int)
	blocks := make(map[uint64]bool)
	accounts := 0
	storage := 0

	// Sample some keys
	count := 0
	for iter.First(); iter.Valid() && count < 200000; iter.Next() {
		key := iter.Key()
		val := iter.Value()

		if len(key) == 0 {
			continue
		}

		// Categorize by length and prefix
		var pattern string
		switch {
		case len(key) == 10 && key[0] == 0x00 && key[9] == 0x6e: // 'n'
			// Canonical hash key: 0x00 + 8 bytes number + 'n'
			pattern = "canonical-hash"
			blockNum := uint64(0)
			for i := 1; i < 9; i++ {
				blockNum = (blockNum << 8) | uint64(key[i])
			}
			blocks[blockNum] = true
			if count < 10 {
				fmt.Printf("Found canonical hash key for block %d: %x\n", blockNum, key)
			}

		case len(key) == 10 && key[0] == 0x00 && key[9] == 0x01:
			pattern = "account-trie"
			accounts++

		case len(key) > 10 && key[0] == 0x00 && key[9] == 0xa3:
			pattern = "storage-trie"
			storage++

		case len(key) == 33 && key[0] == 0x00:
			pattern = "header-by-hash"

		case len(key) == 41 && key[0] == 0x00:
			pattern = "hash-to-number"

		default:
			pattern = fmt.Sprintf("unknown-len-%d-prefix-%02x", len(key), key[0])
		}

		patterns[pattern]++
		count++

		// Try to decode blocks
		if pattern == "header-by-hash" && count < 5 {
			var header types.Header
			if err := rlp.DecodeBytes(val, &header); err == nil {
				fmt.Printf("Decoded header: block=%d hash=%x\n", header.Number.Uint64(), header.Hash())
			}
		}
	}

	fmt.Println("\nKey Pattern Distribution:")
	for pattern, cnt := range patterns {
		fmt.Printf("  %s: %d keys\n", pattern, cnt)
	}

	fmt.Printf("\nBlock Numbers Found: %d\n", len(blocks))
	if len(blocks) > 0 {
		// Find min/max
		min, max := uint64(^uint64(0)), uint64(0)
		for num := range blocks {
			if num < min {
				min = num
			}
			if num > max {
				max = num
			}
		}
		fmt.Printf("Block range: %d to %d\n", min, max)
	}

	fmt.Printf("\nAccount entries: %d\n", accounts)
	fmt.Printf("Storage entries: %d\n", storage)

	// Now let's look for specific keys that geth expects
	fmt.Println("\n=== Checking for Geth Expected Keys ===")

	// Check for LastBlock key
	lastBlockKey := []byte("LastBlock")
	if val, closer, err := db.Get(lastBlockKey); err == nil {
		fmt.Printf("✓ Found LastBlock: %x\n", val)
		closer.Close()
	} else {
		fmt.Printf("✗ LastBlock not found\n")
	}

	// Check for LastHeader key
	lastHeaderKey := []byte("LastHeader")
	if val, closer, err := db.Get(lastHeaderKey); err == nil {
		fmt.Printf("✓ Found LastHeader: %x\n", val)
		closer.Close()
	} else {
		fmt.Printf("✗ LastHeader not found\n")
	}

	// Check for canonical hash key for block 0
	canonicalKey := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6e}
	if val, closer, err := db.Get(canonicalKey); err == nil {
		fmt.Printf("✓ Found canonical hash for block 0: %x\n", val)
		closer.Close()
	} else {
		fmt.Printf("✗ Canonical hash for block 0 not found (key: %x)\n", canonicalKey)
	}
}
