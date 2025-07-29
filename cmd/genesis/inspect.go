package main

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
	"github.com/spf13/cobra"
)

// Inspection flags
var (
	inspectLimit     int
	inspectVerbose   bool
	inspectDecodeRLP bool
	inspectKeyHex    string
)

// NewInspectCommand creates the inspect command with all subcommands
func NewInspectCommand() *cobra.Command {
	inspectCmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect database contents",
		Long: `Inspect various aspects of database contents.
		
Available inspections:
- keys: Inspect database keys
- blocks: Inspect block data
- headers: Inspect block headers
- snowman: Inspect Snowman consensus DB
- prefixes: Scan database prefixes
- tip: Find chain tip`,
	}

	// Add subcommands
	inspectCmd.AddCommand(
		newInspectKeysCmd(),
		newInspectBlocksCmd(),
		newInspectHeadersCmd(),
		newInspectSnowmanCmd(),
		newInspectPrefixesCmd(),
		newInspectTipCmd(),
	)

	return inspectCmd
}

// newInspectKeysCmd creates the keys inspection command
func newInspectKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys <database-path>",
		Short: "Inspect database keys",
		Long: `Inspect database keys and their values.
		
This shows:
- Raw key bytes
- Decoded key meaning
- Value preview
- RLP decoding (if applicable)`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectKeys,
	}

	cmd.Flags().IntVar(&inspectLimit, "limit", 100, "Maximum keys to inspect")
	cmd.Flags().BoolVar(&inspectVerbose, "verbose", false, "Show verbose output")
	cmd.Flags().StringVar(&inspectKeyHex, "key", "", "Inspect specific key (hex)")

	return cmd
}

// newInspectBlocksCmd creates the blocks inspection command
func newInspectBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks <database-path>",
		Short: "Inspect block data",
		Long: `Inspect blockchain block data.
		
This shows:
- Block headers
- Transaction lists
- Uncle blocks
- Block metadata`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectBlocks,
	}

	cmd.Flags().IntVar(&inspectLimit, "limit", 10, "Maximum blocks to inspect")
	cmd.Flags().BoolVar(&inspectDecodeRLP, "decode", true, "Decode RLP data")

	return cmd
}

// newInspectHeadersCmd creates the headers inspection command
func newInspectHeadersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headers <database-path>",
		Short: "Inspect block headers",
		Long: `Inspect block header data.
		
This shows:
- Header fields
- Parent hash
- State root
- Transaction/Receipt roots`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectHeaders,
	}

	cmd.Flags().IntVar(&inspectLimit, "limit", 10, "Maximum headers to inspect")

	return cmd
}

// newInspectSnowmanCmd creates the snowman inspection command
func newInspectSnowmanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snowman <database-path>",
		Short: "Inspect Snowman consensus database",
		Long: `Inspect Snowman consensus database contents.
		
This shows:
- Consensus state
- Block acceptance status
- Chain preferences`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectSnowman,
	}

	return cmd
}

// newInspectPrefixesCmd creates the prefixes inspection command
func newInspectPrefixesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prefixes <database-path>",
		Short: "Scan database prefixes",
		Long: `Scan and categorize all database key prefixes.
		
This provides:
- Complete prefix inventory
- Key count per prefix
- Prefix meanings
- Unusual patterns`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectPrefixes,
	}

	return cmd
}

// newInspectTipCmd creates the tip inspection command
func newInspectTipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tip <database-path>",
		Short: "Find chain tip",
		Long: `Find the highest block in the chain.
		
This shows:
- Highest block number
- Chain head hash
- Total difficulty
- Network statistics`,
		Args: cobra.ExactArgs(1),
		RunE: runInspectTip,
	}

	return cmd
}

// Command implementations

