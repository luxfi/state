package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze blockchain database patterns and statistics",
		Long:  `Deep analysis tools for understanding blockchain data structures`,
	}

	cmd.AddCommand(
		newAnalyzeKeysCmd(),
		newAnalyzePrefixesCmd(),
		newAnalyzeNamespaceCmd(),
		newAnalyzeBlocksCmd(),
		newAnalyzeConsensusCmd(),
	)

	return cmd
}

func newAnalyzeKeysCmd() *cobra.Command {
	var limit int
	var detailed bool

	cmd := &cobra.Command{
		Use:   "keys <db-path>",
		Short: "Deep analysis of key patterns and distribution",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyzeKeys(args[0], limit, detailed)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 100000, "Maximum number of keys to analyze")
	cmd.Flags().BoolVar(&detailed, "detailed", false, "Show detailed key breakdown")

	return cmd
}

func newAnalyzePrefixesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "prefixes <db-path>",
		Short: "Analyze all database prefixes and their usage",
		Args:  cobra.ExactArgs(1),
		RunE:  runAnalyzePrefixes,
	}
}

func newAnalyzeNamespaceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "namespace <db-path>",
		Short: "Analyze namespace patterns in keys",
		Args:  cobra.ExactArgs(1),
		RunE:  runAnalyzeNamespace,
	}
}

func newAnalyzeBlocksCmd() *cobra.Command {
	var start, end uint64

	cmd := &cobra.Command{
		Use:   "blocks <db-path>",
		Short: "Analyze block data patterns and integrity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAnalyzeBlocks(args[0], start, end)
		},
	}

	cmd.Flags().Uint64Var(&start, "start", 0, "Start block number")
	cmd.Flags().Uint64Var(&end, "end", 0, "End block number (0 for latest)")

	return cmd
}

func newAnalyzeConsensusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "consensus <db-path>",
		Short: "Analyze consensus-related data structures",
		Args:  cobra.ExactArgs(1),
		RunE:  runAnalyzeConsensus,
	}
}

func runAnalyzeKeys(dbPath string, limit int, detailed bool) error {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Analyzing key patterns...")

	type keyInfo struct {
		prefix      string
		count       int
		sampleKey   []byte
		sampleValue []byte
	}

	prefixMap := make(map[string]*keyInfo)
	totalKeys := 0

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid() && totalKeys < limit; iter.Next() {
		key := iter.Key()
		val := iter.Value()
		totalKeys++

		// Categorize by prefix
		prefix := getKeyPrefix(key)
		if info, exists := prefixMap[prefix]; exists {
			info.count++
		} else {
			prefixMap[prefix] = &keyInfo{
				prefix:      prefix,
				count:       1,
				sampleKey:   append([]byte{}, key...),
				sampleValue: append([]byte{}, val...),
			}
		}
	}

	// Sort prefixes by count
	var prefixes []string
	for p := range prefixMap {
		prefixes = append(prefixes, p)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		return prefixMap[prefixes[i]].count > prefixMap[prefixes[j]].count
	})

	fmt.Printf("\nAnalyzed %d keys\n", totalKeys)
	fmt.Println("\nKey prefix distribution:")
	fmt.Println("========================")

	for _, prefix := range prefixes {
		info := prefixMap[prefix]
		percentage := float64(info.count) * 100.0 / float64(totalKeys)
		fmt.Printf("%-20s: %8d keys (%5.2f%%)", prefix, info.count, percentage)
		
		if detailed {
			fmt.Printf("\n  Sample: %x", info.sampleKey)
			if len(info.sampleValue) > 0 {
				fmt.Printf("\n  Value: %d bytes", len(info.sampleValue))
			}
		}
		fmt.Println()
	}

	return nil
}

func runAnalyzePrefixes(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Scanning all database prefixes...")

	// Known prefix patterns
	knownPrefixes := map[string]string{
		"0x41": "accountSnapshot",
		"0x42": "storageSnapshot", 
		"0x43": "code",
		"0x48": "header",
		"0x62": "blockBody",
		"0x63": "cliqueSnapshot",
		"0x65": "epochAccumulator",
		"0x66": "blockFullTxLookup",
		"0x68": "canonicalHash",
		"0x48n": "headerNumber",
		"0x68n": "canonicalHash+suffix",
		"0x72": "receipts",
		"0x74": "txLookup",
		"0x75": "bloomBits",
		"evm": "EVM namespace",
	}

	prefixCounts := make(map[string]int)
	totalKeys := 0

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		totalKeys++

		prefix := getDetailedPrefix(key)
		prefixCounts[prefix]++
	}

	fmt.Printf("\nTotal keys: %d\n", totalKeys)
	fmt.Println("\nPrefix analysis:")
	fmt.Println("================")

	// Sort by count
	var sortedPrefixes []string
	for p := range prefixCounts {
		sortedPrefixes = append(sortedPrefixes, p)
	}
	sort.Slice(sortedPrefixes, func(i, j int) bool {
		return prefixCounts[sortedPrefixes[i]] > prefixCounts[sortedPrefixes[j]]
	})

	for _, prefix := range sortedPrefixes {
		count := prefixCounts[prefix]
		description := ""
		if desc, ok := knownPrefixes[prefix]; ok {
			description = fmt.Sprintf(" (%s)", desc)
		}
		fmt.Printf("%-15s: %8d%s\n", prefix, count, description)
	}

	return nil
}

