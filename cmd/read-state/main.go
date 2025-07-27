package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/rlp"
)

// StateAccount represents the Ethereum state account structure
type StateAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

// Allocation represents an account allocation
type Allocation struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

func main() {
	var (
		dbPath     = flag.String("db", "", "Path to extracted pebbledb directory")
		outputPath = flag.String("output", "", "Output file for allocations (JSON)")
		minBalance = flag.String("min", "1000000000", "Minimum balance to include (in wei, default 1 LUX)")
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
	db, err := pebble.Open(*dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Read accounts
	fmt.Println("Reading accounts from state...")
	allocations := make([]Allocation, 0)
	totalBalance := new(big.Int)
	accountCount := 0

	// Create an iterator for account data
	// Account keys have prefix 0x26 (38 in decimal)
	accountPrefix := []byte{0x26}
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: accountPrefix,
		UpperBound: append(accountPrefix, 0xff),
	})
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Skip if key is not the right length (prefix + 32 bytes for address hash)
		if len(key) != 33 {
			continue
		}

		// The key format is: prefix(1) + address_hash(32)
		addressHash := key[1:]

		// We need to look up the preimage to get the actual address
		// For now, we'll try to decode the account data
		var account StateAccount
		if err := rlp.DecodeBytes(value, &account); err != nil {
			continue
		}

		// Skip if balance is below minimum
		if account.Balance == nil || account.Balance.Cmp(minBal) < 0 {
			continue
		}

		// Try to find the address preimage
		// The preimage key format is: 0x01 + hash
		preimageKey := append([]byte{0x01}, addressHash...)
		preimageValue, closer, err := db.Get(preimageKey)
		if err == nil && len(preimageValue) == 20 {
			closer.Close()
			address := common.BytesToAddress(preimageValue).Hex()
			
			allocations = append(allocations, Allocation{
				Address: address,
				Balance: account.Balance.String(),
			})
			totalBalance.Add(totalBalance, account.Balance)
			accountCount++

			if accountCount%1000 == 0 {
				fmt.Printf("  Processed %d accounts...\n", accountCount)
			}
		} else if closer != nil {
			closer.Close()
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	// If we didn't find preimages, try a different approach
	if len(allocations) == 0 {
		fmt.Println("No preimages found, trying alternative approach...")
		
		// Look for known account patterns or use a different key structure
		// This is a fallback - in practice, we'd need the proper preimage mapping
	}

	fmt.Printf("\nFound %d accounts with balance >= %s wei\n", len(allocations), *minBalance)
	fmt.Printf("Total balance: %s wei\n", totalBalance.String())

	// Convert to LUX for display
	luxBalance := new(big.Float).SetInt(totalBalance)
	luxBalance.Quo(luxBalance, big.NewFloat(1e9))
	fmt.Printf("Total balance: %s LUX\n", luxBalance.String())

	// Write output
	if *outputPath != "" {
		output := map[string]interface{}{
			"allocations":  allocations,
			"totalBalance": totalBalance.String(),
			"accountCount": len(allocations),
		}

		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal output: %v", err)
		}

		if err := os.WriteFile(*outputPath, data, 0644); err != nil {
			log.Fatalf("Failed to write output: %v", err)
		}

		fmt.Printf("\nWrote %d allocations to %s\n", len(allocations), *outputPath)
	}
}