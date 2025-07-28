package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	var src = flag.String("src", "", "source database")
	flag.Parse()

	if *src == "" {
		flag.Usage()
		log.Fatal("--src is required")
	}

	// Treasury address
	treasury := "0x9011e888251ab053b7bd1cdb598db4f9ded94714"
	
	// Convert to bytes
	addr, err := hex.DecodeString(treasury[2:])
	if err != nil {
		log.Fatalf("Failed to decode address: %v", err)
	}
	
	// Calculate storage key
	addrHash := crypto.Keccak256(addr)
	
	fmt.Printf("=== Searching for Treasury Account ===\n")
	fmt.Printf("Address: %s\n", treasury)
	fmt.Printf("Address bytes: %x\n", addr)
	fmt.Printf("Address hash: %x\n", addrHash)
	
	db, err := pebble.Open(*src, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	iter, err := db.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	found := false
	accountKeys := 0
	samples := 0
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		if len(key) < 41 {
			continue
		}
		
		logicalKey := key[33:len(key)-8]
		
		// Look for account keys (0x26 prefix)
		if len(logicalKey) > 0 && logicalKey[0] == 0x26 {
			accountKeys++
			
			// Check if this matches our treasury
			if len(logicalKey) >= 33 { // 0x26 + 32 byte hash
				accountHash := logicalKey[1:33]
				if string(accountHash) == string(addrHash) {
					found = true
					fmt.Printf("\nFOUND TREASURY ACCOUNT!\n")
					fmt.Printf("Key: %x\n", key)
					fmt.Printf("Logical key: %x\n", logicalKey)
					fmt.Printf("Value length: %d\n", len(value))
					fmt.Printf("Value: %x\n", value)
				}
			}
			
			// Show some samples
			if samples < 5 {
				samples++
				fmt.Printf("\nAccount sample %d:\n", samples)
				fmt.Printf("  Key: %x\n", logicalKey)
				if len(logicalKey) >= 33 {
					fmt.Printf("  Account hash: %x\n", logicalKey[1:33])
				}
				fmt.Printf("  Value length: %d bytes\n", len(value))
			}
		}
	}
	
	fmt.Printf("\nTotal account keys found: %d\n", accountKeys)
	if !found {
		fmt.Println("Treasury account NOT FOUND in database")
		
		// Try alternative search - look for the address directly
		fmt.Println("\nTrying alternative search patterns...")
		
		iter2, _ := db.NewIter(nil)
		defer iter2.Close()
		
		patterns := 0
		for iter2.First(); iter2.Valid() && patterns < 1000; iter2.Next() {
			key := iter2.Key()
			value := iter2.Value()
			
			// Check if the address appears anywhere in the key or value
			if contains(key, addr) || contains(value, addr) {
				patterns++
				fmt.Printf("\nFound address pattern in:\n")
				fmt.Printf("  Key: %x\n", key)
				if len(value) < 100 {
					fmt.Printf("  Value: %x\n", value)
				} else {
					fmt.Printf("  Value: %x... (truncated, %d bytes)\n", value[:100], len(value))
				}
			}
		}
		
		if patterns == 0 {
			fmt.Println("No patterns found containing the treasury address")
		}
	}
}

func contains(data, pattern []byte) bool {
	if len(pattern) == 0 || len(data) < len(pattern) {
		return false
	}
	
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}