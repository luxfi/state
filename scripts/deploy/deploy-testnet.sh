#!/bin/bash
set -e

echo "=== Deploying Lux Testnet with all historical data ==="
echo ""

# Configuration
NETWORK_ID=96368
OUTPUT_DIR="./lux-testnet-migration"
LUX_CLI="./bin/lux-cli"
GENESIS_BIN="./bin/genesis"

# Step 1: Generate genesis with all configurations
echo "Step 1: Generating testnet genesis configurations..."
$GENESIS_BIN generate --network testnet --output output-testnet

# Step 2: Prepare migration data
echo ""
echo "Step 2: Preparing testnet migration data..."

# Prepare testnet C-Chain (96368) data
echo "  - Preparing C-Chain data from 96368..."
$LUX_CLI migrate prepare \
    --source-db chaindata/lux-testnet-96368/db/pebbledb \
    --network-id $NETWORK_ID \
    --validators 5 \
    --output $OUTPUT_DIR

# Step 3: Bootstrap the testnet network
echo ""
echo "Step 3: Bootstrapping testnet network..."
$LUX_CLI migrate bootstrap \
    --migration-dir $OUTPUT_DIR \
    --genesis output-testnet/genesis-testnet-96368.json \
    --validators 5

# Wait for network to stabilize
echo ""
echo "Waiting for network to stabilize..."
sleep 15

# Step 4: Import historical testnet data
echo ""
echo "Step 4: Importing historical testnet C-Chain data..."
$LUX_CLI migrate import \
    --source chaindata/lux-testnet-96368/db/pebbledb \
    --rpc-url http://localhost:9630/ext/bc/C/rpc

# Step 5: Deploy Zoo testnet L2
echo ""
echo "Step 5: Deploying Zoo testnet L2 subnet..."
./scripts/deploy-zoo-subnet.sh testnet

echo ""
echo "✅ Testnet deployment complete!"
echo ""
echo "Network endpoints:"
echo "  C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "  P-Chain API: http://localhost:9630/ext/P"
echo "  X-Chain API: http://localhost:9630/ext/X"
echo ""
echo "Subnets:"
echo "  Zoo Testnet L2: Check logs for subnet ID and RPC endpoint"