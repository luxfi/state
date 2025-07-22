#!/bin/bash

# Launch Lux Network using lux-cli with proper genesis and L2 deployment

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LUX_CLI="${PROJECT_ROOT}/cli/bin/avalanche"
GENESIS_BUILDER="${PROJECT_ROOT}/genesis/cmd/genesis-builder/genesis-builder"
NETWORK_NAME="lux-mainnet"
NETWORK_TYPE="mainnet"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --testnet)
            NETWORK_NAME="lux-testnet"
            NETWORK_TYPE="testnet"
            shift
            ;;
        --local)
            NETWORK_NAME="lux-local"
            NETWORK_TYPE="local"
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --testnet    Launch testnet instead of mainnet"
            echo "  --local      Launch local development network"
            echo "  --help       Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}=== Lux Network Launch Script ===${NC}"
echo "Network: $NETWORK_NAME ($NETWORK_TYPE)"
echo ""

# Step 1: Build genesis if needed
echo -e "${YELLOW}Step 1: Preparing genesis data...${NC}"
GENESIS_FILE="${PROJECT_ROOT}/genesis/genesis_${NETWORK_TYPE}.json"

if [ ! -f "$GENESIS_FILE" ]; then
    echo "Genesis file not found. Building..."
    cd "${PROJECT_ROOT}/genesis"
    
    # Build genesis builder
    if [ ! -f "cmd/genesis-builder/genesis-builder" ]; then
        echo "Building genesis builder..."
        cd cmd/genesis-builder
        go build
        cd ../..
    fi
    
    # Generate genesis with existing C-Chain data import for mainnet
    if [ "$NETWORK_TYPE" = "mainnet" ]; then
        # Import existing 96369 C-Chain data
        CCHAIN_GENESIS="${PROJECT_ROOT}/genesis/data/unified-genesis/lux-mainnet-96369/genesis.json"
        CCHAIN_ALLOCS="${PROJECT_ROOT}/genesis/data/unified-genesis/lux-mainnet-96369/allocations_combined.json"
        
        if [ -f "$CCHAIN_GENESIS" ] && [ -f "$CCHAIN_ALLOCS" ]; then
            echo "Importing existing C-Chain data..."
            ./cmd/genesis-builder/genesis-builder \
                --network mainnet \
                --import-cchain "$CCHAIN_GENESIS" \
                --import-allocations "$CCHAIN_ALLOCS" \
                --validators configs/mainnet-validators.json \
                --output "$GENESIS_FILE"
        else
            echo "Warning: C-Chain data not found, generating fresh genesis"
            ./cmd/genesis-builder/genesis-builder \
                --network mainnet \
                --validators configs/mainnet-validators.json \
                --output "$GENESIS_FILE"
        fi
    else
        ./cmd/genesis-builder/genesis-builder \
            --network "$NETWORK_TYPE" \
            --output "$GENESIS_FILE"
    fi
fi

# Step 2: Create network with lux-cli
echo -e "${YELLOW}Step 2: Creating primary network with lux-cli...${NC}"

# Check if network already exists
if $LUX_CLI network list | grep -q "$NETWORK_NAME"; then
    echo "Network $NETWORK_NAME already exists. Cleaning up..."
    $LUX_CLI network clean "$NETWORK_NAME" --force
fi

# Create network configuration
cat > /tmp/lux-network-config.json <<EOF
{
  "network_name": "$NETWORK_NAME",
  "network_id": $([ "$NETWORK_TYPE" = "mainnet" ] && echo "96369" || echo "96368"),
  "genesis_file": "$GENESIS_FILE",
  "num_validators": $([ "$NETWORK_TYPE" = "local" ] && echo "5" || echo "11"),
  "bootstrap_nodes": $([ "$NETWORK_TYPE" = "mainnet" ] && echo "true" || echo "false")
}
EOF

# Start the network
echo "Starting network..."
$LUX_CLI network start "$NETWORK_NAME" \
    --genesis-file "$GENESIS_FILE" \
    --num-nodes $([ "$NETWORK_TYPE" = "local" ] && echo "5" || echo "1") \
    --node-config "${SCRIPT_DIR}/node-config-${NETWORK_TYPE}.json" \
    $([ "$NETWORK_TYPE" = "mainnet" ] && echo "--bootstrap-ids $(cat ${SCRIPT_DIR}/bootstrap-ids.txt)" || echo "")

