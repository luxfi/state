#!/bin/bash
# Launch Lux Network with L2s using historic data

set -e

echo "🚀 Launching Lux Network with Historic Data"
echo "=========================================="

# Kill any existing processes
echo "Cleaning up existing processes..."
pkill -f luxd || true
pkill -f avalanche || true
sleep 2

# Launch LUX mainnet using existing script
echo ""
echo "📦 Launching LUX mainnet with POA automining..."
cd /home/z/work/lux
./scripts/run-lux-mainnet-automining.sh &

# Wait for node to start
echo "Waiting for node to start..."
sleep 15

# Check if running
if curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc > /dev/null; then
    echo "✅ LUX mainnet is running"
    
    # Get block number
    BLOCK=$(curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc | jq -r '.result')
    echo "Current block: $BLOCK"
else
    echo "❌ Failed to start LUX mainnet"
    exit 1
fi

# Create and deploy L2s using CLI
echo ""
echo "🚀 Creating and deploying L2s..."
cd /home/z/work/lux/cli

# Configure CLI for local network
export AVALANCHE_NETWORK=Local
export AVALANCHE_CHAIN_ID=96369

# Create ZOO L2
echo ""
echo "Creating ZOO L2..."
./bin/lux blockchain create zoo \
  --evm \
  --chain-id=200200 \
  --token-symbol=ZOO \
  --genesis-file=/home/z/work/lux/genesis/output/import-ready/zoo/L2/genesis.json \
  --force

# Deploy ZOO L2
echo "Deploying ZOO L2..."
./bin/lux blockchain deploy zoo --local --avalanchego-version latest

# Create SPC L2
echo ""
echo "Creating SPC L2..."
./bin/lux blockchain create spc \
  --evm \
  --chain-id=36911 \
  --token-symbol=SPC \
  --genesis-file=/home/z/work/lux/genesis/output/import-ready/spc/L2/genesis.json \
  --force

# Deploy SPC L2
echo "Deploying SPC L2..."
./bin/lux blockchain deploy spc --local --avalanchego-version latest

# Get blockchain info
echo ""
echo "📊 Network Information"
echo "===================="

# Get blockchain IDs
echo ""
echo "Getting blockchain IDs..."
./bin/lux blockchain list

# Show RPC endpoints
echo ""
echo "🌐 RPC Endpoints:"
echo "================"
echo "LUX Mainnet: http://localhost:9650/ext/bc/C/rpc"

# Get ZOO blockchain ID
ZOO_INFO=$(./bin/lux blockchain describe zoo | grep -A1 "Blockchain ID" || echo "Not found")
echo "ZOO L2 Info: $ZOO_INFO"

# Get SPC blockchain ID  
SPC_INFO=$(./bin/lux blockchain describe spc | grep -A1 "Blockchain ID" || echo "Not found")
echo "SPC L2 Info: $SPC_INFO"

echo ""
echo "✅ Network launch complete!"
echo ""
echo "To check network status:"
echo "  curl -X POST --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc"
echo ""
echo "To interact with networks:"
echo "  cast chain-id --rpc-url http://localhost:9650/ext/bc/C/rpc"