func runAnalyzeNamespace(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Analyzing namespace patterns...")

	type namespaceInfo struct {
		count       int
		prefixes    map[string]int
		sampleKeys  []string
	}

	namespaces := make(map[string]*namespaceInfo)

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Extract namespace (everything before first ':')
		namespace := "raw"
		keyStr := string(key)
		if idx := strings.Index(keyStr, ":"); idx > 0 {
			namespace = keyStr[:idx]
		}

		if info, exists := namespaces[namespace]; exists {
			info.count++
			prefix := getKeyPrefix(key)
			info.prefixes[prefix]++
			if len(info.sampleKeys) < 3 {
				info.sampleKeys = append(info.sampleKeys, hex.EncodeToString(key))
			}
		} else {
			namespaces[namespace] = &namespaceInfo{
				count:      1,
				prefixes:   make(map[string]int),
				sampleKeys: []string{hex.EncodeToString(key)},
			}
			namespaces[namespace].prefixes[getKeyPrefix(key)] = 1
		}
	}

	fmt.Println("\nNamespace distribution:")
	fmt.Println("======================")

	for ns, info := range namespaces {
		fmt.Printf("\n%s: %d keys\n", ns, info.count)
		fmt.Println("  Prefixes:")
		for prefix, count := range info.prefixes {
			fmt.Printf("    %-10s: %d\n", prefix, count)
		}
		fmt.Println("  Samples:")
		for _, sample := range info.sampleKeys {
			fmt.Printf("    %s\n", sample)
		}
	}

	return nil
}

func runAnalyzeBlocks(dbPath string, start, end uint64) error {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Printf("Analyzing blocks from %d to %d...\n", start, end)

	// Find the highest block if end is not specified
	if end == 0 {
		end = findHighestBlock(db)
		fmt.Printf("Found highest block: %d\n", end)
	}

	missingHeaders := []uint64{}
	missingBodies := []uint64{}
	missingCanonical := []uint64{}

	for blockNum := start; blockNum <= end && blockNum <= start+1000; blockNum++ {
		// Check header
		if !hasBlockHeader(db, blockNum) {
			missingHeaders = append(missingHeaders, blockNum)
		}

		// Check body
		if !hasBlockBody(db, blockNum) {
			missingBodies = append(missingBodies, blockNum)
		}

		// Check canonical mapping
		if !hasCanonicalMapping(db, blockNum) {
			missingCanonical = append(missingCanonical, blockNum)
		}
	}

	fmt.Println("\nBlock integrity report:")
	fmt.Println("======================")
	fmt.Printf("Missing headers: %d\n", len(missingHeaders))
	if len(missingHeaders) > 0 && len(missingHeaders) < 20 {
		fmt.Printf("  Blocks: %v\n", missingHeaders)
	}

	fmt.Printf("Missing bodies: %d\n", len(missingBodies))
	if len(missingBodies) > 0 && len(missingBodies) < 20 {
		fmt.Printf("  Blocks: %v\n", missingBodies)
	}

	fmt.Printf("Missing canonical: %d\n", len(missingCanonical))
	if len(missingCanonical) > 0 && len(missingCanonical) < 20 {
		fmt.Printf("  Blocks: %v\n", missingCanonical)
	}

	return nil
}

