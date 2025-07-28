#!/bin/bash

# Launch luxd with debug logging and proper genesis
set -e

echo "=== Launching Lux Mainnet in DEBUG Mode ==="
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/mainnet-debug"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Remove old data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

# Create proper C-Chain genesis
mkdir -p $DATA_DIR/configs/chains/C
cp $GENESIS_DIR/cchain-genesis.json $DATA_DIR/configs/chains/C/genesis.json

echo "Starting Lux node with debug logging..."
cd $WORK_DIR

# Set debug environment variables
export AVALANCHE_NETWORK_ID=96369

# Launch with verbose logging
./node/build/luxd \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --sybil-protection-enabled=false \
    --api-admin-enabled=true \
    --log-level=debug \
    --log-display-level=debug 2>&1 | tee $GENESIS_DIR/runtime/mainnet-debug.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/mainnet-debug.log"
echo ""
echo "Watch the log with:"
echo "  tail -f $GENESIS_DIR/runtime/mainnet-debug.log | grep -E '(C Chain|genesis|block|height)'"