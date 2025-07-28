#!/bin/bash

# Launch luxd with chaindata from genesis output directory
set -e

echo "=== Launching Lux Mainnet with Historic Chain Data ==="
echo ""
echo "Chain Data Location: genesis/output/mainnet/C/chaindata-namespaced"
echo "Network ID: 96369"
echo "Blocks found: 14,644 (from namespace extraction)"
echo ""

WORK_DIR="/home/z/work/lux"
GENESIS_DIR="$WORK_DIR/genesis"
DATA_DIR="$GENESIS_DIR/runtime/mainnet"

# Clean up
pkill -f luxd 2>/dev/null || true
sleep 2

# Create runtime directory
mkdir -p $DATA_DIR/chainData/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db

# Link the namespaced chaindata (extracted with state data)
ln -sf $GENESIS_DIR/output/mainnet/C/chaindata-namespaced $DATA_DIR/chainData/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/db/pebbledb

# Create metadata
cat > $DATA_DIR/chainData/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/metadata.json << 'EOF'
{
  "version": "1.0",
  "chainID": "96369",
  "blockchainID": "X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3"
}
EOF

echo "Starting Lux node..."
cd $WORK_DIR

./node/build/luxd \
    --network-id=96369 \
    --http-port=9630 \
    --staking-port=9651 \
    --data-dir=$DATA_DIR \
    --chain-data-dir=$DATA_DIR/chainData \
    --sybil-protection-enabled=false \
    --api-admin-enabled=true \
    --log-level=info 2>&1 | tee $GENESIS_DIR/runtime/mainnet.log &

PID=$!
echo ""
echo "Lux node started with PID: $PID"
echo "Log: $GENESIS_DIR/runtime/mainnet.log"
echo ""
echo "Wait a moment, then test with:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"eth_chainId\",\"params\":[]}' http://localhost:9630/ext/bc/C/rpc"