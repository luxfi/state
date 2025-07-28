#!/bin/bash

# Launch luxd with automining for mainnet
set -e

echo "=== Launching Lux Mainnet with Automining ==="
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/mainnet-automining"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Remove old data
rm -rf $DATA_DIR
mkdir -p $DATA_DIR

echo "Starting Lux node with automining enabled..."
cd $WORK_DIR

# Launch with automining
./node/build/luxd \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --sybil-protection-enabled=false \
    --api-admin-enabled=true \
    --log-level=info \
    --snow-sample-size=1 \
    --snow-quorum-size=1 \
    --snow-concurrent-repolls=1 \
    --snow-optimal-processing=1 \
    --snow-max-processing=1 \
    --snow-max-time-processing=100 \
    --min-stake-duration=1 \
    --tx-fee=1000000 \
    --create-asset-tx-fee=1000000 \
    --create-subnet-tx-fee=100000000 \
    --create-blockchain-tx-fee=100000000 2>&1 | tee $GENESIS_DIR/runtime/mainnet-automining.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/mainnet-automining.log"
echo ""
echo "Wait for initialization, then test with:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_blockNumber\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc"
echo ""
echo "Send a transaction to trigger block production:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_sendTransaction\",\"params\":[{\"from\":\"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC\",\"to\":\"0x9011e888251ab053b7bd1cdb598db4f9ded94714\",\"value\":\"0x1\"}]}' http://localhost:9630/ext/bc/C/rpc"