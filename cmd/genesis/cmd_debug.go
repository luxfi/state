package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newDebugCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Debug tools for blockchain database issues",
		Long:  `Low-level debugging tools for analyzing database problems`,
	}

	cmd.AddCommand(
		newDebugKeysCmd(),
		newDebugStateCmd(),
		newDebugBlockCmd(),
		newDebugPrefixCmd(),
	)

	return cmd
}

func newDebugKeysCmd() *cobra.Command {
	var prefix string
	var limit int
	var raw bool

	cmd := &cobra.Command{
		Use:   "keys <db-path>",
		Short: "Debug specific keys or key ranges",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugKeys(args[0], prefix, limit, raw)
		},
	}

	cmd.Flags().StringVar(&prefix, "prefix", "", "Key prefix to filter (hex or string)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum keys to show")
	cmd.Flags().BoolVar(&raw, "raw", false, "Show raw hex output")

	return cmd
}

func newDebugStateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "state <db-path>",
		Short: "Debug database state and consistency",
		Args:  cobra.ExactArgs(1),
		RunE:  runDebugState,
	}
}

func newDebugBlockCmd() *cobra.Command {
	var blockNum uint64

	cmd := &cobra.Command{
		Use:   "block <db-path>",
		Short: "Debug all data for a specific block",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDebugBlock(args[0], blockNum)
		},
	}

	cmd.Flags().Uint64Var(&blockNum, "number", 0, "Block number to debug")
	cmd.MarkFlagRequired("number")

	return cmd
}

func newDebugPrefixCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prefix <db-path> <prefix>",
		Short: "Debug all keys with a specific prefix",
		Args:  cobra.ExactArgs(2),
		RunE:  runDebugPrefix,
	}
}

func runDebugKeys(dbPath, prefix string, limit int, raw bool) error {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Debugging keys in %s\n", dbPath)
	if prefix != "" {
		fmt.Printf("Filter prefix: %s\n", prefix)
	}
	fmt.Println(strings.Repeat("=", 80))

	var prefixBytes []byte
	if prefix != "" {
		if strings.HasPrefix(prefix, "0x") {
			// Hex prefix
			prefixBytes, err = hex.DecodeString(prefix[2:])
			if err != nil {
				return fmt.Errorf("invalid hex prefix: %w", err)
			}
		} else {
			// String prefix
			prefixBytes = []byte(prefix)
		}
	}

	iterOpts := &pebble.IterOptions{}
	if len(prefixBytes) > 0 {
		iterOpts.LowerBound = prefixBytes
		iterOpts.UpperBound = incrementBytes(prefixBytes)
	}

	iter, _ := db.NewIter(iterOpts)
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < limit; iter.Next() {
		key := iter.Key()
		val := iter.Value()

		if raw {
			fmt.Printf("Key: %x\n", key)
			fmt.Printf("Val: %x\n", val)
		} else {
			fmt.Printf("Key: %s\n", formatKey(key))
			fmt.Printf("Val: %s\n", formatValue(key, val))
		}
		fmt.Println()

		count++
	}

	fmt.Printf("Showed %d keys\n", count)
	return nil
}

func runDebugState(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Database State Debug Report")
	fmt.Println(strings.Repeat("=", 80))

	// Check consensus state
	fmt.Println("\nConsensus State:")
	fmt.Println("----------------")
	checkKey(db, "Height", formatHeight)
	checkKey(db, "LastAccepted", formatHash)
	checkKey(db, "lastAccepted", formatHash)
	checkKey(db, "consensus/accepted", formatHash)

	// Check head pointers
	fmt.Println("\nHead Pointers:")
	fmt.Println("--------------")
	checkKey(db, "head:header", formatHash)
	checkKey(db, "head:block", formatHash)
	checkKey(db, "head:receipts", formatHash)
	checkKey(db, "head:td", formatBigInt)

	// Find chain boundaries
	fmt.Println("\nChain Boundaries:")
	fmt.Println("-----------------")
	
	// Find lowest block
	lowestBlock := findLowestBlock(db)
	fmt.Printf("Lowest block: %d\n", lowestBlock)
	
	// Find highest block
	highestBlock := findHighestBlockDebug(db)
	fmt.Printf("Highest block: %d\n", highestBlock)

	// Check for gaps
	if highestBlock > 0 {
		fmt.Println("\nChecking for gaps (first 100 blocks)...")
		gaps := []uint64{}
		for i := lowestBlock; i <= lowestBlock+100 && i <= highestBlock; i++ {
			if !hasBlock(db, i) {
				gaps = append(gaps, i)
			}
		}
		if len(gaps) > 0 {
			fmt.Printf("Found %d gaps: %v\n", len(gaps), gaps)
		} else {
			fmt.Println("No gaps found in first 100 blocks")
		}
	}

	// Database statistics
	fmt.Println("\nDatabase Statistics:")
	fmt.Println("-------------------")
	
	totalKeys := 0
	totalSize := uint64(0)
	prefixStats := make(map[string]int)

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		totalKeys++
		totalSize += uint64(len(iter.Key()) + len(iter.Value()))
		
		prefix := getKeyPrefix(iter.Key())
		prefixStats[prefix]++
	}

	fmt.Printf("Total keys: %d\n", totalKeys)
	fmt.Printf("Total size: %.2f MB\n", float64(totalSize)/(1024*1024))
	fmt.Println("\nTop prefixes:")
	for prefix, count := range prefixStats {
		if count > 100 {
			fmt.Printf("  %s: %d keys\n", prefix, count)
		}
	}

	return nil
}

