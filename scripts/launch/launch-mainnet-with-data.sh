#!/bin/bash
set -e

echo "=== Launching Lux Mainnet with Historical Chain Data ==="
echo ""

# Configuration
NETWORK_ID=96369
LUXD="../node/build/luxd"
GENESIS="output-mainnet/genesis-mainnet-96369.json"
BASE_DIR="$HOME/.luxd-mainnet"
CHAIN_DATA_DIR="$PWD/chaindata"
HTTP_PORT_BASE=9630
STAKING_PORT_BASE=9651

# Clean previous runs
echo "Cleaning previous network..."
pkill -f "luxd.*network-id=$NETWORK_ID" || true
sleep 2
rm -rf $BASE_DIR

# Create directories for each node
echo "Creating node directories..."
for i in {1..11}; do
    mkdir -p $BASE_DIR/node$i/staking
done

# Copy validator keys
echo "Setting up validator keys..."
for i in {1..11}; do
    cp validator-keys/validator-$i/staking/staker.crt $BASE_DIR/node$i/staking/
    cp validator-keys/validator-$i/staking/staker.key $BASE_DIR/node$i/staking/
    cp validator-keys/validator-$i/bls.key $BASE_DIR/node$i/staking/signer.key
done

# Get node IDs for bootstrap
declare -a NODE_IDS
for i in {1..11}; do
    NODE_ID=$(cat configs/mainnet-validators.json | jq -r ".[$((i-1))].nodeID")
    NODE_IDS[$i]=$NODE_ID
done

# Build bootstrap node list
BOOTSTRAP_IPS=""
BOOTSTRAP_IDS=""
for i in {1..11}; do
    if [ -n "$BOOTSTRAP_IPS" ]; then
        BOOTSTRAP_IPS="${BOOTSTRAP_IPS},"
        BOOTSTRAP_IDS="${BOOTSTRAP_IDS},"
    fi
    BOOTSTRAP_IPS="${BOOTSTRAP_IPS}127.0.0.1:$((STAKING_PORT_BASE + i - 1))"
    BOOTSTRAP_IDS="${BOOTSTRAP_IDS}${NODE_IDS[$i]}"
done

# Start nodes with chain data
echo "Starting nodes with historical chain data..."
for i in {1..11}; do
    echo "Starting node$i (${NODE_IDS[$i]})..."
    
    # For node1, use the existing chain data
    if [ $i -eq 1 ]; then
        echo "  - Node1 will use historical chain data from $CHAIN_DATA_DIR"
        EXTRA_ARGS="--chain-data-dir=$CHAIN_DATA_DIR"
    else
        EXTRA_ARGS=""
    fi
    
    $LUXD \
        --network-id=$NETWORK_ID \
        --data-dir=$BASE_DIR/node$i \
        --genesis-file=$GENESIS \
        --http-host=0.0.0.0 \
        --http-port=$((HTTP_PORT_BASE + i - 1)) \
        --staking-port=$((STAKING_PORT_BASE + i - 1)) \
        --bootstrap-ips="$BOOTSTRAP_IPS" \
        --bootstrap-ids="$BOOTSTRAP_IDS" \
        --log-level=info \
        $EXTRA_ARGS \
        > $BASE_DIR/node$i/node.log 2>&1 &
    
    echo $! > $BASE_DIR/node$i/node.pid
    sleep 1
done

echo ""
echo "✅ 11-node mainnet launched with historical data!"
echo ""
echo "Waiting for network to bootstrap..."
sleep 20

# Check network health
echo "Checking network health..."
HEALTH=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health","params":{}}' \
    -H 'content-type:application/json;' http://localhost:9630/ext/health | \
    jq -r '.result.healthy')

if [ "$HEALTH" = "true" ]; then
    echo "✅ Network is healthy!"
else
    echo "⚠️  Network is still bootstrapping..."
fi

# Check C-Chain block height
echo ""
echo "Checking C-Chain status..."
BLOCK_HEIGHT=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    -H 'content-type:application/json;' http://localhost:9630/ext/bc/C/rpc | \
    jq -r '.result')

echo "C-Chain block height: $BLOCK_HEIGHT"

if [ "$BLOCK_HEIGHT" != "0x0" ] && [ "$BLOCK_HEIGHT" != "null" ]; then
    echo "✅ Historical chain data loaded successfully!"
else
    echo "⚠️  C-Chain appears to be starting from genesis"
fi

echo ""
echo "Network endpoints:"
echo "  C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "  P-Chain API: http://localhost:9630/ext/P"
echo "  X-Chain API: http://localhost:9630/ext/X"
echo ""
echo "To check logs:"
echo "  tail -f $BASE_DIR/node1/node.log"
echo ""
echo "To stop the network:"
echo "  pkill -f 'luxd.*network-id=$NETWORK_ID'"