# Wait for network to be ready
echo "Waiting for network to be ready..."
sleep 10

# Get network status
$LUX_CLI network status "$NETWORK_NAME"

# Step 3: Deploy L2 subnets
echo -e "${YELLOW}Step 3: Deploying L2 subnets...${NC}"

# Function to deploy a subnet
deploy_subnet() {
    local SUBNET_NAME=$1
    local CHAIN_ID=$2
    local VM_TYPE=$3
    local GENESIS_PATH=$4
    
    echo -e "${BLUE}Deploying $SUBNET_NAME (Chain ID: $CHAIN_ID)...${NC}"
    
    # Create subnet configuration
    cat > /tmp/${SUBNET_NAME}-config.json <<EOF
{
  "subnet_name": "$SUBNET_NAME",
  "chain_id": $CHAIN_ID,
  "vm_type": "$VM_TYPE",
  "vm_version": "latest",
  "genesis_file": "$GENESIS_PATH"
}
EOF
    
    # Create the subnet
    $LUX_CLI subnet create "$SUBNET_NAME" \
        --evm \
        --chain-id "$CHAIN_ID" \
        --genesis "$GENESIS_PATH"
    
    # Deploy to network
    $LUX_CLI subnet deploy "$SUBNET_NAME" \
        --network "$NETWORK_NAME" \
        --threshold 1
    
    # Get subnet info
    $LUX_CLI subnet describe "$SUBNET_NAME"
}

# Deploy L2 subnets based on network type
if [ "$NETWORK_TYPE" = "mainnet" ]; then
    # Zoo L2
    ZOO_GENESIS="${PROJECT_ROOT}/genesis/data/unified-genesis/zoo-mainnet-200200/genesis.json"
    if [ -f "$ZOO_GENESIS" ]; then
        deploy_subnet "zoo-mainnet" 200200 "subnet-evm" "$ZOO_GENESIS"
    fi
    
    # SPC L2
    SPC_GENESIS="${PROJECT_ROOT}/genesis/data/unified-genesis/spc-mainnet-36911/genesis.json"
    if [ -f "$SPC_GENESIS" ]; then
        deploy_subnet "spc-mainnet" 36911 "subnet-evm" "$SPC_GENESIS"
    fi
    
    # Hanzo L2 (prepared but not deployed yet)
    echo -e "${YELLOW}Hanzo L2 prepared but not deployed (awaiting deployment decision)${NC}"
    
elif [ "$NETWORK_TYPE" = "testnet" ]; then
    # Deploy testnet L2s
    deploy_subnet "zoo-testnet" 200201 "subnet-evm" "${PROJECT_ROOT}/genesis/genesis_zoo-testnet.json"
    deploy_subnet "hanzo-testnet" 36962 "subnet-evm" "${PROJECT_ROOT}/genesis/genesis_hanzo-testnet.json"
fi

# Step 4: Display network information
echo -e "${GREEN}=== Network Launch Complete ===${NC}"
echo ""
echo "Network Information:"
echo "-------------------"
$LUX_CLI network list
echo ""
echo "RPC Endpoints:"
echo "- C-Chain: http://localhost:9650/ext/bc/C/rpc"
echo ""

# Show subnet endpoints
if [ "$NETWORK_TYPE" != "local" ]; then
    echo "L2 Subnet Endpoints:"
    $LUX_CLI subnet list | while read subnet; do
        if [[ $subnet == *"zoo"* ]] || [[ $subnet == *"spc"* ]] || [[ $subnet == *"hanzo"* ]]; then
            BLOCKCHAIN_ID=$($LUX_CLI subnet describe "$subnet" --json | jq -r '.blockchain_id')
            echo "- $subnet: http://localhost:9650/ext/bc/$BLOCKCHAIN_ID/rpc"
        fi
    done
fi

echo ""
echo -e "${GREEN}Network is ready!${NC}"
echo ""
echo "Useful commands:"
echo "- View status: $LUX_CLI network status $NETWORK_NAME"
echo "- View logs: $LUX_CLI network logs $NETWORK_NAME"
echo "- Stop network: $LUX_CLI network stop $NETWORK_NAME"
echo "- Clean up: $LUX_CLI network clean $NETWORK_NAME"