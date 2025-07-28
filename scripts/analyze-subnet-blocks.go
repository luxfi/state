package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"sort"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := "/home/z/work/lux/genesis/chaindata/lux-mainnet-96369/db/pebbledb"
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Track different types of keys we find
	
	// Scan all keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()

	fmt.Println("=== Analyzing SubnetEVM Block Storage ===")
	fmt.Println()

	// First, let's understand the key structure
	keyCount := 0
	for iter.First(); iter.Valid() && keyCount < 100; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		keyCount++
		
		fmt.Printf("Key %d: %x (len=%d)\n", keyCount, key, len(key))
		fmt.Printf("  Value preview: %x... (len=%d)\n", truncate(value, 32), len(value))
		
		// Try to identify the key type
		keyType := identifyKeyType(key, value)
		fmt.Printf("  Type: %s\n", keyType)
		fmt.Println()
	}

	// Now let's look for block-related patterns
	fmt.Println("\n=== Looking for Block Patterns ===")
	
	// Reset iterator
	iter.Close()
	iter, err = db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}

	// Look for patterns with specific lengths that might be block keys
	blockNumbers := make(map[uint64]int)
	possibleBlockKeys := [][]byte{}
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Look for keys that might contain block numbers
		if nums := extractPossibleBlockNumbers(key); len(nums) > 0 {
			for _, num := range nums {
				if num < 100000 { // Reasonable block number range
					blockNumbers[num]++
					if blockNumbers[num] == 1 {
						possibleBlockKeys = append(possibleBlockKeys, copyBytes(key))
					}
				}
			}
		}
		
		// Also check values for block numbers (e.g., in hash->number mappings)
		if len(value) == 8 {
			num := binary.BigEndian.Uint64(value)
			if num < 100000 {
				blockNumbers[num]++
			}
		}
	}

	// Print block number findings
	fmt.Printf("\nFound %d potential block numbers\n", len(blockNumbers))
	
	// Sort and display block numbers
	var nums []uint64
	for num := range blockNumbers {
		nums = append(nums, num)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	
	if len(nums) > 0 {
		fmt.Printf("Block range: %d - %d\n", nums[0], nums[len(nums)-1])
		fmt.Printf("First 20 blocks: ")
		for i := 0; i < 20 && i < len(nums); i++ {
			fmt.Printf("%d ", nums[i])
		}
		fmt.Println()
		
		fmt.Printf("Last 20 blocks: ")
		start := len(nums) - 20
		if start < 0 {
			start = 0
		}
		for i := start; i < len(nums); i++ {
			fmt.Printf("%d ", nums[i])
		}
		fmt.Println()
	}

	// Analyze key structure for block storage
	fmt.Println("\n=== Analyzing Key Structures ===")
	
	// Look at some specific block keys
	fmt.Println("\nSample block-related keys:")
	for i := 0; i < 10 && i < len(possibleBlockKeys); i++ {
		key := possibleBlockKeys[i]
		fmt.Printf("Key: %x\n", key)
		
		// Try to parse the key structure
		if len(key) >= 32 {
			fmt.Printf("  First 32 bytes: %x\n", key[:32])
			if len(key) > 32 {
				fmt.Printf("  After prefix: %x\n", key[32:])
				if len(key) > 33 {
					fmt.Printf("  Possible type byte: %02x (%c)\n", key[32], key[32])
				}
			}
		}
	}

	// Look for specific patterns
	fmt.Println("\n=== Pattern Analysis ===")
	analyzePatterns(db)
}

func identifyKeyType(key, value []byte) string {
	// Check for 32-byte prefix pattern
	if len(key) >= 33 {
		prefix := key[:32]
		typeChar := key[32]
		
		// Check if prefix is all zeros
		allZeros := true
		for _, b := range prefix {
			if b != 0 {
				allZeros = false
				break
			}
		}
		
		if allZeros {
			switch typeChar {
			case 0x68: // 'h'
				return "Header (32-zero prefix)"
			case 0x62: // 'b'
				return "Body (32-zero prefix)"
			case 0x6e: // 'n'
				return "Number->Hash (32-zero prefix)"
			case 0x48: // 'H'
				return "Hash->Number (32-zero prefix)"
			case 0x72: // 'r'
				return "Receipt (32-zero prefix)"
			case 0x6c: // 'l'
				return "TxLookup (32-zero prefix)"
			default:
				return fmt.Sprintf("Unknown 32-zero prefix type: %02x", typeChar)
			}
		} else {
			// Non-zero prefix
			return fmt.Sprintf("32-byte prefix: %x...", prefix[:8])
		}
	}
	
	// Check for direct rawdb prefixes
	if len(key) > 0 {
		switch key[0] {
		case 0x68:
			return "Header (direct)"
		case 0x62:
			return "Body (direct)"
		case 0x6e:
			return "Number->Hash (direct)"
		case 0x48:
			return "Hash->Number (direct)"
		default:
			return fmt.Sprintf("Unknown type: %02x", key[0])
		}
	}
	
	return "Unknown"
}

