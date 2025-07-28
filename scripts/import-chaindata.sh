#!/bin/bash

# This script launches luxd with a clean C-Chain and then imports historic data

set -e

LUXD="/home/z/work/lux/node/build/luxd"
DATA_DIR="runtime/import-test"
SUBNET_DB="output/mainnet/C/chaindata-prefixed"

echo "=== Starting chain data import process ==="

# Clean up any existing data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Create node configuration
cat > $DATA_DIR/config.json <<EOF
{
  "network-id": "96369",
  "http-port": 9630,
  "staking-enabled": false,
  "health-check-frequency": "30s",
  "chain-config-dir": "/home/z/work/lux/genesis/configs/mainnet",
  "log-level": "info",
  "api-admin-enabled": true,
  "api-eth-enabled": true,
  "api-debug-enabled": true,
  "index-enabled": false,
  "pruning-enabled": false
}
EOF

# Copy genesis files
cp -r ../configs/mainnet/* $DATA_DIR/

echo "Starting luxd..."
$LUXD --data-dir=$DATA_DIR --config-file=$DATA_DIR/config.json &
LUXD_PID=$!

echo "Waiting for node to start..."
sleep 10

# Check if C-Chain is running
curl -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc

echo -e "\n\nNode started. C-Chain should be at block 0."
echo "Now we need to import the historic state data..."

# The challenge is that we have state data but no block headers
# We need to reconstruct the blocks or import the state directly

echo "Press Ctrl+C to stop"
wait $LUXD_PID