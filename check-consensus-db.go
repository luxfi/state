package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	
	"github.com/cockroachdb/pebble"
)

func main() {
	dbPath := os.Args[1]
	
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	
	fmt.Println("Consensus database contents:")
	fmt.Println("===========================")
	
	// Check lastAccepted
	if val, closer, err := db.Get([]byte("lastAccepted")); err == nil {
		fmt.Printf("lastAccepted: %x (%s)\n", val, string(val))
		closer.Close()
	} else {
		fmt.Printf("lastAccepted: NOT FOUND\n")
	}
	
	// Check db:height
	if val, closer, err := db.Get([]byte("db:height")); err == nil && len(val) == 8 {
		height := binary.BigEndian.Uint64(val)
		fmt.Printf("db:height: %d\n", height)
		closer.Close()
	} else {
		fmt.Printf("db:height: NOT FOUND\n")
	}
	
	// Check Height (plain)
	if val, closer, err := db.Get([]byte("Height")); err == nil && len(val) == 8 {
		height := binary.BigEndian.Uint64(val)
		fmt.Printf("Height: %d\n", height)
		closer.Close()
	} else {
		fmt.Printf("Height: NOT FOUND\n")
	}
	
	// Check version
	if val, closer, err := db.Get([]byte("version")); err == nil && len(val) == 8 {
		version := binary.BigEndian.Uint64(val)
		fmt.Printf("version: %d\n", version)
		closer.Close()
	} else {
		fmt.Printf("version: NOT FOUND\n")
	}
	
	// List all keys
	fmt.Println("\nAll keys in database:")
	iter, _ := db.NewIter(nil)
	defer iter.Close()
	
	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		fmt.Printf("  Key: %x (%s) - %d bytes\n", key, string(key), len(iter.Value()))
		count++
		if count > 20 {
			fmt.Println("  ... (truncated)")
			break
		}
	}
}