func runInspectKeys(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Inspecting Keys in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// If specific key requested
	if inspectKeyHex != "" {
		key, err := hex.DecodeString(inspectKeyHex)
		if err != nil {
			return fmt.Errorf("invalid hex key: %w", err)
		}
		
		value, closer, err := db.Get(key)
		if err != nil {
			return fmt.Errorf("key not found: %w", err)
		}
		defer closer.Close()

		fmt.Printf("\nKey: %x\n", key)
		fmt.Printf("Value: %x\n", value)
		fmt.Printf("Key Type: %s\n", identifyKeyType(key))
		
		return nil
	}

	// Iterate through keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < inspectLimit; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		count++

		fmt.Printf("\n[%d] Key: %x (len=%d)\n", count, key, len(key))
		
		keyType := identifyKeyType(key)
		fmt.Printf("    Type: %s\n", keyType)
		
		if inspectVerbose {
			fmt.Printf("    Value preview: %x... (len=%d)\n", truncateBytes(value, 32), len(value))
			
			// Try to decode if it's a known type
			if decoded := tryDecodeValue(keyType, value); decoded != "" {
				fmt.Printf("    Decoded: %s\n", decoded)
			}
		}
	}

	fmt.Printf("\nInspected %d keys\n", count)
	return nil
}

func runInspectBlocks(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Inspecting Blocks in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Find block bodies
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("b"),
		UpperBound: []byte("c"),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < inspectLimit; iter.Next() {
		key := iter.Key()
		if len(key) == 41 && key[0] == 'b' { // Body key
			value := iter.Value()
			blockNum := decodeBlockNumber(key[1:9])
			hash := key[9:]
			
			fmt.Printf("\nBlock %d (hash: %x):\n", blockNum, hash)
			
			if inspectDecodeRLP {
				// Try to decode as block body
				var body types.Body
				if err := rlp.DecodeBytes(value, &body); err == nil {
					fmt.Printf("  Transactions: %d\n", len(body.Transactions))
					fmt.Printf("  Uncles: %d\n", len(body.Uncles))
				} else {
					fmt.Printf("  Failed to decode body: %v\n", err)
				}
			}
			
			count++
		}
	}

	if count == 0 {
		fmt.Println("No blocks found")
	} else {
		fmt.Printf("\nInspected %d blocks\n", count)
	}

	return nil
}

func runInspectHeaders(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Inspecting Headers in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Find headers
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("h"),
		UpperBound: []byte("i"),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < inspectLimit; iter.Next() {
		key := iter.Key()
		if len(key) == 41 && key[0] == 'h' { // Header key
			value := iter.Value()
			blockNum := decodeBlockNumber(key[1:9])
			hash := key[9:]
			
			fmt.Printf("\nHeader %d (hash: %x):\n", blockNum, hash[:8])
			
			// Try to decode header
			var header types.Header
			if err := rlp.DecodeBytes(value, &header); err == nil {
				fmt.Printf("  Parent: %x\n", header.ParentHash[:8])
				fmt.Printf("  State Root: %x\n", header.Root[:8])
				fmt.Printf("  Tx Root: %x\n", header.TxHash[:8])
				fmt.Printf("  Receipt Root: %x\n", header.ReceiptHash[:8])
				fmt.Printf("  Number: %d\n", header.Number.Uint64())
				fmt.Printf("  Gas Limit: %d\n", header.GasLimit)
				fmt.Printf("  Gas Used: %d\n", header.GasUsed)
				fmt.Printf("  Time: %d\n", header.Time)
			} else {
				fmt.Printf("  Failed to decode header: %v\n", err)
			}
			
			count++
		}
	}

	if count == 0 {
		fmt.Println("No headers found")
	} else {
		fmt.Printf("\nInspected %d headers\n", count)
	}

	return nil
}

func runInspectSnowman(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Inspecting Snowman DB in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Common Snowman prefixes
	prefixes := map[string]string{
		"b": "Block",
		"s": "State",
		"v": "Vertex",
		"c": "Choice",
		"a": "Accepted",
	}

	fmt.Println("\nSnowman Key Categories:")
	for prefix, name := range prefixes {
		count := countKeysWithPrefix(db, []byte(prefix))
		if count > 0 {
			fmt.Printf("  %s (%s): %d\n", name, prefix, count)
		}
	}

	// Show sample keys
	fmt.Println("\nSample Snowman Keys:")
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer iter.Close()

	shown := 0
	for iter.First(); iter.Valid() && shown < 10; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		fmt.Printf("  Key: %x (len=%d)\n", key, len(key))
		fmt.Printf("  Value: %x... (len=%d)\n", truncateBytes(value, 16), len(value))
		shown++
	}

	return nil
}

