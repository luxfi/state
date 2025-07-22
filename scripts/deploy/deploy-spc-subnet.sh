#!/bin/bash
set -e

NETWORK=${1:-mainnet}
LUX_CLI="./bin/lux-cli"

echo "=== Deploying SPC L2 Subnet ($NETWORK) ==="

# Configuration
CHAIN_ID=36911
GENESIS_FILE="output-mainnet/spc-mainnet-genesis.json"
DB_PATH="chaindata/spc-mainnet-36911/db/pebbledb"
SUBNET_NAME="spc-mainnet"

# Step 1: Create the subnet
echo "Creating SPC subnet..."
SUBNET_ID=$($LUX_CLI l2 create $SUBNET_NAME \
    --evm \
    --chain-id $CHAIN_ID \
    --genesis $GENESIS_FILE \
    --no-prompt | grep "Subnet ID:" | awk '{print $3}')

echo "Created subnet: $SUBNET_ID"

# Step 2: Deploy the subnet
echo "Deploying SPC subnet..."
$LUX_CLI l2 deploy $SUBNET_NAME \
    --local \
    --no-prompt

# Step 3: Import historical SPC data if available
if [ -d "$DB_PATH" ]; then
    echo "Importing historical SPC data..."
    BLOCKCHAIN_ID=$($LUX_CLI l2 info $SUBNET_NAME | grep "Blockchain ID:" | awk '{print $3}')
    
    $LUX_CLI migrate import \
        --source $DB_PATH \
        --rpc-url http://localhost:9630/ext/bc/$BLOCKCHAIN_ID/rpc
    
    echo "✓ Historical SPC data imported"
else
    echo "⚠️  No historical SPC data found at $DB_PATH"
fi

echo ""
echo "✅ SPC L2 subnet deployed!"
echo "  Subnet ID: $SUBNET_ID"
echo "  Chain ID: $CHAIN_ID"