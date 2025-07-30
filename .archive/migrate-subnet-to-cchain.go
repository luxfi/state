package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// C-Chain blockchain ID
	cchainID = "X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: migrate-subnet-to-cchain <source-subnet-db> <dest-network-db>")
		fmt.Println("Example: migrate-subnet-to-cchain output/mainnet/C/chaindata-namespaced runtime/migrated-cchain/db/network-96369/v1.4.5")
		os.Exit(1)
	}

	sourceDB := os.Args[1]
	destDB := os.Args[2]

	fmt.Println("=== Migrating Subnet Data to C-Chain Format ===")
	fmt.Printf("Source: %s\n", sourceDB)
	fmt.Printf("Destination: %s\n", destDB)
	fmt.Println()

	// Open source database (subnet data)
	src, err := pebble.Open(sourceDB, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer src.Close()

	// Open destination database (network db)
	dst, err := pebble.Open(destDB, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open destination database: %v", err)
	}
	defer dst.Close()

	// Create blockchain ID prefix
	blockchainIDBytes, err := hex.DecodeString(cchainID)
	if err != nil {
		log.Fatalf("Failed to decode blockchain ID: %v", err)
	}

	// VM prefix
	vmPrefix := []byte("vm")

	// Create the full prefix for C-Chain data
	// Format: blockchainID + "vm" + data

	fmt.Println("Analyzing source database...")

	// First pass - analyze what we have
	iter, _ := src.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	stats := struct {
		headers   int
		bodies    int
		receipts  int
		td        int
		canonical int
		accounts  int
		storage   int
		code      int
		other     int
	}{}

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		if len(key) > 0 {
			switch key[0] {
			case 'h': // header
				if len(key) == 41 {
					stats.headers++
				}
			case 'b': // body
				if len(key) == 41 {
					stats.bodies++
				}
			case 'r': // receipt
				if len(key) == 41 {
					stats.receipts++
				}
			case 't': // total difficulty
				stats.td++
			case 'H': // head header hash
				stats.canonical++
			case 0x26: // account
				stats.accounts++
			case 0xa3: // storage
				stats.storage++
			case 'c': // code
				stats.code++
			default:
				stats.other++
			}
		}
	}

	fmt.Printf("\nSource database contents:\n")
	fmt.Printf("  Headers: %d\n", stats.headers)
	fmt.Printf("  Bodies: %d\n", stats.bodies)
	fmt.Printf("  Receipts: %d\n", stats.receipts)
	fmt.Printf("  Accounts: %d\n", stats.accounts)
	fmt.Printf("  Storage: %d\n", stats.storage)
	fmt.Printf("  Code: %d\n", stats.code)
	fmt.Printf("  Other: %d\n", stats.other)

	fmt.Println("\nMigrating data to C-Chain format...")

	// Second pass - migrate data
	iter2, _ := src.NewIter(&pebble.IterOptions{})
	defer iter2.Close()

	batch := dst.NewBatch()
	count := 0
	migratedHeaders := 0
	migratedBodies := 0
	migratedState := 0

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		val := iter2.Value()

		// Create the new key with C-Chain prefixes
		// Format: blockchainID + "vm" + original_key
		newKey := make([]byte, 0, len(blockchainIDBytes)+len(vmPrefix)+len(key))
		newKey = append(newKey, blockchainIDBytes...)
		newKey = append(newKey, vmPrefix...)
		newKey = append(newKey, key...)

		// Write to destination with new key
		if err := batch.Set(newKey, val, nil); err != nil {
			log.Printf("Error setting key: %v", err)
		}

		// Track what we're migrating
		if len(key) > 0 {
			switch key[0] {
			case 'h':
				if len(key) == 41 {
					migratedHeaders++
				}
			case 'b':
				if len(key) == 41 {
					migratedBodies++
				}
			case 0x26, 0xa3, 'c':
				migratedState++
			}
		}

		count++

		// Commit batch periodically
		if count%10000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Printf("Error committing batch: %v", err)
			}
			batch = dst.NewBatch()
			fmt.Printf("  Migrated %d keys (headers: %d, bodies: %d, state: %d)...\n",
				count, migratedHeaders, migratedBodies, migratedState)
		}
	}

	// Commit final batch
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Printf("Error committing final batch: %v", err)
	}

	// Also migrate critical keys that C-Chain expects
	fmt.Println("\nMigrating critical C-Chain keys...")

	criticalKeys := []struct {
		key  []byte
		desc string
	}{
		{[]byte("LastHeader"), "LastHeader"},
		{[]byte("LastBlock"), "LastBlock"},
		{[]byte("LastFinalized"), "LastFinalized"},
		{[]byte{0x48}, "HeadHeaderHash"},
		{[]byte{0x42}, "HeadBlockHash"},
		{[]byte{0x46}, "HeadFastBlockHash"},
	}

	for _, ck := range criticalKeys {
		// Check if it exists in source
		val, closer, err := src.Get(ck.key)
		if err == nil {
			// Migrate with C-Chain prefix
			newKey := append(blockchainIDBytes, vmPrefix...)
			newKey = append(newKey, ck.key...)

			if err := dst.Set(newKey, val, pebble.Sync); err != nil {
				log.Printf("Error migrating %s: %v", ck.desc, err)
			} else {
				fmt.Printf("  ✓ Migrated %s\n", ck.desc)
			}
			closer.Close()
		}
	}

	fmt.Printf("\n✅ Migration complete!\n")
	fmt.Printf("Total keys migrated: %d\n", count)
	fmt.Printf("  Headers: %d\n", migratedHeaders)
	fmt.Printf("  Bodies: %d\n", migratedBodies)
	fmt.Printf("  State data: %d\n", migratedState)
	fmt.Printf("\nDestination: %s\n", destDB)
	fmt.Printf("\nData is now prefixed with:\n")
	fmt.Printf("  Blockchain ID: %s\n", cchainID)
	fmt.Printf("  VM prefix: vm\n")
}
