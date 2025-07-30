package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: analyze-keys-detailed <db-path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]

	// Open database
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("=== Analyzing %s ===\n\n", dbPath)

	// Look at first 20 keys with different patterns
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	count := 0
	accountsFound := 0

	for iter.First(); iter.Valid() && count < 50; iter.Next() {
		key := iter.Key()
		val := iter.Value()

		fmt.Printf("Key %d:\n", count+1)
		fmt.Printf("  Raw key: %x\n", key)
		fmt.Printf("  Key len: %d\n", len(key))
		fmt.Printf("  Val len: %d\n", len(val))

		// Check if it starts with "evm"
		if len(key) >= 3 && string(key[:3]) == "evm" {
			fmt.Printf("  Has 'evm' prefix\n")
			keyWithoutPrefix := key[3:]
			fmt.Printf("  Key without prefix: %x (len=%d)\n", keyWithoutPrefix, len(keyWithoutPrefix))

			// Try to decode value as account
			var acc types.StateAccount
			if err := rlp.DecodeBytes(val, &acc); err == nil && acc.Balance != nil {
				fmt.Printf("  ✓ Valid account! Balance: %s, Nonce: %d\n", acc.Balance.String(), acc.Nonce)
				accountsFound++

				// Try to extract address from key
				if len(keyWithoutPrefix) >= 32 {
					addr := common.BytesToAddress(keyWithoutPrefix[:20])
					fmt.Printf("  Possible address: %s\n", addr.Hex())
				}
			}
		}

		fmt.Println()
		count++
	}

	fmt.Printf("\nAccounts found in first %d keys: %d\n", count, accountsFound)

	// Now let's specifically look for account patterns
	fmt.Println("\n=== Looking for account patterns ===")

	// Count different key lengths
	lengthCounts := make(map[int]int)
	iter2, _ := db.NewIter(&pebble.IterOptions{})

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		lengthCounts[len(key)]++

		// Try to find accounts
		if len(key) == 43 && string(key[:3]) == "evm" { // evm prefix + 40 bytes
			val := iter2.Value()
			var acc types.StateAccount
			if err := rlp.DecodeBytes(val, &acc); err == nil && acc.Balance != nil && acc.Balance.Sign() > 0 {
				accountsFound++
				if accountsFound <= 5 {
					addr := common.BytesToAddress(key[11:31]) // Skip evm + 8 bytes
					fmt.Printf("Account %s: balance=%s\n", addr.Hex(), acc.Balance.String())
				}
			}
		}
	}
	iter2.Close()

	fmt.Println("\nKey length distribution:")
	for length, count := range lengthCounts {
		if count > 100 {
			fmt.Printf("  Length %d: %d keys\n", length, count)
		}
	}

	fmt.Printf("\nTotal accounts with balance > 0: %d\n", accountsFound)
}
