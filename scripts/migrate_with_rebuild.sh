#!/bin/bash
# Migration script with canonical rebuild integration

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -lt 2 ]; then
    echo "Usage: $0 <source-subnet-db> <destination-root>"
    echo "Example: $0 /subnet96369/pebbledb /data/cchain-full"
    exit 1
fi

SRC_DB="$1"
DST_ROOT="$2"

# Ensure bin directory exists
BIN_DIR="$(dirname "$0")/bin"

echo -e "${YELLOW}Starting subnet to C-Chain migration...${NC}"

# Step 1: Migrate with EVM prefix
echo -e "${GREEN}➜ Step 1: Migrating subnet data with EVM prefix${NC}"
mkdir -p "$DST_ROOT/evm/pebbledb"
"$BIN_DIR/migrate_evm" --src "$SRC_DB" --dst "$DST_ROOT/evm/pebbledb"

# Step 2: Rebuild canonical mappings
echo -e "${GREEN}➜ Step 2: Rebuilding canonical mappings${NC}"
"$BIN_DIR/rebuild_canonical" --db "$DST_ROOT/evm/pebbledb"

# Step 3: Check tip height
echo -e "${GREEN}➜ Step 3: Checking tip height${NC}"
TIP=$("$BIN_DIR/peek_tip_v2" --db "$DST_ROOT/evm/pebbledb" 2>/dev/null || echo "0")
echo "Database tip height: $TIP"

# Step 4: Create consensus state
if [ "$TIP" != "0" ]; then
    echo -e "${GREEN}➜ Step 4: Creating consensus state${NC}"
    mkdir -p "$DST_ROOT/state/pebbledb"
    "$BIN_DIR/replay-consensus-pebble" \
        --evm "$DST_ROOT/evm/pebbledb" \
        --state "$DST_ROOT/state/pebbledb" \
        --tip "$TIP"
else
    echo -e "${YELLOW}⚠ No blocks found, skipping consensus replay${NC}"
fi

echo -e "${GREEN}✅ Migration complete!${NC}"
echo ""
echo "To launch luxd:"
echo "  luxd --db-dir $DST_ROOT --network-id 96369 --staking-enabled=false"
echo ""
echo "To verify with RPC:"
echo "  ./tools/rpc_verify.sh"