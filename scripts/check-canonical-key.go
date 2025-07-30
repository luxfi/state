package main

import (
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	db, err := pebble.Open("runtime/lux-96369-fixed/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check what the VM is looking for
	lookingFor := []byte{0x68, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x6e}
	fmt.Printf("VM is looking for key: %x\n", lookingFor)

	if val, closer, err := db.Get(lookingFor); err == nil {
		fmt.Printf("Found value: %x\n", val)
		closer.Close()
	} else {
		fmt.Printf("Not found: %v\n", err)
	}

	// Check our evmn key
	evmnKey := append([]byte("evmn"), make([]byte, 8)...)
	fmt.Printf("\nOur key format: %x\n", evmnKey)

	// List first few keys to see structure
	fmt.Println("\nFirst few keys in database:")
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < 20; iter.Next() {
		key := iter.Key()
		fmt.Printf("Key: %x (len=%d)\n", key, len(key))
		count++
	}
}
