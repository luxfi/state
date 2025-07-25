#!/bin/bash
# One-time import script for migrated blockchain data
# This runs the node with pruning disabled to allow proper import

set -e

echo "=== One-Time Import of Migrated Data ==="
echo "This script will:"
echo "1. Start node with pruning disabled"
echo "2. Allow it to process migrated data"
echo "3. Shut down after import"
echo "4. Then you can start normally with pruning enabled"
echo "========================================"

LUXD="/home/z/work/lux/node/build/luxd"

# Check if luxd exists
if [ ! -f "$LUXD" ]; then
    echo "Error: luxd not found at $LUXD"
    exit 1
fi

echo ""
echo "Starting import process..."
echo "The node will start and process the migrated data."
echo "This may take several minutes..."
echo ""

# Backup original config
cp /home/z/.luxd/configs/chains/C/config.json /home/z/.luxd/configs/chains/C/config.json.backup

# Use import config with pruning disabled
cp /home/z/.luxd/configs/chains/C/config-import.json /home/z/.luxd/configs/chains/C/config.json

# Start node with special flags for import
$LUXD \
    --network-id=96369 \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --db-dir=/home/z/.luxd/db \
    --chain-config-dir=/home/z/.luxd/configs/chains \
    --api-admin-enabled=true \
    --index-enabled=false \
    --index-allow-incomplete=true \
    --log-level=info &

NODE_PID=$!
echo "Node started with PID: $NODE_PID"

# Function to check if C-Chain is ready
check_cchain_ready() {
    curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        http://localhost:9630/ext/bc/C/rpc 2>/dev/null | grep -q "result"
}

# Wait for C-Chain to be ready
echo ""
echo "Waiting for C-Chain to initialize..."
COUNTER=0
while ! check_cchain_ready; do
    if [ $COUNTER -gt 300 ]; then
        echo "Timeout waiting for C-Chain to initialize"
        kill $NODE_PID 2>/dev/null
        exit 1
    fi
    echo -n "."
    sleep 2
    COUNTER=$((COUNTER + 1))
done

echo ""
echo "C-Chain is ready! Checking block height..."

# Get current block number
BLOCK_HEIGHT=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc | jq -r '.result' | xargs printf "%d")

echo "Current block height: $BLOCK_HEIGHT"

if [ $BLOCK_HEIGHT -gt 0 ]; then
    echo "✅ Success! Node recognizes $BLOCK_HEIGHT blocks"
    echo ""
    echo "Waiting 30 seconds for node to create snapshots..."
    sleep 30
else
    echo "⚠️  Warning: Node shows block height 0"
fi

# Gracefully shutdown
echo ""
echo "Shutting down node gracefully..."
curl -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"admin_shutdown","params":[]}' \
    http://localhost:9630/ext/admin 2>/dev/null || true

# Wait for shutdown
sleep 5

# Check if process is still running
if kill -0 $NODE_PID 2>/dev/null; then
    echo "Node still running, forcing shutdown..."
    kill $NODE_PID
fi

# Restore original config
echo "Restoring original configuration..."
cp /home/z/.luxd/configs/chains/C/config.json.backup /home/z/.luxd/configs/chains/C/config.json

echo ""
echo "=== Import Complete ==="
echo "You can now start the node normally with:"
echo "  $LUXD --network-id=96369"
echo ""
echo "Or use the CLI:"
echo "  lux node start --network-id 96369"
echo ""