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

	// The actual subnet prefix found
	subnetPrefix, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")
	
	fmt.Println("=== Finding All Blocks with Subnet Prefix ===")
	fmt.Printf("Subnet prefix: %x\n\n", subnetPrefix)

	// Look for headers (h = 0x68)
	headerPrefix := append(subnetPrefix, 0x68) // 'h'
	
	// Look for hash->number mappings (H = 0x48)
	hashNumPrefix := append(subnetPrefix, 0x48) // 'H'
	
	// Look for number->hash mappings (n = 0x6e)
	// numHashPrefix := append(subnetPrefix, 0x6e) // 'n' - not used for now
	
	// Track all block numbers
	blockNumbers := make(map[uint64]bool)
	
	// First, scan hash->number mappings to understand block range
	fmt.Println("Scanning hash->number mappings...")
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: hashNumPrefix,
		UpperBound: append(hashNumPrefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
			0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		value := iter.Value()
		if len(value) == 8 {
			blockNum := binary.BigEndian.Uint64(value)
			blockNumbers[blockNum] = true
			count++
			if count <= 10 {
				key := iter.Key()
				hash := key[33:] // Skip prefix and type byte
				fmt.Printf("  Block %d: hash=%x\n", blockNum, hash)
			}
		}
	}
	iter.Close()
	fmt.Printf("Found %d hash->number mappings\n\n", count)
	
	// Now scan headers
	fmt.Println("Scanning block headers...")
	iter, err = db.NewIter(&pebble.IterOptions{
		LowerBound: headerPrefix,
		UpperBound: append(headerPrefix, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
	})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	
	headerCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 41 { // prefix(32) + type(1) + number(8)
			blockNum := binary.BigEndian.Uint64(key[33:41])
			blockNumbers[blockNum] = true
			headerCount++
			if headerCount <= 10 {
				fmt.Printf("  Header for block %d found\n", blockNum)
			}
		}
	}
	iter.Close()
	fmt.Printf("Found %d headers\n\n", headerCount)
	
	// Analyze block numbers
	fmt.Println("=== Block Number Analysis ===")
	var nums []uint64
	for num := range blockNumbers {
		nums = append(nums, num)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	
	if len(nums) > 0 {
		fmt.Printf("Total unique blocks found: %d\n", len(nums))
		fmt.Printf("Block range: %d - %d\n", nums[0], nums[len(nums)-1])
		
		// Check for gaps
		gaps := []string{}
		for i := 1; i < len(nums); i++ {
			if nums[i] != nums[i-1]+1 {
				gaps = append(gaps, fmt.Sprintf("%d-%d", nums[i-1]+1, nums[i]-1))
			}
		}
		
		if len(gaps) > 0 {
			fmt.Printf("Missing blocks (gaps): %d gaps found\n", len(gaps))
			if len(gaps) <= 20 {
				for _, gap := range gaps {
					fmt.Printf("  Gap: %s\n", gap)
				}
			} else {
				fmt.Println("  (Too many gaps to display)")
			}
		} else {
			fmt.Println("No gaps found - all blocks are sequential!")
		}
		
		// Show sample blocks
		fmt.Print("\nFirst 20 blocks: ")
		for i := 0; i < 20 && i < len(nums); i++ {
			fmt.Printf("%d ", nums[i])
		}
		fmt.Println()
		
		fmt.Print("Last 20 blocks: ")
		start := len(nums) - 20
		if start < 0 {
			start = 0
		}
		for i := start; i < len(nums); i++ {
			fmt.Printf("%d ", nums[i])
		}
		fmt.Println()
	}
	
	// Let's also check what other prefixes exist in the database
	fmt.Println("\n=== Checking for other key patterns ===")
	
	// Sample first 1000 keys to find patterns
	iter, err = db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatal("Failed to create iterator:", err)
	}
	defer iter.Close()
	
	prefixMap := make(map[string]int)
	sampleCount := 0
	
	for iter.First(); iter.Valid() && sampleCount < 10000; iter.Next() {
		key := iter.Key()
		if len(key) >= 32 {
			prefix := hex.EncodeToString(key[:32])
			prefixMap[prefix]++
		} else {
			prefixMap[fmt.Sprintf("short_%d_bytes", len(key))]++
		}
		sampleCount++
	}
	
	fmt.Printf("\nDifferent 32-byte prefixes found (sample of %d keys):\n", sampleCount)
	for prefix, count := range prefixMap {
		if count > 10 { // Only show significant patterns
			fmt.Printf("  %s: %d keys\n", prefix, count)
		}
	}
	
	// Try to migrate a sample block
	fmt.Println("\n=== Sample Block Migration ===")
	if len(nums) > 0 {
		// Get block 0 header
		blockNum := uint64(0)
		headerKey := append(headerPrefix, binary.BigEndian.AppendUint64(nil, blockNum)...)
		headerKey = append(headerKey, bytes.Repeat([]byte{0}, 32)...) // Add hash padding
		
		value, closer, err := db.Get(headerKey[:41]) // Just prefix + type + number
		if err == nil {
			fmt.Printf("Block 0 header found! Length: %d bytes\n", len(value))
			closer.Close()
		} else {
			// Try without hash
			iter, _ := db.NewIter(&pebble.IterOptions{
				LowerBound: headerKey[:41],
				UpperBound: append(headerKey[:41], 0xff),
			})
			if iter.First() {
				fmt.Printf("Block 0 header found with key: %x\n", iter.Key())
				fmt.Printf("Header length: %d bytes\n", len(iter.Value()))
			}
			iter.Close()
		}
	}
}