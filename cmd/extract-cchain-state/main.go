package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/core/state"
	"github.com/luxfi/geth/core/state/snapshot"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/ethdb"
	"github.com/luxfi/geth/trie"
	"github.com/spf13/cobra"
)

var (
	cfg = struct {
		DataDir    string
		OutputDir  string
		BlockNum   int64
		Format     string
		MinBalance string
	}{
		OutputDir:  "extracted-state",
		Format:     "json",
		MinBalance: "0",
	}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&cfg.DataDir, "datadir", "", "C-Chain data directory")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "Output directory")
	rootCmd.PersistentFlags().Int64Var(&cfg.BlockNum, "block", 0, "Block number to extract (0 for latest)")
	rootCmd.PersistentFlags().StringVar(&cfg.Format, "format", cfg.Format, "Output format (json, csv)")
	rootCmd.PersistentFlags().StringVar(&cfg.MinBalance, "min-balance", cfg.MinBalance, "Minimum balance to include")
}

var rootCmd = &cobra.Command{
	Use:   "extract-cchain-state",
	Short: "Extract C-Chain state from database",
	Long:  "Tool to extract account balances and state from C-Chain database",
	RunE:  runExtract,
}

type AccountData struct {
	Address  string   `json:"address"`
	Balance  string   `json:"balance"`
	Nonce    uint64   `json:"nonce"`
	CodeHash string   `json:"codeHash,omitempty"`
	IsContract bool   `json:"isContract"`
}

func runExtract(cmd *cobra.Command, args []string) error {
	if cfg.DataDir == "" {
		// Try to find C-Chain data in common locations
		possiblePaths := []string{
			"chaindata/lux-mainnet-96369/db/pebbledb",
			"/home/z/.lux-mainnet/chains/C/db",
			"/home/z/.avalanche-cli/runs/network_96369/node1/data/chains/C/db",
		}
		
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				cfg.DataDir = path
				fmt.Printf("Found C-Chain data at: %s\n", path)
				break
			}
		}
		
		if cfg.DataDir == "" {
			return fmt.Errorf("C-Chain data directory not found, please specify with --datadir")
		}
	}

	// Parse minimum balance
	minBalance, ok := new(big.Int).SetString(cfg.MinBalance, 10)
	if !ok {
		return fmt.Errorf("invalid minimum balance: %s", cfg.MinBalance)
	}

	// Open database
	db, err := openDatabase(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get latest block if not specified
	var header *types.Header
	if cfg.BlockNum == 0 {
		header = rawdb.ReadHeadHeader(db)
		if header == nil {
			return fmt.Errorf("failed to read head header")
		}
	} else {
		hash := rawdb.ReadCanonicalHash(db, uint64(cfg.BlockNum))
		if hash == (common.Hash{}) {
			return fmt.Errorf("block %d not found", cfg.BlockNum)
		}
		header = rawdb.ReadHeader(db, hash, uint64(cfg.BlockNum))
	}

	fmt.Printf("Extracting state at block %d (hash: %s)\n", header.Number.Uint64(), header.Hash().Hex())

	// Create state database
	triedb := trie.NewDatabase(db, nil)
	
	// Open state trie
	sdb := state.NewDatabaseWithNodeDB(db, triedb)
	statedb, err := state.New(header.Root, sdb, nil)
	if err != nil {
		// Try with snapshot
		fmt.Println("Direct state access failed, trying with snapshot...")
		
		snap := snapshot.New(snapshot.Config{
			CacheSize:  256,
			Recovery:   false,
			NoBuild:    true,
			AsyncBuild: false,
		}, db, triedb, header.Root)
		
		if snap == nil {
			return fmt.Errorf("failed to create snapshot")
		}
		
		// Extract using snapshot
		return extractFromSnapshot(db, snap, header, minBalance)
	}

	// Extract accounts
	accounts := []AccountData{}
	totalBalance := big.NewInt(0)
	contractCount := 0

	// Iterate through all accounts
	fmt.Println("Iterating through accounts...")
	it := statedb.DumpIterator(nil, nil)
	for it.Next() {
		var acc state.DumpAccount
		if err := json.Unmarshal(it.Value, &acc); err != nil {
			continue
		}

		balance, _ := new(big.Int).SetString(acc.Balance, 10)
		if balance.Cmp(minBalance) < 0 {
			continue
		}

		isContract := acc.Code != ""
		if isContract {
			contractCount++
		}

		accounts = append(accounts, AccountData{
			Address:    acc.Address.Hex(),
			Balance:    balance.String(),
			Nonce:      acc.Nonce,
			CodeHash:   acc.CodeHash,
			IsContract: isContract,
		})

		totalBalance.Add(totalBalance, balance)
	}

	// Sort by balance
	sort.Slice(accounts, func(i, j int) bool {
		bi, _ := new(big.Int).SetString(accounts[i].Balance, 10)
		bj, _ := new(big.Int).SetString(accounts[j].Balance, 10)
		return bi.Cmp(bj) > 0
	})

	fmt.Printf("Found %d accounts (including %d contracts)\n", len(accounts), contractCount)
	fmt.Printf("Total balance: %s wei\n", totalBalance.String())

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save results
	timestamp := time.Now().Format("20060102-150405")
	
	if cfg.Format == "json" || cfg.Format == "both" {
		jsonPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("cchain-state-%d-%s.json", header.Number.Uint64(), timestamp))
		if err := saveJSON(jsonPath, map[string]interface{}{
			"blockNumber":   header.Number.Uint64(),
			"blockHash":     header.Hash().Hex(),
			"stateRoot":     header.Root.Hex(),
			"timestamp":     header.Time,
			"totalAccounts": len(accounts),
			"totalBalance":  totalBalance.String(),
			"contracts":     contractCount,
			"accounts":      accounts,
		}); err != nil {
			return fmt.Errorf("failed to save JSON: %w", err)
		}
		fmt.Printf("Saved JSON to: %s\n", jsonPath)
	}

	if cfg.Format == "csv" || cfg.Format == "both" {
		csvPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("cchain-state-%d-%s.csv", header.Number.Uint64(), timestamp))
		if err := saveCSV(csvPath, accounts); err != nil {
			return fmt.Errorf("failed to save CSV: %w", err)
		}
		fmt.Printf("Saved CSV to: %s\n", csvPath)
	}

	// Save summary
	summaryPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("cchain-summary-%d-%s.json", header.Number.Uint64(), timestamp))
	if err := saveJSON(summaryPath, map[string]interface{}{
		"blockNumber":   header.Number.Uint64(),
		"blockHash":     header.Hash().Hex(),
		"timestamp":     time.Now().Format(time.RFC3339),
		"totalAccounts": len(accounts),
		"totalBalance":  totalBalance.String(),
		"contracts":     contractCount,
		"topHolders":    getTopHolders(accounts, 20),
	}); err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	return nil
}