func extractPossibleBlockNumbers(key []byte) []uint64 {
	numbers := []uint64{}
	
	// Look for 8-byte sequences that could be block numbers
	for i := 0; i <= len(key)-8; i++ {
		num := binary.BigEndian.Uint64(key[i : i+8])
		if num < 100000 { // Reasonable range
			numbers = append(numbers, num)
		}
	}
	
	return numbers
}

func truncate(data []byte, maxLen int) []byte {
	if len(data) <= maxLen {
		return data
	}
	return data[:maxLen]
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

func analyzePatterns(db *pebble.DB) {
	// Look for keys with specific patterns
	patterns := map[string]int{}
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Categorize by length and prefix
		if len(key) == 41 { // 32 + 1 + 8 (prefix + type + number)
			patterns["41-byte keys"]++
		} else if len(key) == 73 { // 32 + 1 + 8 + 32 (prefix + type + number + hash)
			patterns["73-byte keys"]++
		} else if len(key) == 65 { // 32 + 1 + 32 (prefix + type + hash)
			patterns["65-byte keys"]++
		} else if len(key) == 9 { // 1 + 8 (type + number)
			patterns["9-byte keys"]++
		} else if len(key) == 33 { // 1 + 32 (type + hash)
			patterns["33-byte keys"]++
		}
		
		// Check for specific prefixes
		if len(key) >= 32 && bytes.Equal(key[:32], bytes.Repeat([]byte{0}, 32)) {
			patterns["32-zero prefix"]++
		}
		
		if len(key) >= 32 {
			prefix := hex.EncodeToString(key[:4])
			patterns["prefix:"+prefix]++
		}
	}
	
	// Print pattern statistics
	fmt.Println("\nKey pattern distribution:")
	var sortedPatterns []string
	for pattern := range patterns {
		sortedPatterns = append(sortedPatterns, pattern)
	}
	sort.Strings(sortedPatterns)
	
	for _, pattern := range sortedPatterns {
		if patterns[pattern] > 100 { // Only show significant patterns
			fmt.Printf("  %s: %d\n", pattern, patterns[pattern])
		}
	}
	
	// Look specifically for block headers and bodies
	fmt.Println("\n=== Looking for Headers and Bodies ===")
	findHeadersAndBodies(db)
}

func findHeadersAndBodies(db *pebble.DB) {
	// Look for header keys with different patterns
	headerCount := 0
	
	// Pattern 1: 32 zeros + 'h' (0x68) + 8-byte number
	prefix1 := append(bytes.Repeat([]byte{0}, 32), 0x68)
	iter1, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix1,
		UpperBound: append(prefix1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	defer iter1.Close()
	
	fmt.Println("\nHeaders with 32-zero prefix:")
	for iter1.First(); iter1.Valid() && headerCount < 10; iter1.Next() {
		key := iter1.Key()
		if len(key) >= 41 {
			blockNum := binary.BigEndian.Uint64(key[33:41])
			fmt.Printf("  Block %d: key=%x\n", blockNum, key)
			headerCount++
		}
	}
	
	// Count total headers
	for ; iter1.Valid(); iter1.Next() {
		headerCount++
	}
	fmt.Printf("Total headers with 32-zero prefix: %d\n", headerCount)
	
	// Pattern 2: Different prefix patterns
	// Let's check what the actual prefix is
	iter2, _ := db.NewIter(&pebble.IterOptions{})
	defer iter2.Close()
	
	prefixMap := make(map[string]int)
	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		if len(key) >= 41 && key[32] == 0x68 { // header type
			prefix := hex.EncodeToString(key[:32])
			prefixMap[prefix]++
		}
	}
	
	fmt.Println("\nHeader prefixes found:")
	for prefix, count := range prefixMap {
		if count > 0 {
			fmt.Printf("  %s: %d headers\n", prefix, count)
		}
	}
}