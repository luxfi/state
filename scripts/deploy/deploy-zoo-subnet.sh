#!/bin/bash
set -e

NETWORK=${1:-mainnet}
LUX_CLI="./bin/lux-cli"

echo "=== Deploying Zoo L2 Subnet ($NETWORK) ==="

# Configuration based on network
if [ "$NETWORK" == "mainnet" ]; then
    CHAIN_ID=200200
    GENESIS_FILE="output-mainnet/zoo-mainnet-genesis.json"
    DB_PATH="chaindata/zoo-mainnet-200200/db/pebbledb"
    SUBNET_NAME="zoo-mainnet"
else
    CHAIN_ID=200201
    GENESIS_FILE="output-testnet/zoo-testnet-genesis.json"
    DB_PATH="chaindata/zoo-testnet-200201/db/pebbledb"
    SUBNET_NAME="zoo-testnet"
fi

# Step 1: Create the subnet
echo "Creating Zoo subnet..."
SUBNET_ID=$($LUX_CLI l2 create $SUBNET_NAME \
    --evm \
    --chain-id $CHAIN_ID \
    --genesis $GENESIS_FILE \
    --no-prompt | grep "Subnet ID:" | awk '{print $3}')

echo "Created subnet: $SUBNET_ID"

# Step 2: Deploy the subnet
echo "Deploying Zoo subnet..."
$LUX_CLI l2 deploy $SUBNET_NAME \
    --local \
    --no-prompt

# Step 3: Import historical Zoo data if available
if [ -d "$DB_PATH" ]; then
    echo "Importing historical Zoo data..."
    # Get the subnet's blockchain ID
    BLOCKCHAIN_ID=$($LUX_CLI l2 info $SUBNET_NAME | grep "Blockchain ID:" | awk '{print $3}')
    
    # Import the data
    $LUX_CLI migrate import \
        --source $DB_PATH \
        --rpc-url http://localhost:9630/ext/bc/$BLOCKCHAIN_ID/rpc
    
    echo "✓ Historical Zoo data imported"
else
    echo "⚠️  No historical Zoo data found at $DB_PATH"
fi

echo ""
echo "✅ Zoo L2 subnet deployed!"
echo "  Subnet ID: $SUBNET_ID"
echo "  Chain ID: $CHAIN_ID"
echo "  RPC endpoint will be available at: http://localhost:9630/ext/bc/[BLOCKCHAIN_ID]/rpc"