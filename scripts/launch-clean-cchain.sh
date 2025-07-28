#!/bin/bash

# Launch luxd with clean C-Chain for debugging
set -e

echo "=== Launching Clean C-Chain for Debugging ==="
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/clean-cchain"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Remove old data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Copy genesis files
cp -r $GENESIS_DIR/configs/mainnet/* $DATA_DIR/

echo "Starting clean Lux node..."
echo "Data directory: $DATA_DIR"
echo ""

cd $WORK_DIR

# Launch with dev mode for easy mining
./node/build/luxd \
    --dev \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --api-admin-enabled=true \
    --log-level=info 2>&1 | tee $GENESIS_DIR/runtime/clean-cchain.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/clean-cchain.log"
echo ""
echo "Wait for initialization, then test with:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_blockNumber\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc"
echo ""
echo "Check account balance:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_getBalance\",\"params\":[\"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC\",\"latest\"]}' http://localhost:9630/ext/bc/C/rpc"