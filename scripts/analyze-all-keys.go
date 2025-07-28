package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := "/home/z/work/lux/genesis/chaindata/lux-mainnet-96369/db/pebbledb"
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Track key patterns and statistics
	keyPatterns := make(map[string]int)
	prefixCounts := make(map[string]int)
	blockNumbers := make(map[uint64]bool)
	totalKeys := 0
	
	// Known rawdb prefixes
	knownPrefixes := map[byte]string{
		0x48: "HashPrefix",        // H - block hash -> number
		0x62: "BodyPrefix",        // b - block body
		0x68: "HeaderPrefix",      // h - block header
		0x6e: "NumberPrefix",      // n - block number -> hash
		0x72: "ReceiptPrefix",     // r - receipts
		0x6c: "TxLookupPrefix",    // l - tx lookup
		0x54: "TDPrefix",          // T - total difficulty
		0x53: "SecureTriePrefix",  // S - secure trie
		0x74: "CodePrefix",        // t - contract code
		0x00: "PreimagePrefix",    // preimages
		0x42: "BloomBitsPrefix",   // B - bloom bits
	}

	// Analyze all keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()

	fmt.Println("=== Comprehensive Database Analysis ===")
	fmt.Println("Database:", dbPath)
	fmt.Println()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		totalKeys++

		// Track prefix patterns
		if len(key) > 0 {
			prefixCounts[fmt.Sprintf("%02x", key[0])]++
		}
		if len(key) > 1 {
			prefixCounts[fmt.Sprintf("%02x%02x", key[0], key[1])]++
		}
		if len(key) > 2 {
			prefixCounts[fmt.Sprintf("%02x%02x%02x", key[0], key[1], key[2])]++
		}
		if len(key) > 3 {
			prefixCounts[fmt.Sprintf("%02x%02x%02x%02x", key[0], key[1], key[2], key[3])]++
		}

		// Categorize key
		keyType := categorizeKey(key, knownPrefixes)
		keyPatterns[keyType]++

		// Try to extract block numbers
		blockNum := extractBlockNumber(key)
		if blockNum != nil {
			blockNumbers[*blockNum] = true
		}

		// Print sample keys for each pattern (first 5)
		if keyPatterns[keyType] <= 5 {
			value := iter.Value()
			fmt.Printf("Sample %s: key=%x (len=%d), value_len=%d\n", 
				keyType, key, len(key), len(value))
			
			// Try to decode value for certain types
			if strings.Contains(keyType, "Header") && len(value) > 0 {
				decodeHeader(value)
			} else if strings.Contains(keyType, "Hash->Number") && len(value) == 8 {
				num := binary.BigEndian.Uint64(value)
				fmt.Printf("  -> Block number: %d\n", num)
			}
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatal("Iterator error:", err)
	}

	// Print statistics
	fmt.Println("\n=== Key Pattern Statistics ===")
	var patterns []string
	for pattern := range keyPatterns {
		patterns = append(patterns, pattern)
	}
	sort.Strings(patterns)
	for _, pattern := range patterns {
		fmt.Printf("%-40s: %d keys\n", pattern, keyPatterns[pattern])
	}

	fmt.Printf("\nTotal keys in database: %d\n", totalKeys)

	// Print prefix distribution
	fmt.Println("\n=== Prefix Distribution (top 20) ===")
	type prefixCount struct {
		prefix string
		count  int
	}
	var prefixes []prefixCount
	for prefix, count := range prefixCounts {
		prefixes = append(prefixes, prefixCount{prefix, count})
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return prefixes[i].count > prefixes[j].count
	})
	for i := 0; i < 20 && i < len(prefixes); i++ {
		fmt.Printf("%s: %d keys\n", prefixes[i].prefix, prefixes[i].count)
	}

	// Print block number analysis
	fmt.Println("\n=== Block Number Analysis ===")
	if len(blockNumbers) > 0 {
		var nums []uint64
		for num := range blockNumbers {
			nums = append(nums, num)
		}
		sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
		
		fmt.Printf("Found blocks: %d\n", len(nums))
		fmt.Printf("Block range: %d - %d\n", nums[0], nums[len(nums)-1])
		
		// Check for gaps
		gaps := []string{}
		for i := 1; i < len(nums); i++ {
			if nums[i] != nums[i-1]+1 {
				gaps = append(gaps, fmt.Sprintf("%d-%d", nums[i-1]+1, nums[i]-1))
			}
		}
		if len(gaps) > 0 {
			fmt.Printf("Gaps in blocks: %v\n", gaps)
		}
		
		// Show sample block numbers
		fmt.Print("Sample blocks: ")
		for i := 0; i < 10 && i < len(nums); i++ {
			fmt.Printf("%d ", nums[i])
		}
		if len(nums) > 10 {
			fmt.Printf("... %d %d %d", nums[len(nums)-3], nums[len(nums)-2], nums[len(nums)-1])
		}
		fmt.Println()
	}

	// Look for subnet-specific patterns
	fmt.Println("\n=== Subnet-Specific Pattern Analysis ===")
	analyzeSubnetPatterns(db)
}

