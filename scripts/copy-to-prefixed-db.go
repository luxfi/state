package main

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: copy-to-prefixed-db <source-db> <target-db> <prefix>")
		fmt.Println("Example: copy-to-prefixed-db migrated.db network.db cchain")
		os.Exit(1)
	}

	sourceDB := os.Args[1]
	targetDB := os.Args[2]
	prefix := []byte(os.Args[3])

	// Open source database
	src, err := pebble.Open(sourceDB, &pebble.Options{})
	if err != nil {
		fmt.Printf("Failed to open source database: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	// Open target database
	dst, err := pebble.Open(targetDB, &pebble.Options{})
	if err != nil {
		fmt.Printf("Failed to open target database: %v\n", err)
		os.Exit(1)
	}
	defer dst.Close()

	// Create iterator for source database
	iter, err := src.NewIter(nil)
	if err != nil {
		fmt.Printf("Failed to create iterator: %v\n", err)
		os.Exit(1)
	}
	defer iter.Close()

	// Count keys by type
	counts := make(map[string]int)
	totalCopied := 0

	// Batch for efficient writes
	batch := dst.NewBatch()

	// Copy all keys with prefix
	for iter.First(); iter.Valid(); iter.Next() {
		srcKey := iter.Key()
		srcValue := iter.Value()

		// Create prefixed key
		prefixedKey := append(append([]byte{}, prefix...), srcKey...)

		// Write to target database
		if err := batch.Set(prefixedKey, srcValue, nil); err != nil {
			fmt.Printf("Failed to set key: %v\n", err)
			os.Exit(1)
		}

		// Track key types
		if len(srcKey) > 0 {
			switch srcKey[0] {
			case 0x68:
				if len(srcKey) == 9 {
					height := binary.BigEndian.Uint64(srcKey[1:])
					if height == 1082780 {
						fmt.Printf("✓ Copying canonical hash at height 1082780: 0x%x\n", srcValue)
					}
				}
				counts["canonical"]++
			case 0x48, 'H':
				counts["header"]++
			case 0x62, 'b':
				counts["body"]++
			case 0x72, 'r':
				counts["receipt"]++
			case 0x74, 't':
				counts["txLookup"]++
			default:
				counts["other"]++
			}
		}

		totalCopied++

		// Commit batch periodically
		if totalCopied%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				fmt.Printf("Failed to commit batch: %v\n", err)
				os.Exit(1)
			}
			batch.Close()
			batch = dst.NewBatch()
			fmt.Printf("  Copied %d keys...\n", totalCopied)
		}
	}

	// Commit final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		fmt.Printf("Failed to commit final batch: %v\n", err)
		os.Exit(1)
	}
	batch.Close()

	fmt.Printf("\nCopied %d keys with prefix '%s':\n", totalCopied, prefix)
	for keyType, count := range counts {
		fmt.Printf("  %s: %d\n", keyType, count)
	}

	// Verify the key was copied
	testKey := append(append([]byte{}, prefix...), []byte{0x68, 0, 0, 0, 0, 0, 0x10, 0x89, 0x9c}...)
	if val, closer, err := dst.Get(testKey); err == nil {
		defer closer.Close()
		fmt.Printf("\n✓ Verified: Found canonical hash at height 1082780 in prefixed location\n")
		fmt.Printf("  Key: %x\n", testKey)
		fmt.Printf("  Value: 0x%x\n", val)
	} else {
		fmt.Printf("\n✗ Warning: Could not verify canonical hash at height 1082780\n")
	}
}
