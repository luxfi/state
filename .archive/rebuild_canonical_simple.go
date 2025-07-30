package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/rlp"
)

// Header represents an Ethereum block header
type Header struct {
	ParentHash  [32]byte
	UncleHash   [32]byte
	Coinbase    [20]byte
	Root        [32]byte
	TxHash      [32]byte
	ReceiptHash [32]byte
	Bloom       [256]byte
	Difficulty  []byte
	Number      []byte
	GasLimit    []byte
	GasUsed     []byte
	Time        []byte
	Extra       []byte
	MixDigest   [32]byte
	Nonce       [8]byte
	// Optional fields for newer versions
	BaseFee          []byte `rlp:"optional"`
	WithdrawalsHash  []byte `rlp:"optional"`
	BlobGasUsed      []byte `rlp:"optional"`
	ExcessBlobGas    []byte `rlp:"optional"`
	ParentBeaconRoot []byte `rlp:"optional"`
}

func main() {
	dbPath := flag.String("db", "", "Path to EVM PebbleDB")
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("--db required")
	}

	// Open the database
	opts := &pebble.Options{
		ReadOnly: false,
	}
	db, err := pebble.Open(*dbPath, opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Step 1: Find the tip by scanning headers
	log.Println("Step 1: Finding tip by scanning headers...")
	tipNum := uint64(0)
	var tipHash [32]byte

	// Create iterator for headers (evmh prefix)
	iter := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"), // next prefix
	})
	defer iter.Close()

	headerCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 44 && string(key[:4]) == "evmh" {
			// Key format: evmh + 8-byte number + 32-byte hash
			num := binary.BigEndian.Uint64(key[4:12])
			copy(tipHash[:], key[12:44])

			if num > tipNum {
				tipNum = num
			}
			headerCount++
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	log.Printf("Found %d headers, tip at height %d, hash %s", headerCount, tipNum, hex.EncodeToString(tipHash[:]))

	if tipNum == 0 {
		log.Fatal("No headers found in database")
	}

	// Step 2: Walk back and build canonical chain map
	log.Println("Step 2: Building canonical chain by walking back from tip...")
	canon := make(map[uint64][32]byte)
	hash := tipHash
	num := tipNum

	for {
		canon[num] = hash
		if num == 0 {
			break
		}

		// Read the header
		key := append([]byte("evmh"), make([]byte, 40)...)
		binary.BigEndian.PutUint64(key[4:12], num)
		copy(key[12:44], hash[:])

		val, closer, err := db.Get(key)
		if err != nil {
			log.Fatalf("Missing header at height %d, hash %s", num, hex.EncodeToString(hash[:]))
		}
		defer closer.Close()

		// Decode header to get parent hash
		var header Header
		if err := rlp.DecodeBytes(val, &header); err != nil {
			log.Printf("Warning: Failed to decode header at height %d: %v", num, err)
			// Try simpler approach - parent hash is first 32 bytes after RLP list prefix
			if len(val) > 33 {
				// Skip RLP prefix and get parent hash
				copy(hash[:], val[3:35])
			} else {
				log.Fatalf("Header too short at height %d", num)
			}
		} else {
			hash = header.ParentHash
		}

		num--

		if num%10000 == 0 {
			log.Printf("Progress: at height %d", num)
		}
	}

	log.Printf("Built canonical chain with %d blocks", len(canon))

	// Step 3: Write evmn (canonical number -> hash) mappings
	log.Println("Step 3: Writing canonical number->hash mappings...")
	batch := db.NewBatch()
	written := 0

	for num, hash := range canon {
		// Key format: evmn + 8-byte big-endian number
		key := make([]byte, 12)
		copy(key[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(key[4:], num)

		if err := batch.Set(key, hash[:], nil); err != nil {
			log.Fatalf("Failed to write mapping for height %d: %v", num, err)
		}

		written++
		if written%10000 == 0 {
			// Flush batch periodically
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to write batch: %v", err)
			}
			batch = db.NewBatch()
			log.Printf("Written %d mappings...", written)
		}
	}

	// Write final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to write final batch: %v", err)
	}

	log.Printf("Successfully wrote %d canonical mappings", written)
	log.Printf("Canonical chain tip: height=%d, hash=%s", tipNum, hex.EncodeToString(tipHash[:]))

	// Verify by reading back a sample
	testNum := tipNum
	testKey := make([]byte, 12)
	copy(testKey[:4], []byte("evmn"))
	binary.BigEndian.PutUint64(testKey[4:], testNum)

	if val, closer, err := db.Get(testKey); err == nil {
		defer closer.Close()
		log.Printf("Verification: height %d -> hash %s", testNum, hex.EncodeToString(val))
	}
}
