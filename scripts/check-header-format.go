package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-header-format <db_path>")
		os.Exit(1)
	}

	dbPath := os.Args[1]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	fmt.Println("=== Checking Header Format ===")
	
	// Check a few headers to understand the key format
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68}, // 'h'
		UpperBound: []byte{0x69},
	})
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid() && count < 10; iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		fmt.Printf("\nHeader %d:\n", count+1)
		fmt.Printf("  Key: %x (len=%d)\n", key, len(key))
		fmt.Printf("  Key breakdown:\n")
		fmt.Printf("    Prefix: %02x (%c)\n", key[0], key[0])
		
		if len(key) >= 9 {
			num := binary.BigEndian.Uint64(key[1:9])
			fmt.Printf("    Number: %d (0x%x)\n", num, key[1:9])
		}
		
		if len(key) >= 41 {
			fmt.Printf("    Hash: %x\n", key[9:41])
		}
		
		fmt.Printf("  Value length: %d bytes\n", len(value))
		
		count++
	}
	
	// Now check if we can lookup block 0 and block 1082780
	fmt.Println("\n=== Testing Specific Lookups ===")
	
	// Try to find block 0
	fmt.Println("\nLooking for block 0...")
	findBlock(db, 0)
	
	// Try to find block 1082780
	fmt.Println("\nLooking for block 1082780...")
	findBlock(db, 1082780)
	
	// Check the number->hash mapping
	fmt.Println("\n=== Checking Number->Hash Mappings ===")
	checkNumberMapping(db, 0)
	checkNumberMapping(db, 1082780)
}

func findBlock(db *pebble.DB, blockNum uint64) {
	// First get the hash from number->hash mapping
	nKey := make([]byte, 9)
	nKey[0] = 0x6e // 'n'
	binary.BigEndian.PutUint64(nKey[1:], blockNum)
	
	hash, closer, err := db.Get(nKey)
	if err != nil {
		fmt.Printf("  No number->hash mapping found for block %d\n", blockNum)
		return
	}
	closer.Close()
	
	fmt.Printf("  Hash from n mapping: %x\n", hash)
	
	// Now try different header key formats
	
	// Format 1: h + number + hash
	hKey1 := make([]byte, 41)
	hKey1[0] = 0x68 // 'h'
	binary.BigEndian.PutUint64(hKey1[1:9], blockNum)
	copy(hKey1[9:41], hash)
	
	if val, closer, err := db.Get(hKey1); err == nil {
		fmt.Printf("  Found header with format h+num+hash, length: %d\n", len(val))
		closer.Close()
	} else {
		fmt.Printf("  No header with format h+num+hash\n")
		
		// Try finding any header for this block number
		prefix := make([]byte, 9)
		prefix[0] = 0x68 // 'h'
		binary.BigEndian.PutUint64(prefix[1:], blockNum)
		
		iter, _ := db.NewIter(&pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: incrementBytes(prefix),
		})
		
		if iter.First() {
			key := iter.Key()
			fmt.Printf("  Found header with key: %x\n", key)
			if len(key) >= 41 {
				fmt.Printf("    Stored hash: %x\n", key[9:41])
				fmt.Printf("    Expected hash: %x\n", hash)
				fmt.Printf("    Match: %v\n", hex.EncodeToString(key[9:41]) == hex.EncodeToString(hash))
			}
		} else {
			fmt.Printf("  No headers found for block %d\n", blockNum)
		}
		iter.Close()
	}
}

func checkNumberMapping(db *pebble.DB, blockNum uint64) {
	fmt.Printf("\nBlock %d number->hash mapping:\n", blockNum)
	
	nKey := make([]byte, 9)
	nKey[0] = 0x6e // 'n'
	binary.BigEndian.PutUint64(nKey[1:], blockNum)
	
	if hash, closer, err := db.Get(nKey); err == nil {
		fmt.Printf("  Key: %x\n", nKey)
		fmt.Printf("  Hash: %x\n", hash)
		closer.Close()
		
		// Also check the reverse mapping
		HKey := append([]byte{0x48}, hash...) // 'H' + hash
		if numBytes, closer2, err := db.Get(HKey); err == nil {
			reverseNum := binary.BigEndian.Uint64(numBytes)
			fmt.Printf("  Reverse mapping (H+hash): block %d\n", reverseNum)
			fmt.Printf("  Bidirectional mapping OK: %v\n", reverseNum == blockNum)
			closer2.Close()
		}
	} else {
		fmt.Printf("  Not found\n")
	}
}

func incrementBytes(b []byte) []byte {
	result := make([]byte, len(b))
	copy(result, b)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 255 {
			result[i]++
			break
		}
		result[i] = 0
	}
	return result
}