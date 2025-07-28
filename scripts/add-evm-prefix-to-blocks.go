package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/cockroachdb/pebble"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <source-db> <destination-db>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	srcPath := flag.Arg(0)
	dstPath := flag.Arg(1)

	// Open source database
	srcDB, err := pebble.Open(srcPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Create destination database
	dstDB, err := pebble.Open(dstPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create destination database: %v", err)
	}
	defer dstDB.Close()

	// Copy all keys with "evm" prefix
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	evmPrefix := []byte("evm")
	batch := dstDB.NewBatch()
	count := 0

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		value := iter.Value()
		
		// Add "evm" prefix if not already present
		newKey := key
		if len(key) < 3 || string(key[:3]) != "evm" {
			newKey = append(evmPrefix, key...)
		}
		
		if err := batch.Set(newKey, value, nil); err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}
		
		count++
		if count%10000 == 0 {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Migrated %d keys...\n", count)
		}
	}

	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	fmt.Printf("Migration complete: %d keys migrated\n", count)
}