func runInspectPrefixes(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Scanning Prefixes in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Track all unique prefixes with examples
	prefixCounts := make(map[string]int)
	prefixExamples := make(map[string]string)
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer iter.Close()

	totalKeys := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		totalKeys++
		
		// Get various prefix lengths
		if len(key) >= 1 {
			prefix1 := fmt.Sprintf("1-byte: %02x", key[0])
			prefixCounts[prefix1]++
			if _, exists := prefixExamples[prefix1]; !exists {
				prefixExamples[prefix1] = hex.EncodeToString(key)
			}
			
			// Check for known single-byte prefixes
			knownPrefix := fmt.Sprintf("Known: 0x%02x", key[0])
			switch key[0] {
			case 0x48:
				knownPrefix += " (H)"
			case 0x68:
				knownPrefix += " (h)"
			case 0x62:
				knownPrefix += " (b)"
			case 0x72:
				knownPrefix += " (r)"
			case 0x6e:
				knownPrefix += " (n)"
			default:
				knownPrefix = ""
			}
			if knownPrefix != "" {
				prefixCounts[knownPrefix]++
				if _, exists := prefixExamples[knownPrefix]; !exists {
					prefixExamples[knownPrefix] = hex.EncodeToString(key)
				}
			}
		}
		if len(key) >= 2 {
			prefix2 := fmt.Sprintf("2-byte: %02x%02x", key[0], key[1])
			prefixCounts[prefix2]++
			if _, exists := prefixExamples[prefix2]; !exists {
				prefixExamples[prefix2] = hex.EncodeToString(key)
			}
		}
		if len(key) >= 3 {
			prefix3 := fmt.Sprintf("3-byte: %02x%02x%02x", key[0], key[1], key[2])
			prefixCounts[prefix3]++
			if _, exists := prefixExamples[prefix3]; !exists {
				prefixExamples[prefix3] = hex.EncodeToString(key)
			}
		}
		if len(key) >= 4 {
			prefix4 := fmt.Sprintf("4-byte: %02x%02x%02x%02x", key[0], key[1], key[2], key[3])
			prefixCounts[prefix4]++
			if _, exists := prefixExamples[prefix4]; !exists {
				prefixExamples[prefix4] = hex.EncodeToString(key)
			}
			
			// Check for ASCII prefixes
			if isTextPrefix(key[:4]) {
				asciiPrefix := fmt.Sprintf("ASCII: %s", string(key[:4]))
				prefixCounts[asciiPrefix]++
				if _, exists := prefixExamples[asciiPrefix]; !exists {
					prefixExamples[asciiPrefix] = hex.EncodeToString(key)
				}
			}
		}
	}

	// Sort prefixes by count (descending)
	type prefixInfo struct {
		prefix  string
		count   int
		example string
	}
	
	var prefixes []prefixInfo
	for prefix, count := range prefixCounts {
		prefixes = append(prefixes, prefixInfo{
			prefix:  prefix,
			count:   count,
			example: prefixExamples[prefix],
		})
	}
	
	sort.Slice(prefixes, func(i, j int) bool {
		if prefixes[i].count != prefixes[j].count {
			return prefixes[i].count > prefixes[j].count
		}
		return prefixes[i].prefix < prefixes[j].prefix
	})

	// Display results
	fmt.Printf("\nTotal Keys: %d\n", totalKeys)
	fmt.Println("\nKey Prefix Analysis (sorted by count):")
	fmt.Println("=====================================")
	
	for _, p := range prefixes {
		percentage := float64(p.count) * 100.0 / float64(totalKeys)
		fmt.Printf("%-20s: %8d keys (%6.2f%%) - Example: %s\n", 
			p.prefix, p.count, percentage, p.example)
		
		// Show structure for significant 1-byte prefixes
		if percentage > 1.0 && strings.HasPrefix(p.prefix, "1-byte:") && len(p.example) >= 24 {
			keyBytes, _ := hex.DecodeString(p.example)
			if len(keyBytes) >= 12 {
				fmt.Printf("  → Structure: prefix(%02x) + ", keyBytes[0])
				if len(keyBytes) == 41 || len(keyBytes) == 40 {
					fmt.Printf("8-byte-num + 32-byte-hash")
				}
				fmt.Println()
			}
		}
	}
	
	// Check for EVM patterns
	fmt.Println("\nEVM Database Pattern Check:")
	fmt.Println("===========================")
	patterns := map[string]string{
		"ASCII: evmh": "EVM Headers (hash->header)",
		"ASCII: evmH": "EVM Hash->Number mapping", 
		"ASCII: evmn": "EVM Canonical (number->hash)",
		"ASCII: evmb": "EVM Bodies",
		"ASCII: evmr": "EVM Receipts",
		"ASCII: evmt": "EVM Transactions",
		"Known: 0x48 (H)": "Headers (raw prefix)",
		"Known: 0x68 (h)": "Canonical hash (raw prefix)",
		"Known: 0x62 (b)": "Bodies (raw prefix)",
	}
	
	found := false
	for pattern, description := range patterns {
		for _, p := range prefixes {
			if p.prefix == pattern && p.count > 0 {
				fmt.Printf("✓ Found %s: %d keys\n", description, p.count)
				found = true
				break
			}
		}
	}
	
	if !found {
		fmt.Println("⚠ No standard EVM database patterns found")
		fmt.Println("  This might be a namespaced or non-EVM database")
	}

	return nil
}

