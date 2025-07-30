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

	fmt.Println("Checking for consensus/blockchain keys:")
	fmt.Println("======================================")

	// Expected namespace
	expectedNamespace := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
	nsBytes, _ := hex.DecodeString(expectedNamespace)

	// Subnet EVM might use different key prefixes
	// Let's check all possible single-byte prefixes
	fmt.Println("\nChecking all possible key type prefixes:")

	for keyType := byte(0); keyType < 255; keyType++ {
		prefix := append(nsBytes, keyType)

		iter, _ := db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: append(prefix, 0xff),
		})

		count := 0
		var firstKey []byte
		for iter.First(); iter.Valid() && count < 1; iter.Next() {
			firstKey = iter.Key()
			count++
		}
		iter.Close()

		if count > 0 && len(firstKey) > 33 {
			actualKey := firstKey[33:]
			fmt.Printf("Found keys with type 0x%02x: first_key_len=%d, sample=%s\n",
				keyType, len(actualKey), hex.EncodeToString(actualKey[:min(len(actualKey), 20)]))
		}
	}

	// Check without namespace - maybe consensus keys don't use namespace
	fmt.Println("\nChecking keys without namespace prefix:")

	iter, _ := db.NewIter(nil)
	defer iter.Close()

	nonNamespacedKeys := 0
	for iter.First(); iter.Valid() && nonNamespacedKeys < 20; iter.Next() {
		key := iter.Key()

		// Check if it doesn't start with our namespace
		if len(key) < 32 || hex.EncodeToString(key[:32]) != expectedNamespace[:64] {
			fmt.Printf("Non-namespaced key: %s (len=%d)\n",
				hex.EncodeToString(key[:min(len(key), 40)]), len(key))
			nonNamespacedKeys++
		}
	}

	// Try to find consensus database keys
	fmt.Println("\nLooking for Snowman consensus keys:")

	// Snowman consensus might store under different patterns
	consensusPatterns := []string{
		"lastAccepted",
		"LastAccepted",
		"vm/lastAccepted",
		"snowman/lastAccepted",
		"height",
		"Height",
	}

	for _, pattern := range consensusPatterns {
		// Try with namespace
		for keyType := byte(0); keyType < 10; keyType++ {
			testKey := append(nsBytes, keyType)
			testKey = append(testKey, []byte(pattern)...)

			if _, closer, err := db.Get(testKey); err == nil {
				fmt.Printf("Found '%s' with namespace and type 0x%02x\n", pattern, keyType)
				closer.Close()
			}
		}

		// Try without namespace
		if val, closer, err := db.Get([]byte(pattern)); err == nil {
			fmt.Printf("Found '%s' without namespace: %s\n",
				pattern, hex.EncodeToString(val[:min(len(val), 20)]))
			closer.Close()
		}
	}

	// Check the actual structure of type 0x00 keys more carefully
	fmt.Println("\nAnalyzing type 0x00 key structure:")

	prefix := append(nsBytes, 0x00)
	iter2, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	defer iter2.Close()

	for i := 0; iter2.Valid() && i < 5; i++ {
		iter2.First()
		for j := 0; j < i && iter2.Valid(); j++ {
			iter2.Next()
		}

		if iter2.Valid() {
			key := iter2.Key()
			value := iter2.Value()
			actualKey := key[33:]

			fmt.Printf("\nKey %d:\n", i)
			fmt.Printf("  Full key: %s\n", hex.EncodeToString(actualKey))
			fmt.Printf("  First 20 bytes (address?): %s\n", hex.EncodeToString(actualKey[:20]))
			fmt.Printf("  Last 11 bytes: %s\n", hex.EncodeToString(actualKey[20:]))
			fmt.Printf("  Value length: %d\n", len(value))
			fmt.Printf("  Value start: %s\n", hex.EncodeToString(value[:min(len(value), 40)]))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