func categorizeKey(key []byte, knownPrefixes map[byte]string) string {
	if len(key) == 0 {
		return "Empty"
	}

	// Check for ASCII prefixes (like "evm/")
	if len(key) > 4 && isASCII(key[:4]) {
		prefix := string(key[:4])
		if strings.HasPrefix(prefix, "evm/") {
			return fmt.Sprintf("SubnetEVM_%s", prefix)
		}
		return fmt.Sprintf("ASCII_%s", prefix)
	}

	// Check for 32-byte prefix pattern (subnet prefix)
	if len(key) > 32 {
		// Check if it's all zeros or a specific pattern
		allZeros := true
		for i := 0; i < 32; i++ {
			if key[i] != 0 {
				allZeros = false
				break
			}
		}
		
		if allZeros && len(key) > 32 {
			// This looks like subnet-prefixed data
			if name, ok := knownPrefixes[key[32]]; ok {
				return fmt.Sprintf("Subnet_32zeros_%s", name)
			}
			return fmt.Sprintf("Subnet_32zeros_%02x", key[32])
		} else {
			// Non-zero 32-byte prefix
			return fmt.Sprintf("32BytePrefix_%x...", key[:4])
		}
	}

	// Standard rawdb prefix
	if name, ok := knownPrefixes[key[0]]; ok {
		return name
	}

	// Unknown pattern
	if len(key) < 10 {
		return fmt.Sprintf("Unknown_%x", key)
	}
	return fmt.Sprintf("Unknown_%x...", key[:10])
}

func isASCII(data []byte) bool {
	for _, b := range data {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

func extractBlockNumber(key []byte) *uint64 {
	// Try various patterns to extract block number
	
	// Pattern 1: NumberPrefix (0x6e) + hash -> block number in value
	if len(key) > 0 && key[0] == 0x6e {
		return nil // Number is in value, not key
	}
	
	// Pattern 2: 32-byte prefix + NumberPrefix
	if len(key) > 32 && key[32] == 0x6e {
		return nil // Number is in value
	}
	
	// Pattern 3: HeaderPrefix/BodyPrefix with embedded number
	if len(key) >= 41 { // 32 prefix + 1 type + 8 number
		if key[32] == 0x68 || key[32] == 0x62 { // header or body
			num := binary.BigEndian.Uint64(key[33:41])
			return &num
		}
	}
	
	// Pattern 4: Direct header/body with number
	if len(key) >= 9 && (key[0] == 0x68 || key[0] == 0x62) {
		num := binary.BigEndian.Uint64(key[1:9])
		return &num
	}
	
	return nil
}

func decodeHeader(data []byte) {
	// Try to decode header manually
	if len(data) > 100 {
		fmt.Printf("  -> Header data preview: %x... (len=%d)\n", data[:32], len(data))
	}
}

func analyzeSubnetPatterns(db *pebble.DB) {
	// Look for specific subnet patterns
	patterns := []struct {
		name   string
		prefix []byte
	}{
		{"ASCII evm/", []byte("evm/")},
		{"32 zeros", bytes.Repeat([]byte{0}, 32)},
		{"Specific subnet prefix", []byte{0x00, 0x00, 0x00, 0x00}}, // Add known subnet prefixes
	}
	
	for _, pattern := range patterns {
		count := 0
		iter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: pattern.prefix,
			UpperBound: append(pattern.prefix, 0xff),
		})
		if err != nil {
			fmt.Printf("Error creating iterator for %s: %v\n", pattern.name, err)
			continue
		}
		defer iter.Close()
		
		for iter.First(); iter.Valid() && count < 10; iter.Next() {
			if count == 0 {
				fmt.Printf("\n%s pattern examples:\n", pattern.name)
			}
			key := iter.Key()
			fmt.Printf("  key: %x (len=%d)\n", key, len(key))
			count++
		}
		
		if count > 0 {
			// Count total
			total := count
			for ; iter.Valid(); iter.Next() {
				total++
			}
			fmt.Printf("  Total keys with this pattern: %d\n", total)
		}
	}
}