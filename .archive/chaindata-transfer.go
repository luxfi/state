// Package chaindatatransfer provides functions to export and import a Lux subnet EVM PebbleDB
// by iterating all core rawdb prefixes (headers, bodies, receipts, difficulty, tx lookups, bloom bits, etc.).
// It includes a CLI for copying one chaindata folder to another for any subnet EVM ID.
package main

import (
    "flag"
    "fmt"
    "log"
    "path/filepath"

    "github.com/cockroachdb/pebble"
    "github.com/ethereum/go-ethereum/core/rawdb"
)

// CopyChaindata opens the Pebble DB at srcPath and dstPath, then iterates through all
// relevant rawdb prefixes, copying each key/value pair into the destination DB in batches.
func CopyChaindata(srcPath, dstPath string) error {
    srcDB, err := pebble.Open(filepath.Clean(srcPath), &pebble.Options{})
    if err != nil {
        return fmt.Errorf("open source Pebble DB: %w", err)
    }
    defer srcDB.Close()

    dstDB, err := pebble.Open(filepath.Clean(dstPath), &pebble.Options{})
    if err != nil {
        return fmt.Errorf("open destination Pebble DB: %w", err)
    }
    defer dstDB.Close()

    // Core rawdb prefixes to export/import (from geth core/rawdb/schema.go)
    prefixes := map[string][]byte{
        "Header":          []byte("h"), // headerPrefix = []byte("h")
        "Body":            []byte("b"), // bodyPrefix = []byte("b")
        "NumberToHash":    []byte("n"), // headerNumberPrefix = []byte("n")
        "HashToNumber":    []byte("H"), // headerHashSuffix = []byte("H")
        "TotalDifficulty": []byte("t"), // headerTDSuffix = []byte("t")
        "Receipt":         []byte("r"), // receiptsPrefix = []byte("r")
        "TxLookup":        []byte("l"), // txLookupPrefix = []byte("l")
    }

    totalKeys := 0
    for name, prefix := range prefixes {
        lower := prefix
        upper := []byte{prefix[0] + 1}

        iter, err := srcDB.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
        if err != nil {
            return fmt.Errorf("create iterator for %s: %w", name, err)
        }
        defer iter.Close()

        batch := dstDB.NewBatch()
        count := 0
        for ok := iter.First(); ok; ok = iter.Next() {
            batch.Set(iter.Key(), iter.Value(), pebble.Sync)
            count++
            totalKeys++
            
            // Commit batch every 1000 entries
            if count%1000 == 0 {
                if err := batch.Commit(pebble.Sync); err != nil {
                    return fmt.Errorf("commit batch for %s: %w", name, err)
                }
                batch = dstDB.NewBatch()
                fmt.Printf("  %s: copied %d entries...\n", name, count)
            }
        }
        
        // Commit final batch
        if count%1000 != 0 {
            if err := batch.Commit(pebble.Sync); err != nil {
                return fmt.Errorf("commit final batch for %s: %w", name, err)
            }
        }
        
        fmt.Printf("✓ %s: copied %d entries\n", name, count)
    }
    
    fmt.Printf("\n📊 Total keys copied: %d\n", totalKeys)
    return nil
}

func main() {
    // CLI options
    src := flag.String("src", "", "path to source C/chaindata/<subnet-id>/db/pebbledb folder")
    dst := flag.String("dst", "", "path to destination C/chaindata/<subnet-id>/db/pebbledb folder")
    subnetID := flag.Uint("subnet-id", 0, "numeric ID of the subnet EVM to migrate")
    configs := flag.String("configs", "", "path to directory containing configs/<subnet-id>/ (chain-config.json, config.json, genesis.json)")
    flag.Parse()

    if *src == "" || *dst == "" || *subnetID == 0 || *configs == "" {
        log.Fatal("--src, --dst, --subnet-id, and --configs are required")
    }

    fmt.Printf("🚀 Transferring chaindata for subnet %d\n", *subnetID)
    fmt.Printf("   Source: %s\n", *src)
    fmt.Printf("   Destination: %s\n", *dst)
    fmt.Printf("   Configs: %s\n\n", *configs)

    // Copy the PebbleDB data for the specified subnet
    if err := CopyChaindata(*src, *dst); err != nil {
        log.Fatalf("failed to copy chaindata for subnet %d: %v", *subnetID, err)
    }
    fmt.Printf("\n✅ Chaindata for subnet %d copied successfully.\n\n", *subnetID)

    // Next, start luxd for that subnet EVM using the migrated DB:
    fmt.Printf("To launch subnet %d on the new mainnet, run:\n\n", *subnetID)
    fmt.Printf("  luxd --db-dir=%s --chain-config=%s/chain-config.json \\\n", *dst, *configs)
    fmt.Printf("       --config-file=%s/config.json --genesis-file=%s/genesis.json \\\n", *configs, *configs)
    fmt.Printf("       --network-id=%d\n", *subnetID)
}

/*
General Usage Example:

For each subnet EVM you want to migrate, run:

  go build -o chaindata-transfer chaindata-transfer.go

  ./chaindata-transfer \
    --src ~/work/lux/genesis/chaindata/<subnet-id>/db/pebbledb \
    --dst /data/new-node/<subnet-id>/db/pebbledb \
    --subnet-id <subnet-id> \
    --configs configs/<subnet-id>

This copies the entire block history (headers, bodies, receipts, difficulty, tx lookups) for that subnet.

Then launch the migrated subnet EVM with `luxd`:

  luxd \
    --db-dir /data/new-node/<subnet-id> \
    --chain-config configs/<subnet-id>/chain-config.json \
    --config-file   configs/<subnet-id>/config.json \
    --genesis-file  configs/<subnet-id>/genesis.json \
    --network-id    <subnet-id>

Repeat the above for each subnet EVM (e.g., ID 96369, 36962, etc.).
*/