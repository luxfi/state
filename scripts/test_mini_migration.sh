#!/usr/bin/env bash
set -euo pipefail

echo "=== Testing mini migration pipeline ==="

# Use a small test database
SRC=/home/z/archived/restored-blockchain-data/chainData/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/db/pebbledb
ROOT=.tmp/mini-test-$(date +%s)

echo "➜ Source: $SRC"
echo "➜ Root: $ROOT"

# Step 1: Migrate keys
echo ""
echo "=== Step 1: Migrating keys ==="
bin/migrate_evm \
    --src  $SRC \
    --dst  $ROOT/evm/pebbledb \
    --verbose

# Step 2: Find tip height
echo ""
echo "=== Step 2: Finding tip height ==="
TIP=$(bin/find_actual_max_block --db $ROOT/evm/pebbledb 2>/dev/null || echo "0")
echo "➜ Tip height: $TIP"

# Step 3: Check if we have any canonical mappings
echo ""
echo "=== Step 3: Checking canonical mappings ==="
bin/check-canonical-keys --db $ROOT/evm/pebbledb 2>&1 | head -20

echo ""
echo "=== Test complete ==="
echo "Database migrated to: $ROOT/evm/pebbledb"