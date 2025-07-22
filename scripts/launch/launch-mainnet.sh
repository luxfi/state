#!/bin/bash
set -e

echo "=== Launching Lux Mainnet with Historical Chain Data ==="
echo ""

# Configuration
NETWORK_ID=96369
LUXD="../node/build/luxd"
GENESIS="output-mainnet/genesis-mainnet-96369.json"
BASE_DIR="$HOME/.luxd-mainnet"
CHAIN_DATA_DIR="$(pwd)/chaindata"
HTTP_PORT_BASE=9630
STAKING_PORT_BASE=9651

# C-Chain blockchain ID
C_CHAIN_ID="dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"

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

# Copy chain data for node1
echo "Setting up historical chain data for node1..."
if [ -d "$CHAIN_DATA_DIR/lux-mainnet-96369/db/pebbledb" ]; then
    echo "  - Copying C-Chain data from $CHAIN_DATA_DIR/lux-mainnet-96369..."
    mkdir -p $BASE_DIR/node1/chainData/$C_CHAIN_ID
    cp -r $CHAIN_DATA_DIR/lux-mainnet-96369/db/pebbledb $BASE_DIR/node1/chainData/$C_CHAIN_ID/
    echo "  - C-Chain data copied successfully"
else
    echo "  ⚠️  Warning: C-Chain data not found at $CHAIN_DATA_DIR/lux-mainnet-96369/db/pebbledb"
fi

# Start nodes
echo "Starting 11-node network..."
for i in {1..11}; do
    echo "Starting node$i (${NODE_IDS[$i]})..."
    
    $LUXD \
        --network-id=$NETWORK_ID \
        --data-dir=$BASE_DIR/node$i \
        --chain-data-dir=$BASE_DIR/node$i/chainData \
        --genesis-file=$GENESIS \
        --http-host=0.0.0.0 \
        --http-port=$((HTTP_PORT_BASE + i - 1)) \
        --staking-port=$((STAKING_PORT_BASE + i - 1)) \
        --bootstrap-ips="$BOOTSTRAP_IPS" \
        --bootstrap-ids="$BOOTSTRAP_IDS" \
        --log-level=info \
        > $BASE_DIR/node$i/node.log 2>&1 &
    
    echo $! > $BASE_DIR/node$i/node.pid
    sleep 1
done

echo ""
echo "✅ 11-node mainnet launched!"
echo ""
echo "Waiting for network to bootstrap..."
sleep 20

# Check network health
echo "Checking network health..."
HEALTH=$(curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health","params":{}}' \
    -H 'content-type:application/json;' http://localhost:9630/ext/health | \
    jq -r '.result.healthy' 2>/dev/null || echo "false")

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
    jq -r '.result' 2>/dev/null || echo "null")

if [ "$BLOCK_HEIGHT" != "null" ] && [ "$BLOCK_HEIGHT" != "" ]; then
    echo "C-Chain block height: $BLOCK_HEIGHT"
    DECIMAL_HEIGHT=$((16#${BLOCK_HEIGHT#0x}))
    if [ $DECIMAL_HEIGHT -gt 0 ]; then
        echo "✅ Historical chain data loaded! Block height: $DECIMAL_HEIGHT"
    else
        echo "⚠️  C-Chain starting from genesis (block 0)"
    fi
else
    echo "⚠️  C-Chain not responding yet"
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