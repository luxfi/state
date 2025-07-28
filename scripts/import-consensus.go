package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/node/database"
	"github.com/luxfi/node/database/prefixdb"
	"github.com/luxfi/node/database/versiondb"
)

// Simple mock for consensus state writes that mimics what Accept() would do
type ConsensusStateWriter struct {
	vdb      *versiondb.Database
	evmDB    *pebble.DB
}

// Key prefixes used by Snowman consensus
var (
	blkBytesPrefix   = []byte{0x00}
	blkStatusPrefix  = []byte{0x01}
	blkIDIndexPrefix = []byte{0x02}
	lastAcceptedKey  = []byte("last_accepted")
	statusAccepted   = byte(0x02)
)

func main() {
	var (
		chainDBPath  = flag.String("chain-db", "", "Path to chain database (e.g., runtime/mainnet/chainData/<ID>/db)")
		evmDBPath    = flag.String("evm-db", "", "Path to EVM database with blocks")
		tipHeight    = flag.Uint64("tip", 0, "Highest block height to import")
		batchSize    = flag.Int("batch", 10000, "Commit batch size")
	)
	flag.Parse()

	if *chainDBPath == "" || *evmDBPath == "" || *tipHeight == 0 {
		flag.Usage()
		os.Exit(1)
	}

	if err := importConsensus(*chainDBPath, *evmDBPath, *tipHeight, *batchSize); err != nil {
		log.Fatalf("Import failed: %v", err)
	}
}

func importConsensus(chainDBPath, evmDBPath string, tipHeight uint64, batchSize int) error {
	fmt.Printf("=== Consensus State Import ===\n")
	fmt.Printf("Chain DB: %s\n", chainDBPath)
	fmt.Printf("EVM DB: %s\n", evmDBPath) 
	fmt.Printf("Tip Height: %d\n", tipHeight)
	fmt.Printf("Batch Size: %d\n\n", batchSize)

	// 1. Open the chain's logical DB layers exactly like AvalancheGo
	fmt.Println("Opening chain database...")
	
	// Create pebble options
	opts := &pebble.Options{}
	
	// Open base database
	baseDB, err := pebble.Open(chainDBPath, opts)
	if err != nil {
		return fmt.Errorf("failed to open chain DB: %w", err)
	}
	defer baseDB.Close()

	// Wrap with database.Database interface
	base := WrapPebbleDB(baseDB)
	
	// Add prefix layer (state prefix)
	prefixed := prefixdb.New([]byte("state"), base)
	
	// Add version layer
	vdb := versiondb.New(prefixed)

	// 2. Open EVM database
	fmt.Println("Opening EVM database...")
	evmDB, err := pebble.Open(evmDBPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open EVM DB: %w", err)
	}
	defer evmDB.Close()

	// Create consensus writer
	writer := &ConsensusStateWriter{
		vdb:   vdb,
		evmDB: evmDB,
	}

	// 3. Import blocks
	fmt.Printf("\nImporting blocks 0-%d...\n", tipHeight)
	startTime := time.Now()

	for height := uint64(0); height <= tipHeight; height++ {
		// Get canonical hash for this height
		numKey := make([]byte, 12)
		copy(numKey[:4], []byte("evmn"))
		binary.BigEndian.PutUint64(numKey[4:], height)
		
		hash, closer, err := evmDB.Get(numKey)
		if err != nil {
			log.Printf("Warning: No canonical hash for height %d: %v", height, err)
			continue
		}
		ethHash := make([]byte, len(hash))
		copy(ethHash, hash)
		closer.Close()

		// Import this block
		if err := writer.AcceptBlock(height, ethHash); err != nil {
			return fmt.Errorf("failed to accept block %d: %w", height, err)
		}

		// Progress reporting
		if height%1000 == 0 {
			elapsed := time.Since(startTime)
			rate := float64(height) / elapsed.Seconds()
			eta := time.Duration(float64(tipHeight-height) / rate * float64(time.Second))
			fmt.Printf("  Height %d (%.0f blocks/sec, ETA: %v)\n", height, rate, eta)
		}

		// Commit periodically
		if height%uint64(batchSize) == 0 && height > 0 {
			if err := vdb.Commit(); err != nil {
				return fmt.Errorf("failed to commit at height %d: %w", height, err)
			}
		}
	}

	// Final commit
	fmt.Println("\nFinal commit...")
	if err := vdb.Commit(); err != nil {
		return fmt.Errorf("failed to final commit: %w", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\n=== Import Complete ===\n")
	fmt.Printf("Total blocks: %d\n", tipHeight+1)
	fmt.Printf("Total time: %v\n", elapsed)
	fmt.Printf("Average rate: %.0f blocks/sec\n", float64(tipHeight+1)/elapsed.Seconds())

	return nil
}

// AcceptBlock mimics what the consensus engine does when accepting a block
func (w *ConsensusStateWriter) AcceptBlock(height uint64, ethHash []byte) error {
	// Generate deterministic Snowman ID (same as our previous approach)
	snowmanID := generateSnowmanID(ethHash, height)

	// Create simple block bytes (we don't need full structure for consensus)
	blockBytes := createMinimalBlockBytes(snowmanID, height, ethHash)

	// 1. Store block bytes
	bytesKey := append(blkBytesPrefix, snowmanID[:]...)
	if err := w.vdb.Put(bytesKey, blockBytes); err != nil {
		return fmt.Errorf("failed to put block bytes: %w", err)
	}

	// 2. Mark block as accepted
	statusKey := append(blkStatusPrefix, snowmanID[:]...)
	if err := w.vdb.Put(statusKey, []byte{statusAccepted}); err != nil {
		return fmt.Errorf("failed to put status: %w", err)
	}

	// 3. Store height -> ID mapping
	heightKey := make([]byte, 9)
	copy(heightKey, blkIDIndexPrefix)
	binary.BigEndian.PutUint64(heightKey[1:], height)
	if err := w.vdb.Put(heightKey, snowmanID[:]); err != nil {
		return fmt.Errorf("failed to put height index: %w", err)
	}

	// 4. Update last accepted
	if err := w.vdb.Put(lastAcceptedKey, snowmanID[:]); err != nil {
		return fmt.Errorf("failed to put last accepted: %w", err)
	}

	return nil
}

// Same ID generation as before
func generateSnowmanID(ethHash []byte, height uint64) [32]byte {
	data := make([]byte, 8+len(ethHash))
	binary.BigEndian.PutUint64(data[:8], height)
	copy(data[8:], ethHash)
	return common.BytesToHash(data) // Using common.Hash as [32]byte
}

// Create minimal block bytes that consensus can parse
func createMinimalBlockBytes(snowmanID [32]byte, height uint64, ethHash []byte) []byte {
	// Simple format: [height(8)] [timestamp(8)] [ethHash(32)] [id(32)]
	blockBytes := make([]byte, 8+8+32+32)
	
	offset := 0
	// Height
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height)
	offset += 8
	
	// Timestamp (use height*12 for ~12 second blocks)
	binary.BigEndian.PutUint64(blockBytes[offset:offset+8], height*12)
	offset += 8
	
	// Ethereum hash
	copy(blockBytes[offset:offset+32], ethHash)
	offset += 32
	
	// Snowman ID
	copy(blockBytes[offset:offset+32], snowmanID[:])
	
	return blockBytes
}

// WrapPebbleDB wraps a pebble.DB to implement database.Database
type pebbleWrapper struct {
	db *pebble.DB
}

func WrapPebbleDB(db *pebble.DB) database.Database {
	return &pebbleWrapper{db: db}
}

func (p *pebbleWrapper) Has(key []byte) (bool, error) {
	_, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return true, nil
}

func (p *pebbleWrapper) Get(key []byte) ([]byte, error) {
	value, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return append([]byte{}, value...), nil
}

func (p *pebbleWrapper) Put(key []byte, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *pebbleWrapper) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *pebbleWrapper) NewBatch() database.Batch {
	return &pebbleBatch{
		batch: p.db.NewBatch(),
		db:    p.db,
	}
}

