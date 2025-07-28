package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
	"github.com/spf13/cobra"
)

// importSubnetCmd imports subnet chain data as C-Chain continuation
func importSubnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subnet [source-db] [dest-db]",
		Short: "Import subnet chain data as C-Chain continuation",
		Long: `Import subnet blockchain data with proper chain continuity.
		
This command:
1. Reads all blocks from the subnet database
2. Updates accepted block markers
3. Sets proper chain pointers for continuity
4. Ensures the node recognizes all historic blocks`,
		Args: cobra.ExactArgs(2),
		RunE: runImportSubnet,
	}
	
	cmd.Flags().Bool("verify", true, "Verify block continuity")
	cmd.Flags().Int("start-block", 0, "Starting block number")
	
	return cmd
}

func runImportSubnet(cmd *cobra.Command, args []string) error {
	// Resolve paths relative to work directory
	srcPath := ResolvePath(args[0])
	dstPath := ResolvePath(args[1])
	
	fmt.Printf("📦 Importing subnet data as C-Chain continuation\n")
	fmt.Printf("   Source: %s\n", srcPath)
	fmt.Printf("   Destination: %s\n", dstPath)
	
	// Check if source has namespace prefix (subnet data)
	hasNamespace, err := checkForNamespacePrefix(srcPath)
	if err != nil {
		return fmt.Errorf("failed to check source database: %w", err)
	}
	
	var extractedPath string
	if hasNamespace {
		fmt.Println("\n🔍 Detected namespaced subnet data, extracting...")
		
		// Extract to temporary directory
		extractedPath = dstPath + "-extracted"
		
		// Run extract state command
		extractCmd := exec.Command(os.Args[0], "extract", "state", srcPath, extractedPath, "--network", "96369")
		extractCmd.Stdout = os.Stdout
		extractCmd.Stderr = os.Stderr
		
		if err := extractCmd.Run(); err != nil {
			return fmt.Errorf("failed to extract state: %w", err)
		}
		
		// Use extracted data as source
		srcPath = extractedPath
		defer os.RemoveAll(extractedPath) // Clean up temp data
	}
	
	// Copy all data from source to destination
	fmt.Println("\n📋 Copying blockchain data...")
	
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()
	
	// Create destination directory
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()
	
	// Copy all keys
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	count := 0
	batch := dstDB.NewBatch()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		
		if err := batch.Set(key, value, nil); err != nil {
			return fmt.Errorf("failed to set key: %w", err)
		}
		
		count++
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch.Close()
			batch = dstDB.NewBatch()
			fmt.Printf("   Copied %d keys...\n", count)
		}
	}
	
	// Commit final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("failed to commit final batch: %w", err)
	}
	batch.Close()
	
	fmt.Printf("   Total keys copied: %d\n", count)
	
	// Find the highest block
	highestBlock, highestHash, err := findHighestBlock(dstDB)
	if err != nil {
		return fmt.Errorf("failed to find highest block: %w", err)
	}
	
	fmt.Printf("\n📊 Found highest block: %d (0x%s)\n", highestBlock, hex.EncodeToString(highestHash[:]))
	
	// Set chain continuity markers
	fmt.Println("\n🔗 Setting chain continuity markers...")
	
	// Set all the necessary pointers for C-Chain
	pointers := map[string][]byte{
		// EVM pointers
		"LastAcceptedKey": highestHash[:],
		"lastAcceptedKey": highestHash[:],
		"LastAccepted":    highestHash[:],
		"lastAccepted":    highestHash[:],
		"LastBlock":       highestHash[:],
		"LastHeader":      highestHash[:],
		"LastFast":        highestHash[:],
		"Height":          encodeBlockNumber(highestBlock),
		
		// Geth-specific pointers
		string(append([]byte("h"), encodeBlockNumber(highestBlock)...)): highestHash[:], // head block hash
		string([]byte("LastFinalized")): highestHash[:],
		string([]byte("LastSafe")):       highestHash[:],
	}
	
	for key, value := range pointers {
		if err := dstDB.Set([]byte(key), value, pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("   ✓ Set %s\n", key)
		}
	}
	
	fmt.Printf("\n✅ Import complete! Chain ready for C-Chain at block %d\n", highestBlock)
	fmt.Println("\n📍 Next steps:")
	fmt.Println("   1. Launch luxd with: ./bin/genesis launch L1")
	fmt.Println("   2. Verify with: ./bin/genesis launch verify")
	
	return nil
}

func findHighestBlock(db *pebble.DB) (uint64, common.Hash, error) {
	var highestNum uint64
	var highestHash common.Hash
	
	// Scan headers to find highest block
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x48}, // 'H' - number to hash mapping
		UpperBound: []byte{0x49},
	})
	if err != nil {
		return 0, highestHash, err
	}
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) < 9 { // prefix(1) + number(8)
			continue
		}
		
		blockNum := binary.BigEndian.Uint64(key[1:9])
		if blockNum > highestNum {
			highestNum = blockNum
			value := iter.Value()
			if len(value) >= 32 {
				copy(highestHash[:], value[:32])
			}
		}
	}
	
	return highestNum, highestHash, nil
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}

func checkForNamespacePrefix(dbPath string) (bool, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return false, err
	}
	defer db.Close()
	
	// Check first few keys to see if they have 32-byte namespace prefix
	iter, err := db.NewIter(nil)
	if err != nil {
		return false, err
	}
	defer iter.Close()
	
	// Sample a few keys
	count := 0
	var firstPrefix []byte
	allSamePrefix := true
	
	for iter.First(); iter.Valid() && count < 100; iter.Next() {
		key := iter.Key()
		
		// Subnet EVM uses 33-byte prefix (32 bytes namespace + 1 byte key type)
		if len(key) >= 33 {
			prefix := key[:32]
			
			if count == 0 {
				firstPrefix = make([]byte, 32)
				copy(firstPrefix, prefix)
			} else {
				// Check if all keys have same 32-byte prefix
				for i := 0; i < 32; i++ {
					if prefix[i] != firstPrefix[i] {
						allSamePrefix = false
						break
					}
				}
			}
		} else {
			// Key too short to have namespace
			allSamePrefix = false
		}
		count++
	}
	
	// If we found keys and they all have same 32-byte prefix, it's namespaced
	return count > 0 && allSamePrefix && len(firstPrefix) == 32, nil
}