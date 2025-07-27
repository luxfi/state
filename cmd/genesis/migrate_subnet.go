package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	localrawdb "github.com/luxfi/genesis/pkg/rawdb"
	"github.com/spf13/cobra"
)

var migrateSubnetCmd = &cobra.Command{
	Use:   "migrate-subnet",
	Short: "Migrate subnet EVM data to C-Chain format",
	Long: `Migrate subnet EVM data to C-Chain format.
	
This command reads a subnet EVM PebbleDB and converts it to the standard
Coreth/Geth database format used by the C-Chain.

The tool handles:
- Subnet prefix removal (337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1)
- Key pattern discovery and mapping
- Conversion to standard rawdb prefixes
- Head pointer and chain config migration`,
	Run: runMigrateSubnet,
}

var (
	srcPath    string
	dstPath    string
	dryRun     bool
	verbose    bool
	maxKeys    int
)

func init() {
	migrateSubnetCmd.Flags().StringVar(&srcPath, "source", "", "Source subnet database path")
	migrateSubnetCmd.Flags().StringVar(&dstPath, "target", "", "Target C-Chain database path")
	migrateSubnetCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Analyze without migrating")
	migrateSubnetCmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed progress")
	migrateSubnetCmd.Flags().IntVar(&maxKeys, "max-keys", 0, "Limit number of keys to process (0 = all)")
	
	migrateSubnetCmd.MarkFlagRequired("source")
}

// The subnet prefix for chain 96369
var subnetPrefix = []byte{
	0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 
	0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e,
	0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a,
	0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1,
}

func runMigrateSubnet(cmd *cobra.Command, args []string) {
	if dryRun {
		fmt.Println("=== DRY RUN MODE - No data will be written ===")
		if dstPath == "" {
			dstPath = "/tmp/migration-test"
		}
	} else if dstPath == "" {
		log.Fatal("--target is required unless using --dry-run")
	}
	
	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{
		ReadOnly: dryRun || true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()
	
	fmt.Printf("=== Migrating Subnet Data ===\n")
	fmt.Printf("Source: %s\n", srcPath)
	fmt.Printf("Target: %s\n", dstPath)
	fmt.Printf("Mode: %s\n\n", func() string {
		if dryRun {
			return "DRY RUN"
		}
		return "MIGRATE"
	}())
	
	// Phase 1: Analyze database to understand key patterns
	fmt.Println("Phase 1: Analyzing database patterns...")
	patterns := analyzeDatabase(srcDB)
	
	// Phase 2: Build prefix mapping
	fmt.Println("\nPhase 2: Building prefix mapping...")
	mapping := buildPrefixMapping(patterns)
	
	if dryRun {
		fmt.Println("\n=== DRY RUN COMPLETE ===")
		return
	}
	
	// Phase 3: Migrate data
	fmt.Println("\nPhase 3: Migrating data...")
	if err := migrateData(srcDB, dstPath, mapping); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	
	fmt.Println("\n=== MIGRATION COMPLETE ===")
}

type keyPattern struct {
	prefix      []byte
	prefixStr   string
	count       int
	examples    [][]byte
	description string
}

func analyzeDatabase(db *pebble.DB) map[string]*keyPattern {
	patterns := make(map[string]*keyPattern)
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		count++
		
		if maxKeys > 0 && count > maxKeys {
			break
		}
		
		// Identify pattern
		pattern := identifyPattern(key)
		if pattern != nil {
			if p, exists := patterns[pattern.prefixStr]; exists {
				p.count++
				if len(p.examples) < 3 {
					p.examples = append(p.examples, copyBytes(key))
				}
			} else {
				pattern.count = 1
				pattern.examples = [][]byte{copyBytes(key)}
				patterns[pattern.prefixStr] = pattern
			}
		}
		
		if count%100000 == 0 {
			fmt.Printf("  Analyzed %d keys...\n", count)
		}
	}
	
	fmt.Printf("  Total keys analyzed: %d\n", count)
	fmt.Printf("  Unique patterns found: %d\n", len(patterns))
	
	// Display patterns
	fmt.Println("\n  Key patterns discovered:")
	for _, p := range patterns {
		fmt.Printf("    %s: %d keys (%s)\n", p.prefixStr, p.count, p.description)
		if verbose && len(p.examples) > 0 {
			fmt.Printf("      Example: %s\n", hex.EncodeToString(p.examples[0][:min(64, len(p.examples[0]))]))
		}
	}
	
	return patterns
}

