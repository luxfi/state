package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb"
)

// PebbleDBWrapper wraps PebbleDB to implement ethdb.Database
type PebbleDBWrapper struct {
	db *pebble.DB
}

func NewPebbleDBWrapper(path string) (*PebbleDBWrapper, error) {
	db, err := pebble.Open(path, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, err
	}
	return &PebbleDBWrapper{db: db}, nil
}

func (p *PebbleDBWrapper) Has(key []byte) (bool, error) {
	val, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return val != nil, nil
}

func (p *PebbleDBWrapper) Get(key []byte) ([]byte, error) {
	val, closer, err := p.db.Get(key)
	if err == pebble.ErrNotFound {
		return nil, fmt.Errorf("not found")
	}
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	result := make([]byte, len(val))
	copy(result, val)
	return result, nil
}

func (p *PebbleDBWrapper) Put(key []byte, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *PebbleDBWrapper) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *PebbleDBWrapper) DeleteRange(start, end []byte) error {
	batch := p.db.NewBatch()
	defer batch.Close()
	if err := batch.DeleteRange(start, end, nil); err != nil {
		return err
	}
	return batch.Commit(pebble.Sync)
}

func (p *PebbleDBWrapper) NewBatch() ethdb.Batch {
	return &pebbleBatch{db: p.db, b: p.db.NewBatch()}
}

func (p *PebbleDBWrapper) NewBatchWithSize(size int) ethdb.Batch {
	return &pebbleBatch{db: p.db, b: p.db.NewBatch()}
}

func (p *PebbleDBWrapper) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	opts := &pebble.IterOptions{}
	if prefix != nil {
		opts.LowerBound = prefix
		opts.UpperBound = append(prefix, 0xff)
	}
	if start != nil {
		opts.LowerBound = start
	}
	iter, err := p.db.NewIter(opts)
	if err != nil {
		// Return a dummy iterator that immediately returns false on Next()
		return &pebbleIterator{iter: nil}
	}
	return &pebbleIterator{iter: iter}
}

func (p *PebbleDBWrapper) Stat() (string, error) {
	return "", nil
}

func (p *PebbleDBWrapper) Compact(start []byte, limit []byte) error {
	return nil
}

func (p *PebbleDBWrapper) NewSnapshot() error {
	return fmt.Errorf("snapshots not implemented")
}

func (p *PebbleDBWrapper) Close() error {
	return p.db.Close()
}

// Ancient methods - not implemented for PebbleDB
func (p *PebbleDBWrapper) Ancient(kind string, number uint64) ([]byte, error) {
	return nil, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) AncientSize(kind string) (uint64, error) {
	return 0, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) Ancients() (uint64, error) {
	return 0, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) Tail() (uint64, error) {
	return 0, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	return nil, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) ReadAncients(fn func(op ethdb.AncientReaderOp) error) error {
	return fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) AncientDatadir() (string, error) {
	return "", fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) ModifyAncients(f func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, fmt.Errorf("ancient not supported")
}

func (p *PebbleDBWrapper) SyncAncient() error {
	return fmt.Errorf("ancient not supported")
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

func (b *pebbleBatch) DeleteRange(start, end []byte) error {
	// PebbleDB supports range deletes
	return b.b.DeleteRange(start, end, nil)
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
	iter *pebble.Iterator
}

func (i *pebbleIterator) Next() bool {
	if i.iter == nil {
		return false
	}
	return i.iter.Next()
}

func (i *pebbleIterator) Error() error {
	if i.iter == nil {
		return nil
	}
	return i.iter.Error()
}

func (i *pebbleIterator) Key() []byte {
	if i.iter == nil {
		return nil
	}
	return i.iter.Key()
}

func (i *pebbleIterator) Value() []byte {
	if i.iter == nil {
		return nil
	}
	return i.iter.Value()
}

func (i *pebbleIterator) Release() {
	if i.iter != nil {
		i.iter.Close()
	}
}

// Account represents an Ethereum account
type Account struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
	Nonce   uint64   `json:"nonce"`
}

func main() {
	var (
		dbPath     = flag.String("db", "", "Path to pebbledb directory")
		outputPath = flag.String("output", "", "Output file for allocations (JSON)")
		minBalance = flag.String("min", "0", "Minimum balance to include (in wei)")
	)
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("Database path is required (-db)")
	}

	// Parse minimum balance
	minBal := new(big.Int)
	if _, ok := minBal.SetString(*minBalance, 10); !ok {
		log.Fatalf("Invalid minimum balance: %s", *minBalance)
	}

	// Open database
	db, err := NewPebbleDBWrapper(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Get latest block header
	headHash := rawdb.ReadHeadBlockHash(db)
	if headHash == (common.Hash{}) {
		log.Fatal("No head block found in database")
	}

	headerNumber := rawdb.ReadHeaderNumber(db, headHash)
	if headerNumber == nil {
		log.Fatal("Failed to read header number")
	}
	header := rawdb.ReadHeader(db, headHash, *headerNumber)
	if header == nil {
		log.Fatal("Failed to read head header")
	}

	fmt.Printf("Latest block: %d (hash: %s)\n", header.Number, header.Hash().Hex())
	fmt.Printf("State root: %s\n", header.Root.Hex())

	// Create trie database
	tdb := triedb.NewDatabase(db, nil)
	
	// Open state trie
	sdb := state.NewDatabase(tdb, nil)
	stateDB, err := state.New(header.Root, sdb)
	if err != nil {
		log.Fatalf("Failed to open state: %v", err)
	}

	// Extract accounts
	fmt.Println("\nExtracting accounts...")
	accounts := make([]Account, 0)
	totalBalance := new(big.Int)
	accountCount := 0

	// Use state dump to extract accounts
	dump := stateDB.RawDump(nil)
	
	for addr, account := range dump.Accounts {
		// Parse balance
		balance := new(big.Int)
		if account.Balance != "" {
			balance.SetString(account.Balance, 10)
		}
		
		// Skip if balance is below minimum
		if balance.Cmp(minBal) < 0 {
			continue
		}

		accounts = append(accounts, Account{
			Address: addr,
			Balance: balance,
			Nonce:   account.Nonce,
		})
		
		totalBalance.Add(totalBalance, balance)
		accountCount++
		
		if accountCount%1000 == 0 {
			fmt.Printf("Processed %d accounts...\n", accountCount)
		}
	}

	fmt.Printf("\nExtracted %d accounts with total balance: %s wei\n", accountCount, totalBalance.String())
	
	// Write output
	if *outputPath != "" {
		output := map[string]interface{}{
			"chainId":      header.Number.String(),
			"blockNumber":  header.Number.String(),
			"stateRoot":    header.Root.Hex(),
			"totalBalance": totalBalance.String(),
			"accounts":     accounts,
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal output: %v", err)
		}

		if err := os.WriteFile(*outputPath, data, 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}

		fmt.Printf("\nWrote %d accounts to %s\n", len(accounts), *outputPath)
	}

	fmt.Printf("\nExtraction complete:\n")
	fmt.Printf("  Total accounts: %d\n", accountCount)
	fmt.Printf("  Total balance: %s wei\n", totalBalance.String())
}