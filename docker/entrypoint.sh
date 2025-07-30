#!/bin/bash
set -euo pipefail

# Environment variables with defaults
CHAIN_ID="${CHAIN_ID:-X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3}"
VM_ID="${VM_ID:-mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6}"
NETWORK_ID="${NETWORK_ID:-96369}"
TIP_HEIGHT="${TIP_HEIGHT:-1082780}"
TIP_HASH="${TIP_HASH:-0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0}"

# Paths
RUNTIME_DIR="/app/runtime"
CHAINDATA_DIR="/app/chaindata"
NODE_DATA_DIR="$RUNTIME_DIR/node-data"
TARGET_EVM="$NODE_DATA_DIR/db/chains/$CHAIN_ID/evm"

echo "🚀 Lux Genesis Migration & Launch"
echo "================================="
echo "Chain ID: $CHAIN_ID"
echo "VM ID: $VM_ID"
echo "Network ID: $NETWORK_ID"
echo "Target Height: $TIP_HEIGHT"
echo ""

# Check if migration is needed
if [ ! -d "$TARGET_EVM" ] || [ -z "$(ls -A "$TARGET_EVM" 2>/dev/null)" ]; then
    echo "▶ Migration needed - starting genesis pipeline"
    
    # Ensure directories exist
    mkdir -p "$TARGET_EVM"
    
    # Check for source chaindata
    if [ ! -d "$CHAINDATA_DIR/db/pebbledb" ] && [ ! -d "$CHAINDATA_DIR/lux-mainnet-96369/db/pebbledb" ]; then
        echo "❌ Error: No chaindata found at $CHAINDATA_DIR"
        echo "Please mount the chaindata directory with -v /path/to/chaindata:/app/chaindata"
        exit 1
    fi
    
    # Determine source path
    SRC_DB="$CHAINDATA_DIR/db/pebbledb"
    if [ ! -d "$SRC_DB" ]; then
        SRC_DB="$CHAINDATA_DIR/lux-mainnet-96369/db/pebbledb"
    fi
    
    echo "▶ Source database: $SRC_DB"
    
    # Run migration pipeline
    echo "▶ Step 1: Importing subnet data with namespace stripping"
    /app/bin/genesis import subnet "$SRC_DB" "$TARGET_EVM"
    
    echo "▶ Step 2: Cleaning 10-byte canonical keys (removing 0x6e suffix)"
    /app/bin/genesis repair delete-suffix "$TARGET_EVM" 6e --prefix 68
    
    echo "▶ Step 3: Rebuilding canonical mappings with 9-byte keys"
    /app/bin/genesis rebuild-canonical "$TARGET_EVM"
    
    echo "▶ Step 4: Copying to node structure with consensus markers"
    /app/bin/genesis copy-to-node \
        --chain-id "$CHAIN_ID" \
        --vm-id "$VM_ID" \
        --evm-db "$TARGET_EVM" \
        --node-dir "$NODE_DATA_DIR" \
        --height "$TIP_HEIGHT" \
        --hash "$TIP_HASH"
    
    echo "✅ Migration complete!"
else
    echo "▶ Using existing migrated data at $TARGET_EVM"
fi

echo ""
echo "▶ Launching luxd with migrated chain data"
echo "  RPC endpoint: http://localhost:9630/ext/bc/C/rpc"
echo "  Expected height: $TIP_HEIGHT"
echo ""

# Launch luxd with the migrated data
exec /app/bin/luxd \
    --network-id="$NETWORK_ID" \
    --data-dir="$NODE_DATA_DIR" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --dev \
    --log-level=info \
    "$@"