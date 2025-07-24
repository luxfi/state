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

func main() {
	srcPath := flag.String("src", "", "Path to original Pebble DB")
	dstPath := flag.String("dst", "", "Path to new Pebble DB (must not exist)")
	oldChainID := flag.String("old-chain-id", "", "Old blockchain ID (e.g., dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ)")
	newChainID := flag.String("new-chain-id", "", "New blockchain ID (e.g., 2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC)")
	includeState := flag.Bool("state", true, "Include state data (accounts, storage)")

	flag.Parse()
	if *srcPath == "" || *dstPath == "" || *oldChainID == "" || *newChainID == "" {
		log.Fatal("usage: migrate-chain -src /old/db -dst /new/db -old-chain-id <id> -new-chain-id <id> [-state]")
	}

	// Parse blockchain IDs
	oldID, err := ids.FromString(*oldChainID)
	if err != nil {
		log.Fatalf("Invalid old chain ID: %v", err)
	}
	
	newID, err := ids.FromString(*newChainID)
	if err != nil {
		log.Fatalf("Invalid new chain ID: %v", err)
	}

	oldChainBytes := oldID[:]
	newChainBytes := newID[:]

	log.Printf("Migration Configuration:")
	log.Printf("  Old Chain ID: %s (%x)", *oldChainID, oldChainBytes)
	log.Printf("  New Chain ID: %s (%x)", *newChainID, newChainBytes)
	log.Printf("  Include state: %v", *includeState)

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
		0x64: "difficulty",   // total difficulty
		0x44: "Difficulty",   // block difficulty
	}

	// Optionally include state data
	if *includeState {
		validSuffixes[0x26] = "accounts" // account data
		validSuffixes[0xa3] = "storage"  // contract storage
		validSuffixes[0x6f] = "objects"  // state objects
		validSuffixes[0x73] = "state"    // state trie
		validSuffixes[0x63] = "code"     // contract code
		validSuffixes[0x41] = "Account"  // account trie
		validSuffixes[0x53] = "Storage"  // storage trie
	}

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

	// Copy keys, replacing old chain ID with new chain ID
	iter, err := src.NewIter(nil)
	if err != nil {
		log.Fatalf("create iterator: %v", err)
	}
	defer iter.Close()

	batch := dst.NewBatch()
	count := 0
	skipped := 0
	start := time.Now()

	// Metadata keys that should be copied as-is
	metadataKeys := []string{
		"LastAccepted",
		"last_accepted_key",
		"lastAccepted",
		"lastFinalized",
		"lastBlock",
		"vm_lastAccepted",
		"TrieDB.scheme",
		"snapshotDisabled",
		"snapshotRecovery",
		"snapshotJournal",
		"snapshotGenerator",
		"snapshotRoot",
		"snapshotBlock",
	}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

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
			if err := batch.Set(key, value, nil); err != nil {
				log.Fatalf("set metadata key: %v", err)
			}
			count++
			log.Printf("Found metadata key: %s", key)
		} else if len(key) >= 33 && bytes.HasPrefix(key, oldChainBytes) {
			// Key has old chain ID prefix
			suffix := key[32]

			// Only process valid suffixes
			if _, valid := validSuffixes[suffix]; valid {
				// Create new key with new chain ID prefix
				newKey := make([]byte, len(key))
				copy(newKey, newChainBytes)        // New chain ID (32 bytes)
				copy(newKey[32:], key[32:])        // Suffix and rest of key

				// Also check if the value contains the old chain ID and replace it
				newValue := bytes.ReplaceAll(value, oldChainBytes, newChainBytes)

				if err := batch.Set(newKey, newValue, nil); err != nil {
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
			} else {
				skipped++
			}
		} else if len(key) > 0 {
			// Check if this is a key without chain ID prefix that we should keep
			firstByte := key[0]
			keepKey := false
			
			// Common prefixes for blockchain data
			switch firstByte {
			case 'H', 'h', 'b', 'B', 'r', 'R', 'l', 'L', 't', 'T', 'n', 'd':
				keepKey = true
			}

			if keepKey {
				// Check if the value contains the old chain ID and replace it
				newValue := bytes.ReplaceAll(value, oldChainBytes, newChainBytes)
				
				if err := batch.Set(key, newValue, nil); err != nil {
					log.Fatalf("set unprefixed key: %v", err)
				}
				count++
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

	log.Printf("\n✅ Migration Complete!")
	log.Printf("Total keys migrated: %d in %.1f seconds (%.0f keys/sec)", count, elapsed, rate)
	log.Printf("Keys skipped: %d", skipped)

	log.Printf("\nKeys per type:")
	for suffix, name := range validSuffixes {
		if count := suffixCounts[suffix]; count > 0 {
			log.Printf("  %s (0x%02x): %d keys", name, suffix, count)
		}
	}

	// Analyze the migrated data
	log.Printf("\nAnalyzing migrated blockchain...")
	analyzeBlocks(dst, newChainBytes)
}

func analyzeBlocks(db *pebble.DB, chainIDBytes []byte) {
	iter, err := db.NewIter(nil)
	if err != nil {
		log.Printf("Failed to create iterator for analysis: %v", err)
		return
	}
	defer iter.Close()

	headerCount := 0
	var maxBlockNum uint64
	hasLastAccepted := false

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		
		// Check for metadata keys
		if string(key) == "LastAccepted" || string(key) == "lastAccepted" {
			hasLastAccepted = true
			log.Printf("Found LastAccepted: %x", iter.Value())
		}

		// Count headers with new chain ID prefix
		if len(key) >= 33 && bytes.HasPrefix(key, chainIDBytes) && key[32] == 0x68 {
			headerCount++
		}
	}

	log.Printf("Migration Analysis:")
	log.Printf("  Headers found: %d", headerCount)
	log.Printf("  Has LastAccepted: %v", hasLastAccepted)
	if headerCount > 0 {
		log.Printf("  Estimated blocks: ~%d", headerCount)
	}
}