func runAnalyzeConsensus(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Analyzing consensus data structures...")

	// Check all consensus-related keys
	consensusKeys := []string{
		"Height",
		"LastAccepted",
		"lastAccepted",
		"consensus/accepted",
		"consensus/lastAccepted",
		"LastBlock",
		"LastHeader",
		"head:header",
		"head:block",
		"head:receipts",
		"head:td",
	}

	fmt.Println("\nConsensus key analysis:")
	fmt.Println("======================")

	for _, key := range consensusKeys {
		if val, closer, err := db.Get([]byte(key)); err == nil {
			fmt.Printf("%s: ", key)
			if len(val) == 8 {
				height := binary.BigEndian.Uint64(val)
				fmt.Printf("%d (0x%x)\n", height, height)
			} else if len(val) == 32 {
				fmt.Printf("0x%s\n", hex.EncodeToString(val))
			} else {
				fmt.Printf("%d bytes\n", len(val))
			}
			closer.Close()
		} else {
			fmt.Printf("%s: not found\n", key)
		}
	}

	// Check for namespaced consensus keys
	fmt.Println("\nChecking namespaced consensus keys:")
	prefixes := []string{"evm:", "subnet-evm:", "blockchain:"}
	for _, prefix := range prefixes {
		for _, key := range consensusKeys {
			namespacedKey := prefix + key
			if val, closer, err := db.Get([]byte(namespacedKey)); err == nil {
				fmt.Printf("%s: %d bytes\n", namespacedKey, len(val))
				closer.Close()
			}
		}
	}

	return nil
}

// Helper functions
func getKeyPrefix(key []byte) string {
	if len(key) == 0 {
		return "empty"
	}

	// Check for string prefixes
	if len(key) >= 3 {
		if key[0] >= 'a' && key[0] <= 'z' {
			// Look for word boundary
			for i := 1; i < len(key) && i < 10; i++ {
				if key[i] == ':' || key[i] == '/' {
					return string(key[:i])
				}
			}
			if len(key) >= 4 && key[0] == 'e' && key[1] == 'v' && key[2] == 'm' {
				return string(key[:4])
			}
		}
	}

	// Single byte prefix
	return fmt.Sprintf("0x%02x", key[0])
}

func getDetailedPrefix(key []byte) string {
	if len(key) == 0 {
		return "empty"
	}

	// Check for EVM prefixes
	if len(key) >= 4 && string(key[:3]) == "evm" {
		return string(key[:4])
	}

	// Check for string prefixes
	keyStr := string(key)
	if idx := strings.IndexAny(keyStr, ":/"); idx > 0 && idx < 20 {
		return keyStr[:idx]
	}

	// Check for known patterns
	if len(key) >= 2 {
		prefix := fmt.Sprintf("0x%02x", key[0])
		if key[0] == 0x48 && len(key) >= 10 && key[9] == 0x6e {
			return "0x48n"
		}
		if key[0] == 0x68 && len(key) == 10 && key[9] == 0x6e {
			return "0x68n"
		}
		return prefix
	}

	return fmt.Sprintf("0x%02x", key[0])
}

func findHighestBlock(db *pebble.DB) uint64 {
	maxBlock := uint64(0)

	// Check evmh prefix
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer iter.Close()

	for iter.Last(); iter.Valid(); iter.Prev() {
		key := iter.Key()
		if len(key) >= 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
			break
		}
	}

	// Check 0x48 prefix (headers)
	iter2, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x48},
		UpperBound: []byte{0x49},
	})
	defer iter2.Close()

	for iter2.Last(); iter2.Valid(); iter2.Prev() {
		key := iter2.Key()
		if len(key) >= 9 {
			blockNum := binary.BigEndian.Uint64(key[1:9])
			if blockNum > maxBlock && blockNum < 100000000 {
				maxBlock = blockNum
			}
			break
		}
	}

	return maxBlock
}

func hasBlockHeader(db *pebble.DB, blockNum uint64) bool {
	// Check evmh key
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)
	evmhKey := append([]byte("evmh"), blockBytes...)
	
	if _, closer, err := db.Get(evmhKey); err == nil {
		closer.Close()
		return true
	}

	// Check 0x48 key
	headerKey := append([]byte{0x48}, blockBytes...)
	if _, closer, err := db.Get(headerKey); err == nil {
		closer.Close()
		return true
	}

	return false
}

func hasBlockBody(db *pebble.DB, blockNum uint64) bool {
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)
	evmbKey := append([]byte("evmb"), blockBytes...)
	
	if _, closer, err := db.Get(evmbKey); err == nil {
		closer.Close()
		return true
	}

	return false
}

func hasCanonicalMapping(db *pebble.DB, blockNum uint64) bool {
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, blockNum)
	
	// Check evmn key
	evmnKey := append([]byte("evmn"), blockBytes...)
	if _, closer, err := db.Get(evmnKey); err == nil {
		closer.Close()
		return true
	}

	// Check 9-byte canonical
	canonicalKey := append([]byte{0x68}, blockBytes...)
	if _, closer, err := db.Get(canonicalKey); err == nil {
		closer.Close()
		return true
	}

	return false
}