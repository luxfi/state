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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: decode-rlp-data <db-path>")
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

	fmt.Println("=== Decoding RLP Data from Subnet Database ===")

	// The subnet prefix we found
	subnetPrefix := []byte{0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1}

	// Scan for keys with this prefix
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: subnetPrefix,
	})
	defer iter.Close()

	examined := 0
	for iter.First(); iter.Valid() && examined < 5; iter.Next() {
		key := iter.Key()
		val := iter.Value()

		// Check if key starts with our prefix
		if !bytes.HasPrefix(key, subnetPrefix) {
			break
		}

		fmt.Printf("\n=== Entry %d ===\n", examined+1)

		// Try to decode as different types
		suffix := key[len(subnetPrefix):]
		fmt.Printf("Key suffix: %s\n", hex.EncodeToString(suffix))
		fmt.Printf("Value length: %d bytes\n", len(val))

		// Try as block header
		var header types.Header
		if err := rlp.DecodeBytes(val, &header); err == nil {
			fmt.Println("Successfully decoded as Header!")
			fmt.Printf("  Number: %d\n", header.Number)
			fmt.Printf("  Hash: %s\n", header.Hash().Hex())
			fmt.Printf("  Parent: %s\n", header.ParentHash.Hex())
			fmt.Printf("  Root: %s\n", header.Root.Hex())
			fmt.Printf("  Time: %d\n", header.Time)
			examined++
			continue
		}

		// Try as block
		var block types.Block
		if err := rlp.DecodeBytes(val, &block); err == nil {
			fmt.Println("Successfully decoded as Block!")
			fmt.Printf("  Number: %d\n", block.Number())
			fmt.Printf("  Hash: %s\n", block.Hash().Hex())
			fmt.Printf("  Transactions: %d\n", len(block.Transactions()))
			examined++
			continue
		}

		// Try as transaction
		var tx types.Transaction
		if err := rlp.DecodeBytes(val, &tx); err == nil {
			fmt.Println("Successfully decoded as Transaction!")
			fmt.Printf("  Hash: %s\n", tx.Hash().Hex())
			fmt.Printf("  Value: %s\n", tx.Value())
			if tx.To() != nil {
				fmt.Printf("  To: %s\n", tx.To().Hex())
			}
			examined++
			continue
		}

		// Try as receipt
		var receipt types.Receipt
		if err := rlp.DecodeBytes(val, &receipt); err == nil {
			fmt.Println("Successfully decoded as Receipt!")
			fmt.Printf("  Status: %d\n", receipt.Status)
			fmt.Printf("  Gas Used: %d\n", receipt.GasUsed)
			examined++
			continue
		}

		// Try as account
		type Account struct {
			Nonce    uint64
			Balance  []byte
			Root     common.Hash
			CodeHash []byte
		}
		var account Account
		if err := rlp.DecodeBytes(val, &account); err == nil && len(val) < 200 {
			fmt.Println("Possibly an Account!")
			fmt.Printf("  Nonce: %d\n", account.Nonce)
			if len(account.Balance) > 0 {
				fmt.Printf("  Balance: %s\n", hex.EncodeToString(account.Balance))
			}
			examined++
			continue
		}

		// Try as a generic RLP list
		var list []interface{}
		if err := rlp.DecodeBytes(val, &list); err == nil {
			fmt.Printf("Decoded as generic RLP list with %d elements\n", len(list))
			for i, elem := range list {
				if i >= 3 {
					fmt.Printf("  ... and %d more elements\n", len(list)-3)
					break
				}
				switch v := elem.(type) {
				case []byte:
					if len(v) == 32 {
						fmt.Printf("  [%d]: 32-byte value (hash?): %s\n", i, hex.EncodeToString(v))
					} else if len(v) == 20 {
						fmt.Printf("  [%d]: 20-byte value (address?): %s\n", i, hex.EncodeToString(v))
					} else {
						fmt.Printf("  [%d]: %d-byte value\n", i, len(v))
					}
				case uint64:
					fmt.Printf("  [%d]: uint64: %d\n", i, v)
				default:
					fmt.Printf("  [%d]: %T\n", i, v)
				}
			}
			examined++
			continue
		}

		fmt.Println("Could not decode as any known type")
		examined++
	}

	fmt.Printf("\n=== Decoded %d entries ===\n", examined)
}
