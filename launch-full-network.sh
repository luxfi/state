#!/bin/bash
# Launch full Lux mainnet + all L2s with historic data

set -e

echo "🚀 Lux Network Full Launch Script"
echo "================================="
echo "This will launch:"
echo "  - LUX Mainnet (96369) with historic data"
echo "  - ZOO L2 (200200) with BSC migration"
echo "  - SPC L2 (36911) with bootstrap genesis"
echo ""

# Configuration
GENESIS_DIR="/home/z/work/lux/genesis"
NODE_DIR="/home/z/work/lux/node"
CLI_DIR="/home/z/work/lux/cli"
DATA_DIR="/home/z/.luxd"
OUTPUT_DIR="$GENESIS_DIR/output"

# Create output directories
mkdir -p "$OUTPUT_DIR"/{exports,genesis,analysis,networks}

# Step 1: Build tools if needed
echo "📦 Step 1: Building tools..."
cd "$NODE_DIR"
if [ ! -f "build/luxd" ]; then
    echo "Building luxd..."
    ./scripts/build.sh
fi

cd "$CLI_DIR"
if [ ! -f "bin/lux" ]; then
    echo "Building lux CLI..."
    go build -o bin/lux cmd/main.go
fi

# Step 2: Generate genesis files from historic data
echo ""
echo "🏗️  Step 2: Generating genesis files from historic data..."
cd "$GENESIS_DIR"

# LUX Mainnet - Use existing chaindata
echo "Generating LUX mainnet genesis..."
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    echo "✅ Found LUX mainnet chaindata"
    # Copy the existing genesis
    cp chaindata/configs/lux-mainnet-96369/genesis.json "$OUTPUT_DIR/genesis/lux-mainnet-genesis.json"
else
    echo "⚠️  No LUX mainnet chaindata found, using default genesis"
fi

# ZOO L2 - Include BSC migration data
echo "Generating ZOO L2 genesis with BSC migration..."
if [ -f "exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.json" ]; then
    echo "✅ Found ZOO egg holder data"
    # Use existing ZOO genesis with egg allocations
    cp chaindata/configs/zoo-mainnet-200200/genesis.json "$OUTPUT_DIR/genesis/zoo-mainnet-genesis.json"
else
    echo "⚠️  No ZOO migration data found, using default genesis"
fi

# SPC L2 - Bootstrap genesis
echo "Generating SPC L2 bootstrap genesis..."
cp chaindata/configs/spc-mainnet-36911/genesis.json "$OUTPUT_DIR/genesis/spc-mainnet-genesis.json"

# Step 3: Launch LUX mainnet
echo ""
echo "🚀 Step 3: Launching LUX mainnet..."
cd "$NODE_DIR"

# Kill any existing luxd
pkill -f luxd || true
sleep 2

# Launch with POA automining configuration
echo "Starting luxd with POA automining..."
nohup ./build/luxd \
  --network-id=96369 \
  --data-dir="$DATA_DIR" \
  --chain-config-content='{"C": {"state-sync-enabled": false, "pruning-enabled": false}}' \
  --http-host=0.0.0.0 \
  --http-port=9650 \
  --staking-enabled=false \
  --sybil-protection-enabled=false \
  --bootstrap-ips="" \
  --bootstrap-ids="" \
  --public-ip=127.0.0.1 \
  --snow-sample-size=1 \
  --snow-quorum-size=1 \
  --snow-virtuous-commit-threshold=1 \
  --snow-rogue-commit-threshold=1 \
  --snow-concurrent-repolls=1 \
  --index-enabled \
  --db-dir="$DATA_DIR/db" \
  > "$OUTPUT_DIR/networks/lux-mainnet.log" 2>&1 &

echo "Waiting for LUX mainnet to start..."
sleep 10

# Check if running
if curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc > /dev/null; then
    echo "✅ LUX mainnet is running"
else
    echo "❌ Failed to start LUX mainnet"
    exit 1
fi

# Step 4: Create and deploy L2s
echo ""
echo "🚀 Step 4: Creating and deploying L2s..."
cd "$CLI_DIR"

# Create ZOO L2
echo "Creating ZOO L2..."
./bin/lux blockchain create zoo \
  --evm \
  --genesis-file "$OUTPUT_DIR/genesis/zoo-mainnet-genesis.json" \
  --use-defaults

# Deploy ZOO L2
echo "Deploying ZOO L2..."
./bin/lux blockchain deploy zoo \
  --local \
  --avalanchego-version latest

# Create SPC L2
echo "Creating SPC L2..."
./bin/lux blockchain create spc \
  --evm \
  --genesis-file "$OUTPUT_DIR/genesis/spc-mainnet-genesis.json" \
  --use-defaults

# Deploy SPC L2
echo "Deploying SPC L2..."
./bin/lux blockchain deploy spc \
  --local \
  --avalanchego-version latest

# Step 5: Get network information
echo ""
echo "📊 Step 5: Network Information"
echo "=============================="

# Get blockchain IDs
ZOO_BLOCKCHAIN_ID=$(./bin/lux blockchain describe zoo --local | grep -A1 "Blockchain ID" | tail -1 | xargs)
SPC_BLOCKCHAIN_ID=$(./bin/lux blockchain describe spc --local | grep -A1 "Blockchain ID" | tail -1 | xargs)

echo "LUX Mainnet:"
echo "  Chain ID: 96369"
echo "  RPC: http://localhost:9650/ext/bc/C/rpc"
echo "  Explorer: http://localhost:9650/ext/bc/C/explorer"

echo ""
echo "ZOO L2:"
echo "  Chain ID: 200200"
echo "  Blockchain ID: $ZOO_BLOCKCHAIN_ID"
echo "  RPC: http://localhost:9650/ext/bc/$ZOO_BLOCKCHAIN_ID/rpc"

echo ""
echo "SPC L2:"
echo "  Chain ID: 36911"
echo "  Blockchain ID: $SPC_BLOCKCHAIN_ID"
echo "  RPC: http://localhost:9650/ext/bc/$SPC_BLOCKCHAIN_ID/rpc"

# Save network info
cat > "$OUTPUT_DIR/networks/network-info.json" <<EOF
{
  "lux": {
    "chainId": 96369,
    "rpc": "http://localhost:9650/ext/bc/C/rpc",
    "explorer": "http://localhost:9650/ext/bc/C/explorer"
  },
  "zoo": {
    "chainId": 200200,
    "blockchainId": "$ZOO_BLOCKCHAIN_ID",
    "rpc": "http://localhost:9650/ext/bc/$ZOO_BLOCKCHAIN_ID/rpc"
  },
  "spc": {
    "chainId": 36911,
    "blockchainId": "$SPC_BLOCKCHAIN_ID",
    "rpc": "http://localhost:9650/ext/bc/$SPC_BLOCKCHAIN_ID/rpc"
  }
}
EOF

echo ""
echo "✅ All networks launched successfully!"
echo "Network info saved to: $OUTPUT_DIR/networks/network-info.json"
echo ""
echo "To interact with the networks:"
echo "  LUX: cast block-number --rpc-url http://localhost:9650/ext/bc/C/rpc"
echo "  ZOO: cast block-number --rpc-url http://localhost:9650/ext/bc/$ZOO_BLOCKCHAIN_ID/rpc"
echo "  SPC: cast block-number --rpc-url http://localhost:9650/ext/bc/$SPC_BLOCKCHAIN_ID/rpc"