func runDebugBlock(dbPath string, blockNum uint64) error {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Debugging block %d\n", blockNum)
	fmt.Println(strings.Repeat("=", 80))

	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)

	// Check all possible keys for this block
	keys := []struct {
		name   string
		prefix []byte
		suffix []byte
	}{
		{"EVM Header", []byte("evmh"), nil},
		{"EVM Body", []byte("evmb"), nil},
		{"EVM Receipts", []byte("evmr"), nil},
		{"EVM TD", []byte("evmt"), nil},
		{"EVM Canonical", []byte("evmn"), nil},
		{"Header (0x48)", []byte{0x48}, nil},
		{"Header (0x48n)", []byte{0x48}, []byte{0x6e}},
		{"Body (0x62)", []byte{0x62}, nil},
		{"Receipts (0x72)", []byte{0x72}, nil},
		{"TD (0x74)", []byte{0x74}, []byte{0x64}},
		{"Canonical (0x68)", []byte{0x68}, nil},
		{"Canonical (0x68n)", []byte{0x68}, []byte{0x6e}},
	}

	found := false
	for _, k := range keys {
		key := append(k.prefix, blockBytes...)
		if k.suffix != nil {
			key = append(key, k.suffix...)
		}

		if val, closer, err := db.Get(key); err == nil {
			found = true
			fmt.Printf("\n%s:\n", k.name)
			fmt.Printf("  Key: %x\n", key)
			fmt.Printf("  Value: %d bytes\n", len(val))
			if len(val) == 32 {
				fmt.Printf("  Hash: 0x%s\n", hex.EncodeToString(val))
			} else if len(val) < 100 {
				fmt.Printf("  Hex: %x\n", val)
			}
			closer.Close()
		}
	}

	if !found {
		fmt.Println("\nNo data found for this block!")
		
		// Try to find nearby blocks
		fmt.Println("\nSearching for nearby blocks...")
		for delta := uint64(1); delta <= 10; delta++ {
			if blockNum > delta && hasBlock(db, blockNum-delta) {
				fmt.Printf("  Found block %d\n", blockNum-delta)
				break
			}
			if hasBlock(db, blockNum+delta) {
				fmt.Printf("  Found block %d\n", blockNum+delta)
				break
			}
		}
	}

	return nil
}

func runDebugPrefix(cmd *cobra.Command, args []string) error {
	dbPath, prefixStr := args[0], args[1]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	var prefix []byte
	if strings.HasPrefix(prefixStr, "0x") {
		prefix, err = hex.DecodeString(prefixStr[2:])
		if err != nil {
			return fmt.Errorf("invalid hex prefix: %w", err)
		}
	} else {
		prefix = []byte(prefixStr)
	}

	fmt.Printf("Debugging prefix: %x (%s)\n", prefix, prefixStr)
	fmt.Println(strings.Repeat("=", 80))

	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: incrementBytes(prefix),
	})
	defer iter.Close()

	count := 0
	sizeTotal := uint64(0)
	samples := []string{}

	for iter.First(); iter.Valid(); iter.Next() {
		count++
		key := iter.Key()
		val := iter.Value()
		sizeTotal += uint64(len(key) + len(val))

		if count <= 5 {
			samples = append(samples, fmt.Sprintf("  %x => %d bytes", key, len(val)))
		}
	}

	fmt.Printf("Found %d keys with this prefix\n", count)
	fmt.Printf("Total size: %.2f KB\n", float64(sizeTotal)/1024)
	
	if len(samples) > 0 {
		fmt.Println("\nSample keys:")
		for _, s := range samples {
			fmt.Println(s)
		}
	}

	// Analyze structure
	if count > 0 {
		fmt.Println("\nStructure analysis:")
		analyzeKeyStructure(db, prefix)
	}

	return nil
}

// Helper functions
func formatKey(key []byte) string {
	// Try to identify key type
	if len(key) > 0 {
		switch key[0] {
		case 0x48:
			if len(key) == 10 && key[9] == 0x6e {
				return fmt.Sprintf("headerNumber: block=%d", binary.BigEndian.Uint64(key[1:9]))
			} else if len(key) == 9 {
				return fmt.Sprintf("header: block=%d", binary.BigEndian.Uint64(key[1:9]))
			}
		case 0x68:
			if len(key) == 10 && key[9] == 0x6e {
				return fmt.Sprintf("canonicalHash(10-byte): block=%d", binary.BigEndian.Uint64(key[1:9]))
			} else if len(key) == 9 {
				return fmt.Sprintf("canonicalHash(9-byte): block=%d", binary.BigEndian.Uint64(key[1:9]))
			}
		}
	}

	// Check string prefixes
	keyStr := string(key)
	if strings.HasPrefix(keyStr, "evm") && len(key) >= 12 {
		blockNum := binary.BigEndian.Uint64(key[4:12])
		return fmt.Sprintf("%s: block=%d", key[:4], blockNum)
	}

	// Default hex
	return fmt.Sprintf("%x", key)
}

