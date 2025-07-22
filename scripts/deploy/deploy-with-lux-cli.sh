#!/bin/bash

# Deploy Lux Network and L2s using lux-cli
# This script uses the actual lux-cli (avalanche CLI) commands

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LUX_CLI="${PROJECT_ROOT}/cli/bin/avalanche"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
NETWORK_TYPE="mainnet"
SKIP_PRIMARY=false
SKIP_L2=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --testnet)
            NETWORK_TYPE="testnet"
            shift
            ;;
        --local)
            NETWORK_TYPE="local"
            shift
            ;;
        --skip-primary)
            SKIP_PRIMARY=true
            shift
            ;;
        --skip-l2)
            SKIP_L2=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --testnet      Deploy testnet configuration"
            echo "  --local        Deploy local development network"
            echo "  --skip-primary Skip primary network deployment"
            echo "  --skip-l2      Skip L2 subnet deployment"
            echo "  --help         Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}=== Lux Network Deployment with lux-cli ===${NC}"
echo "Network Type: $NETWORK_TYPE"
echo ""

# Check if lux-cli exists
if [ ! -f "$LUX_CLI" ]; then
    echo -e "${RED}Error: lux-cli not found at $LUX_CLI${NC}"
    echo "Please build it first: cd cli && go build -o bin/avalanche"
    exit 1
fi

# Step 1: Deploy Primary Network
if [ "$SKIP_PRIMARY" = false ]; then
    echo -e "${YELLOW}Step 1: Starting Primary Network...${NC}"
    
    # Generate genesis if needed
    GENESIS_FILE="${PROJECT_ROOT}/genesis/genesis_${NETWORK_TYPE}.json"
    if [ ! -f "$GENESIS_FILE" ]; then
        echo "Generating genesis file..."
        cd "${PROJECT_ROOT}/genesis"
        if [ ! -f "cmd/genesis-builder/genesis-builder" ]; then
            cd cmd/genesis-builder && go build && cd ../..
        fi
        
        if [ "$NETWORK_TYPE" = "mainnet" ]; then
            # Import existing C-Chain data for mainnet
            ./cmd/genesis-builder/genesis-builder \
                --network mainnet \
                --import-cchain "data/unified-genesis/lux-mainnet-96369/genesis.json" \
                --import-allocations "data/unified-genesis/lux-mainnet-96369/allocations_combined.json" \
                --validators configs/mainnet-validators.json \
                --output "$GENESIS_FILE"
        else
            ./cmd/genesis-builder/genesis-builder \
                --network "$NETWORK_TYPE" \
                --output "$GENESIS_FILE"
        fi
    fi
    
    # Start network with lux-cli
    echo "Starting network..."
    cd "$PROJECT_ROOT/cli"
    
    # For local development
    if [ "$NETWORK_TYPE" = "local" ]; then
        ./bin/avalanche network start \
            --avalanchego-path ../node/build/luxd \
            --custom-network-genesis "$GENESIS_FILE"
    else
        # For mainnet/testnet with specific config
        CONFIG_FILE="${SCRIPT_DIR}/lux-cli-config-${NETWORK_TYPE}.json"
        if [ -f "$CONFIG_FILE" ]; then
            ./bin/avalanche --config "$CONFIG_FILE" network start
        else
            ./bin/avalanche network start
        fi
    fi
    
    # Wait for network
    echo "Waiting for network to be ready..."
    sleep 10
    
    # Check status
    ./bin/avalanche network status
fi

# Step 2: Deploy L2 Subnets
if [ "$SKIP_L2" = false ] && [ "$NETWORK_TYPE" != "local" ]; then
    echo -e "${YELLOW}Step 2: Deploying L2 Subnets...${NC}"
    cd "$PROJECT_ROOT/cli"
    
    # Function to create and deploy subnet
    deploy_l2() {
        local NAME=$1
        local CHAIN_ID=$2
        local GENESIS_PATH=$3
        
        echo -e "${BLUE}Deploying $NAME (Chain ID: $CHAIN_ID)...${NC}"
        
        # Check if subnet exists
        if ./bin/avalanche subnet list | grep -q "$NAME"; then
            echo "$NAME already exists, skipping..."
            return
        fi
        
        # Create subnet with Subnet-EVM
        ./bin/avalanche subnet create "$NAME" \
            --evm \
            --chain-id "$CHAIN_ID" \
            --custom-subnet-evm-genesis "$GENESIS_PATH"
        
        # Deploy subnet
        if [ "$NETWORK_TYPE" = "mainnet" ]; then
            # For mainnet, deploy with validators
            ./bin/avalanche subnet deploy "$NAME" \
                --mainnet \
                --validators "$(cat ${SCRIPT_DIR}/mainnet-validators.txt)"
        else
            # For testnet/local
            ./bin/avalanche subnet deploy "$NAME" \
                --local
        fi
        
        # Show info
        ./bin/avalanche subnet describe "$NAME"
    }
    
    # Deploy based on network type
    if [ "$NETWORK_TYPE" = "mainnet" ]; then
        # Zoo L2
        if [ -f "${PROJECT_ROOT}/genesis/data/unified-genesis/zoo-mainnet-200200/genesis.json" ]; then
            deploy_l2 "zoo" 200200 "${PROJECT_ROOT}/genesis/data/unified-genesis/zoo-mainnet-200200/genesis.json"
        fi
        
        # SPC L2
        if [ -f "${PROJECT_ROOT}/genesis/data/unified-genesis/spc-mainnet-36911/genesis.json" ]; then
            deploy_l2 "spc" 36911 "${PROJECT_ROOT}/genesis/data/unified-genesis/spc-mainnet-36911/genesis.json"
        fi
        
        echo -e "${YELLOW}Note: Hanzo L2 is prepared but not deployed yet${NC}"
        
    elif [ "$NETWORK_TYPE" = "testnet" ]; then
        # Generate testnet L2 genesis files if needed
        deploy_l2 "zoo-testnet" 200201 "${PROJECT_ROOT}/genesis/genesis_zoo-testnet.json"
        deploy_l2 "hanzo-testnet" 36962 "${PROJECT_ROOT}/genesis/genesis_hanzo-testnet.json"
    fi
fi

# Step 3: Show Summary
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo ""
echo "Network Status:"
cd "$PROJECT_ROOT/cli"
./bin/avalanche network status

echo ""
echo "Subnet Status:"
./bin/avalanche subnet list

echo ""
echo -e "${GREEN}RPC Endpoints:${NC}"
echo "C-Chain: http://localhost:9650/ext/bc/C/rpc"

# Get subnet endpoints
if [ "$SKIP_L2" = false ]; then
    for subnet in $(./bin/avalanche subnet list --json | jq -r '.[].name'); do
        if [[ "$subnet" == "zoo"* ]] || [[ "$subnet" == "spc"* ]] || [[ "$subnet" == "hanzo"* ]]; then
            BLOCKCHAIN_ID=$(./bin/avalanche subnet describe "$subnet" --json | jq -r '.blockchainID')
            echo "$subnet: http://localhost:9650/ext/bc/$BLOCKCHAIN_ID/rpc"
        fi
    done
fi

echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo "cd $PROJECT_ROOT/cli"
echo "./bin/avalanche network status"
echo "./bin/avalanche subnet list"
echo "./bin/avalanche network stop"
echo "./bin/avalanche network clean"