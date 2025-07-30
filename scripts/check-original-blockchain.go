package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("chaindata/lux-mainnet-96369/db/pebbledb", &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Checking original database for blockchain data:")
	fmt.Println("==============================================")

	// Expected namespace for chain 96369
	expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
	nsBytes, _ := hex.DecodeString(expectedNamespace)

	// Count key types
	keyTypes := make(map[byte]int)
	totalKeys := 0

	iter, _ := db.NewIter(nil)
	defer iter.Close()

	for iter.First(); iter.Valid() && totalKeys < 1000000; iter.Next() {
		key := iter.Key()
		totalKeys++

		if len(key) >= 33 {
			// Check if it has our namespace
			hasNamespace := true
			for i := 0; i < 32; i++ {
				if key[i] != nsBytes[i] {
					hasNamespace = false
					break
				}
			}

			if hasNamespace {
				keyType := key[32]
				keyTypes[keyType]++
			}
		}
	}

	fmt.Printf("Scanned %d keys\n", totalKeys)
	fmt.Println("\nKey type distribution:")

	typeNames := map[byte]string{
		0x68: "headers (h)",
		0x62: "bodies (b)",
		0x72: "receipts (r)",
		0x6e: "canonical (n)",
		0x48: "hash->number (H)",
		0x74: "transactions (t)",
		0x26: "accounts (&)",
		0x73: "state (s)",
		0x00: "consensus (\\x00)",
	}

	for keyType, count := range keyTypes {
		name, ok := typeNames[keyType]
		if !ok {
			name = fmt.Sprintf("unknown (0x%02x)", keyType)
		}
		fmt.Printf("  %s: %d\n", name, count)
	}

	// Check for Height key
	heightKey := append(nsBytes, 0x00) // consensus keys use 0x00
	heightKey = append(heightKey, []byte("Height")...)
	value, closer, err := db.Get(heightKey)
	if err == nil {
		fmt.Printf("\nHeight key found: %s\n", hex.EncodeToString(value))
		closer.Close()
	} else {
		fmt.Printf("\nHeight key not found\n")
	}
}