func formatValue(key, val []byte) string {
	// Hash values (32 bytes)
	if len(val) == 32 {
		return fmt.Sprintf("hash: 0x%s", hex.EncodeToString(val))
	}

	// Height values (8 bytes)
	if len(val) == 8 {
		height := binary.BigEndian.Uint64(val)
		return fmt.Sprintf("uint64: %d (0x%x)", height, height)
	}

	// Large values
	if len(val) > 100 {
		return fmt.Sprintf("%d bytes (large value)", len(val))
	}

	// Small values - show hex
	return fmt.Sprintf("%x (%d bytes)", val, len(val))
}

func checkKey(db *pebble.DB, key string, formatter func([]byte) string) {
	if val, closer, err := db.Get([]byte(key)); err == nil {
		fmt.Printf("%-20s: %s\n", key, formatter(val))
		closer.Close()
	} else {
		fmt.Printf("%-20s: not found\n", key)
	}
}

func formatHeight(val []byte) string {
	if len(val) == 8 {
		height := binary.BigEndian.Uint64(val)
		return fmt.Sprintf("%d", height)
	}
	return fmt.Sprintf("invalid (%d bytes)", len(val))
}

func formatHash(val []byte) string {
	if len(val) == 32 {
		return fmt.Sprintf("0x%s", hex.EncodeToString(val))
	}
	return fmt.Sprintf("invalid (%d bytes)", len(val))
}

func formatBigInt(val []byte) string {
	return fmt.Sprintf("%d bytes", len(val))
}

func incrementBytes(b []byte) []byte {
	result := make([]byte, len(b))
	copy(result, b)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 0xff {
			result[i]++
			break
		}
		result[i] = 0
		if i == 0 {
			// Overflow - append a byte
			result = append(result, 1)
		}
	}
	return result
}

func findLowestBlock(db *pebble.DB) uint64 {
	// Check evmh prefix
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer iter.Close()

	if iter.First() && iter.Valid() {
		key := iter.Key()
		if len(key) >= 12 {
			return binary.BigEndian.Uint64(key[4:12])
		}
	}

	return 0
}

func findHighestBlockDebug(db *pebble.DB) uint64 {
	maxBlock := uint64(0)

	// Check various sources
	sources := []struct {
		name string
		find func(*pebble.DB) uint64
	}{
		{"Height key", func(db *pebble.DB) uint64 {
			if val, closer, err := db.Get([]byte("Height")); err == nil {
				defer closer.Close()
				if len(val) == 8 {
					return binary.BigEndian.Uint64(val)
				}
			}
			return 0
		}},
		{"evmh prefix", func(db *pebble.DB) uint64 {
			iter, _ := db.NewIter(&pebble.IterOptions{
				LowerBound: []byte("evmh"),
				UpperBound: []byte("evmi"),
			})
			defer iter.Close()
			
			max := uint64(0)
			for iter.Last(); iter.Valid(); iter.Prev() {
				key := iter.Key()
				if len(key) >= 12 {
					blockNum := binary.BigEndian.Uint64(key[4:12])
					if blockNum > max {
						max = blockNum
					}
					break
				}
			}
			return max
		}},
	}

	for _, src := range sources {
		if block := src.find(db); block > maxBlock {
			maxBlock = block
		}
	}

	return maxBlock
}

func hasBlock(db *pebble.DB, blockNum uint64) bool {
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)

	// Check evmh
	evmhKey := append([]byte("evmh"), blockBytes...)
	if _, closer, err := db.Get(evmhKey); err == nil {
		closer.Close()
		return true
	}

	// Check 0x48
	headerKey := append([]byte{0x48}, blockBytes...)
	if _, closer, err := db.Get(headerKey); err == nil {
		closer.Close()
		return true
	}

	return false
}

func analyzeKeyStructure(db *pebble.DB, prefix []byte) {
	// Analyze the structure of keys with this prefix
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: incrementBytes(prefix),
	})
	defer iter.Close()

	lengths := make(map[int]int)
	patterns := make(map[string]int)

	count := 0
	for iter.First(); iter.Valid() && count < 1000; iter.Next() {
		key := iter.Key()
		count++

		// Track key lengths
		lengths[len(key)]++

		// Track patterns (last few bytes)
		if len(key) > len(prefix)+1 {
			suffix := key[len(prefix):]
			if len(suffix) <= 4 {
				patterns[fmt.Sprintf("suffix:%x", suffix)]++
			}
		}
	}

	fmt.Println("  Key lengths:")
	for length, cnt := range lengths {
		fmt.Printf("    %d bytes: %d keys\n", length, cnt)
	}

	if len(patterns) > 0 && len(patterns) < 20 {
		fmt.Println("  Common patterns:")
		for pattern, cnt := range patterns {
			fmt.Printf("    %s: %d keys\n", pattern, cnt)
		}
	}
}