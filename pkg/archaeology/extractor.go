package archaeology

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// DatabaseType represents the type of database
type DatabaseType int

const (
	DBTypeLevelDB DatabaseType = iota
	DBTypePebbleDB
)

// Extractor handles extraction of blockchain data from various database formats
type Extractor struct {
	config  ChainConfig
	options ExtractionOptions
	logger  *log.Logger
}

// NewExtractor creates a new extractor for a specific chain
func NewExtractor(config ChainConfig, options ExtractionOptions) *Extractor {
	return &Extractor{
		config:  config,
		options: options,
		logger:  log.Default(),
	}
}

// DetectDatabaseType attempts to detect the database type
func DetectDatabaseType(path string) (DatabaseType, error) {
	// Try opening as PebbleDB first
	db, err := pebble.Open(path, &pebble.Options{ReadOnly: true})
	if err == nil {
		db.Close()
		return DBTypePebbleDB, nil
	}

	// Try LevelDB
	ldb, err := leveldb.OpenFile(path, &opt.Options{ReadOnly: true})
	if err == nil {
		ldb.Close()
		return DBTypeLevelDB, nil
	}

	return -1, fmt.Errorf("could not detect database type: %v", err)
}

// Extract performs the extraction based on the configured options
func (e *Extractor) Extract(srcPath, dstPath string) (*ExtractionResult, error) {
	dbType, err := DetectDatabaseType(srcPath)
	if err != nil {
		return nil, fmt.Errorf("detect database type: %w", err)
	}

	e.logger.Printf("Detected database type: %s", e.dbTypeName(dbType))

	switch dbType {
	case DBTypePebbleDB:
		return e.extractPebbleDB(srcPath, dstPath)
	case DBTypeLevelDB:
		return e.extractLevelDB(srcPath, dstPath)
	default:
		return nil, fmt.Errorf("unsupported database type")
	}
}

// extractPebbleDB handles extraction from PebbleDB databases
func (e *Extractor) extractPebbleDB(srcPath, dstPath string) (*ExtractionResult, error) {
	start := time.Now()
	result := &ExtractionResult{
		KeysByType:     make(map[KeyType]int),
		ExtractedState: make(map[string]AccountState),
	}

	// Open source database
	src, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	// Create destination database
	dst, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("open destination: %w", err)
	}
	defer dst.Close()

	// Determine which key types to extract
	validTypes := e.getValidKeyTypes()

	// Parse chain ID if provided
	var chainBytes []byte
	if e.config.ChainIDHex != "" {
		chainBytes, err = hex.DecodeString(e.config.ChainIDHex)
		if err != nil {
			return nil, fmt.Errorf("decode chain ID: %w", err)
		}
	}

	// Iterate through source database
	iter, err := src.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("create iterator: %w", err)
	}
	defer iter.Close()

	batch := dst.NewBatch()
	count := 0

	// Metadata keys to look for
	metadataKeys := []string{
		"LastAccepted", "lastAccepted", "lastFinalized",
		"lastBlock", "vm_lastAccepted", "last_accepted_key",
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Check for metadata keys
		keyStr := string(key)
		isMetadata := false
		for _, mk := range metadataKeys {
			if keyStr == mk {
				isMetadata = true
				e.logger.Printf("Found metadata: %s = %x", keyStr, value)
				break
			}
		}

		if isMetadata && e.options.IncludeMetadata {
			if err := batch.Set(key, value, nil); err != nil {
				return nil, fmt.Errorf("set metadata key: %w", err)
			}
			count++
			result.TotalKeys++
		} else if len(chainBytes) > 0 && len(key) >= 33 && bytes.HasPrefix(key, chainBytes) {
			// Namespaced key - extract based on suffix
			suffix := KeyType(key[32])

			if _, valid := validTypes[suffix]; valid {
				newKey := key[33:]
				if len(newKey) > 0 {
					if err := batch.Set(newKey, value, nil); err != nil {
						return nil, fmt.Errorf("set key: %w", err)
					}

					count++
					result.TotalKeys++
					result.KeysByType[suffix]++

					// Process specific key types
					e.processKey(suffix, newKey, value, result)

					if count%100000 == 0 {
						e.logger.Printf("Processed %d keys", count)
						if err := batch.Commit(nil); err != nil {
							return nil, fmt.Errorf("commit batch: %w", err)
						}
						batch = dst.NewBatch()
					}
				}
			}
		} else if len(chainBytes) == 0 && len(key) > 0 {
			// No namespace - process based on first byte
			keyType := KeyType(key[0])

			if _, valid := validTypes[keyType]; valid {
				if err := batch.Set(key, value, nil); err != nil {
					return nil, fmt.Errorf("set key: %w", err)
				}

				count++
				result.TotalKeys++
				result.KeysByType[keyType]++

				e.processKey(keyType, key[1:], value, result)

				if count%100000 == 0 {
					e.logger.Printf("Processed %d keys", count)
					if err := batch.Commit(nil); err != nil {
						return nil, fmt.Errorf("commit batch: %w", err)
					}
					batch = dst.NewBatch()
				}
			}
		}
	}

	// Final commit
	if err := batch.Commit(nil); err != nil {
		return nil, fmt.Errorf("final commit: %w", err)
	}

	result.Duration = time.Since(start).Seconds()

	e.logger.Printf("\nExtraction complete!")
	e.logger.Printf("Total keys: %d", result.TotalKeys)
	e.logger.Printf("Duration: %.1f seconds", result.Duration)
	e.logger.Printf("\nKeys by type:")
	for keyType, count := range result.KeysByType {
		e.logger.Printf("  %s: %d", GetKeyTypeName(byte(keyType)), count)
	}

	if result.HighestBlock > 0 {
		e.logger.Printf("\nBlock range: %d - %d", result.LowestBlock, result.HighestBlock)
	}

	return result, nil
}