func openDatabase(path string) (ethdb.Database, error) {
	// Check if it's a Pebble database
	if _, err := os.Stat(filepath.Join(path, "CURRENT")); err == nil {
		// Open Pebble database
		db, err := pebble.Open(path, &pebble.Options{
			ReadOnly: true,
		})
		if err != nil {
			return nil, err
		}
		return &pebbleDB{db: db}, nil
	}

	// Try LevelDB
	return rawdb.NewLevelDBDatabase(path, 256, 16, "", true)
}

// pebbleDB implements ethdb.Database for Pebble
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
	data, closer, err := p.db.Get(key)
	if err != nil {
		return nil, err
	}
	defer closer.Close()
	return append([]byte{}, data...), nil
}

func (p *pebbleDB) Put(key []byte, value []byte) error {
	return p.db.Set(key, value, pebble.Sync)
}

func (p *pebbleDB) Delete(key []byte) error {
	return p.db.Delete(key, pebble.Sync)
}

func (p *pebbleDB) NewBatch() ethdb.Batch {
	return &pebbleBatch{batch: p.db.NewBatch()}
}

func (p *pebbleDB) NewBatchWithSize(size int) ethdb.Batch {
	return &pebbleBatch{batch: p.db.NewBatch()}
}

func (p *pebbleDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	iter, _ := p.db.NewIter(&pebble.IterOptions{
		LowerBound: append(prefix, start...),
		UpperBound: incrementBytes(prefix),
	})
	return &pebbleIterator{iter: iter, prefix: prefix}
}

func (p *pebbleDB) NewSnapshot() (ethdb.Snapshot, error) {
	return p, nil
}

func (p *pebbleDB) Stat(property string) (string, error) {
	return "", nil
}

func (p *pebbleDB) Compact(start []byte, limit []byte) error {
	return nil
}

func (p *pebbleDB) Close() error {
	return p.db.Close()
}

// pebbleBatch implements ethdb.Batch
type pebbleBatch struct {
	batch *pebble.Batch
}

func (b *pebbleBatch) Put(key []byte, value []byte) error {
	return b.batch.Set(key, value, nil)
}

func (b *pebbleBatch) Delete(key []byte) error {
	return b.batch.Delete(key, nil)
}

func (b *pebbleBatch) ValueSize() int {
	return int(b.batch.Len())
}

func (b *pebbleBatch) Write() error {
	return b.batch.Commit(pebble.Sync)
}

func (b *pebbleBatch) Reset() {
	b.batch.Reset()
}

func (b *pebbleBatch) Replay(w ethdb.KeyValueWriter) error {
	return nil
}

// pebbleIterator implements ethdb.Iterator
type pebbleIterator struct {
	iter   *pebble.Iterator
	prefix []byte
}

func (it *pebbleIterator) Next() bool {
	return it.iter.Next()
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

func incrementBytes(b []byte) []byte {
	result := make([]byte, len(b))
	copy(result, b)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 255 {
			result[i]++
			break
		}
		result[i] = 0
		if i == 0 {
			result = append([]byte{1}, result...)
		}
	}
	return result
}

func extractFromSnapshot(db ethdb.Database, snap snapshot.Snapshot, header *types.Header, minBalance *big.Int) error {
	// This would require implementing snapshot iteration
	// For now, return an error suggesting to use a different approach
	return fmt.Errorf("snapshot extraction not fully implemented, try running on a node with state available")
}

func saveJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, jsonData, 0644)
}

func saveCSV(path string, accounts []AccountData) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"address", "balance_wei", "nonce", "is_contract", "code_hash"}); err != nil {
		return err
	}

	// Write data
	for _, acc := range accounts {
		record := []string{
			acc.Address,
			acc.Balance,
			fmt.Sprintf("%d", acc.Nonce),
			fmt.Sprintf("%t", acc.IsContract),
			acc.CodeHash,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

func getTopHolders(accounts []AccountData, limit int) []map[string]string {
	holders := []map[string]string{}
	
	for i := 0; i < len(accounts) && i < limit; i++ {
		balance, _ := new(big.Int).SetString(accounts[i].Balance, 10)
		
		// Convert to human readable
		eth := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			new(big.Float).SetInt(big.NewInt(1e18)),
		)
		
		holders = append(holders, map[string]string{
			"rank":       fmt.Sprintf("%d", i+1),
			"address":    accounts[i].Address,
			"balance":    accounts[i].Balance,
			"balanceETH": fmt.Sprintf("%.6f", eth),
			"isContract": fmt.Sprintf("%t", accounts[i].IsContract),
		})
	}
	
	return holders
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}