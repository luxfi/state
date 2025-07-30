package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"

	"github.com/cockroachdb/pebble"
	"golang.org/x/crypto/sha3"
)

func main() {
	var (
		db      = flag.String("db", "", "path to pebbledb")
		address = flag.String("address", "0x9011e888251ab053b7bd1cdb598db4f9ded94714", "address to check")
	)
	flag.Parse()

	if *db == "" {
		flag.Usage()
		log.Fatal("--db is required")
	}

	// Remove 0x prefix if present
	addr := *address
	if len(addr) >= 2 && addr[:2] == "0x" {
		addr = addr[2:]
	}

	// Convert to bytes
	addrBytes, err := hex.DecodeString(addr)
	if err != nil {
		log.Fatalf("Invalid address: %v", err)
	}

	// Open database
	database, err := pebble.Open(*db, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Calculate storage key for balance
	// Balance is stored at slot 0 for most ERC20 contracts
	// Key = keccak256(address . uint256(0))
	balanceKey := getBalanceStorageKey(addrBytes)

	// Try different key formats that might be used
	prefixes := [][]byte{
		[]byte("evms"),    // state
		[]byte("evm\x73"), // state (0x73)
		[]byte("evm\x26"), // accounts (0x26)
		[]byte("evm\xa3"), // storage (0xa3)
	}

	fmt.Printf("Checking balance for address: %s\n", *address)
	fmt.Printf("Address bytes: %x\n", addrBytes)
	fmt.Printf("Balance storage key: %x\n", balanceKey)

	for _, prefix := range prefixes {
		fmt.Printf("\nTrying prefix: %x (%s)\n", prefix, string(prefix[:3]))

		// Try account data first
		accountKey := append(prefix, addrBytes...)
		value, closer, err := database.Get(accountKey)
		if err == nil {
			fmt.Printf("Found account data! Length: %d\n", len(value))
			fmt.Printf("Raw value: %x\n", value)
			closer.Close()

			// Try to decode as RLP-encoded account
			if len(value) >= 32 {
				// Simple extraction - in production use proper RLP decoding
				// Balance is typically the first field after nonce
				possibleBalance := value[len(value)-32:]
				balance := new(big.Int).SetBytes(possibleBalance)
				fmt.Printf("Possible balance: %s wei\n", balance.String())
				fmt.Printf("In LUX (18 decimals): %s\n", weiToEther(balance))
			}
		}

		// Try storage slot for balance
		storageKey := append(prefix, balanceKey...)
		value, closer, err = database.Get(storageKey)
		if err == nil {
			fmt.Printf("Found storage value! Length: %d\n", len(value))
			balance := new(big.Int).SetBytes(value)
			fmt.Printf("Balance: %s wei\n", balance.String())
			fmt.Printf("In LUX (18 decimals): %s\n", weiToEther(balance))
			closer.Close()
		}
	}

	// Also try to find any keys related to this address
	fmt.Printf("\nSearching for any keys containing the address...\n")
	iter, err := database.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	count := 0
	found := 0
	for iter.First(); iter.Valid() && count < 1000000; iter.Next() {
		key := iter.Key()
		if len(key) >= 20 {
			// Check if key contains address bytes
			for i := 0; i <= len(key)-20; i++ {
				if bytesEqual(key[i:i+20], addrBytes) {
					found++
					if found <= 10 {
						fmt.Printf("Found key containing address: %x\n", key)
						value := iter.Value()
						if len(value) > 0 && len(value) <= 32 {
							num := new(big.Int).SetBytes(value)
							fmt.Printf("  Value as number: %s\n", num.String())
						}
					}
					break
				}
			}
		}
		count++
	}

	fmt.Printf("\nScanned %d keys, found %d containing the address\n", count, found)
}

func getBalanceStorageKey(address []byte) []byte {
	// For simple balance mapping at slot 0:
	// key = keccak256(address . uint256(0))
	data := make([]byte, 64)
	copy(data[12:32], address) // address padded to 32 bytes
	// slot 0 is already zeros

	hash := sha3.NewLegacyKeccak256()
	hash.Write(data)
	return hash.Sum(nil)
}

func weiToEther(wei *big.Int) string {
	// 1 ether = 10^18 wei
	ether := new(big.Float).SetInt(wei)
	divisor := new(big.Float).SetFloat64(1e18)
	ether.Quo(ether, divisor)
	return ether.Text('f', 6) + " LUX"
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
