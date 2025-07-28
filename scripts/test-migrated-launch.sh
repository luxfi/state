#!/bin/bash

# Test launching luxd with migrated data
echo "=== Testing C-Chain VM with Migrated Data ==="

cd /home/z/work/lux/genesis

# Kill any existing processes
pkill -f luxd || true
sleep 2

# Clear fresh data dir
rm -rf runtime/test-migrated
mkdir -p runtime/test-migrated

# Create directories for C-Chain blockchain ID
C_CHAIN_ID="QJ1f5XDaMmEBAMtcoNFM5dhqWVwyqnb95uCEvCmbxah3iXmuS"
mkdir -p runtime/test-migrated/chainData/$C_CHAIN_ID

# Copy the migrated data
echo "Copying migrated chaindata to C-Chain location..."
cp -r /tmp/migrated-chaindata/pebbledb runtime/test-migrated/chainData/$C_CHAIN_ID/db

# Check if Snowman state exists and copy it
if [ -d "/tmp/snowman-state/pebbledb" ]; then
    echo "Copying Snowman consensus state..."
    mkdir -p runtime/test-migrated/chains/$C_CHAIN_ID
    cp -r /tmp/snowman-state/pebbledb runtime/test-migrated/chains/$C_CHAIN_ID/snowmanDB
else
    echo "Warning: Snowman state not found, will be created fresh"
fi

# Launch luxd with POA settings
echo "Launching luxd..."
../node/build/luxd \
    --network-id=96369 \
    --data-dir=runtime/test-migrated \
    --sybil-protection-enabled=false \
    --public-ip=127.0.0.1 \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9631 \
    --log-level=debug &

PID=$!
echo "luxd PID: $PID"

# Wait for node to start
echo "Waiting for node to start..."
sleep 10

# Check C-Chain block height
echo -e "\n=== Checking C-Chain Block Height ==="
curl -s -X POST --data '{
    "jsonrpc": "2.0",
    "method": "eth_blockNumber",
    "params": [],
    "id": 1
}' -H 'Content-Type: application/json' http://localhost:9630/ext/bc/C/rpc | jq

echo -e "\n=== Checking Latest Block ==="
curl -s -X POST --data '{
    "jsonrpc": "2.0",
    "method": "eth_getBlockByNumber",
    "params": ["latest", false],
    "id": 1
}' -H 'Content-Type: application/json' http://localhost:9630/ext/bc/C/rpc | jq

# Give time to check logs
echo -e "\nPress Ctrl+C to stop..."
wait $PID