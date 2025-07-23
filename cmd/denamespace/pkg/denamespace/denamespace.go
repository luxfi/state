package denamespace

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
)

// Options for the denamespace operation
type Options struct {
	Source      string
	Destination string
	NetworkID   uint64
	State       bool
	Limit       int
}

var chainIDs = map[uint64]string{
	96369:  "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1",
	96368:  "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1",
	200200: "6078e156c49594d6f65dc1f49a2d2a96f2a59e7c9e8f7e5c4f3a2b1c0d9e8f7a", // ZOO
	36911:  "5f4e3d2c1b0a9f8e7d6c5b4a3f2e1d0c9b8a7f6e5d4c3b2a1f0e9d8c7b6a5f4e", // SPC
}

// Extract performs the denamespace operation
func Extract(opts Options) error {
	chainHex, ok := chainIDs[opts.NetworkID]
	if !ok {
		return fmt.Errorf("unknown network ID: %d", opts.NetworkID)
	}

	chainBytes, err := hex.DecodeString(chainHex)
	if err != nil {
		return fmt.Errorf("decode chain hex: %v", err)
	}

	// Define the suffixes we want to copy
	validSuffixes := map[byte]string{
		0x68: "headers",      // block headers
		0x6c: "last values",  // last accepted, etc
		0x48: "Headers",      // hash->number mappings
		0x72: "receipts",     // transaction receipts
		0x62: "bodies",       // block bodies
		0x42: "Bodies",       // alternative bodies
		0x6e: "number->hash", // block number to hash
		0x74: "transactions", // transaction data
		0xfd: "metadata",     // chain metadata
	}

	// Optionally include state data
	if opts.State {
		validSuffixes[0x26] = "accounts" // account data
		validSuffixes[0xa3] = "storage"  // contract storage
		validSuffixes[0x6f] = "objects"  // state objects
		validSuffixes[0x73] = "state"    // state trie
		validSuffixes[0x63] = "code"     // contract code
	}

	log.Printf("Network %d Selective Migration", opts.NetworkID)
	log.Printf("Chain hex: %s", chainHex)
	log.Printf("Include state: %v", opts.State)
	log.Printf("Processing blockchain data prefixes...")

	// Open source DB
	src, err := pebble.Open(opts.Source, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("open source: %v", err)
	}
	defer src.Close()

	// Create destination DB
	dst, err := pebble.Open(opts.Destination, &pebble.Options{})
	if err != nil {
		return fmt.Errorf("open dest: %v", err)
	}
	defer dst.Close()

	// Track counts per suffix
	suffixCounts := make(map[byte]int)

	// Copy only keys with valid prefixes
	iter, err := src.NewIter(nil)
	if err != nil {
		return fmt.Errorf("create iterator: %v", err)
	}
	defer iter.Close()

	batch := dst.NewBatch()
	count := 0
	skipped := 0
	start := time.Now()

	// Also copy keys without namespace prefix (metadata keys)
	metadataKeys := []string{
		"LastAccepted",
		"last_accepted_key",
		"lastAccepted",
		"lastFinalized",
		"LastFinalizedKey",
		"vm_state",
		"chain_state",
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		// Check limit if specified
		if opts.Limit > 0 && count >= opts.Limit {
			break
		}

		// Check if it's a metadata key (no namespace)
		isMetadata := false
		for _, mk := range metadataKeys {
			if bytes.Equal(key, []byte(mk)) || bytes.HasPrefix(key, []byte(mk)) {
				isMetadata = true
				break
			}
		}

		if isMetadata {
			if err := batch.Set(key, iter.Value(), nil); err != nil {
				return fmt.Errorf("set metadata key: %v", err)
			}
			count++
			if count%10000 == 0 {
				log.Printf("Progress: %d keys copied", count)
			}
			continue
		}

		// Check if key has namespace prefix
		if len(key) < 33 {
			continue
		}

		// Check if it's our chain's namespace
		if !bytes.Equal(key[:32], chainBytes) {
			skipped++
			continue
		}

		// Check if it has a valid suffix
		suffix := key[32]
		if _, ok := validSuffixes[suffix]; !ok {
			skipped++
			continue
		}

		// Remove namespace prefix and copy
		newKey := key[33:]
		if err := batch.Set(newKey, iter.Value(), nil); err != nil {
			return fmt.Errorf("set key: %v", err)
		}

		suffixCounts[suffix]++
		count++

		if count%10000 == 0 {
			log.Printf("Progress: %d keys copied", count)
		}

		// Commit batch periodically
		if batch.Len() >= 1000 {
			if err := batch.Commit(pebble.Sync); err != nil {
				return fmt.Errorf("commit batch: %v", err)
			}
			batch = dst.NewBatch()
		}
	}

	// Final batch commit
	if batch.Len() > 0 {
		if err := batch.Commit(pebble.Sync); err != nil {
			return fmt.Errorf("final commit: %v", err)
		}
	}

	elapsed := time.Since(start)
	log.Printf("\nMigration complete!")
	log.Printf("Total keys copied: %d", count)
	log.Printf("Total keys skipped: %d", skipped)
	log.Printf("Time elapsed: %v", elapsed)
	log.Printf("\nKeys copied per suffix:")
	for suffix, cnt := range suffixCounts {
		if name, ok := validSuffixes[suffix]; ok {
			log.Printf("  0x%02x (%s): %d", suffix, name, cnt)
		}
	}

	return nil
}
