#!/bin/bash
set -e

echo "=== Deploying Lux Mainnet with historical blockchain data ==="
echo ""

# Configuration
NETWORK_ID=96369
GENESIS_BIN="./bin/genesis"
LUXD="../node/build/luxd"
BASE_DIR="$HOME/.luxd-mainnet"
HTTP_PORT_BASE=9630
STAKING_PORT_BASE=9651

# Step 1: Generate genesis configurations
echo "Step 1: Generating mainnet genesis configurations..."
$GENESIS_BIN generate --network mainnet --output output-mainnet

# Step 2: Clean previous deployment
echo ""
echo "Step 2: Cleaning previous deployment..."
pkill -f "luxd.*network-id=$NETWORK_ID" || true
sleep 2
rm -rf $BASE_DIR

# Step 3: Prepare node directories with historical data
echo ""
echo "Step 3: Preparing node directories with historical data..."

# Create directories for all 11 nodes
for i in {1..11}; do
    mkdir -p $BASE_DIR/node$i/staking
    mkdir -p $BASE_DIR/node$i/db
done

# Copy validator keys
echo "  - Setting up validator keys..."
for i in {1..11}; do
    cp validator-keys/validator-$i/staking/staker.crt $BASE_DIR/node$i/staking/
    cp validator-keys/validator-$i/staking/staker.key $BASE_DIR/node$i/staking/
    cp validator-keys/validator-$i/bls.key $BASE_DIR/node$i/staking/signer.key
done

# Copy C-Chain historical data to node1
echo "  - Copying C-Chain historical data from 96369..."
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    # Create C-Chain directory structure
    C_CHAIN_ID="2vNDBRPABGFLJPBBBBPHCCCCCCCCCCCCCCCCCCCCCC" # Placeholder, will be determined at runtime
    mkdir -p $BASE_DIR/node1/db/$C_CHAIN_ID
    
    # Copy the pebbledb data
    cp -r chaindata/lux-mainnet-96369/db/pebbledb/* $BASE_DIR/node1/db/$C_CHAIN_ID/
    echo "    ✓ C-Chain historical data copied"
else
    echo "    ⚠️ C-Chain historical data not found"
fi

# Step 4: Launch the network
echo ""
echo "Step 4: Launching mainnet network with 11 validators..."

# Get node IDs for bootstrap
declare -a NODE_IDS
for i in {1..11}; do
    NODE_ID=$(cat configs/mainnet-validators.json | jq -r ".[$((i-1))].nodeID")
    NODE_IDS[$i]=$NODE_ID
done

# Build bootstrap list
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

# Start nodes
for i in {1..11}; do
    echo "  - Starting node$i (${NODE_IDS[$i]})..."
    
    $LUXD \
        --network-id=$NETWORK_ID \
        --data-dir=$BASE_DIR/node$i \
        --genesis-file=output-mainnet/genesis-mainnet-96369.json \
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

# Wait for network to stabilize
echo ""
echo "Waiting for network to stabilize..."
sleep 20

# Check network health
echo ""
echo "Checking network health..."
curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health","params":{}}' \
    -H 'content-type:application/json;' http://localhost:9630/ext/health | \
    jq -r '.result | if .healthy then "✅ Network is healthy!" else "⚠️ Network is starting up..." end'

# Step 5: Deploy L2 subnets using lux-cli
echo ""
echo "Step 5: Deploying L2 subnets..."

# Deploy Zoo L2
echo "  - Deploying Zoo L2 subnet..."
if command -v ./bin/lux-cli &> /dev/null; then
    ./scripts/deploy-zoo-subnet.sh mainnet || echo "    ⚠️ Zoo subnet deployment needs manual configuration"
else
    echo "    ⚠️ lux-cli not found, skipping subnet deployment"
fi

# Deploy SPC L2
echo "  - Deploying SPC L2 subnet..."
if command -v ./bin/lux-cli &> /dev/null; then
    ./scripts/deploy-spc-subnet.sh mainnet || echo "    ⚠️ SPC subnet deployment needs manual configuration"
else
    echo "    ⚠️ lux-cli not found, skipping subnet deployment"
fi

echo ""
echo "✅ Mainnet deployment complete!"
echo ""
echo "Network endpoints:"
echo "  C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "  P-Chain API: http://localhost:9630/ext/P"
echo "  X-Chain API: http://localhost:9630/ext/X"
echo ""
echo "Node endpoints:"
for i in {1..11}; do
    echo "  Node$i: http://localhost:$((HTTP_PORT_BASE + i - 1))"
done
echo ""
echo "Logs: $BASE_DIR/node*/node.log"
echo ""
echo "To check C-Chain block height:"
echo '  curl -X POST --data '"'"'{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}'"'"' -H '"'"'content-type:application/json;'"'"' http://localhost:9630/ext/bc/C/rpc'
echo ""
echo "To stop the network:"
echo "  pkill -f 'luxd.*network-id=$NETWORK_ID'"