// Package db provides common database operations for genesis migration
package db

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
)

// DB wraps a pebble database with common operations
type DB struct {
	*pebble.DB
	path string
}

// Open opens a pebble database at the given path
func Open(path string, opts *pebble.Options) (*DB, error) {
	if opts == nil {
		opts = &pebble.Options{}
	}

	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open database at %s: %w", path, err)
	}

	return &DB{DB: db, path: path}, nil
}

// OpenReadOnly opens a database in read-only mode
func OpenReadOnly(path string) (*DB, error) {
	return Open(path, &pebble.Options{ReadOnly: true})
}

// Path returns the database path
func (db *DB) Path() string {
	return db.path
}

// IteratePrefix iterates over all keys with the given prefix
func (db *DB) IteratePrefix(prefix []byte, fn func(key, value []byte) error) error {
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: keyUpperBound(prefix),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		if err := fn(iter.Key(), iter.Value()); err != nil {
			return err
		}
	}

	return iter.Error()
}

// CountKeys counts the number of keys with the given prefix
func (db *DB) CountKeys(prefix []byte) (int, error) {
	count := 0
	err := db.IteratePrefix(prefix, func(_, _ []byte) error {
		count++
		return nil
	})
	return count, err
}

// CopyTo copies all keys from this database to another
func (db *DB) CopyTo(dst *DB) error {
	iter, err := db.NewIter(nil)
	if err != nil {
		return err
	}
	defer iter.Close()

	batch := dst.NewBatch()
	defer batch.Close()

	const batchSize = 10000
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		if err := batch.Set(iter.Key(), iter.Value(), nil); err != nil {
			return err
		}

		count++
		if count%batchSize == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return err
			}
			batch.Reset()
		}
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return batch.Commit(pebble.Sync)
}

// EnsureDir ensures the parent directory exists
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// Uint64ToBigEndian converts uint64 to big endian bytes
func Uint64ToBigEndian(n uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, n)
	return b
}

// BigEndianToUint64 converts big endian bytes to uint64
func BigEndianToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

// keyUpperBound returns the upper bound for prefix iteration
func keyUpperBound(prefix []byte) []byte {
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] != 0xff {
			end[i]++
			return end
		}
	}
	return nil // no upper bound
}