func identifyPattern(key []byte) *keyPattern {
	// Check if it has subnet prefix
	hasSubnetPrefix := false
	if bytes.HasPrefix(key, subnetPrefix) {
		hasSubnetPrefix = true
		key = key[len(subnetPrefix):]
	}
	
	if len(key) == 0 {
		return nil
	}
	
	pattern := &keyPattern{}
	
	// Check for "evm/" prefix patterns
	if bytes.HasPrefix(key, []byte("evm/")) {
		if len(key) >= 6 {
			switch {
			case bytes.HasPrefix(key, []byte("evm/h/")):
				pattern.prefix = []byte("evm/h/")
				pattern.prefixStr = "evm/h/"
				pattern.description = "block headers"
			case bytes.HasPrefix(key, []byte("evm/b/")):
				pattern.prefix = []byte("evm/b/")
				pattern.prefixStr = "evm/b/"
				pattern.description = "block bodies"
			case bytes.HasPrefix(key, []byte("evm/n/")):
				pattern.prefix = []byte("evm/n/")
				pattern.prefixStr = "evm/n/"
				pattern.description = "number->hash"
			case bytes.HasPrefix(key, []byte("evm/H/")):
				pattern.prefix = []byte("evm/H/")
				pattern.prefixStr = "evm/H/"
				pattern.description = "hash->number"
			case bytes.HasPrefix(key, []byte("evm/T/")):
				pattern.prefix = []byte("evm/T/")
				pattern.prefixStr = "evm/T/"
				pattern.description = "total difficulty"
			case bytes.HasPrefix(key, []byte("evm/r/")):
				pattern.prefix = []byte("evm/r/")
				pattern.prefixStr = "evm/r/"
				pattern.description = "receipts"
			case bytes.HasPrefix(key, []byte("evm/l/")):
				pattern.prefix = []byte("evm/l/")
				pattern.prefixStr = "evm/l/"
				pattern.description = "tx lookups"
			case bytes.HasPrefix(key, []byte("evm/secure/")):
				pattern.prefix = []byte("evm/secure/")
				pattern.prefixStr = "evm/secure/"
				pattern.description = "secure trie nodes"
			case bytes.HasPrefix(key, []byte("evm/bloombits/")):
				pattern.prefix = []byte("evm/bloombits/")
				pattern.prefixStr = "evm/bloombits/"
				pattern.description = "bloom filter bits"
			default:
				pattern.prefix = []byte("evm/")
				pattern.prefixStr = "evm/other"
				pattern.description = "other evm data"
			}
		}
		return pattern
	}
	
	// Check for single-byte rawdb prefix (already in correct format)
	if len(key) > 0 {
		firstByte := key[0]
		switch firstByte {
		case 0x68: // 'h'
			pattern.prefixStr = "rawdb-h"
			pattern.description = "headers (already rawdb format)"
		case 0x62: // 'b'
			pattern.prefixStr = "rawdb-b"
			pattern.description = "bodies (already rawdb format)"
		case 0x6e: // 'n'
			pattern.prefixStr = "rawdb-n"
			pattern.description = "number->hash (already rawdb format)"
		case 0x48: // 'H'
			pattern.prefixStr = "rawdb-H"
			pattern.description = "hash->number (already rawdb format)"
		case 0x54: // 'T'
			pattern.prefixStr = "rawdb-T"
			pattern.description = "total difficulty (already rawdb format)"
		case 0x72: // 'r'
			pattern.prefixStr = "rawdb-r"
			pattern.description = "receipts (already rawdb format)"
		case 0x6c: // 'l'
			pattern.prefixStr = "rawdb-l"
			pattern.description = "tx lookups (already rawdb format)"
		case 0x53: // 'S'
			pattern.prefixStr = "rawdb-S"
			pattern.description = "secure trie (already rawdb format)"
		default:
			// Check if it's a state trie node (usually 0x00-0x0f range)
			if firstByte <= 0x0f {
				pattern.prefixStr = fmt.Sprintf("trie-%02x", firstByte)
				pattern.description = "state trie node"
			} else {
				pattern.prefixStr = fmt.Sprintf("unknown-%02x", firstByte)
				pattern.description = "unknown data"
			}
		}
		pattern.prefix = []byte{firstByte}
	}
	
	if hasSubnetPrefix {
		pattern.prefixStr = "subnet+" + pattern.prefixStr
	}
	
	return pattern
}

func buildPrefixMapping(patterns map[string]*keyPattern) map[string][]byte {
	mapping := make(map[string][]byte)
	
	// Standard mappings for "evm/" prefixes
	mapping["evm/h/"] = []byte{localrawdb.HeaderPrefix[0]}
	mapping["evm/b/"] = []byte{localrawdb.BlockBodyPrefix[0]}
	mapping["evm/n/"] = []byte{localrawdb.HeaderNumberPrefix[0]}
	mapping["evm/H/"] = []byte{localrawdb.HashPrefix[0]}
	mapping["evm/T/"] = []byte{localrawdb.HeaderTDSuffix[0]}
	mapping["evm/r/"] = []byte{localrawdb.BlockReceiptsPrefix[0]}
	mapping["evm/l/"] = []byte{localrawdb.TxLookupPrefix[0]}
	mapping["evm/secure/"] = []byte{0x53} // SecureTriePrefix
	
	// Special mappings for head pointers and config
	mapping["evm/head_header"] = localrawdb.HeadHeaderKey
	mapping["evm/head_block"] = localrawdb.HeadBlockKey
	mapping["evm/head_fast_block"] = localrawdb.HeadFastBlockKey
	mapping["evm/chain_config"] = []byte("ethereum-config-")
	
	fmt.Println("\n  Prefix mapping table:")
	for old, new := range mapping {
		if len(new) == 1 {
			fmt.Printf("    %s -> 0x%02x\n", old, new[0])
		} else {
			fmt.Printf("    %s -> %s\n", old, string(new))
		}
	}
	
	return mapping
}

