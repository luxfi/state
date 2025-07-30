package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"sort"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("chaindata/lux-mainnet-96369/db/pebbledb", &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Deep analysis of database keys:")
	fmt.Println("==============================")

	// Expected namespace
	expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
	nsBytes, _ := hex.DecodeString(expectedNamespace)

	// Collect statistics about key patterns
	keyLengths := make(map[byte]map[int]int) // keyType -> length -> count
	valueLengths := make(map[byte]map[int]int)

	// Look for specific patterns
	iter, _ := db.NewIter(nil)
	defer iter.Close()

	totalKeys := 0
	for iter.First(); iter.Valid() && totalKeys < 500000; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		totalKeys++

		if len(key) >= 33 {
			// Check namespace
			hasNamespace := true
			for i := 0; i < 32; i++ {
				if key[i] != nsBytes[i] {
					hasNamespace = false
					break
				}
			}

			if hasNamespace {
				keyType := key[32]
				actualKey := key[33:]

				// Track key lengths
				if keyLengths[keyType] == nil {
					keyLengths[keyType] = make(map[int]int)
				}
				keyLengths[keyType][len(actualKey)]++

				// Track value lengths
				if valueLengths[keyType] == nil {
					valueLengths[keyType] = make(map[int]int)
				}
				valueLengths[keyType][len(value)]++

				// Look for keys that start with 8-byte numbers
				if len(actualKey) >= 8 && totalKeys < 1000 {
					num := binary.BigEndian.Uint64(actualKey[:8])
					if num > 1000000 && num < 1100000 { // Around block 1082781
						fmt.Printf("Key type 0x%02x with high number: %d (0x%x), full_key=%s, value_len=%d\n",
							keyType, num, num, hex.EncodeToString(actualKey[:min(len(actualKey), 16)]), len(value))
					}
				}
			}
		}
	}

	fmt.Printf("\nAnalyzed %d keys\n\n", totalKeys)

	// Print key length patterns
	fmt.Println("Key length patterns by type:")
	for keyType := byte(0); keyType <= 0x09; keyType++ {
		if lengths, ok := keyLengths[keyType]; ok && len(lengths) > 0 {
			fmt.Printf("\nType 0x%02x:\n", keyType)

			// Sort lengths
			var sortedLengths []int
			for l := range lengths {
				sortedLengths = append(sortedLengths, l)
			}
			sort.Ints(sortedLengths)

			for _, l := range sortedLengths {
				count := lengths[l]
				if count > 100 { // Only show significant patterns
					fmt.Printf("  Length %d: %d occurrences", l, count)

					// Identify common patterns
					switch l {
					case 8:
						fmt.Print(" (block number?)")
					case 20:
						fmt.Print(" (address?)")
					case 32:
						fmt.Print(" (hash?)")
					case 40: // 8 + 32
						fmt.Print(" (block + hash?)")
					case 31: // versiondb suffix?
						fmt.Print(" (address + version?)")
					}
					fmt.Println()
				}
			}
		}
	}

	// Look specifically for Height key
	fmt.Println("\nLooking for Height/LastBlock patterns:")

	// Try different encodings
	heightPatterns := []string{"Height", "height", "LastBlock", "lastBlock", "HEAD", "head"}

	for _, pattern := range heightPatterns {
		for keyType := byte(0); keyType <= 0x09; keyType++ {
			testKey := append(nsBytes, keyType)
			testKey = append(testKey, []byte(pattern)...)

			if value, closer, err := db.Get(testKey); err == nil {
				fmt.Printf("Found '%s' with type 0x%02x: value=%s\n",
					pattern, keyType, hex.EncodeToString(value[:min(len(value), 20)]))
				closer.Close()
			}
		}
	}

	// Check if any keys contain the target block number in their value
	fmt.Printf("\nSearching for block number 1082781 (0x10859d) in values:\n")
	targetBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(targetBytes, 1082781)

	iter2, _ := db.NewIter(nil)
	defer iter2.Close()

	found := 0
	checked := 0
	for iter2.First(); iter2.Valid() && found < 10 && checked < 100000; iter2.Next() {
		value := iter2.Value()
		checked++

		// Look for the block number in the value
		if len(value) >= 8 {
			for i := 0; i <= len(value)-8; i++ {
				if string(value[i:i+8]) == string(targetBytes) {
					key := iter2.Key()
					keyType := byte(0xff)
					if len(key) >= 33 {
						keyType = key[32]
					}
					fmt.Printf("Found in value at offset %d, key type=0x%02x, key=%s\n",
						i, keyType, hex.EncodeToString(key[:min(len(key), 40)]))
					found++
					break
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
