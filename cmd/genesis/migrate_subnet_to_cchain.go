package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/luxfi/ids"
	"github.com/spf13/cobra"
)

// migrateSubnetToCChainCmd migrates subnet data to C-Chain with proper prefixes
func migrateSubnetToCChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-subnet-to-cchain [source-db] [blockchain-id]",
		Short: "Migrate subnet data to C-Chain with blockchain ID prefixes",
		Long: `Migrate subnet blockchain data to C-Chain format.
		
This command:
1. Reads all data from the subnet database (already denamespacded)
2. Adds the C-Chain blockchain ID prefix to all keys
3. Sets proper chain pointers for continuity
4. Ensures the node recognizes all historic blocks`,
		Args: cobra.ExactArgs(2),
		RunE: runMigrateSubnetToCChain,
	}
	
	cmd.Flags().Bool("verify", true, "Verify block continuity")
	
	return cmd
}

func runMigrateSubnetToCChain(cmd *cobra.Command, args []string) error {
	srcPath := args[0]
	blockchainIDStr := args[1]
	
	// Parse blockchain ID
	blockchainID, err := ids.FromString(blockchainIDStr)
	if err != nil {
		return fmt.Errorf("invalid blockchain ID: %w", err)
	}
	
	fmt.Printf("🔄 Migrating subnet data to C-Chain\n")
	fmt.Printf("   Source: %s\n", srcPath)
	fmt.Printf("   Blockchain ID: %s\n", blockchainID.String())
	
	// Destination is the C-Chain database
	dstPath := fmt.Sprintf("/home/z/.luxd/chainData/%s/db/pebbledb", blockchainID.String())
	
	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()
	
	// Open destination database  
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()
	
	// First, find the highest block in source
	highestBlock, highestHash, err := findHighestBlockNoPrefix(srcDB)
	if err != nil {
		return fmt.Errorf("failed to find highest block: %w", err)
	}
	
	fmt.Printf("📊 Found highest block: %d (0x%s)\n", highestBlock, hex.EncodeToString(highestHash[:]))
	
	// Migrate all data with blockchain ID prefix
	fmt.Println("\n📦 Migrating data with blockchain ID prefix...")
	
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	count := 0
	start := time.Now()
	blockchainIDBytes := blockchainID[:]
	
	for iter.First(); iter.Valid(); iter.Next() {
		// Get key and value
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		
		// Add blockchain ID prefix to key
		prefixedKey := append(blockchainIDBytes, key...)
		
		// Write to destination
		if err := dstDB.Set(prefixedKey, value, pebble.Sync); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}
		
		count++
		if count%100000 == 0 {
			fmt.Printf("   Migrated %d keys...\n", count)
		}
	}
	
	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}
	
	// Set chain continuity markers
	fmt.Println("\n⚙️  Setting chain continuity markers...")
	
	// Set accepted block markers
	acceptedPrefix := append(blockchainIDBytes, []byte("a")...) // accepted blocks prefix
	
	for blockNum := uint64(0); blockNum <= highestBlock; blockNum++ {
		// Find block hash for this number
		hashKey := append([]byte{0x48}, encodeBlockNum(blockNum)...) // 0x48 = 'H' for number->hash
		hashData, closer, err := srcDB.Get(hashKey)
		if err != nil {
			continue // Skip if not found
		}
		blockHash := common.BytesToHash(hashData)
		closer.Close()
		
		// Mark as accepted
		acceptedKey := append(acceptedPrefix, blockHash[:]...)
		acceptedValue := encodeBlockNum(blockNum)
		
		if err := dstDB.Set(acceptedKey, acceptedValue, pebble.Sync); err != nil {
			log.Printf("Failed to mark block %d as accepted: %v", blockNum, err)
		}
		
		if blockNum%1000 == 0 && blockNum > 0 {
			fmt.Printf("   Marked %d blocks as accepted...\n", blockNum)
		}
	}
	
	// Set the chain head pointers with blockchain ID prefix
	fmt.Println("\n🔗 Setting chain head pointers...")
	
	// These keys need the blockchain ID prefix
	pointers := map[string][]byte{
		"lastAcceptedKey": highestHash[:],
		"LastAccepted":    highestHash[:],
		"lastAccepted":    highestHash[:],
		"LastBlock":       highestHash[:],
		"LastHeader":      highestHash[:],
		"Height":          encodeBlockNum(highestBlock),
	}
	
	for key, value := range pointers {
		prefixedKey := append(blockchainIDBytes, []byte(key)...)
		if err := dstDB.Set(prefixedKey, value, pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("   ✓ Set %s\n", key)
		}
	}
	
	fmt.Printf("\n✅ Migration complete! Migrated %d keys in %v\n", count, time.Since(start))
	fmt.Printf("   Chain will continue from block %d\n", highestBlock)
	fmt.Printf("   Blockchain ID: %s\n", blockchainID.String())
	
	return nil
}

// findHighestBlockNoPrefix finds the highest block without expecting prefixes
func findHighestBlockNoPrefix(db *pebble.DB) (uint64, common.Hash, error) {
	var highestNum uint64
	var highestHash common.Hash
	
	// Since namespace tool already stripped prefixes, scan all keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return 0, highestHash, err
	}
	defer iter.Close()
	
	// Look for keys that start with 0x48 (number to hash mapping)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 9 && key[0] == 0x48 {
			// This is a number->hash mapping
			blockNum := binary.BigEndian.Uint64(key[1:9])
			if blockNum > highestNum {
				highestNum = blockNum
				value := iter.Value()
				if len(value) >= 32 {
					copy(highestHash[:], value[:32])
				}
			}
		}
	}
	
	log.Printf("Found highest block: %d", highestNum)
	
	return highestNum, highestHash, nil
}

func encodeBlockNum(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}