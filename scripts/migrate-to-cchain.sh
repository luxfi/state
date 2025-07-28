#!/bin/bash

# Migrate historic chain data to C-Chain location
set -e

echo "=== Migrating Historic Chain Data to C-Chain ==="
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/migrated-cchain"
SOURCE_CHAINDATA="$GENESIS_DIR/output/mainnet/C/chaindata-namespaced"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Remove old data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Copy genesis files
cp -r $GENESIS_DIR/configs/mainnet/* $DATA_DIR/

# Extract genesis from our chaindata to ensure we use the right one
echo "Extracting genesis from historic chaindata..."
cd $GENESIS_DIR

# Use the cchain-genesis.json we created earlier
if [ -f "$GENESIS_DIR/cchain-genesis.json" ]; then
    echo "Using existing cchain-genesis.json"
    cp $GENESIS_DIR/cchain-genesis.json $DATA_DIR/C/genesis.json
else
    echo "Error: cchain-genesis.json not found"
    exit 1
fi

echo ""
echo "Starting Lux node to initialize database structure..."
cd $WORK_DIR

# Launch node briefly to create database structure
./node/build/luxd \
    --dev \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --api-admin-enabled=true \
    --log-level=info > $GENESIS_DIR/runtime/migrate-init.log 2>&1 &

PID=$!
echo "Waiting for node to initialize (PID: $PID)..."

# Wait for C-Chain to initialize
for i in {1..30}; do
    if curl -s -X POST -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        http://localhost:9630/ext/bc/C/rpc 2>/dev/null | grep -q result; then
        echo "C-Chain initialized successfully"
        break
    fi
    sleep 1
done

# Kill the node
kill $PID 2>/dev/null || true
sleep 2

echo ""
echo "Node stopped. Now migrating historic chain data..."

# The C-Chain blockchain ID
CCHAIN_ID="X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3"

# Create the chainData directory for C-Chain if it doesn't exist
mkdir -p $DATA_DIR/chainData/$CCHAIN_ID/db

# Copy our historic chaindata
echo "Copying historic chaindata to C-Chain location..."
cp -r $SOURCE_CHAINDATA $DATA_DIR/chainData/$CCHAIN_ID/db/

echo ""
echo "Migration complete!"
echo "Data directory: $DATA_DIR"
echo "C-Chain data: $DATA_DIR/chainData/$CCHAIN_ID/db/chaindata-namespaced"
echo ""
echo "Now launching with migrated data..."

# Launch with migrated data
./node/build/luxd \
    --dev \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --api-admin-enabled=true \
    --log-level=debug 2>&1 | tee $GENESIS_DIR/runtime/migrated-cchain.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/migrated-cchain.log"
echo ""
echo "Wait for initialization, then test with:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_blockNumber\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc"
echo ""
echo "Check treasury balance:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getBalance\",\"params\":[\"0x9011e888251ab053b7bd1cdb598db4f9ded94714\",\"latest\"]}' http://localhost:9630/ext/bc/C/rpc"