package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/params"
	"github.com/luxfi/geth/trie"
	"github.com/luxfi/node/ids"
)

func main() {
	var (
		subnetPath = flag.String("subnet", "", "path to subnet EVM database")
		cchainPath = flag.String("cchain", "", "path to output C-Chain database")
		startBlock = flag.Uint64("start", 0, "starting block number")
		endBlock   = flag.Uint64("end", 0, "ending block number (0 = all)")
	)
	flag.Parse()

	if *subnetPath == "" || *cchainPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	if err := replaySubnetToChain(*subnetPath, *cchainPath, *startBlock, *endBlock); err != nil {
		log.Fatalf("Replay failed: %v", err)
	}
}

func replaySubnetToChain(subnetPath, cchainPath string, startBlock, endBlock uint64) error {
	fmt.Println("=== Subnet to C-Chain Replay ===")
	fmt.Printf("Subnet DB: %s\n", subnetPath)
	fmt.Printf("C-Chain DB: %s\n", cchainPath)
	fmt.Printf("Block range: %d to %d\n", startBlock, endBlock)

	// Open subnet database
	fmt.Println("\nOpening subnet database...")
	subnetDB, err := rawdb.NewLevelDBDatabase(subnetPath, 0, 0, "", false)
	if err != nil {
		return fmt.Errorf("failed to open subnet DB: %w", err)
	}
	defer subnetDB.Close()

	// Create new C-Chain database
	fmt.Println("Creating C-Chain database...")
	os.RemoveAll(cchainPath)
	cchainDB, err := rawdb.NewLevelDBDatabase(cchainPath, 0, 0, "", false)
	if err != nil {
		return fmt.Errorf("failed to create C-Chain DB: %w", err)
	}
	defer cchainDB.Close()

	// Find the highest block in subnet
	if endBlock == 0 {
		endBlock = findHighestBlock(subnetDB)
		fmt.Printf("Found highest block: %d\n", endBlock)
	}

	// Create genesis for C-Chain
	genesis := createGenesisFromSubnet(subnetDB, startBlock)
	
	// Initialize the C-Chain with genesis
	fmt.Println("\nInitializing C-Chain with genesis...")
	chainConfig := genesis.Config
	if chainConfig == nil {
		chainConfig = params.AllEthashProtocolChanges
	}
	
	// Create the blockchain
	chain, err := core.NewBlockChain(cchainDB, nil, genesis, nil, chainConfig, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to create blockchain: %w", err)
	}
	defer chain.Stop()

	// Replay blocks
	fmt.Printf("\nReplaying blocks %d to %d...\n", startBlock+1, endBlock)
	startTime := time.Now()

	for blockNum := startBlock + 1; blockNum <= endBlock; blockNum++ {
		// Read block from subnet
		block := readSubnetBlock(subnetDB, blockNum)
		if block == nil {
			log.Printf("Warning: Block %d not found in subnet", blockNum)
			continue
		}

		// Create a proper block for C-Chain
		cchainBlock := convertSubnetBlock(block, chain.GetHeaderByNumber(blockNum-1))
		
		// Insert block into C-Chain
		if _, err := chain.InsertChain([]*types.Block{cchainBlock}); err != nil {
			log.Printf("Error inserting block %d: %v", blockNum, err)
			// Try to continue with next block
			continue
		}

		// Also need to write state if available
		if err := copyBlockState(subnetDB, cchainDB, block.Root()); err != nil {
			log.Printf("Warning: Failed to copy state for block %d: %v", blockNum, err)
		}

		// Progress reporting
		if blockNum%1000 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(blockNum-startBlock) / elapsed.Seconds()
			eta := time.Duration(float64(endBlock-blockNum) / rate * float64(time.Second))
			fmt.Printf("  Block %d (%.0f blocks/sec, ETA: %v)\n", blockNum, rate, eta)
		}
	}

	// Write Snowman consensus metadata
	fmt.Println("\nWriting consensus metadata...")
	if err := writeConsensusMetadata(cchainDB, chain); err != nil {
		return fmt.Errorf("failed to write consensus metadata: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Replay Complete ===\n")
	fmt.Printf("Total blocks: %d\n", endBlock-startBlock)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(endBlock-startBlock)/elapsed.Seconds())

	return nil
}

