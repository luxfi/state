#!/bin/bash

# Launch Lux Mainnet (96369) with proper genesis

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GENESIS_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
LUX_CLI="${GENESIS_ROOT}/bin/lux-cli"
LUXD_PATH="${GENESIS_ROOT}/../node/build/luxd"
GENESIS_BUILDER="${GENESIS_ROOT}/bin/genesis-builder"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Lux Network Mainnet Launch ===${NC}"
echo "Network ID: 96369"
echo "Chain ID: 96369"
echo ""

# Step 1: Generate Genesis with C-Chain Import
echo -e "${YELLOW}Step 1: Generating mainnet genesis...${NC}"

GENESIS_FILE="${GENESIS_ROOT}/genesis_mainnet_96369.json"
CCHAIN_GENESIS="${GENESIS_ROOT}/data/unified-genesis/lux-mainnet-96369/genesis.json"
CCHAIN_ALLOCS="${GENESIS_ROOT}/data/unified-genesis/lux-mainnet-96369/allocations_combined.json"

# Check if C-Chain data exists
if [ ! -f "$CCHAIN_GENESIS" ] || [ ! -f "$CCHAIN_ALLOCS" ]; then
    echo -e "${RED}Error: C-Chain data not found!${NC}"
    echo "Expected files:"
    echo "  - $CCHAIN_GENESIS"
    echo "  - $CCHAIN_ALLOCS"
    echo ""
    echo "Please ensure you have extracted the 96369 chain data."
    exit 1
fi

# Generate genesis with real validator keys
# NOTE: You need to replace the placeholder keys in mainnet-validators.json with real BLS keys
if [ ! -f "$GENESIS_FILE" ]; then
    echo "Generating genesis file..."
    "$GENESIS_BUILDER" \
        --network mainnet \
        --import-cchain "$CCHAIN_GENESIS" \
        --import-allocations "$CCHAIN_ALLOCS" \
        --validators "${GENESIS_ROOT}/configs/mainnet-validators-real.json" \
        --output "$GENESIS_FILE"
fi

# Step 2: Launch Primary Network
echo -e "${YELLOW}Step 2: Starting Lux Network...${NC}"

# Check if luxd exists
if [ ! -f "$LUXD_PATH" ]; then
    echo -e "${RED}Error: luxd not found at $LUXD_PATH${NC}"
    echo "Please build it first: cd ../node && go build -o build/luxd ./main.go"
    exit 1
fi

# Create data directory
DATA_DIR="${HOME}/.lux-mainnet"
mkdir -p "$DATA_DIR"

# Copy genesis
cp "$GENESIS_FILE" "$DATA_DIR/genesis.json"

# Start with lux-cli
echo "Starting network with lux-cli..."
"$LUX_CLI" network start \
    --luxd-path "$LUXD_PATH" \
    --network-id 96369 \
    --custom-network-genesis "$GENESIS_FILE" \
    --node-config "${SCRIPT_DIR}/node-config-mainnet.json"

# Wait for network to be ready
echo "Waiting for network to be ready..."
sleep 30

# Check status
"$LUX_CLI" network status

# Step 3: Deploy L2 Subnets
echo -e "${YELLOW}Step 3: Deploying L2 subnets...${NC}"

# Deploy Zoo L2
if [ -f "${GENESIS_ROOT}/data/unified-genesis/zoo-mainnet-200200/genesis.json" ]; then
    echo "Deploying Zoo L2 (200200)..."
    "$LUX_CLI" l2 create zoo \
        --evm \
        --chain-id 200200 \
        --custom-subnet-evm-genesis "${GENESIS_ROOT}/data/unified-genesis/zoo-mainnet-200200/genesis.json"
    
    "$LUX_CLI" l2 deploy zoo --mainnet
fi

# Deploy SPC L2
if [ -f "${GENESIS_ROOT}/data/unified-genesis/spc-mainnet-36911/genesis.json" ]; then
    echo "Deploying SPC L2 (36911)..."
    "$LUX_CLI" l2 create spc \
        --evm \
        --chain-id 36911 \
        --custom-subnet-evm-genesis "${GENESIS_ROOT}/data/unified-genesis/spc-mainnet-36911/genesis.json"
    
    "$LUX_CLI" l2 deploy spc --mainnet
fi

echo -e "${GREEN}=== Mainnet Launch Complete ===${NC}"
echo ""
echo "Network Information:"
echo "-------------------"
"$LUX_CLI" network list

echo ""
echo "RPC Endpoints:"
echo "- C-Chain: http://localhost:9650/ext/bc/C/rpc"

# Get L2 endpoints
for subnet in zoo spc; do
    if "$LUX_CLI" l2 list | grep -q "$subnet"; then
        BLOCKCHAIN_ID=$("$LUX_CLI" l2 describe "$subnet" --json | jq -r '.blockchainID')
        echo "- $subnet: http://localhost:9650/ext/bc/$BLOCKCHAIN_ID/rpc"
    fi
done

echo ""
echo -e "${BLUE}Important: This is MAINNET!${NC}"
echo "Ensure you have:"
echo "1. Real validator BLS keys in configs/mainnet-validators-real.json"
echo "2. Proper network security configured"
echo "3. Monitoring and alerting set up"
echo "4. Backup procedures in place"