func (p *pebbleWrapper) NewIterator() database.Iterator {
	iter, _ := p.db.NewIter(nil)
	return &pebbleIterator{iter: iter}
}

func (p *pebbleWrapper) NewIteratorWithPrefix(prefix []byte) database.Iterator {
	iter, _ := p.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	return &pebbleIterator{iter: iter}
}

func (p *pebbleWrapper) NewIteratorWithStartAndPrefix(start, prefix []byte) database.Iterator {
	iter, _ := p.db.NewIter(&pebble.IterOptions{
		LowerBound: start,
		UpperBound: append(prefix, 0xff),
	})
	return &pebbleIterator{iter: iter}
}

func (p *pebbleWrapper) Close() error {
	return p.db.Close()
}

func (p *pebbleWrapper) Compact(start []byte, limit []byte) error {
	// Pebble automatically compacts, so this is a no-op
	return nil
}

func (p *pebbleWrapper) HealthCheck(ctx context.Context) (interface{}, error) {
	return map[string]interface{}{"status": "ok"}, nil
}

// Batch implementation
type pebbleBatch struct {
	batch *pebble.Batch
	db    *pebble.DB
}

func (b *pebbleBatch) Put(key, value []byte) error {
	return b.batch.Set(key, value, nil)
}

func (b *pebbleBatch) Delete(key []byte) error {
	return b.batch.Delete(key, nil)
}

func (b *pebbleBatch) Write() error {
	return b.batch.Commit(pebble.Sync)
}

func (b *pebbleBatch) Reset() {
	b.batch.Close()
	b.batch = b.db.NewBatch()
}

func (b *pebbleBatch) Replay(w database.KeyValueWriterDeleter) error {
	// Not implemented for this use case
	return nil
}

func (b *pebbleBatch) Inner() database.Batch {
	return b
}

func (b *pebbleBatch) Size() (int, error) {
	return len(b.batch.Repr()), nil
}

// Iterator implementation
type pebbleIterator struct {
	iter *pebble.Iterator
}

func (i *pebbleIterator) Next() bool {
	return i.iter.Next()
}

func (i *pebbleIterator) Error() error {
	return i.iter.Error()
}

func (i *pebbleIterator) Key() []byte {
	return i.iter.Key()
}

func (i *pebbleIterator) Value() []byte {
	return i.iter.Value()
}

func (i *pebbleIterator) Release() {
	i.iter.Close()
}