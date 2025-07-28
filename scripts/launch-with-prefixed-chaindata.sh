#!/bin/bash

# Launch luxd with prefixed chaindata
set -e

echo "=== Launching Lux with Prefixed Chaindata ==="
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/mainnet-prefixed"
CHAINDATA_SRC="$GENESIS_DIR/output/mainnet/C/chaindata-prefixed"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Remove old data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Create chain config directory
mkdir -p $DATA_DIR/chains/C

# Link the prefixed chaindata
ln -s $CHAINDATA_SRC $DATA_DIR/chains/C/db

# Copy genesis files
cp -r $GENESIS_DIR/configs/mainnet/* $DATA_DIR/

echo "Starting Lux node with prefixed chaindata..."
echo "Chaindata: $CHAINDATA_SRC"
echo ""

cd $WORK_DIR

# Launch with minimal parameters - use dev mode
./node/build/luxd \
    --dev \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --api-admin-enabled=true \
    --log-level=debug 2>&1 | tee $GENESIS_DIR/runtime/mainnet-prefixed.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/mainnet-prefixed.log"
echo ""
echo "Wait for initialization, then test with:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_blockNumber\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc"
echo ""
echo "Check account balance:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getBalance\",\"params\":[\"0x9011e888251ab053b7bd1cdb598db4f9ded94714\",\"latest\"]}' http://localhost:9630/ext/bc/C/rpc"