func findHighestBlock(db ethdb.Database) uint64 {
	// Try to find the highest block number
	var highest uint64
	
	// Check head block pointers
	headKeys := [][]byte{
		[]byte("LastBlock"),
		[]byte("LastHeader"),
		[]byte("LastFast"),
	}
	
	for _, key := range headKeys {
		if hash, _ := db.Get(key); len(hash) == common.HashLength {
			if num := rawdb.ReadHeaderNumber(db, common.BytesToHash(hash)); num != nil {
				if *num > highest {
					highest = *num
				}
			}
		}
	}

	// If no head pointers, scan for blocks
	if highest == 0 {
		fmt.Println("Scanning for highest block...")
		for i := uint64(1); i < 10000000; i *= 10 {
			if rawdb.HasHeader(db, common.Hash{}, i) {
				highest = i
			} else if i > 1 {
				// Binary search for exact highest
				low, high := highest, i
				for low < high-1 {
					mid := (low + high) / 2
					if rawdb.HasHeader(db, common.Hash{}, mid) {
						low = mid
					} else {
						high = mid
					}
				}
				highest = low
				break
			}
		}
	}

	return highest
}

func createGenesisFromSubnet(db ethdb.Database, blockNum uint64) *core.Genesis {
	genesis := &core.Genesis{
		Config:     params.AllEthashProtocolChanges,
		Timestamp:  uint64(time.Now().Unix()),
		ExtraData:  []byte("Lux C-Chain"),
		GasLimit:   8000000,
		Difficulty: big.NewInt(0),
		Alloc:      make(core.GenesisAlloc),
	}

	// If starting from block 0, try to read genesis state
	if blockNum == 0 {
		// Read genesis block if available
		if block := rawdb.ReadBlock(db, rawdb.ReadCanonicalHash(db, 0), 0); block != nil {
			genesis.Timestamp = block.Time()
			genesis.GasLimit = block.GasLimit()
			genesis.Difficulty = block.Difficulty()
			genesis.ExtraData = block.Extra()
			
			// TODO: Read state from genesis block
		}
	} else {
		// Starting from existing block - read that state
		if header := rawdb.ReadHeader(db, rawdb.ReadCanonicalHash(db, blockNum), blockNum); header != nil {
			genesis.Timestamp = header.Time
			genesis.GasLimit = header.GasLimit
			
			// TODO: Read state from this block
		}
	}

	// Set chain ID to 96369 for Lux mainnet
	genesis.Config.ChainID = big.NewInt(96369)
	
	return genesis
}

func readSubnetBlock(db ethdb.Database, number uint64) *types.Block {
	hash := rawdb.ReadCanonicalHash(db, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return rawdb.ReadBlock(db, hash, number)
}

func convertSubnetBlock(subnetBlock *types.Block, parent *types.Header) *types.Block {
	// Create new header based on subnet block
	header := &types.Header{
		ParentHash:  parent.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    subnetBlock.Coinbase(),
		Root:        subnetBlock.Root(),
		TxHash:      subnetBlock.TxHash(),
		ReceiptHash: subnetBlock.ReceiptHash(),
		Bloom:       subnetBlock.Bloom(),
		Difficulty:  big.NewInt(0),
		Number:      subnetBlock.Number(),
		GasLimit:    subnetBlock.GasLimit(),
		GasUsed:     subnetBlock.GasUsed(),
		Time:        subnetBlock.Time(),
		Extra:       subnetBlock.Extra(),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     subnetBlock.BaseFee(),
	}

	// Create block with transactions
	return types.NewBlockWithHeader(header).WithBody(subnetBlock.Transactions(), nil)
}

func copyBlockState(srcDB, dstDB ethdb.Database, stateRoot common.Hash) error {
	// This is a simplified version - in reality, we'd need to properly copy the entire state trie
	// For now, just copy the trie nodes we can find
	
	srcTrie, err := trie.New(trie.StateTrieID(stateRoot), trie.NewDatabase(srcDB))
	if err != nil {
		return err
	}

	// Create iterator
	it := srcTrie.NodeIterator(nil)
	batch := dstDB.NewBatch()
	count := 0

	for it.Next(true) {
		if it.Hash() != (common.Hash{}) {
			if node, err := srcDB.Get(it.Hash().Bytes()); err == nil {
				batch.Put(it.Hash().Bytes(), node)
				count++
				
				if count%1000 == 0 {
					if err := batch.Write(); err != nil {
						return err
					}
					batch.Reset()
				}
			}
		}
	}

	return batch.Write()
}

func writeConsensusMetadata(db ethdb.Database, chain *core.BlockChain) error {
	// Write last accepted block info for Snowman consensus
	current := chain.CurrentBlock()
	if current == nil {
		return fmt.Errorf("no current block")
	}

	// Convert to Avalanche block ID
	blockID := ids.ID(current.Hash())
	
	// Write consensus keys (simplified - would need proper versiondb in production)
	batch := db.NewBatch()
	
	// Last accepted ID
	batch.Put([]byte("lastAcceptedID"), blockID[:])
	
	// Last accepted height
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, current.Number.Uint64())
	batch.Put([]byte("lastAcceptedHeight"), heightBytes)
	
	// Consensus initialization flag
	batch.Put([]byte("initialized"), []byte{0x01})
	
	return batch.Write()
}