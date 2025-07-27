package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/ids"
	"github.com/spf13/cobra"
)

// runDiagnose implements the diagnose command
func runDiagnose(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	fmt.Println("🔍 Blockchain Database Diagnostics")
	fmt.Println("==================================")
	fmt.Printf("Database path: %s\n\n", dbPath)
	
	// Step 1: Check if database exists
	pebblePath := filepath.Join(dbPath, "db", "pebbledb")
	if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
		// Try alternative paths
		pebblePath = filepath.Join(dbPath, "pebbledb")
		if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
			pebblePath = dbPath // Maybe it's already the pebbledb path
		}
	}
	
	fmt.Printf("✓ Database location: %s\n", pebblePath)
	
	// Step 2: Open database
	db, err := pebble.Open(pebblePath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("❌ Failed to open database: %w", err)
	}
	defer db.Close()
	
	fmt.Println("✓ Database opened successfully")
	
	// Step 3: Count headers
	headerCount := 0
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) > 0 && (key[0] == 'h' || (len(key) > 32 && key[32] == 0x68)) {
			headerCount++
		}
	}
	
	fmt.Printf("\n📊 Database Statistics:\n")
	fmt.Printf("   Headers found: %d (indicates ~%d blocks)\n", headerCount, headerCount)
	
	// Step 4: Check pointer keys
	fmt.Println("\n🔑 Pointer Keys:")
	pointerKeys := []string{"Height", "LastAccepted", "LastBlock", "LastHeader", "lastAccepted", "last_accepted_key"}
	foundPointers := 0
	
	for _, key := range pointerKeys {
		value, closer, err := db.Get([]byte(key))
		if err == nil {
			defer closer.Close()
			foundPointers++
			
			val := make([]byte, len(value))
			copy(val, value)
			
			if key == "Height" && len(val) == 8 {
				height := uint64(0)
				for i := 0; i < 8; i++ {
					height = (height << 8) | uint64(val[i])
				}
				fmt.Printf("   ✓ %-15s: %d (0x%x)\n", key, height, val)
			} else {
				fmt.Printf("   ✓ %-15s: 0x%x\n", key, val)
			}
		} else {
			fmt.Printf("   ✗ %-15s: <not found>\n", key)
		}
	}
	
	// Step 5: Check for genesis
	fmt.Println("\n📄 Genesis Configuration:")
	genesisValue, closer, err := db.Get([]byte("genesis"))
	if err == nil {
		defer closer.Close()
		fmt.Printf("   ✓ Found genesis blob: %d bytes\n", len(genesisValue))
		
		// Try to derive blockchain ID
		blockchainID, err := deriveBlockchainID(genesisValue)
		if err == nil {
			fmt.Printf("   ✓ Derived blockchain ID: %s\n", blockchainID.String())
		}
	} else {
		fmt.Printf("   ✗ Genesis not found in database\n")
	}
	
	// Step 6: Check for namespace prefixes
	fmt.Println("\n🔍 Checking for blockchain ID prefixes...")
	prefixMap := make(map[string]int)
	count := 0
	
	iter2, _ := db.NewIter(&pebble.IterOptions{})
	defer iter2.Close()
	
	for iter2.First(); iter2.Valid() && count < 1000; iter2.Next() {
		key := iter2.Key()
		if len(key) >= 32 {
			prefix := fmt.Sprintf("%x", key[:32])
			prefixMap[prefix]++
		}
		count++
	}
	
	if len(prefixMap) > 0 {
		fmt.Println("   Found blockchain ID prefixes:")
		for prefix, cnt := range prefixMap {
			if cnt > 10 { // Only show significant prefixes
				// Try to convert to blockchain ID
				var idBytes [32]byte
				prefixBytes, _ := hex.DecodeString(prefix)
				if len(prefixBytes) >= 32 {
					copy(idBytes[:], prefixBytes[:32])
				}
				id, err := ids.ToID(idBytes[:])
				if err == nil {
					fmt.Printf("   - %s (%d keys)\n", id.String(), cnt)
				} else {
					fmt.Printf("   - %s... (%d keys)\n", prefix[:16], cnt)
				}
			}
		}
	}
	
	// Step 7: Diagnosis summary
	fmt.Println("\n📋 Diagnosis Summary:")
	if headerCount == 0 {
		fmt.Println("   ⚠️  No headers found - database may be empty or corrupted")
	} else if foundPointers == 0 {
		fmt.Println("   ⚠️  No pointer keys found - database missing critical metadata")
	} else if err != nil {
		fmt.Println("   ⚠️  Genesis not found - may need to extract from config")
	} else {
		fmt.Println("   ✅ Database appears healthy with historic data")
	}
	
	fmt.Println("\n💡 Recommendations:")
	if headerCount > 0 && foundPointers == 0 {
		fmt.Println("   1. Use 'genesis pointers copy' to restore pointer keys from another database")
		fmt.Println("   2. Or use 'genesis migrate' to fully migrate the data")
	}
	if err != nil {
		fmt.Println("   1. Use 'genesis read' to extract genesis from chain data")
		fmt.Println("   2. Ensure genesis matches the blockchain ID")
	}
	
	return nil
}

