#!/usr/bin/env bash
set -euo pipefail

echo "=== Simple ZOO Subnet Test ==="

ROOT=.tmp/zoo-test-$(date +%s)
SRC=/home/z/archived/restored-blockchain-data/chainData/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/db/pebbledb

# Step 1: Migrate
echo "Step 1: Migrating database..."
bin/migrate_evm --src $SRC --dst $ROOT/evm/pebbledb

# Step 2: Check what we have
echo -e "\nStep 2: Checking migrated data..."
echo "Canonical mappings:"
go run -args $ROOT/evm/pebbledb - <<'EOF'
package main
import (
    "encoding/binary"
    "fmt"
    "os"
    "github.com/cockroachdb/pebble"
)
func main() {
    db, _ := pebble.Open(os.Args[1], nil)
    defer db.Close()
    
    // Check evmn keys
    iter := db.NewIter(nil)
    defer iter.Close()
    
    count := 0
    maxHeight := uint64(0)
    
    prefix := []byte("evmn")
    for iter.SeekGE(prefix); iter.Valid() && count < 10; iter.Next() {
        key := iter.Key()
        if len(key) >= 4 && string(key[:4]) == "evmn" {
            if len(key) == 12 { // proper format
                height := binary.BigEndian.Uint64(key[4:])
                fmt.Printf("  Block %d -> hash %x\n", height, iter.Value())
                if height > maxHeight {
                    maxHeight = height
                }
                count++
            }
        }
    }
    
    fmt.Printf("\nMax height with canonical mapping: %d\n", maxHeight)
    
    // Check if we have headers
    hCount := 0
    hPrefix := []byte("evmh")
    for iter.SeekGE(hPrefix); iter.Valid() && hCount < 5; iter.Next() {
        key := iter.Key()
        if len(key) >= 4 && string(key[:4]) == "evmh" {
            hCount++
        }
    }
    fmt.Printf("Headers found: %d\n", hCount)
}
EOF

# Step 3: Create consensus state
echo -e "\nStep 3: Creating consensus state..."
bin/replay-consensus-pebble \
    --evm $ROOT/evm/pebbledb \
    --state $ROOT/state/pebbledb \
    --tip 475 2>&1 | tail -20

echo -e "\n=== Test Complete ==="
echo "Data location: $ROOT"