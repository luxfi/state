#!/bin/bash
set -e

echo "=== Deploying Lux Mainnet with all historical data ==="
echo ""

# Configuration
NETWORK_ID=96369
OUTPUT_DIR="./lux-mainnet-migration"
LUX_CLI="./bin/lux-cli"
GENESIS_BIN="./bin/genesis"

# Step 1: Generate genesis with all configurations
echo "Step 1: Generating mainnet genesis configurations..."
$GENESIS_BIN generate --network mainnet --output output-mainnet

# Step 2: Prepare migration data for all chains
echo ""
echo "Step 2: Preparing migration data..."

# Prepare main C-Chain (96369) data
echo "  - Preparing C-Chain data from 96369..."
$LUX_CLI migrate prepare \
    --source-db chaindata/lux-mainnet-96369/db/pebbledb \
    --network-id $NETWORK_ID \
    --validators 11 \
    --output $OUTPUT_DIR

# Step 3: Bootstrap the network with migrated data
echo ""
echo "Step 3: Bootstrapping mainnet network..."
$LUX_CLI migrate bootstrap \
    --migration-dir $OUTPUT_DIR \
    --genesis output-mainnet/genesis-mainnet-96369.json \
    --validators 11

# Wait for network to stabilize
echo ""
echo "Waiting for network to stabilize..."
sleep 15

# Step 4: Import historical data into C-Chain
echo ""
echo "Step 4: Importing historical C-Chain data..."
$LUX_CLI migrate import \
    --source chaindata/lux-mainnet-96369/db/pebbledb \
    --rpc-url http://localhost:9630/ext/bc/C/rpc

# Step 5: Deploy Zoo L2 subnet
echo ""
echo "Step 5: Deploying Zoo L2 subnet..."
./scripts/deploy-zoo-subnet.sh mainnet

# Step 6: Deploy SPC L2 subnet
echo ""
echo "Step 6: Deploying SPC L2 subnet..."
./scripts/deploy-spc-subnet.sh mainnet

echo ""
echo "✅ Mainnet deployment complete!"
echo ""
echo "Network endpoints:"
echo "  C-Chain RPC: http://localhost:9630/ext/bc/C/rpc"
echo "  P-Chain API: http://localhost:9630/ext/P"
echo "  X-Chain API: http://localhost:9630/ext/X"
echo ""
echo "Subnets:"
echo "  Zoo L2: Check logs for subnet ID and RPC endpoint"
echo "  SPC L2: Check logs for subnet ID and RPC endpoint"