func runInspectTip(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Finding Chain Tip in %s ===\n", dbPath)
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Look for LastHeader, LastBlock, LastFast
	keys := []string{
		"LastHeader",
		"LastBlock", 
		"LastFast",
	}

	for _, key := range keys {
		value, closer, err := db.Get([]byte(key))
		if err == nil {
			defer closer.Close()
			
			if len(value) == 32 {
				fmt.Printf("%s: %x\n", key, value)
				
				// Try to find the block number
				hashKey := append([]byte("H"), value...)
				if numValue, closer2, err := db.Get(hashKey); err == nil {
					defer closer2.Close()
					if len(numValue) == 8 {
						num := decodeBlockNumber(numValue)
						fmt.Printf("  Block Number: %d\n", num)
					}
				}
			}
		}
	}

	// Also scan for highest block by iterating headers
	fmt.Println("\nScanning for highest block...")
	maxBlock := uint64(0)
	
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("h"),
		UpperBound: []byte("i"),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 41 && key[0] == 'h' {
			blockNum := decodeBlockNumber(key[1:9])
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
		}
	}

	if maxBlock > 0 {
		fmt.Printf("\nHighest Block Found: %d\n", maxBlock)
	}

	return nil
}

// Helper functions

func identifyKeyType(key []byte) string {
	if len(key) == 0 {
		return "empty"
	}

	// Check text prefixes
	if len(key) >= 4 {
		prefix := string(key[:4])
		switch prefix {
		case "evmn":
			if len(key) == 12 {
				return "canonical-number"
			}
			return "canonical-hash"
		case "snap":
			return "snapshot"
		case "stat":
			return "statistics"
		}
	}

	// Check single-byte prefixes
	switch key[0] {
	case 'h':
		return "header"
	case 'b':
		return "body"
	case 'r':
		return "receipts"
	case 'H':
		return "hash-to-number"
	case 'n':
		return "number-to-hash"
	case 't':
		return "transaction"
	case 'l':
		return "transaction-lookup"
	case 'B':
		return "bloom-bits"
	case 's':
		return "state"
	case 'c':
		return "code"
	case 'D':
		return "difficulty"
	}

	return "unknown"
}

func tryDecodeValue(keyType string, value []byte) string {
	switch keyType {
	case "canonical-number", "hash-to-number":
		if len(value) == 8 {
			num := decodeBlockNumber(value)
			return fmt.Sprintf("Block #%d", num)
		}
	case "number-to-hash", "canonical-hash":
		if len(value) == 32 {
			return fmt.Sprintf("Hash: %x", value[:8])
		}
	}
	return ""
}

func truncateBytes(b []byte, maxLen int) []byte {
	if len(b) <= maxLen {
		return b
	}
	return b[:maxLen]
}

func isTextPrefix(b []byte) bool {
	for _, c := range b {
		if c < 32 || c > 126 {
			return false
		}
	}
	return true
}

func displayPrefixStats(prefixMap map[string]int, prefixLen int) {
	for prefix, count := range prefixMap {
		if len(prefix) == prefixLen {
			fmt.Printf("  %s: %d\n", prefix, count)
		}
	}
}

func displayTextPrefixes(prefixMap map[string]int) {
	for prefix, count := range prefixMap {
		if len(prefix) == 4 && isTextPrefix([]byte(prefix)) {
			fmt.Printf("  %s: %d\n", prefix, count)
		}
	}
}