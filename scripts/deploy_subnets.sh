#!/bin/bash

# Deploy subnets to a running LUX network
# This script can be used standalone or called from docker-compose

set -e

# Default RPC endpoint
RPC_ENDPOINT="${LUX_RPC:-http://localhost:9630}"
LUX_CLI="${LUX_CLI:-bin/lux}"

echo "Deploying subnets to $RPC_ENDPOINT"

# Check if primary network is healthy
echo "Checking primary network health..."
if ! curl -sf "$RPC_ENDPOINT/ext/health" > /dev/null; then
    echo "❌ Primary network is not healthy at $RPC_ENDPOINT"
    exit 1
fi

echo "✅ Primary network is healthy"

# Function to deploy a subnet
deploy_subnet() {
    local name=$1
    local chain_id=$2
    local genesis_path=$3
    local config_path=$4
    
    echo ""
    echo "Deploying $name subnet (chain ID: $chain_id)..."
    
    if [ ! -f "$genesis_path" ]; then
        echo "❌ Genesis file not found: $genesis_path"
        return 1
    fi
    
    if [ ! -f "$config_path" ]; then
        echo "❌ Config file not found: $config_path"
        return 1
    fi
    
    # Deploy the subnet
    $LUX_CLI subnet deploy "$name" \
        --genesis-file "$genesis_path" \
        --config-file "$config_path" \
        --endpoint "$RPC_ENDPOINT" \
        --yes
    
    echo "✅ $name subnet deployed"
}

# Deploy all subnets
deploy_subnet "zoo" "200200" \
    "configs/zoo-mainnet-200200/genesis.json" \
    "configs/zoo-mainnet-200200/chain.json"

deploy_subnet "spc" "36911" \
    "configs/spc-mainnet-36911/genesis.json" \
    "configs/spc-mainnet-36911/chain.json"

deploy_subnet "hanzo" "36963" \
    "configs/hanzo-mainnet-36963/genesis.json" \
    "configs/hanzo-mainnet-36963/chain.json"

echo ""
echo "✅ All subnets deployed successfully!"
echo ""
echo "RPC Endpoints:"
echo "  Primary: $RPC_ENDPOINT/ext/bc/C/rpc"
echo "  ZOO: $RPC_ENDPOINT/ext/bc/zoo/rpc"
echo "  SPC: $RPC_ENDPOINT/ext/bc/spc/rpc"
echo "  Hanzo: $RPC_ENDPOINT/ext/bc/hanzo/rpc"