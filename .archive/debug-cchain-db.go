package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: debug-cchain-db <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open PebbleDB
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("=== C-Chain Database Debug ===")
	fmt.Printf("Database: %s\n\n", dbPath)

	// Check critical keys that C-Chain looks for
	criticalKeys := []struct {
		name string
		key  []byte
	}{
		// Genesis related
		{"LastHeader", []byte("LastHeader")},
		{"LastBlock", []byte("LastBlock")},
		{"LastFinalized", []byte("LastFinalized")},

		// Canonical hash for block 0
		{"Block 0 canonical", append([]byte{0x68}, append(make([]byte, 8), 0x6e)...)},

		// Head references
		{"HeadHeaderHash", []byte{0x48}},    // 'H'
		{"HeadBlockHash", []byte{0x42}},     // 'B'
		{"HeadFastBlockHash", []byte{0x46}}, // 'F'

		// Schema version
		{"databaseVersion", []byte("databaseVersion")},
		{"LastPivot", []byte("LastPivot")},

		// EVM prefixed keys
		{"evm-LastHeader", append([]byte("evm"), []byte("LastHeader")...)},
		{"evm-LastBlock", append([]byte("evm"), []byte("LastBlock")...)},
	}

	for _, ck := range criticalKeys {
		val, closer, err := db.Get(ck.key)
		if err == nil {
			fmt.Printf("✓ %s: %s", ck.name, hex.EncodeToString(val))
			if len(val) <= 8 {
				fmt.Printf(" (len=%d)", len(val))
			}
			fmt.Println()
			closer.Close()
		} else {
			fmt.Printf("✗ %s: not found\n", ck.name)
		}
	}

	// Check for any keys with "evm" prefix
	fmt.Println("\n=== Keys with 'evm' prefix ===")
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evm"),
		UpperBound: []byte("evn"), // next prefix
	})
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		val := iter.Value()
		fmt.Printf("Key: %s, Value len: %d\n", string(key), len(val))
		count++
	}

	if count == 0 {
		fmt.Println("No 'evm' prefixed keys found")
	}

	// Check total key count
	fmt.Println("\n=== Database Statistics ===")
	iter2, _ := db.NewIter(&pebble.IterOptions{})
	defer iter2.Close()

	totalKeys := 0
	for iter2.First(); iter2.Valid() && totalKeys < 1000000; iter2.Next() {
		totalKeys++
	}

	fmt.Printf("Total keys (up to 1M): %d\n", totalKeys)
}
