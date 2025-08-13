#!/bin/bash

# Lux Mainnet Validator Startup Script
# This script boots a full mainnet consensus validator with staking

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Lux Mainnet Validator...${NC}"

# Configuration
LUXD_PATH="/Users/z/work/lux/node/build/luxd"
DATA_DIR="/Users/z/.luxd"
CHAIN_DATA_DIR="${DATA_DIR}/chainData"
STATE_DIR="/Users/z/work/lux/state"
GENESIS_DIR="${STATE_DIR}/mainnet-genesis"
SUBNET_DATA="${STATE_DIR}/chaindata/lux-mainnet-96369"

# Create necessary directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p "${DATA_DIR}"
mkdir -p "${CHAIN_DATA_DIR}"
mkdir -p "${CHAIN_DATA_DIR}/mainnet"

# Copy genesis data for P, C, X chains
echo -e "${YELLOW}Setting up genesis data...${NC}"
if [ -d "${GENESIS_DIR}/P" ]; then
    echo "Copying P-Chain genesis..."
    cp -r "${GENESIS_DIR}/P" "${CHAIN_DATA_DIR}/mainnet/"
fi

if [ -d "${GENESIS_DIR}/C" ]; then
    echo "Copying C-Chain genesis..."
    cp -r "${GENESIS_DIR}/C" "${CHAIN_DATA_DIR}/mainnet/"
fi

if [ -d "${GENESIS_DIR}/X" ]; then
    echo "Copying X-Chain genesis..."
    cp -r "${GENESIS_DIR}/X" "${CHAIN_DATA_DIR}/mainnet/"
fi

# Copy subnet EVM data (96369) to C-Chain using BadgerDB
echo -e "${YELLOW}Setting up C-Chain with subnet EVM data (chain 96369)...${NC}"
if [ -d "${SUBNET_DATA}/db" ]; then
    echo "Copying subnet EVM chaindata to C-Chain..."
    # Create C-Chain data directory if it doesn't exist
    mkdir -p "${CHAIN_DATA_DIR}/mainnet/C/db"
    
    # Copy the subnet data as the C-Chain database
    cp -r "${SUBNET_DATA}/db/"* "${CHAIN_DATA_DIR}/mainnet/C/db/" 2>/dev/null || true
    
    echo -e "${GREEN}Subnet EVM data (96369) copied to C-Chain${NC}"
fi

# Create staking keys if they don't exist
STAKING_DIR="${DATA_DIR}/staking"
if [ ! -d "${STAKING_DIR}" ]; then
    echo -e "${YELLOW}Generating staking keys...${NC}"
    mkdir -p "${STAKING_DIR}"
    # The node will generate keys automatically if not present
fi

# Start the node
echo -e "${GREEN}Starting Lux node with mainnet configuration...${NC}"
echo -e "${YELLOW}Node will be available at: http://localhost:9630${NC}"
echo -e "${YELLOW}C-Chain RPC will be available at: http://localhost:9630/ext/bc/C/rpc${NC}"

# Check if luxd exists
if [ ! -f "${LUXD_PATH}" ]; then
    echo -e "${RED}Error: luxd not found at ${LUXD_PATH}${NC}"
    echo -e "${YELLOW}Please build the node first: cd /Users/z/work/lux/node && make build${NC}"
    exit 1
fi

# Start the node as the PRIMARY MAINNET BOOTSTRAP NODE
# This node IS the mainnet - it doesn't bootstrap from anyone
exec "${LUXD_PATH}" \
    --network-id=mainnet \
    --db-type=badgerdb \
    --data-dir="${DATA_DIR}" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-port=9651 \
    --public-ip=127.0.0.1 \
    --log-level=info \
    --min-validator-stake=2000000000000