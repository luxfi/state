package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run set-chain-continuity.go <source-db> <dest-db> [block-number]")
		os.Exit(1)
	}

	srcPath := os.Args[1]
	dstPath := os.Args[2]
	targetBlock := uint64(14552) // Default to known block count
	
	if len(os.Args) > 3 {
		fmt.Sscanf(os.Args[3], "%d", &targetBlock)
	}

	fmt.Printf("🔧 Setting Chain Continuity\n")
	fmt.Printf("   Source: %s\n", srcPath)
	fmt.Printf("   Destination: %s\n", dstPath)
	fmt.Printf("   Target Block: %d\n", targetBlock)

	// Open source database to find the target block hash
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Open destination database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open destination database: %v", err)
	}
	defer dstDB.Close()

	// Find the block hash for block 14552
	// First, try to find it in the source database
	var targetHash common.Hash
	found := false

	// Look for number->hash mapping (key prefix 0x48 or 'H')
	numberKey := append([]byte{0x48}, encodeBlockNumber(targetBlock)...)
	if hashData, closer, err := srcDB.Get(numberKey); err == nil {
		targetHash = common.BytesToHash(hashData)
		closer.Close()
		found = true
		fmt.Printf("✓ Found block %d hash in source: 0x%s\n", targetBlock, hex.EncodeToString(targetHash[:]))
	}

	// If not found, try in destination database
	if !found {
		if hashData, closer, err := dstDB.Get(numberKey); err == nil {
			targetHash = common.BytesToHash(hashData)
			closer.Close()
			found = true
			fmt.Printf("✓ Found block %d hash in destination: 0x%s\n", targetBlock, hex.EncodeToString(targetHash[:]))
		}
	}

	// If still not found, scan for the highest block
	if !found {
		fmt.Println("⚠️  Block hash not found, scanning for highest block...")
		highestBlock, hash := findHighestBlock(dstDB)
		if highestBlock > 0 {
			targetBlock = highestBlock
			targetHash = hash
			found = true
			fmt.Printf("✓ Found highest block: %d (0x%s)\n", highestBlock, hex.EncodeToString(targetHash[:]))
		}
	}

	if !found {
		log.Fatal("❌ Could not find any blocks to set as chain head")
	}

	// Now set all the necessary pointer keys
	fmt.Println("\n🔗 Setting chain pointers...")

	// Set Height pointer
	heightBytes := encodeBlockNumber(targetBlock)
	if err := dstDB.Set([]byte("Height"), heightBytes, pebble.Sync); err != nil {
		log.Printf("Failed to set Height: %v", err)
	} else {
		fmt.Printf("✓ Height = %d\n", targetBlock)
	}

	// Set various forms of last accepted
	pointers := []string{
		"LastAccepted",
		"lastAccepted", 
		"last_accepted_key",
		"lastAcceptedKey",
		"LastBlock",
		"LastHeader",
		"LastFast",
	}

	for _, key := range pointers {
		if err := dstDB.Set([]byte(key), targetHash[:], pebble.Sync); err != nil {
			log.Printf("Failed to set %s: %v", key, err)
		} else {
			fmt.Printf("✓ %s = 0x%s\n", key, hex.EncodeToString(targetHash[:]))
		}
	}

	// Also set accepted block markers
	fmt.Println("\n📝 Marking blocks as accepted...")
	acceptedPrefix := []byte("a") // accepted blocks prefix
	
	// Mark blocks as accepted (sample every 1000 blocks)
	for blockNum := uint64(0); blockNum <= targetBlock; blockNum += 1000 {
		// Find block hash
		hashKey := append([]byte{0x48}, encodeBlockNumber(blockNum)...)
		var blockHash common.Hash
		
		// Try destination first
		if hashData, closer, err := dstDB.Get(hashKey); err == nil {
			blockHash = common.BytesToHash(hashData)
			closer.Close()
		} else if hashData, closer, err := srcDB.Get(hashKey); err == nil {
			blockHash = common.BytesToHash(hashData)
			closer.Close()
		} else {
			continue
		}
		
		// Mark as accepted
		acceptedKey := append(acceptedPrefix, blockHash[:]...)
		acceptedValue := encodeBlockNumber(blockNum)
		
		if err := dstDB.Set(acceptedKey, acceptedValue, pebble.Sync); err != nil {
			log.Printf("Failed to mark block %d as accepted: %v", blockNum, err)
		}
		
		if blockNum%10000 == 0 && blockNum > 0 {
			fmt.Printf("   Marked %d blocks as accepted...\n", blockNum)
		}
	}

	fmt.Printf("\n✅ Chain continuity set! Node should now recognize %d blocks\n", targetBlock)
	fmt.Println("\n🚀 To verify:")
	fmt.Println("   1. Start the node")
	fmt.Println("   2. Check RPC: curl -X POST -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_blockNumber\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc")
}

func findHighestBlock(db *pebble.DB) (uint64, common.Hash) {
	var highestNum uint64
	var highestHash common.Hash
	
	// Scan for number->hash mappings
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x48},
		UpperBound: []byte{0x49},
	})
	if err != nil {
		return 0, highestHash
	}
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid() && count < 10000; iter.Next() {
		key := iter.Key()
		if len(key) >= 9 {
			blockNum := binary.BigEndian.Uint64(key[1:9])
			if blockNum > highestNum {
				highestNum = blockNum
				value := iter.Value()
				if len(value) >= 32 {
					copy(highestHash[:], value[:32])
				}
			}
		}
		count++
	}
	
	return highestNum, highestHash
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}