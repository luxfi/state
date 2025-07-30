package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/cockroachdb/pebble"
)

func main() {
	var src = flag.String("src", "", "source pebbledb path")
	var dst = flag.String("dst", "", "destination pebbledb path")
	var limit = flag.Int("limit", 10000, "max keys to migrate")
	flag.Parse()

	if *src == "" || *dst == "" {
		flag.Usage()
		log.Fatal("Both --src and --dst are required")
	}

	fmt.Println("=== Limited EVM Key Migration ===")
	fmt.Printf("Source: %s\n", *src)
	fmt.Printf("Destination: %s\n", *dst)
	fmt.Printf("Limit: %d keys\n", *limit)

	start := time.Now()

	// Open source database
	srcDB, err := pebble.Open(*src, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer srcDB.Close()

	// Create destination database
	dstDB, err := pebble.Open(*dst, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to create destination database: %v", err)
	}
	defer dstDB.Close()

	// Create iterator for source database
	iter, err := srcDB.NewIter(nil)
	if err != nil {
		log.Fatalf("Failed to create iterator: %v", err)
	}
	defer iter.Close()

	evmPrefix := []byte("evm")
	count := 0
	headers := 0
	bodies := 0
	receipts := 0
	numbers := 0
	hashes := 0

	// Batch for performance
	batch := dstDB.NewBatch()

	for iter.First(); iter.Valid() && count < *limit; iter.Next() {
		key := iter.Key()

		// Skip short keys
		if len(key) < 34 {
			continue
		}

		// Strip namespace prefix (33 bytes)
		actualKey := key[33:]

		// Create new key with evm prefix
		newKey := make([]byte, len(evmPrefix)+len(actualKey))
		copy(newKey, evmPrefix)
		copy(newKey[len(evmPrefix):], actualKey)

		// Copy value
		value := iter.Value()
		if err := batch.Set(newKey, value, nil); err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}

		// Track key types
		if len(actualKey) > 0 {
			switch actualKey[0] {
			case 0x68: // 'h' headers
				headers++
			case 0x62: // 'b' bodies
				bodies++
			case 0x72: // 'r' receipts
				receipts++
			case 0x6e: // 'n' number->hash
				numbers++
			case 0x48: // 'H' hash->number
				hashes++
			}
		}

		count++

		// Commit batch periodically
		if count%1000 == 0 {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = dstDB.NewBatch()
			fmt.Printf("Migrated %d keys (h:%d H:%d b:%d r:%d n:%d)...\n",
				count, headers, hashes, bodies, receipts, numbers)
		}
	}

	// Commit final batch
	if err := batch.Commit(nil); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	fmt.Printf("\nFinal Statistics:\n")
	fmt.Printf("Total keys migrated: %d\n", count)
	fmt.Printf("- Headers (h):       %d\n", headers)
	fmt.Printf("- Hash->Height (H):  %d\n", hashes)
	fmt.Printf("- Bodies (b):        %d\n", bodies)
	fmt.Printf("- Receipts (r):      %d\n", receipts)
	fmt.Printf("- Numbers (n):       %d\n", numbers)

	fmt.Printf("\n=== Migration Complete in %s ===\n", time.Since(start))
}
