#!/bin/bash
set -e

echo "Launching Lux Network with historical blockchain data..."

# Set paths
LUXD="../node/build/luxd"
GENESIS="genesis-mainnet-96369.json"
BASE_DIR="$HOME/.luxd"
HTTP_PORT_BASE=9630
STAKING_PORT_BASE=9651

# Clean previous runs
echo "Cleaning previous network..."
pkill -f "luxd.*network-id=96369" || true
sleep 2
rm -rf $BASE_DIR

# Create directories for each node
for i in {1..11}; do
    mkdir -p $BASE_DIR/node$i/staking
    mkdir -p $BASE_DIR/node$i/db
done

# Copy C-Chain data to first node (will sync to others)
echo "Copying C-Chain historical data from 96369..."
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    # The chaindata is in pebbledb format, but luxd expects it in a specific location
    # We need to copy it to the C-Chain data directory
    mkdir -p $BASE_DIR/node1/db/C
    cp -r chaindata/lux-mainnet-96369/db/pebbledb $BASE_DIR/node1/db/C/
    echo "✓ Copied 96369 C-Chain data to node1"
else
    echo "⚠️  Warning: 96369 chaindata not found"
fi

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

# Start nodes
echo "Starting nodes..."
for i in {1..11}; do
    echo "Starting node$i (${NODE_IDS[$i]})..."
    
    # For node1, we use the chaindata directly
    # For others, they will sync from node1
    $LUXD \
        --network-id=96369 \
        --data-dir=$BASE_DIR/node$i \
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
echo "✅ Network launched with historical data!"
echo ""
echo "Node endpoints:"
for i in {1..11}; do
    echo "  Node$i: http://localhost:$((HTTP_PORT_BASE + i - 1))"
done
echo ""
echo "Note: Node1 has the historical 96369 C-Chain data"
echo "Other nodes will sync from it"
echo ""
echo "To deploy Zoo L2:"
echo "  ./scripts/deploy-zoo-subnet.sh"