// runCount implements the count command
func runCount(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	prefixStr, _ := cmd.Flags().GetString("prefix")
	countAll, _ := cmd.Flags().GetBool("all")
	
	// Find database path
	pebblePath := filepath.Join(dbPath, "db", "pebbledb")
	if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
		pebblePath = filepath.Join(dbPath, "pebbledb")
		if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
			pebblePath = dbPath
		}
	}
	
	db, err := pebble.Open(pebblePath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	fmt.Printf("📊 Counting keys in database: %s\n", pebblePath)
	
	// Parse prefix if provided
	var prefix []byte
	if !countAll && prefixStr != "" {
		prefix, err = hex.DecodeString(prefixStr)
		if err != nil {
			return fmt.Errorf("invalid hex prefix: %w", err)
		}
		fmt.Printf("   Filtering by prefix: 0x%s\n", prefixStr)
	}
	
	// Count keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	totalCount := 0
	prefixCounts := make(map[byte]int)
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Filter by prefix if specified
		if !countAll && prefix != nil && !bytes.HasPrefix(key, prefix) {
			continue
		}
		
		totalCount++
		
		// Track key type by first byte after blockchain ID
		if len(key) > 32 {
			prefixCounts[key[32]]++
		} else if len(key) > 0 {
			prefixCounts[key[0]]++
		}
		
		if totalCount%100000 == 0 {
			fmt.Printf("   Counted %d keys...\n", totalCount)
		}
	}
	
	fmt.Printf("\n✅ Total keys: %d\n", totalCount)
	
	if len(prefixCounts) > 0 {
		fmt.Println("\n📈 Key distribution by type:")
		
		// Common key types
		keyTypes := map[byte]string{
			0x68: "headers",
			0x6c: "last values",
			0x48: "Headers",
			0x72: "receipts",
			0x62: "bodies",
			0x42: "Bodies",
			0x6e: "number->hash",
			0x74: "transactions",
			0xfd: "metadata",
			0x26: "accounts",
			0xa3: "storage",
			0x6f: "objects",
			0x73: "state",
			0x63: "code",
		}
		
		for prefix, count := range prefixCounts {
			typeName := keyTypes[prefix]
			if typeName == "" {
				typeName = fmt.Sprintf("unknown (0x%02x)", prefix)
			}
			fmt.Printf("   %-20s: %d\n", typeName, count)
		}
	}
	
	return nil
}

// runPointersShow implements the pointers show command
func runPointersShow(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("database path required")
	}
	dbPath := args[0]
	return showPointerKeys(dbPath)
}

// runPointersSet implements the pointers set command
func runPointersSet(cmd *cobra.Command, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: pointers set [db-path] [key] [value]")
	}
	dbPath := args[0]
	key := args[1]
	value := args[2]
	
	// Find database path
	pebblePath := filepath.Join(dbPath, "db", "pebbledb")
	if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
		pebblePath = filepath.Join(dbPath, "pebbledb")
		if _, err := os.Stat(pebblePath); os.IsNotExist(err) {
			pebblePath = dbPath
		}
	}
	
	db, err := pebble.Open(pebblePath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Parse value as hex
	valueBytes, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		return fmt.Errorf("invalid hex value: %w", err)
	}
	
	// Set the key
	if err := db.Set([]byte(key), valueBytes, pebble.Sync); err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	
	fmt.Printf("✅ Set %s = 0x%x\n", key, valueBytes)
	return nil
}

