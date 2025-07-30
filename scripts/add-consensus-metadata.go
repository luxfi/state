package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"math/big"

	"github.com/cockroachdb/pebble"
)

func main() {
	// Open the consensus database
	db, err := pebble.Open("runtime/lux-96369-vm-ready/state", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Add version marker
	versionKey := []byte("version")
	versionValue := make([]byte, 8)
	binary.BigEndian.PutUint64(versionValue, 24) // Current DB format version

	if err := db.Set(versionKey, versionValue, nil); err != nil {
		log.Fatal("Failed to set version:", err)
	}
	fmt.Println("✅ Set version = 24")

	// Add currentSupply (1.9 trillion LUX)
	supplyKey := []byte("currentSupply")
	supply := new(big.Int)
	supply.SetString("1900000000000000000000000000", 10) // 1.9T LUX with 18 decimals
	supplyBytes := supply.Bytes()

	if err := db.Set(supplyKey, supplyBytes, nil); err != nil {
		log.Fatal("Failed to set currentSupply:", err)
	}
	fmt.Printf("✅ Set currentSupply = %s\n", supply.String())

	// Add timestamp (unix timestamp of block 1082780)
	timestampKey := []byte("timestamp")
	timestampValue := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampValue, 1717148410)

	if err := db.Set(timestampKey, timestampValue, nil); err != nil {
		log.Fatal("Failed to set timestamp:", err)
	}
	fmt.Println("✅ Set timestamp = 1717148410")

	// Verify lastAccepted exists
	if val, closer, err := db.Get([]byte("lastAccepted")); err == nil {
		fmt.Printf("✅ Found lastAccepted: %x\n", val)
		closer.Close()
	} else {
		fmt.Println("⚠️  Warning: lastAccepted not found")
	}

	// Verify height exists
	if val, closer, err := db.Get([]byte("db:height")); err == nil {
		height := binary.BigEndian.Uint64(val)
		fmt.Printf("✅ Found db:height: %d\n", height)
		closer.Close()
	} else {
		fmt.Println("⚠️  Warning: db:height not found")
	}

	fmt.Println("\n✅ Consensus metadata added successfully!")
}
