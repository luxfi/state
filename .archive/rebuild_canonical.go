package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// pebbleDB implements a simple ethdb.Database wrapper
type pebbleDB struct {
	db *pebble.DB
}

func (p *pebbleDB) Has(key []byte) (bool, error) {
	val, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return len(val) > 0, nil
}

func (p *pebbleDB) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return append([]byte(nil), val...), nil
}

func (p *pebbleDB) Put(key []byte, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *pebbleDB) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *pebbleDB) NewBatch() ethdb.Batch {
	return &pebbleBatch{db: p.db, b: p.db.NewBatch()}
}

func (p *pebbleDB) NewBatchWithSize(size int) ethdb.Batch {
	return &pebbleBatch{db: p.db, b: p.db.NewBatch()}
}

func (p *pebbleDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	opts := &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	}
	iter := p.db.NewIter(opts)
	if start != nil {
		iter.SeekGE(start)
	} else {
		iter.First()
	}
	return &pebbleIterator{iter: iter, prefix: prefix}
}

func (p *pebbleDB) NewSnapshot() (ethdb.Snapshot, error) {
	return nil, fmt.Errorf("snapshots not implemented")
}

func (p *pebbleDB) Stat(property string) (string, error) {
	return "", fmt.Errorf("stats not implemented")
}

func (p *pebbleDB) Compact(start []byte, limit []byte) error {
	return nil
}

func (p *pebbleDB) Close() error {
	return p.db.Close()
}

// pebbleBatch implements ethdb.Batch
type pebbleBatch struct {
	db   *pebble.DB
	b    *pebble.Batch
	size int
}

func (b *pebbleBatch) Put(key, value []byte) error {
	b.size += len(key) + len(value)
	return b.b.Set(key, value, nil)
}

func (b *pebbleBatch) Delete(key []byte) error {
	b.size += len(key)
	return b.b.Delete(key, nil)
}

func (b *pebbleBatch) ValueSize() int {
	return b.size
}

func (b *pebbleBatch) Write() error {
	return b.b.Commit(pebble.Sync)
}

func (b *pebbleBatch) Reset() {
	b.b.Close()
	b.b = b.db.NewBatch()
	b.size = 0
}

func (b *pebbleBatch) Replay(w ethdb.KeyValueWriter) error {
	return fmt.Errorf("replay not implemented")
}

// pebbleIterator implements ethdb.Iterator
type pebbleIterator struct {
	iter   *pebble.Iterator
	prefix []byte
}

func (it *pebbleIterator) Next() bool {
	it.iter.Next()
	return it.iter.Valid()
}

func (it *pebbleIterator) Error() error {
	return it.iter.Error()
}

func (it *pebbleIterator) Key() []byte {
	return it.iter.Key()
}

func (it *pebbleIterator) Value() []byte {
	return it.iter.Value()
}

func (it *pebbleIterator) Release() {
	it.iter.Close()
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
	base, err := pebble.Open(*dbPath, opts)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer base.Close()

	// Wrap with ethdb interface
	db := &pebbleDB{db: base}

	// Step 1: Find the tip by scanning headers
	log.Println("Step 1: Finding tip by scanning headers...")
	tipNum := uint64(0)
	var tipHash common.Hash

	// Create iterator for headers (evmh prefix)
	iter := base.NewIter(&pebble.IterOptions{
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
			hash := common.BytesToHash(key[12:44])

			if num > tipNum {
				tipNum = num
				tipHash = hash
			}
			headerCount++
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	log.Printf("Found %d headers, tip at height %d, hash %s", headerCount, tipNum, tipHash.Hex())

	if tipNum == 0 {
		log.Fatal("No headers found in database")
	}

	// Step 2: Walk back and build canonical chain map
	log.Println("Step 2: Building canonical chain by walking back from tip...")
	canon := make(map[uint64]common.Hash)
	hash := tipHash
	num := tipNum

	for {
		canon[num] = hash
		if num == 0 {
			break
		}

		// Read the header
		header := rawdb.ReadHeader(db, hash, num)
		if header == nil {
			// Try to read raw header data if ReadHeader fails
			key := append([]byte("evmh"), make([]byte, 40)...)
			binary.BigEndian.PutUint64(key[4:12], num)
			copy(key[12:44], hash.Bytes())

			val, err := db.Get(key)
			if err != nil {
				log.Fatalf("Missing header at height %d, hash %s", num, hash.Hex())
			}

			// Decode header
			header = new(types.Header)
			if err := rlp.DecodeBytes(val, header); err != nil {
				log.Fatalf("Failed to decode header at height %d: %v", num, err)
			}
		}

		hash = header.ParentHash
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

		if err := batch.Put(key, hash.Bytes()); err != nil {
			log.Fatalf("Failed to write mapping for height %d: %v", num, err)
		}

		written++
		if written%10000 == 0 {
			// Flush batch periodically
			if err := batch.Write(); err != nil {
				log.Fatalf("Failed to write batch: %v", err)
			}
			batch.Reset()
			log.Printf("Written %d mappings...", written)
		}
	}

	// Write final batch
	if err := batch.Write(); err != nil {
		log.Fatalf("Failed to write final batch: %v", err)
	}

	log.Printf("Successfully wrote %d canonical mappings", written)
	log.Printf("Canonical chain tip: height=%d, hash=%s", tipNum, tipHash.Hex())

	// Verify by reading back a sample
	testNum := tipNum
	testKey := make([]byte, 12)
	copy(testKey[:4], []byte("evmn"))
	binary.BigEndian.PutUint64(testKey[4:], testNum)

	if val, err := db.Get(testKey); err == nil {
		readHash := common.BytesToHash(val)
		log.Printf("Verification: height %d -> hash %s", testNum, readHash.Hex())
	}
}
