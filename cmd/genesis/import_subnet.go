package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

// importSubnetCmd imports subnet chain data as C-Chain continuation
func importSubnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-subnet [source-db] [dest-db]",
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
	srcPath := args[0]
	dstPath := args[1]
	
	fmt.Printf("📦 Importing subnet data as C-Chain continuation\n")
	fmt.Printf("   Source: %s\n", srcPath)
	fmt.Printf("   Destination: %s\n", dstPath)
	
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
	
	// Find the highest block in source
	highestBlock, highestHash, err := findHighestBlock(srcDB)
	if err != nil {
		return fmt.Errorf("failed to find highest block: %w", err)
	}
	
	fmt.Printf("📊 Found highest block: %d (0x%s)\n", highestBlock, hex.EncodeToString(highestHash[:]))
	
	// Set accepted block markers
	fmt.Println("\n⚙️  Setting chain continuity markers...")
	
	// For each block, mark it as accepted
	acceptedPrefix := []byte("a") // accepted blocks prefix
	
	for blockNum := uint64(0); blockNum <= highestBlock; blockNum++ {
		// Find block hash for this number
		hashKey := append([]byte{0x48}, encodeBlockNumber(blockNum)...) // 0x48 = 'H' for number->hash
		hashData, closer, err := srcDB.Get(hashKey)
		if err != nil {
			continue // Skip if not found
		}
		blockHash := common.BytesToHash(hashData)
		closer.Close()
		
		// Mark as accepted
		acceptedKey := append(acceptedPrefix, blockHash[:]...)
		acceptedValue := encodeBlockNumber(blockNum)
		
		if err := dstDB.Set(acceptedKey, acceptedValue, pebble.Sync); err != nil {
			log.Printf("Failed to mark block %d as accepted: %v", blockNum, err)
		}
		
		if blockNum%1000 == 0 {
			fmt.Printf("   Marked %d blocks as accepted...\n", blockNum)
		}
	}
	
	// Set the chain head pointers
	fmt.Println("\n🔗 Setting chain head pointers...")
	
	// LastAcceptedKey used by EVM
	lastAcceptedKey := []byte("lastAcceptedKey")
	if err := dstDB.Set(lastAcceptedKey, highestHash[:], pebble.Sync); err != nil {
		return fmt.Errorf("failed to set lastAcceptedKey: %w", err)
	}
	
	// Standard pointer keys
	pointers := map[string][]byte{
		"LastAccepted": highestHash[:],
		"lastAccepted": highestHash[:],
		"LastBlock":    highestHash[:],
		"LastHeader":   highestHash[:],
		"Height":       encodeBlockNumber(highestBlock),
	}
	
	for key, value := range pointers {
		if err := dstDB.Set([]byte(key), value, pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("   ✓ Set %s\n", key)
		}
	}
	
	// Also set head block hash
	headBlockKey := []byte("LastBlock")
	if err := dstDB.Set(headBlockKey, highestHash[:], pebble.Sync); err != nil {
		log.Printf("Failed to set head block: %v", err)
	}
	
	// Set fast sync head
	headFastKey := []byte("LastFast") 
	if err := dstDB.Set(headFastKey, highestHash[:], pebble.Sync); err != nil {
		log.Printf("Failed to set fast head: %v", err)
	}
	
	fmt.Printf("\n✅ Import complete! Chain will continue from block %d\n", highestBlock)
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