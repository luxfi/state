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

// The subnet prefix for chain 96369
var subnetPrefix = []byte{
	0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c,
	0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e,
	0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a,
	0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1,
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: migrate-subnet-tool <source-db>")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	
	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	fmt.Printf("=== Analyzing Subnet Database: %s ===\n\n", srcPath)
	
	// Phase 1: Look for standard rawdb prefixes
	fmt.Println("Phase 1: Checking for standard rawdb prefixes...")
	checkRawdbPrefixes(srcDB)
	
	// Phase 2: Check for evm/ string prefixes
	fmt.Println("\nPhase 2: Checking for 'evm/' string prefixes...")
	checkEvmPrefixes(srcDB)
	
	// Phase 3: Check for simple height keys
	fmt.Println("\nPhase 3: Checking for simple height keys...")
	checkSimpleHeightKeys(srcDB)
	
	// Phase 4: Analyze general patterns
	fmt.Println("\nPhase 4: Analyzing general key patterns...")
	analyzePatterns(srcDB)
}

func checkRawdbPrefixes(db *pebble.DB) {
	// Standard rawdb prefixes
	prefixes := map[byte]string{
		0x68: "headers (h)",
		0x62: "bodies (b)",
		0x6e: "number->hash (n)",
		0x48: "hash->number (H)",
		0x54: "total difficulty (T)",
		0x72: "receipts (r)",
		0x6c: "tx lookups (l)",
		0x53: "secure trie (S)",
	}
	
	for prefix, name := range prefixes {
		// Check with subnet prefix
		checkKey := append(subnetPrefix, prefix)
		iter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: checkKey,
			UpperBound: append(checkKey, 0xff),
		})
		if err != nil {
			continue
		}
		
		count := 0
		for iter.First(); iter.Valid() && count < 3; iter.Next() {
			count++
			if count == 1 {
				fmt.Printf("  Found %s with subnet prefix!\n", name)
				fmt.Printf("    Example key: %s\n", hex.EncodeToString(iter.Key()[:min(64, len(iter.Key()))]))
				fmt.Printf("    Value size: %d bytes\n", len(iter.Value()))
			}
		}
		iter.Close()
		
		// Also check without subnet prefix
		iter2, err := db.NewIter(&pebble.IterOptions{
			LowerBound: []byte{prefix},
			UpperBound: []byte{prefix + 1},
		})
		if err == nil {
			count2 := 0
			for iter2.First(); iter2.Valid() && count2 < 3; iter2.Next() {
				count2++
				if count2 == 1 {
					fmt.Printf("  Found %s without subnet prefix!\n", name)
					fmt.Printf("    Example key: %s\n", hex.EncodeToString(iter2.Key()[:min(64, len(iter2.Key()))]))
				}
			}
			iter2.Close()
		}
	}
}

func checkEvmPrefixes(db *pebble.DB) {
	patterns := []string{
		"evm/h/",      // headers
		"evm/b/",      // bodies  
		"evm/n/",      // number->hash
		"evm/H/",      // hash->number
		"evm/T/",      // total difficulty
		"evm/r/",      // receipts
		"evm/l/",      // tx lookups
		"evm/secure/", // state trie
		"evm/head_header",
		"evm/head_block",
		"evm/chain_config",
	}
	
	for _, pattern := range patterns {
		// With subnet prefix
		key := append(subnetPrefix, []byte(pattern)...)
		iter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: key,
			UpperBound: append(key, 0xff),
		})
		if err == nil {
			if iter.First() {
				fmt.Printf("  Found %s with subnet prefix!\n", pattern)
				fmt.Printf("    Key: %s\n", hex.EncodeToString(iter.Key()[:min(64, len(iter.Key()))]))
			}
			iter.Close()
		}
		
		// Without subnet prefix
		iter2, err := db.NewIter(&pebble.IterOptions{
			LowerBound: []byte(pattern),
			UpperBound: append([]byte(pattern), 0xff),
		})
		if err == nil {
			if iter2.First() {
				fmt.Printf("  Found %s without subnet prefix!\n", pattern)
			}
			iter2.Close()
		}
	}
}

func checkSimpleHeightKeys(db *pebble.DB) {
	// Check for blocks stored as 8-byte big-endian height
	for height := uint64(0); height < 10; height++ {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, height)
		
		// With subnet prefix
		keyWithPrefix := append(subnetPrefix, key...)
		val, closer, err := db.Get(keyWithPrefix)
		if err == nil {
			fmt.Printf("  Found block at height %d with subnet prefix! Size: %d bytes\n", height, len(val))
			closer.Close()
		}
		
		// Without subnet prefix
		val2, closer2, err2 := db.Get(key)
		if err2 == nil {
			fmt.Printf("  Found block at height %d without prefix! Size: %d bytes\n", height, len(val2))
			closer2.Close()
		}
	}
}

func analyzePatterns(db *pebble.DB) {
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return
	}
	defer iter.Close()
	
	patterns := make(map[string]int)
	count := 0
	
	for iter.First(); iter.Valid() && count < 10000; iter.Next() {
		key := iter.Key()
		count++
		
		// Determine pattern
		pattern := identifyPattern(key)
		patterns[pattern]++
	}
	
	fmt.Printf("\n  Total keys analyzed: %d\n", count)
	fmt.Println("  Top patterns found:")
	
	// Show patterns with significant counts
	for p, c := range patterns {
		if c > 10 {
			fmt.Printf("    %s: %d occurrences\n", p, c)
		}
	}
}

func identifyPattern(key []byte) string {
	// Check if it has subnet prefix
	hasSubnetPrefix := false
	if bytes.HasPrefix(key, subnetPrefix) {
		hasSubnetPrefix = true
		key = key[len(subnetPrefix):]
	}
	
	if len(key) == 0 {
		return "empty"
	}
	
	// Check first byte
	firstByte := key[0]
	pattern := ""
	
	switch {
	case firstByte == 0x68:
		pattern = "rawdb-header(h)"
	case firstByte == 0x62:
		pattern = "rawdb-body(b)"
	case firstByte == 0x6e:
		pattern = "rawdb-num2hash(n)"
	case firstByte == 0x48:
		pattern = "rawdb-hash2num(H)"
	case firstByte == 0x54:
		pattern = "rawdb-td(T)"
	case firstByte == 0x72:
		pattern = "rawdb-receipt(r)"
	case firstByte == 0x6c:
		pattern = "rawdb-txlookup(l)"
	case firstByte == 0x53:
		pattern = "rawdb-trie(S)"
	case firstByte >= 0x00 && firstByte <= 0x0f:
		pattern = fmt.Sprintf("trie-node-%02x", firstByte)
	case firstByte >= 0x20 && firstByte <= 0x7e:
		// Printable ASCII - check for string prefix
		end := 1
		for end < len(key) && end < 10 && key[end] >= 0x20 && key[end] <= 0x7e {
			end++
		}
		pattern = fmt.Sprintf("ascii-%s", string(key[:end]))
	default:
		pattern = fmt.Sprintf("byte-%02x", firstByte)
	}
	
	if hasSubnetPrefix {
		pattern = "subnet+" + pattern
	}
	
	return pattern
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}