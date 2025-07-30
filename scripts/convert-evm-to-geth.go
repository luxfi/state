package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

// Mapping of EVM prefixes to Geth prefixes
var prefixMap = map[string]byte{
	"evmh": 0x48, // headers
	"evmn": 0x68, // canonical
	"evmb": 0x62, // bodies
	"evmr": 0x72, // receipts
	"evmt": 0x74, // transactions
	"evmR": 0x52, // block receipts
	"evmd": 0x44, // diffs
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <source-db> <dest-db>\n", os.Args[0])
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]

	log.Printf("Converting EVM database to Geth format")
	log.Printf("Source: %s", srcPath)
	log.Printf("Destination: %s", dstPath)

	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Open destination database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open destination database: %v", err)
	}
	defer dstDB.Close()

	// Convert all keys
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	batch := dstDB.NewBatch()
	convertCount := 0
	copyCount := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Check if key starts with any EVM prefix
		var newKey []byte
		converted := false

		for evmPrefix, gethPrefix := range prefixMap {
			if bytes.HasPrefix(key, []byte(evmPrefix)) {
				// Convert EVM prefix to Geth prefix
				newKey = make([]byte, len(key)-len(evmPrefix)+1)
				newKey[0] = gethPrefix
				copy(newKey[1:], key[len(evmPrefix):])
				converted = true
				convertCount++
				break
			}
		}

		if !converted {
			// Copy key as-is if not an EVM prefix
			newKey = make([]byte, len(key))
			copy(newKey, key)
			copyCount++
		}

		// Write to destination
		valueCopy := make([]byte, len(value))
		copy(valueCopy, value)

		if err := batch.Set(newKey, valueCopy, nil); err != nil {
			log.Printf("Failed to set key %x: %v", newKey, err)
			continue
		}

		// Commit batch every 10000 keys
		if (convertCount+copyCount)%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
			log.Printf("Progress: %d keys converted, %d copied", convertCount, copyCount)
		}
	}

	// Commit final batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	log.Printf("Conversion complete!")
	log.Printf("Total keys converted: %d", convertCount)
	log.Printf("Total keys copied: %d", copyCount)
	log.Printf("Grand total: %d", convertCount+copyCount)

	// Verify canonical hash at height 1082780
	heightBytes := make([]byte, 8)
	heightBytes[0] = 0x00
	heightBytes[1] = 0x00
	heightBytes[2] = 0x00
	heightBytes[3] = 0x00
	heightBytes[4] = 0x00
	heightBytes[5] = 0x10
	heightBytes[6] = 0x89
	heightBytes[7] = 0x9c

	canonicalKey := append([]byte{0x68}, heightBytes...)
	val, closer, err := dstDB.Get(canonicalKey)
	if err == nil {
		defer closer.Close()
		log.Printf("✓ Verified canonical hash at height 1082780: %x", val)
	} else {
		log.Printf("✗ Could not find canonical hash at height 1082780")
	}
}