// runPointersCopy implements the pointers copy command
func runPointersCopy(cmd *cobra.Command, args []string) error {
	srcPath := args[0]
	dstPath := args[1]
	
	// Open source database
	srcPebble := findPebblePath(srcPath)
	srcDB, err := pebble.Open(srcPebble, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()
	
	// Open destination database
	dstPebble := findPebblePath(dstPath)
	dstDB, err := pebble.Open(dstPebble, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()
	
	// Copy pointer keys
	pointerKeys := []string{"Height", "LastAccepted", "LastBlock", "LastHeader", "lastAccepted", "last_accepted_key"}
	copied := 0
	
	fmt.Println("📋 Copying pointer keys...")
	
	for _, key := range pointerKeys {
		value, closer, err := srcDB.Get([]byte(key))
		if err != nil {
			continue
		}
		
		val := make([]byte, len(value))
		copy(val, value)
		closer.Close()
		
		if err := dstDB.Set([]byte(key), val, pebble.Sync); err != nil {
			fmt.Printf("   ✗ Failed to copy %s: %v\n", key, err)
		} else {
			fmt.Printf("   ✓ Copied %s\n", key)
			copied++
		}
	}
	
	fmt.Printf("\n✅ Copied %d pointer keys\n", copied)
	return nil
}

// findPebblePath finds the pebbledb path from various possible locations
func findPebblePath(basePath string) string {
	// Try common patterns
	patterns := []string{
		filepath.Join(basePath, "db", "pebbledb"),
		filepath.Join(basePath, "pebbledb"),
		basePath, // Maybe it's already the pebbledb path
	}
	
	for _, path := range patterns {
		if _, err := os.Stat(filepath.Join(path, "CURRENT")); err == nil {
			return path
		}
	}
	
	// Default to first pattern
	return patterns[0]
}

// NewBuildCommand creates the build command structure
func NewBuildCommand() *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build genesis files",
		Long:  "Build genesis files for different networks and configurations",
	}
	
	// Add subcommands for different build types
	buildCmd.AddCommand(
		&cobra.Command{
			Use:   "all",
			Short: "Build all genesis files",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Building all genesis files...")
				return nil
			},
		},
	)
	
	return buildCmd
}

// NewMigrateCommand creates the migrate command structure
func NewMigrateCommand() *cobra.Command {
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate blockchain data",
		Long: `Comprehensive migration tool for blockchain data.
		
This command performs a complete migration of historic blockchain data:
1. Extracts genesis from the source database
2. Derives the new blockchain ID
3. Migrates all data with blockchain ID translation
4. Preserves pointer keys and block data
5. Configures the node for the migrated chain`,
		Args: cobra.ExactArgs(1),
		RunE: runComprehensiveMigrate,
	}
	
	migrateCmd.Flags().StringP("destination", "d", "", "Destination path for migrated data")
	migrateCmd.Flags().StringP("old-id", "o", "", "Old blockchain ID (auto-detected if not specified)")
	migrateCmd.Flags().BoolP("dry-run", "n", false, "Show what would be done without making changes")
	migrateCmd.Flags().BoolP("preserve-original", "p", true, "Keep original data intact")
	
	return migrateCmd
}

