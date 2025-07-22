#!/bin/bash

# Launch luxd with proper genesis configuration

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LUXD_PATH="${PROJECT_ROOT}/node/build/luxd"
GENESIS_PATH="${PROJECT_ROOT}/genesis/genesis_mainnet.json"
DATA_DIR="${HOME}/.luxd"
NETWORK="mainnet"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --network)
            NETWORK="$2"
            shift 2
            ;;
        --data-dir)
            DATA_DIR="$2"
            shift 2
            ;;
        --genesis)
            GENESIS_PATH="$2"
            shift 2
            ;;
        --build)
            BUILD_FIRST=true
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --network <name>    Network to run (mainnet, testnet, local)"
            echo "  --data-dir <path>   Data directory (default: ~/.luxd)"
            echo "  --genesis <path>    Path to genesis file"
            echo "  --build             Build luxd before running"
            echo "  --help              Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if genesis needs to be generated
if [ ! -f "$GENESIS_PATH" ]; then
    echo -e "${YELLOW}Genesis file not found. Generating...${NC}"
    cd "$PROJECT_ROOT/genesis"
    
    # Build genesis builder if needed
    if [ ! -f "cmd/genesis-builder/genesis-builder" ]; then
        echo "Building genesis builder..."
        cd cmd/genesis-builder
        go build
        cd ../..
    fi
    
    # Generate genesis
    if [ "$NETWORK" = "mainnet" ]; then
        ./cmd/genesis-builder/genesis-builder \
            --network mainnet \
            --validators configs/mainnet-validators.json \
            --output "genesis_${NETWORK}.json"
    else
        ./cmd/genesis-builder/genesis-builder \
            --network "$NETWORK" \
            --output "genesis_${NETWORK}.json"
    fi
    
    GENESIS_PATH="${PROJECT_ROOT}/genesis/genesis_${NETWORK}.json"
fi

# Build luxd if requested
if [ "$BUILD_FIRST" = true ]; then
    echo -e "${YELLOW}Building luxd...${NC}"
    cd "$PROJECT_ROOT"
    make build-node
fi

# Check if luxd exists
if [ ! -f "$LUXD_PATH" ]; then
    echo -e "${RED}Error: luxd not found at $LUXD_PATH${NC}"
    echo "Run with --build flag or build manually with 'make build-node'"
    exit 1
fi

# Load network configuration
source "${SCRIPT_DIR}/network-config.sh" "$NETWORK"

# Create data directory
mkdir -p "$DATA_DIR"

# Copy genesis to data directory
cp "$GENESIS_PATH" "$DATA_DIR/genesis.json"

# Launch luxd
echo -e "${GREEN}Launching luxd for $NETWORK...${NC}"
echo "Data directory: $DATA_DIR"
echo "Genesis: $GENESIS_PATH"
echo "Network ID: $NETWORK_ID"
echo "Bootstrap IPs: $BOOTSTRAP_IPS"
echo "Bootstrap IDs: $BOOTSTRAP_IDS"

exec "$LUXD_PATH" \
    --data-dir="$DATA_DIR" \
    --network-id="$NETWORK_ID" \
    --bootstrap-ips="$BOOTSTRAP_IPS" \
    --bootstrap-ids="$BOOTSTRAP_IDS" \
    --staking-enabled="$STAKING_ENABLED" \
    --sybil-protection-enabled="$SYBIL_PROTECTION_ENABLED" \
    --snow-sample-size="$SNOW_SAMPLE_SIZE" \
    --snow-quorum-size="$SNOW_QUORUM_SIZE" \
    --snow-virtuous-commit-threshold="$SNOW_VIRTUOUS_COMMIT_THRESHOLD" \
    --snow-rogue-commit-threshold="$SNOW_ROGUE_COMMIT_THRESHOLD" \
    --http-host=0.0.0.0 \
    --http-port=9650 \
    --staking-port=9651 \
    --log-level=info \
    "$@"