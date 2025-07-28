#!/bin/bash

# Direct state import approach - launch luxd and inject state data

set -e

LUXD="/home/z/work/lux/node/build/luxd"
DATA_DIR="runtime/state-import"
PORT=9630

echo "=== Direct State Import Test ==="

# Clean up
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Create config
cat > $DATA_DIR/config.json <<EOF
{
  "network-id": "96369",
  "http-port": $PORT,
  "staking-enabled": false,
  "health-check-frequency": "30s",
  "log-level": "debug",
  "api-admin-enabled": true,
  "api-eth-enabled": true,
  "api-debug-enabled": true,
  "index-enabled": false,
  "pruning-enabled": false
}
EOF

# Copy genesis files
cp -r /home/z/work/lux/genesis/configs/mainnet/* $DATA_DIR/

echo "Starting luxd on port $PORT..."
$LUXD --data-dir=$DATA_DIR --config-file=$DATA_DIR/config.json &
LUXD_PID=$!

echo "Waiting for node to start..."
sleep 15

# Check current block height
echo -e "\nChecking initial block height..."
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:$PORT/ext/bc/C/rpc | jq .

# Get the C-Chain blockchain ID
echo -e "\nGetting C-Chain blockchain ID..."
CCHAIN_ID=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"platform.getBlockchains","params":[]}' \
    http://localhost:$PORT/ext/bc/P | jq -r '.result.blockchains[] | select(.name=="C-Chain") | .id')

echo "C-Chain blockchain ID: $CCHAIN_ID"

# The database should be at:
DB_PATH="$DATA_DIR/db/network-96369/v1.4.5"
echo -e "\nNetwork database path: $DB_PATH"

# Check if the database exists
if [ -d "$DB_PATH" ]; then
    echo "Network database exists"
    ls -la "$DB_PATH"
else
    echo "Network database not found!"
fi

echo -e "\nNode is running. Press Ctrl+C to stop."
echo "Next steps:"
echo "1. Find where C-Chain stores its data within the network DB"
echo "2. Copy the state data to that location"
echo "3. Restart the node to see if it recognizes the state"

wait $LUXD_PID