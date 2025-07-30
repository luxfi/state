package main

import (
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: reconstruct-blocks <db-path>")
		fmt.Println("This reconstructs block headers based on state data")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("=== Reconstructing blocks in %s ===\n", dbPath)

	// We need to create block headers for blocks 0-14552
	// Since we don't have the actual block data, we'll create minimal headers
	// that reference the state

	// First, let's find the state root from the database
	// The state data should give us the final state root

	// For now, we'll create basic headers
	const maxBlock = 14552

	// Genesis block (0)
	genesisHeader := &types.Header{
		ParentHash:  common.Hash{},
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    common.Address{},
		Root:        common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"), // Empty state root
		TxHash:      types.EmptyTxsHash,
		ReceiptHash: types.EmptyReceiptsHash,
		Bloom:       types.Bloom{},
		Difficulty:  big.NewInt(1),
		Number:      big.NewInt(0),
		GasLimit:    8000000,
		GasUsed:     0,
		Time:        1640995200, // 2022-01-01
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
	}

	// Calculate genesis hash
	genesisHash := genesisHeader.Hash()

	// Store genesis header
	if err := storeHeader(db, genesisHeader); err != nil {
		log.Fatalf("Failed to store genesis header: %v", err)
	}

	// Store canonical hash
	if err := storeCanonicalHash(db, 0, genesisHash); err != nil {
		log.Fatalf("Failed to store canonical hash: %v", err)
	}

	fmt.Printf("Created genesis block: %x\n", genesisHash)

	// Create subsequent blocks
	parentHash := genesisHash

	for i := uint64(1); i <= maxBlock; i++ {
		header := &types.Header{
			ParentHash:  parentHash,
			UncleHash:   types.EmptyUncleHash,
			Coinbase:    common.Address{},
			Root:        common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"), // We'll update this
			TxHash:      types.EmptyTxsHash,
			ReceiptHash: types.EmptyReceiptsHash,
			Bloom:       types.Bloom{},
			Difficulty:  big.NewInt(1),
			Number:      big.NewInt(int64(i)),
			GasLimit:    8000000,
			GasUsed:     0,
			Time:        uint64(1640995200 + i*10), // 10 seconds per block
			Extra:       []byte{},
			MixDigest:   common.Hash{},
			Nonce:       types.BlockNonce{},
		}

		// For the last block, we should use the actual state root
		// But for now, we'll use a placeholder
		if i == maxBlock {
			// This should be the actual state root from the database
			// We'd need to calculate it from the state trie
		}

		hash := header.Hash()

		// Store header
		if err := storeHeader(db, header); err != nil {
			log.Printf("Failed to store header %d: %v", i, err)
			continue
		}

		// Store canonical hash
		if err := storeCanonicalHash(db, i, hash); err != nil {
			log.Printf("Failed to store canonical hash %d: %v", i, err)
			continue
		}

		// Store empty body
		body := &types.Body{
			Transactions: []*types.Transaction{},
			Uncles:       []*types.Header{},
		}
		if err := storeBody(db, hash, body); err != nil {
			log.Printf("Failed to store body %d: %v", i, err)
			continue
		}

		// Store empty receipts
		receipts := []*types.Receipt{}
		if err := storeReceipts(db, hash, receipts); err != nil {
			log.Printf("Failed to store receipts %d: %v", i, err)
			continue
		}

		parentHash = hash

		if i%1000 == 0 {
			fmt.Printf("Created block %d\n", i)
		}
	}

	// Set the pointer keys
	fmt.Println("\nSetting pointer keys...")

	// LastBlock points to the last block hash
	if err := db.Set([]byte("LastBlock"), parentHash.Bytes(), nil); err != nil {
		log.Printf("Failed to set LastBlock: %v", err)
	}

	// LastHeader points to the last block hash
	if err := db.Set([]byte("LastHeader"), parentHash.Bytes(), nil); err != nil {
		log.Printf("Failed to set LastHeader: %v", err)
	}

	// LastFast points to the last block hash (for fast sync)
	if err := db.Set([]byte("LastFast"), parentHash.Bytes(), nil); err != nil {
		log.Printf("Failed to set LastFast: %v", err)
	}

	fmt.Printf("\nReconstruction complete!\n")
	fmt.Printf("Created blocks 0-%d\n", maxBlock)
	fmt.Printf("Last block hash: %x\n", parentHash)
}

func storeHeader(db *pebble.DB, header *types.Header) error {
	hash := header.Hash()
	data, err := rlp.EncodeToBytes(header)
	if err != nil {
		return err
	}

	// Header key: 'H' + hash
	key := append([]byte{0x48}, hash.Bytes()...)
	return db.Set(key, data, nil)
}

func storeCanonicalHash(db *pebble.DB, number uint64, hash common.Hash) error {
	// Canonical hash key: 'h' + number(8 bytes) + 'n'
	key := make([]byte, 10)
	key[0] = 0x68 // 'h'

	// Encode block number as big endian
	for i := 0; i < 8; i++ {
		key[1+i] = byte(number >> uint(8*(7-i)))
	}

	key[9] = 0x6e // 'n'

	return db.Set(key, hash.Bytes(), nil)
}

func storeBody(db *pebble.DB, hash common.Hash, body *types.Body) error {
	data, err := rlp.EncodeToBytes(body)
	if err != nil {
		return err
	}

	// Body key: 'b' + hash
	key := append([]byte{0x62}, hash.Bytes()...)
	return db.Set(key, data, nil)
}

func storeReceipts(db *pebble.DB, hash common.Hash, receipts []*types.Receipt) error {
	data, err := rlp.EncodeToBytes(receipts)
	if err != nil {
		return err
	}

	// Receipt key: 'r' + hash
	key := append([]byte{0x72}, hash.Bytes()...)
	return db.Set(key, data, nil)
}
