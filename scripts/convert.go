package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func main() {
	var (
		chainID    = flag.String("chain", "", "Chain ID (7777 or 96369)")
		inputPath  = flag.String("input", "", "Input LevelDB path (defaults to chaindata/{chain}/)")
		outputPath = flag.String("output", "", "Output PebbleDB path (defaults to pebbledb/{chain}/)")
		dryRun     = flag.Bool("dry-run", false, "Only show statistics without converting")
	)
	flag.Parse()

	if *chainID == "" {
		log.Fatal("Chain ID is required (-chain 7777 or -chain 96369)")
	}

	// Default paths based on chain ID
	if *inputPath == "" {
		*inputPath = filepath.Join("chaindata", fmt.Sprintf("lux-%s", *chainID))
	}
	if *outputPath == "" {
		*outputPath = filepath.Join("pebbledb", fmt.Sprintf("lux-%s", *chainID))
	}

	fmt.Printf("Converting LUX %s chain data:\n", *chainID)
	fmt.Printf("  Input:  %s\n", *inputPath)
	fmt.Printf("  Output: %s\n", *outputPath)

	// Open LevelDB
	ldb, err := leveldb.OpenFile(*inputPath, &opt.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open LevelDB: %v", err)
	}
	defer ldb.Close()

	// Statistics
	var keyCount, totalKeySize, totalValueSize int64

	if *dryRun {
		// Just count keys
		iter := ldb.NewIterator(nil, nil)
		defer iter.Release()

		for iter.Next() {
			keyCount++
			totalKeySize += int64(len(iter.Key()))
			totalValueSize += int64(len(iter.Value()))
		}

		fmt.Printf("\nLevelDB Statistics:\n")
		fmt.Printf("  Total Keys: %d\n", keyCount)
		fmt.Printf("  Total Key Size: %.2f MB\n", float64(totalKeySize)/1024/1024)
		fmt.Printf("  Total Value Size: %.2f MB\n", float64(totalValueSize)/1024/1024)
		fmt.Printf("  Average Key Size: %d bytes\n", totalKeySize/keyCount)
		fmt.Printf("  Average Value Size: %d bytes\n", totalValueSize/keyCount)
		return
	}

	// Create output directory
	if err := os.MkdirAll(*outputPath, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Open PebbleDB
	pdb, err := pebble.Open(*outputPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open PebbleDB: %v", err)
	}
	defer pdb.Close()

	// Convert data
	iter := ldb.NewIterator(nil, nil)
	defer iter.Release()

	batch := pdb.NewBatch()
	batchSize := 0
	const maxBatchSize = 100 * 1024 * 1024 // 100MB batches

	fmt.Println("\nConverting LevelDB to PebbleDB...")
	
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		// Make copies since iterator reuses slices
		keyCopy := make([]byte, len(key))
		valueCopy := make([]byte, len(value))
		copy(keyCopy, key)
		copy(valueCopy, value)

		if err := batch.Set(keyCopy, valueCopy, nil); err != nil {
			log.Fatalf("Failed to set key: %v", err)
		}

		keyCount++
		batchSize += len(keyCopy) + len(valueCopy)

		// Commit batch if it's getting large
		if batchSize >= maxBatchSize {
			if err := batch.Commit(nil); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = pdb.NewBatch()
			batchSize = 0
			fmt.Printf("  Converted %d keys...\n", keyCount)
		}
	}

	// Commit final batch
	if batchSize > 0 {
		if err := batch.Commit(nil); err != nil {
			log.Fatalf("Failed to commit final batch: %v", err)
		}
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	fmt.Printf("\nConversion complete!\n")
	fmt.Printf("  Total keys converted: %d\n", keyCount)
	
	// Get PebbleDB metrics
	metrics := pdb.Metrics()
	fmt.Printf("  PebbleDB size: %.2f MB\n", float64(metrics.DiskSpaceUsage())/1024/1024)
}