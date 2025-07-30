package main

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
    "github.com/ethereum/go-ethereum/core/rawdb"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethdb"
    "github.com/ethereum/go-ethereum/rlp"
)

// pebbleDB wraps a Pebble database
type pebbleDB struct {
    db *pebble.DB
}

func (p *pebbleDB) Has(key []byte) (bool, error) {
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

func (p *pebbleDB) Get(key []byte) ([]byte, error) {
    val, closer, err := p.db.Get(key)
    if err != nil {
        return nil, err
    }
    defer closer.Close()
    result := make([]byte, len(val))
    copy(result, val)
    return result, nil
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
    opts := &pebble.IterOptions{}
    if prefix != nil {
        opts.LowerBound = prefix
        opts.UpperBound = append(prefix, 0xff)
    }
    iter, _ := p.db.NewIter(opts)
    return &pebbleIterator{iter: iter}
}

func (p *pebbleDB) Stat(property string) (string, error) {
    return "", nil
}

func (p *pebbleDB) NewSnapshot() (ethdb.Snapshot, error) {
    return p, nil
}

func (p *pebbleDB) Compact(start []byte, limit []byte) error {
    return nil
}

func (p *pebbleDB) Close() error {
    return p.db.Close()
}

type pebbleBatch struct {
    db *pebble.DB
    b  *pebble.Batch
}

func (b *pebbleBatch) Put(key []byte, value []byte) error {
    return b.b.Set(key, value, nil)
}

func (b *pebbleBatch) Delete(key []byte) error {
    return b.b.Delete(key, nil)
}

func (b *pebbleBatch) ValueSize() int {
    return int(b.b.Len())
}

func (b *pebbleBatch) Write() error {
    return b.b.Commit(pebble.Sync)
}

func (b *pebbleBatch) Reset() {
    b.b.Reset()
}

func (b *pebbleBatch) Replay(w ethdb.KeyValueWriter) error {
    return nil
}

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

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: convert-subnet-to-cchain <source-db> <dest-db>")
        os.Exit(1)
    }
    
    sourceDB := os.Args[1]
    destDB := os.Args[2]
    
    // Open source database (namespaced)
    srcPebble, err := pebble.Open(sourceDB, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open source database: %v", err)
    }
    defer srcPebble.Close()
    
    // Create destination database
    dstPebble, err := pebble.Open(destDB, &pebble.Options{})
    if err != nil {
        log.Fatalf("Failed to create destination database: %v", err)
    }
    defer dstPebble.Close()
    
    src := &pebbleDB{db: srcPebble}
    dst := &pebbleDB{db: dstPebble}
    
    fmt.Println("=== Converting Subnet Data to C-Chain Format ===")
    
    // First, let's find what block numbers we have
    fmt.Println("Scanning for block headers...")
    
    highestBlock := uint64(0)
    blockHashes := make(map[uint64][]byte)
    
    // Scan all keys to understand the structure
    iter := src.NewIterator(nil, nil)
    defer iter.Release()
    
    prefixCounts := make(map[byte]int)
    for iter.Next() {
        key := iter.Key()
        if len(key) > 0 {
            prefixCounts[key[0]]++
        }
    }
    
    fmt.Println("\nKey prefix distribution:")
    for prefix, count := range prefixCounts {
        fmt.Printf("  0x%02x: %d keys\n", prefix, count)
    }
    
    // Now let's process the data
    // For subnet data that was extracted with namespace tool, 
    // we need to find and convert the block data
    
    fmt.Println("\nLooking for block data...")
    
    // Try to read block 0 directly
    // In extracted data, canonical hash keys might be: h + number + n
    canonicalKey := append([]byte{0x68}, make([]byte, 8)...) // h + 00000000
    canonicalKey = append(canonicalKey, 0x6e) // + n
    
    if hash, err := src.Get(canonicalKey); err == nil {
        fmt.Printf("Found canonical hash for block 0: %s\n", hex.EncodeToString(hash))
        blockHashes[0] = hash
        
        // Write it to destination
        rawdb.WriteCanonicalHash(dst, hash, 0)
    }
    
    // Let's try a different approach - look for any headers
    iter2 := src.NewIterator([]byte{0x68}, nil) // h prefix
    defer iter2.Release()
    
    headerCount := 0
    for iter2.Next() && headerCount < 100 {
        key := iter2.Key()
        val := iter2.Value()
        
        if len(key) >= 9 {
            // Try to extract block number from key
            // Key format might be: h + number (8 bytes) + hash
            if len(key) > 9 {
                blockNum := binary.BigEndian.Uint64(key[1:9])
                if blockNum < 1000000 { // reasonable block number
                    // Try to decode as header
                    var header types.Header
                    if err := rlp.DecodeBytes(val, &header); err == nil {
                        fmt.Printf("Found header for block %d\n", blockNum)
                        
                        // Write to destination in proper format
                        rawdb.WriteHeader(dst, &header)
                        rawdb.WriteCanonicalHash(dst, header.Hash(), blockNum)
                        
                        if blockNum > highestBlock {
                            highestBlock = blockNum
                        }
                        headerCount++
                    }
                }
            }
        }
    }
    
    fmt.Printf("\nFound %d headers, highest block: %d\n", headerCount, highestBlock)
    
    // Copy other essential data
    fmt.Println("\nCopying state data...")
    
    // Copy accounts (prefix 0x26)
    copyPrefix(src, dst, []byte{0x26}, "accounts")
    
    // Copy storage (prefix 0xa3)
    copyPrefix(src, dst, []byte{0xa3}, "storage")
    
    // Copy code (prefix 0x63)
    copyPrefix(src, dst, []byte{0x63}, "code")
    
    // Set head block pointers
    if highestBlock > 0 {
        fmt.Printf("\nSetting head block to %d\n", highestBlock)
        
        // Get the hash of the highest block
        if hash, err := rawdb.ReadCanonicalHash(dst, highestBlock); err == nil {
            rawdb.WriteHeadHeaderHash(dst, hash)
            rawdb.WriteHeadBlockHash(dst, hash)
            rawdb.WriteHeadFastBlockHash(dst, hash)
        }
    }
    
    fmt.Println("\n✅ Conversion complete!")
}

func copyPrefix(src, dst ethdb.Database, prefix []byte, name string) {
    iter := src.NewIterator(prefix, nil)
    defer iter.Release()
    
    count := 0
    batch := dst.NewBatch()
    
    for iter.Next() {
        key := iter.Key()
        val := iter.Value()
        
        batch.Put(key, val)
        count++
        
        if count%1000 == 0 {
            batch.Write()
            batch.Reset()
            if count%10000 == 0 {
                fmt.Printf("  Copied %d %s entries...\n", count, name)
            }
        }
    }
    
    batch.Write()
    fmt.Printf("  Total %s copied: %d\n", name, count)
}