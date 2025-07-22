#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Lux Network with lux-cli...${NC}"

# Build latest tools
echo -e "${YELLOW}Building latest lux-cli and luxd...${NC}"
make check-luxd check-lux-cli

# Generate genesis files and validators
echo -e "${YELLOW}Generating genesis configuration...${NC}"
MNEMONIC="test test test test test test test test test test test junk" make generate-validators generate-all-genesis

# Clean any existing network
echo -e "${YELLOW}Cleaning existing network...${NC}"
./bin/lux-cli network clean --hard || true

# Start local network with genesis
echo -e "${YELLOW}Starting local network...${NC}"
./bin/lux-cli network start \
    --lux-path ../node/build/luxd \
    --blockchain-specs '[{
        "vm_name": "evm",
        "genesis": "./genesis_mainnet_96369.json"
    }]'

# Wait for network to be ready
echo -e "${YELLOW}Waiting for network to be ready...${NC}"
sleep 10

# Check network status
echo -e "${GREEN}Network Status:${NC}"
./bin/lux-cli network status

# Show RPC endpoints
echo -e "${GREEN}RPC Endpoints:${NC}"
echo "Primary Network C-Chain: http://localhost:9630/ext/bc/C/rpc"
echo "P-Chain: http://localhost:9630/ext/bc/P"
echo "X-Chain: http://localhost:9630/ext/bc/X"

# Test network
echo -e "${YELLOW}Testing network connection...${NC}"
curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc | jq .

echo -e "${GREEN}✅ Network launched successfully!${NC}"
echo -e "${YELLOW}To stop: ./bin/lux-cli network stop${NC}"
echo -e "${YELLOW}To clean: ./bin/lux-cli network clean${NC}"