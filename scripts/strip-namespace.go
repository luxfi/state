package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: strip-namespace <source-db> <dest-db>")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]

	fmt.Printf("Stripping namespace prefixes from %s to %s\n", srcPath, dstPath)

	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open source DB: %v", err)
	}
	defer srcDB.Close()

	// Create destination database
	os.MkdirAll(dstPath, 0755)
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create destination DB: %v", err)
	}
	defer dstDB.Close()

	// Expected namespace for chain 96369
	expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
	nsBytes, _ := hex.DecodeString(expectedNamespace)

	iter, err := srcDB.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	totalKeys := 0
	strippedKeys := 0
	batch := dstDB.NewBatch()
	batchSize := 0

	for iter.First(); iter.Valid(); iter.Next() {
		totalKeys++
		key := iter.Key()
		value := iter.Value()

		// Check if key has namespace prefix
		if len(key) >= 33 {
			// Check if first 32 bytes match expected namespace
			hasNamespace := true
			for i := 0; i < 32; i++ {
				if key[i] != nsBytes[i] {
					hasNamespace = false
					break
				}
			}

			if hasNamespace {
				// Strip the 33-byte prefix (32-byte namespace + 1-byte key type)
				keyType := key[32]
				actualKey := key[33:]

				// Add EVM prefix based on key type
				var newKey []byte
				switch keyType {
				case 0x68: // 'h' - headers
					newKey = append([]byte("evmh"), actualKey...)
				case 0x62: // 'b' - bodies
					newKey = append([]byte("evmb"), actualKey...)
				case 0x72: // 'r' - receipts
					newKey = append([]byte("evmr"), actualKey...)
				case 0x6e: // 'n' - canonical (number->hash)
					newKey = append([]byte("evmn"), actualKey...)
				case 0x48: // 'H' - hash->number
					newKey = append([]byte("evmH"), actualKey...)
				case 0x74: // 't' - transactions
					newKey = append([]byte("evmt"), actualKey...)
				case 0x26: // account state
					newKey = actualKey // No prefix for accounts
				case 0x73: // 's' - state
					newKey = actualKey // No prefix for state
				default:
					// For other keys, just strip namespace without adding prefix
					newKey = actualKey
				}

				batch.Set(newKey, value, nil)
				strippedKeys++
			} else {
				// Not our namespace, copy as-is
				batch.Set(key, value, nil)
			}
		} else {
			// Key too short to have namespace, copy as-is
			batch.Set(key, value, nil)
		}

		batchSize++
		if batchSize >= 1000 {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
			batchSize = 0

			if totalKeys%100000 == 0 {
				fmt.Printf("Progress: %d keys processed, %d stripped\n", totalKeys, strippedKeys)
			}
		}
	}

	// Commit final batch
	if batchSize > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Fatalf("Failed to commit final batch: %v", err)
		}
	}

	fmt.Printf("\nCompleted: %d total keys, %d stripped of namespace\n", totalKeys, strippedKeys)
}