// extractLevelDB handles extraction from LevelDB databases
func (e *Extractor) extractLevelDB(srcPath, dstPath string) (*ExtractionResult, error) {
	// Similar to extractPebbleDB but using LevelDB API
	// This is a placeholder - full implementation would follow similar pattern
	return nil, fmt.Errorf("LevelDB extraction not yet implemented")
}

// getValidKeyTypes returns which key types to extract based on options
func (e *Extractor) getValidKeyTypes() map[KeyType]bool {
	types := make(map[KeyType]bool)

	if e.options.IncludeHeaders {
		types[KeyTypeHeader] = true
		types[KeyTypeHashToNumber] = true
		types[KeyTypeNumberToHash] = true
	}

	if e.options.IncludeBodies {
		types[KeyTypeBody] = true
		types[KeyTypeBodyAlt] = true
	}

	if e.options.IncludeReceipts {
		types[KeyTypeReceipt] = true
	}

	if e.options.IncludeTransactions {
		types[KeyTypeTransaction] = true
		types[KeyTypeTxLookup] = true
	}

	if e.options.IncludeState {
		types[KeyTypeAccount] = true
		types[KeyTypeStorage] = true
		types[KeyTypeCode] = true
		types[KeyTypeState] = true
		types[KeyTypeStateObject] = true
	}

	if e.options.IncludeMetadata {
		types[KeyTypeMetadata] = true
		types[KeyTypeLastValues] = true
	}

	return types
}

// processKey processes specific key types to extract additional information
func (e *Extractor) processKey(keyType KeyType, key, value []byte, result *ExtractionResult) {
	switch keyType {
	case KeyTypeHashToNumber:
		// Extract block number from hash->number mapping
		if len(value) >= 8 {
			blockNum, _ := ParseBlockNumber(value)
			if result.HighestBlock == 0 || blockNum > result.HighestBlock {
				result.HighestBlock = blockNum
			}
			if result.LowestBlock == 0 || blockNum < result.LowestBlock {
				result.LowestBlock = blockNum
			}
		}

	case KeyTypeAccount:
		// Extract account balance if within target addresses
		if e.options.IncludeState && len(e.options.TargetAddresses) > 0 {
			// Implementation would decode account data
			// This is a placeholder
		}
	}
}

func (e *Extractor) dbTypeName(dbType DatabaseType) string {
	switch dbType {
	case DBTypeLevelDB:
		return "LevelDB"
	case DBTypePebbleDB:
		return "PebbleDB"
	default:
		return "Unknown"
	}
}
