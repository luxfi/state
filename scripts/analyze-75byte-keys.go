package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	
	"github.com/cockroachdb/pebble"
)

func main() {
	sourceDB := "/home/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb"
	source, err := pebble.Open(sourceDB, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer source.Close()

	namespace := []byte{
		0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31,
		0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b,
		0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c,
		0x6f, 0xd1, 0x64, 0xf1, 0xd1,
	}

	iter, err := source.NewIter(nil)
	if err != nil {
		log.Fatal(err)
	}
	defer iter.Close()

	fmt.Println("Analyzing 75-byte keys in SubnetEVM database...")
	fmt.Printf("Namespace: %x\n\n", namespace)
	
	count75 := 0
	samples75 := [][]byte{}
	maxBlock := uint64(0)
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		keyLen := len(key)
		
		// Focus on 75-byte keys
		if keyLen == 75 {
			count75++
			if len(samples75) < 10 {
				samples75 = append(samples75, append([]byte(nil), key...))
			}
			
			// Check if it's namespaced
			if bytes.HasPrefix(key, namespace) {
				actualKey := key[32:]
				// 75 - 32 = 43 bytes for actual key
				
				// Try to interpret as block-related
				if len(actualKey) >= 9 {
					// Check for patterns like H<num><hash> or h<num><hash>n
					if actualKey[0] == 'H' || actualKey[0] == 'h' {
						num := binary.BigEndian.Uint64(actualKey[1:9])
						if num > maxBlock && num < 2000000 { // Sanity check
							maxBlock = num
						}
					}
				}
			}
		}
		
		// Also check 65-byte keys (might be 32 + 33)
		if keyLen == 65 {
			if bytes.HasPrefix(key, namespace) {
				actualKey := key[32:]
				// 65 - 32 = 33 bytes
				if actualKey[0] == 'H' && len(actualKey) == 33 {
					// This might be H<hash> -> num mapping
					value, err := iter.ValueAndErr()
					if err == nil && len(value) == 8 {
						num := binary.BigEndian.Uint64(value)
						if num > maxBlock && num < 2000000 {
							maxBlock = num
							fmt.Printf("Found H<hash> mapping for block %d\n", num)
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("\n=== Analysis Results ===\n")
	fmt.Printf("Found %d keys of 75 bytes\n", count75)
	fmt.Printf("Max block number found: %d\n", maxBlock)
	
	fmt.Printf("\nSample 75-byte keys:\n")
	for i, key := range samples75 {
		fmt.Printf("\nSample %d:\n", i+1)
		fmt.Printf("  Full key: %x\n", key)
		
		if bytes.HasPrefix(key, namespace) {
			actualKey := key[32:]
			fmt.Printf("  After namespace strip (43 bytes): %x\n", actualKey)
			
			// Try to decode
			if len(actualKey) >= 1 {
				fmt.Printf("  First byte: '%c' (0x%02x)\n", actualKey[0], actualKey[0])
			}
			if len(actualKey) >= 9 {
				num := binary.BigEndian.Uint64(actualKey[1:9])
				fmt.Printf("  Bytes 1-9 as uint64: %d\n", num)
			}
			if len(actualKey) == 43 {
				// Could be h<num><hash>n (1 + 8 + 32 + 1 + 1 = 43)
				// Or H<num><hash> (1 + 8 + 32 + 2 extra = 43)
				fmt.Printf("  Possible structure: prefix(1) + num(8) + hash(32) + suffix(2)\n")
			}
		} else {
			fmt.Printf("  Does NOT start with expected namespace\n")
			fmt.Printf("  First 32 bytes: %x\n", key[:32])
		}
	}
	
	// Now let's specifically look for the target block 1082780
	fmt.Printf("\n\nSearching for block 1082780 (0x%x)...\n", 1082780)
	targetNum := make([]byte, 8)
	binary.BigEndian.PutUint64(targetNum, 1082780)
	
	iter2, _ := source.NewIter(nil)
	defer iter2.Close()
	
	found := false
	for iter2.First(); iter2.Valid() && !found; iter2.Next() {
		key := iter2.Key()
		// Look for targetNum anywhere in the key
		if bytes.Contains(key, targetNum) {
			fmt.Printf("Found key containing block 1082780:\n")
			fmt.Printf("  Key length: %d\n", len(key))
			fmt.Printf("  Full key: %x\n", key)
			if bytes.HasPrefix(key, namespace) {
				fmt.Printf("  After namespace: %x\n", key[32:])
			}
			found = true
		}
	}
	
	if !found {
		fmt.Printf("Block 1082780 not found in any key\n")
	}
}