func migrateData(srcDB *pebble.DB, dstPath string, mapping map[string][]byte) error {
	// Create target database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to create target database: %w", err)
	}
	defer dstDB.Close()
	
	// Create batch for efficient writes
	batch := dstDB.NewBatch()
	defer batch.Close()
	
	// Iterate through all keys
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	count := 0
	migrated := 0
	batchSize := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		count++
		
		if maxKeys > 0 && count > maxKeys {
			break
		}
		
		// Migrate the key
		newKey := migrateKey(key, mapping)
		if newKey != nil {
			if err := batch.Set(newKey, value, pebble.Sync); err != nil {
				return fmt.Errorf("failed to set key: %w", err)
			}
			migrated++
			batchSize++
			
			if verbose && migrated <= 10 {
				fmt.Printf("  Migrated: %s -> %s\n", 
					hex.EncodeToString(key[:min(32, len(key))]),
					hex.EncodeToString(newKey[:min(32, len(newKey))]))
			}
		}
		
		// Commit batch periodically
		if batchSize >= 10000 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch = dstDB.NewBatch()
			batchSize = 0
			
			if count%100000 == 0 {
				fmt.Printf("  Processed %d keys, migrated %d...\n", count, migrated)
			}
		}
	}
	
	// Final batch commit
	if batchSize > 0 {
		if err := batch.Commit(pebble.Sync); err != nil {
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
	}
	
	// Also migrate special keys that might not have prefixes
	if err := migrateSpecialKeys(srcDB, dstDB); err != nil {
		return fmt.Errorf("failed to migrate special keys: %w", err)
	}
	
	fmt.Printf("\n  Migration complete: %d keys processed, %d migrated\n", count, migrated)
	
	return nil
}

func migrateKey(key []byte, mapping map[string][]byte) []byte {
	// Remove subnet prefix if present
	if bytes.HasPrefix(key, subnetPrefix) {
		key = key[len(subnetPrefix):]
	}
	
	// Check each mapping
	for oldPrefix, newPrefix := range mapping {
		if bytes.HasPrefix(key, []byte(oldPrefix)) {
			suffix := key[len(oldPrefix):]
			return append(newPrefix, suffix...)
		}
	}
	
	// If key already starts with a rawdb prefix, keep it
	if len(key) > 0 {
		firstByte := key[0]
		if firstByte == 0x68 || firstByte == 0x62 || firstByte == 0x6e ||
		   firstByte == 0x48 || firstByte == 0x54 || firstByte == 0x72 ||
		   firstByte == 0x6c || firstByte == 0x53 {
			return key // Already in rawdb format
		}
		
		// State trie nodes (0x00-0x0f) should be kept as-is
		if firstByte <= 0x0f {
			return key
		}
	}
	
	// Unknown pattern - skip
	return nil
}

func migrateSpecialKeys(srcDB *pebble.DB, dstDB *pebble.DB) error {
	// List of special keys to check
	specialKeys := []struct {
		oldKey []byte
		newKey []byte
		name   string
	}{
		{localrawdb.HeadHeaderKey, localrawdb.HeadHeaderKey, "head header"},
		{localrawdb.HeadBlockKey, localrawdb.HeadBlockKey, "head block"},
		{localrawdb.HeadFastBlockKey, localrawdb.HeadFastBlockKey, "head fast block"},
		{[]byte("ethereum-config-"), []byte("ethereum-config-"), "chain config"},
		{[]byte("LastBlock"), localrawdb.HeadBlockKey, "last block (legacy)"},
		{[]byte("LastHeader"), localrawdb.HeadHeaderKey, "last header (legacy)"},
	}
	
	for _, sk := range specialKeys {
		val, closer, err := srcDB.Get(sk.oldKey)
		if err == nil {
			newVal := make([]byte, len(val))
			copy(newVal, val)
			closer.Close()
			
			if err := dstDB.Set(sk.newKey, newVal, pebble.Sync); err != nil {
				return fmt.Errorf("failed to migrate %s: %w", sk.name, err)
			}
			fmt.Printf("  Migrated special key: %s\n", sk.name)
		}
	}
	
	return nil
}

func copyBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}

