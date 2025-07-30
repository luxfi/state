package main

import (
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: check-keys <db-path>")
		os.Exit(1)
	}

	db, err := pebble.Open(os.Args[1], &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	prefixes := make(map[byte]int)
	count := 0

	for iter.First(); iter.Valid() && count < 100000; iter.Next() {
		key := iter.Key()
		if len(key) > 0 {
			prefixes[key[0]]++
		}
		count++
	}

	fmt.Printf("Analyzed %d keys\n", count)
	fmt.Println("Key prefixes:")
	for p, c := range prefixes {
		fmt.Printf("  0x%02x ('%c'): %d keys\n", p, p, c)
	}
}
