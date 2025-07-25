package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/node/ids"
)

var chainIDs = map[string]string{
	"96369": "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1",
	"96368": "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1", // Same chain ID for testnet
}

// Map old blockchain IDs to new ones for migration
var blockchainIDMap = map[string]string{
	// Old LUX mainnet subnet blockchain ID -> New C-Chain blockchain ID
	"dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ": "2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC",
}

func main() {
	srcPath := flag.String("src", "", "Path to original Pebble DB")
	dstPath := flag.String("dst", "", "Path to new Pebble DB (must not exist)")
	network := flag.String("network", "96369", "Network ID (96369 for mainnet, 96368 for testnet)")
	includeState := flag.Bool("state", false, "Include state data (accounts, storage)")
	migrateBlockchainID := flag.Bool("migrate-id", false, "Migrate from old blockchain ID to new C-Chain ID")
	oldBlockchainID := flag.String("old-id", "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ", "Old blockchain ID to migrate from")
	newBlockchainID := flag.String("new-id", "2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC", "New blockchain ID to migrate to")

	flag.Parse()
	if *srcPath == "" || *dstPath == "" {
		log.Fatal("usage: namespace -src /old/db -dst /new/db [-network 96369] [-state]")
	}

	var chainBytes []byte
	var migrateToChainBytes []byte
	
	if *migrateBlockchainID {
		// Parse old and new blockchain IDs
		oldID, err := ids.FromString(*oldBlockchainID)
		if err != nil {
			log.Fatalf("Invalid old blockchain ID: %v", err)
		}
		newID, err := ids.FromString(*newBlockchainID)
		if err != nil {
			log.Fatalf("Invalid new blockchain ID: %v", err)
		}
		chainBytes = oldID[:]
		migrateToChainBytes = newID[:]
		
		log.Printf("Blockchain ID Migration Mode:")
		log.Printf("  From: %s", *oldBlockchainID)
		log.Printf("  To:   %s", *newBlockchainID)
	} else {
		chainHex, ok := chainIDs[*network]
		if !ok {
			log.Fatalf("Unknown network: %s", *network)
		}

		var err error
		chainBytes, err = hex.DecodeString(chainHex)
		if err != nil {
			log.Fatalf("decode chain hex: %v", err)
		}
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
	if *includeState {
		validSuffixes[0x26] = "accounts" // account data
		validSuffixes[0xa3] = "storage"  // contract storage
		validSuffixes[0x6f] = "objects"  // state objects
		validSuffixes[0x73] = "state"    // state trie
		validSuffixes[0x63] = "code"     // contract code
	}

	log.Printf("Network %s Selective Migration", *network)
	log.Printf("Include state: %v", *includeState)
	log.Printf("Processing blockchain data prefixes...")

	// Open source DB
	src, err := pebble.Open(*srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("open source: %v", err)
	}
	defer src.Close()

	// Create destination DB
	dst, err := pebble.Open(*dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("open dest: %v", err)
	}
	defer dst.Close()

	// Track counts per suffix
	suffixCounts := make(map[byte]int)

	// Copy only keys with valid prefixes
	iter, err := src.NewIter(nil)
	if err != nil {
		log.Fatalf("create iterator: %v", err)
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
		"lastBlock",
		"vm_lastAccepted",
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		// Check for metadata keys without namespace
		isMetadata := false
		for _, mk := range metadataKeys {
			if string(key) == mk {
				isMetadata = true
				break
			}
		}

		if isMetadata {
			// Copy metadata key as-is
			if err := batch.Set(key, iter.Value(), nil); err != nil {
				log.Fatalf("set metadata key: %v", err)
			}
			count++
			log.Printf("Found metadata key: %s", key)
		} else if len(key) >= 33 && bytes.HasPrefix(key, chainBytes) {
			// Check if key has our chain ID prefix + suffix
			suffix := key[32]

			// Only process valid suffixes
			if _, valid := validSuffixes[suffix]; valid {
				var newKey []byte
				value := iter.Value()
				
				if *migrateBlockchainID {
					// Replace old blockchain ID with new one
					newKey = make([]byte, len(key))
					copy(newKey, migrateToChainBytes)  // New blockchain ID
					copy(newKey[32:], key[32:])        // Rest of the key
					
					// Also replace blockchain ID in the value if it contains it
					if bytes.Contains(value, chainBytes) {
						value = bytes.ReplaceAll(value, chainBytes, migrateToChainBytes)
					}
				} else {
					// Strip the 33-byte prefix (32 bytes chain ID + 1 byte suffix)
					newKey = key[33:]
				}

				if len(newKey) > 0 {
					if err := batch.Set(newKey, value, nil); err != nil {
						log.Fatalf("set key: %v", err)
					}

					count++
					suffixCounts[suffix]++

					if count%100000 == 0 {
						elapsed := time.Since(start).Seconds()
						rate := float64(count) / elapsed
						log.Printf("Processed %d keys (%.0f keys/sec)", count, rate)

						if err := batch.Commit(nil); err != nil {
							log.Fatalf("commit batch: %v", err)
						}
						batch = dst.NewBatch()
					}
				}
			} else {
				skipped++
			}
		} else {
			skipped++
		}
	}

	// Final batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("final commit: %v", err)
	}

	elapsed := time.Since(start).Seconds()
	rate := float64(count) / elapsed

	log.Printf("\n✅ Complete!")
	log.Printf("Total keys copied: %d in %.1f seconds (%.0f keys/sec)", count, elapsed, rate)
	log.Printf("Keys skipped: %d", skipped)

	log.Printf("\nKeys per type:")
	for suffix, name := range validSuffixes {
		if count := suffixCounts[suffix]; count > 0 {
			log.Printf("  %s (0x%02x): %d keys", name, suffix, count)
		}
	}

	// Try to find the highest block number
	log.Printf("\nAnalyzing block data...")
	analyzeBlocks(dst)
}

func analyzeBlocks(db *pebble.DB) {
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Printf("Failed to create iterator for analysis: %v", err)
		return
	}
	defer iter.Close()

	headerCount := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 0 {
			continue
		}

		keyType := key[0]

		// Count headers
		if keyType == 'h' {
			headerCount++
		}

		// Check for metadata keys
		if string(key) == "LastAccepted" || string(key) == "lastAccepted" {
			log.Printf("Found LastAccepted: %x", iter.Value())
		}
	}

	log.Printf("Total headers found: %d (suggests ~%d blocks)", headerCount, headerCount)
}
