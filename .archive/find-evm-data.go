package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: find-evm-data <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open PebbleDB
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Finding EVM Data in Subnet Database ===")
	fmt.Println("Database:", dbPath)
	fmt.Println()

	// Subnet EVM uses different key patterns
	// Looking for:
	// - "lastAccepted" keys
	// - Block data with specific prefixes
	// - Height markers
	// - Finalized blocks

	interestingKeys := []string{}
	keyPatterns := make(map[string]int)

	// Scan database
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	totalKeys := 0
	for iter.First(); iter.Valid() && totalKeys < 100000; iter.Next() {
		key := iter.Key()
		val := iter.Value()

		keyHex := hex.EncodeToString(key)

		// Look for text-based keys (subnet style)
		if bytes.Contains(key, []byte("last")) ||
			bytes.Contains(key, []byte("height")) ||
			bytes.Contains(key, []byte("block")) ||
			bytes.Contains(key, []byte("finalized")) ||
			bytes.Contains(key, []byte("accepted")) {
			fmt.Printf("Found text key: %s = %s\n", string(key), hex.EncodeToString(val))
			interestingKeys = append(interestingKeys, string(key))
		}

		// Check for block-like structures
		if len(val) > 100 && len(val) < 10000 {
			// Could be RLP-encoded block
			if val[0] == 0xf9 || val[0] == 0xfa || val[0] == 0xfb {
				fmt.Printf("Possible RLP block at key %s (size: %d)\n", keyHex, len(val))
				keyPatterns["rlp_block"]++
			}
		}

		// Check for 32-byte values (hashes)
		if len(val) == 32 {
			keyPatterns["32_byte_values"]++
			if keyPatterns["32_byte_values"] <= 5 {
				fmt.Printf("32-byte value at key %s: %s\n", keyHex, hex.EncodeToString(val))
			}
		}

		// Check for height-like keys (8-byte big-endian numbers)
		if len(key) == 8 {
			height := binary.BigEndian.Uint64(key)
			if height < 1000000 { // reasonable block height
				keyPatterns["height_keys"]++
				if keyPatterns["height_keys"] <= 5 {
					fmt.Printf("Possible height key %d: value size %d\n", height, len(val))
				}
			}
		}

		// Check specific prefixes
		if len(key) > 0 {
			prefix := key[0]
			keyPatterns[fmt.Sprintf("prefix_%02x", prefix)]++
		}

		totalKeys++
		if totalKeys%10000 == 0 {
			fmt.Printf("Scanned %d keys...\n", totalKeys)
		}
	}

	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total keys scanned: %d\n", totalKeys)

	fmt.Println("\n=== Key Patterns ===")
	for pattern, count := range keyPatterns {
		if count > 100 {
			fmt.Printf("%s: %d\n", pattern, count)
		}
	}

	fmt.Println("\n=== Interesting Keys Found ===")
	for _, k := range interestingKeys {
		fmt.Println(k)
	}

	// Try to read specific subnet keys
	fmt.Println("\n=== Checking Known Subnet Keys ===")

	subnetKeys := []string{
		"lastAccepted",
		"lastAcceptedKey",
		"vm/lastAcceptedKey",
		"height",
		"finalized",
		"proposervm_height",
		"proposervm_lastAccepted",
	}

	for _, k := range subnetKeys {
		if val, closer, err := db.Get([]byte(k)); err == nil {
			fmt.Printf("%s: %s\n", k, hex.EncodeToString(val))
			closer.Close()
		}
	}
}