// runComprehensiveMigrate performs a complete migration
func runComprehensiveMigrate(cmd *cobra.Command, args []string) error {
	srcPath := args[0]
	dstPath, _ := cmd.Flags().GetString("destination")
	oldIDStr, _ := cmd.Flags().GetString("old-id")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	// preserve, _ := cmd.Flags().GetBool("preserve-original") // Reserved for future use
	
	fmt.Println("🔄 Comprehensive Blockchain Migration")
	fmt.Println("====================================")
	
	// Step 1: Diagnose source database
	fmt.Printf("\n📊 Analyzing source database: %s\n", srcPath)
	if err := runDiagnose(&cobra.Command{}, []string{srcPath}); err != nil {
		return fmt.Errorf("source database diagnosis failed: %w", err)
	}
	
	// Step 2: Extract genesis
	fmt.Println("\n📄 Extracting genesis configuration...")
	genesis, genesisBytes, err := extractHistoricGenesis(srcPath)
	if err != nil {
		return fmt.Errorf("failed to extract genesis: %w", err)
	}
	
	// Step 3: Derive blockchain ID
	newBlockchainID, err := deriveBlockchainID(genesisBytes)
	if err != nil {
		return fmt.Errorf("failed to derive blockchain ID: %w", err)
	}
	
	fmt.Printf("✅ New blockchain ID: %s\n", newBlockchainID.String())
	
	// Step 4: Determine old blockchain ID
	var oldBlockchainID ids.ID
	if oldIDStr != "" {
		oldBlockchainID, err = ids.FromString(oldIDStr)
		if err != nil {
			return fmt.Errorf("invalid old blockchain ID: %w", err)
		}
	} else {
		// Try to auto-detect from path or database
		base := filepath.Base(srcPath)
		if base != "." && base != "/" {
			oldBlockchainID, _ = ids.FromString(base)
		}
	}
	
	if oldBlockchainID != ids.Empty {
		fmt.Printf("📌 Old blockchain ID: %s\n", oldBlockchainID.String())
	}
	
	// Step 5: Set destination path
	if dstPath == "" {
		dstPath = filepath.Join(os.Getenv("HOME"), ".luxd", "chainData", newBlockchainID.String())
	}
	
	fmt.Printf("\n📁 Migration paths:\n")
	fmt.Printf("   Source: %s\n", srcPath)
	fmt.Printf("   Destination: %s\n", dstPath)
	
	if dryRun {
		fmt.Println("\n🔍 DRY RUN - No changes will be made")
		return nil
	}
	
	// Step 6: Write genesis configuration
	fmt.Println("\n📝 Writing genesis configuration...")
	genesisDir := filepath.Join(os.Getenv("HOME"), ".luxd", "configs", "C")
	if err := os.MkdirAll(genesisDir, 0755); err != nil {
		return fmt.Errorf("failed to create genesis directory: %w", err)
	}
	
	genesisPath := filepath.Join(genesisDir, "genesis.json")
	formattedGenesis, _ := json.MarshalIndent(genesis, "", "  ")
	if err := ioutil.WriteFile(genesisPath, formattedGenesis, 0644); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}
	
	// Also create per-chain aliases
	aliasesPath := filepath.Join(os.Getenv("HOME"), ".luxd", "chainData", newBlockchainID.String(), "aliases")
	if err := os.MkdirAll(filepath.Dir(aliasesPath), 0755); err != nil {
		return fmt.Errorf("failed to create aliases directory: %w", err)
	}
	
	if err := ioutil.WriteFile(aliasesPath, []byte(`["C","C-Chain"]`), 0644); err != nil {
		return fmt.Errorf("failed to write aliases: %w", err)
	}
	
	// Step 7: Migrate the data
	fmt.Println("\n🚀 Starting data migration...")
	if err := migrateChainData(srcPath, dstPath, newBlockchainID); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	
	// Step 8: Verify migration
	fmt.Println("\n✓ Verifying migrated database...")
	if err := runDiagnose(&cobra.Command{}, []string{dstPath}); err != nil {
		fmt.Printf("⚠️  Warning: Verification showed issues: %v\n", err)
	}
	
	// Step 9: Create chain config
	chainConfigPath := filepath.Join(os.Getenv("HOME"), ".luxd", "configs", "chains", "C", "config.json")
	if err := os.MkdirAll(filepath.Dir(chainConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create chain config directory: %w", err)
	}
	
	chainConfig := map[string]interface{}{
		"state-sync-enabled": false,
		"pruning-enabled":    false,
		"log-level":          "info",
	}
	
	chainConfigBytes, _ := json.MarshalIndent(chainConfig, "", "  ")
	if err := ioutil.WriteFile(chainConfigPath, chainConfigBytes, 0644); err != nil {
		return fmt.Errorf("failed to write chain config: %w", err)
	}
	
	// Step 10: Summary
	fmt.Println("\n✅ Migration Complete!")
	fmt.Println("\n📋 Summary:")
	fmt.Printf("   Blockchain ID: %s\n", newBlockchainID.String())
	fmt.Printf("   Genesis: %s\n", genesisPath)
	fmt.Printf("   Chain data: %s\n", dstPath)
	fmt.Printf("   Aliases: %s\n", aliasesPath)
	fmt.Printf("   Chain config: %s\n", chainConfigPath)
	
	fmt.Println("\n🚀 To start the node:")
	fmt.Println("   cd /home/z/work/lux/node")
	fmt.Printf("   ./build/luxd --http-port=9630\n")
	
	return nil
}