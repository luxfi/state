#!/bin/bash
# Lux Network Deployment Script
# Usage: ./deploy.sh [environment] [network]
# Example: ./deploy.sh mainnet lux

set -e

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CONFIGS_DIR="$SCRIPT_DIR/configs"
LUXD_PATH="${LUXD_PATH:-/home/z/work/lux/node/build/luxd}"
LUX_CLI_PATH="${LUX_CLI_PATH:-/home/z/work/lux/cli/bin/lux}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Functions
usage() {
    echo "Lux Network Deployment Script"
    echo ""
    echo "Usage: $0 [environment] [network] [action]"
    echo ""
    echo "Environments:"
    echo "  mainnet    - Production mainnet"
    echo "  testnet    - Public testnet"
    echo "  local      - Local development"
    echo ""
    echo "Networks:"
    echo "  lux        - Primary network (P/C/X chains)"
    echo "  zoo        - ZOO L2 network"
    echo "  spc        - SPC L2 network"
    echo "  hanzo      - Hanzo L2 network"
    echo ""
    echo "Actions:"
    echo "  start      - Start the network (default)"
    echo "  stop       - Stop the network"
    echo "  info       - Show network information"
    echo ""
    echo "Examples:"
    echo "  $0 mainnet lux        # Start LUX mainnet"
    echo "  $0 local zoo start    # Start local ZOO L2"
    echo "  $0 testnet all        # Start testnet with all L2s"
}

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    if [[ ! -f "$LUXD_PATH" ]]; then
        error "luxd not found at $LUXD_PATH. Please set LUXD_PATH environment variable."
    fi
    
    if [[ ! -f "$LUX_CLI_PATH" ]] && [[ "$2" != "lux" ]]; then
        warn "lux-cli not found at $LUX_CLI_PATH. L2 deployment may fail."
    fi
}

# Get network configuration
get_network_config() {
    local env=$1
    local network=$2
    
    case $network in
        lux)
            case $env in
                mainnet) NETWORK_ID=96369 ;;
                testnet) NETWORK_ID=96368 ;;
                local) NETWORK_ID=12345 ;;
            esac
            ;;
        zoo)
            case $env in
                mainnet) NETWORK_ID=200200; CHAIN_ID=200200 ;;
                testnet) NETWORK_ID=200201; CHAIN_ID=200201 ;;
                local) NETWORK_ID=200202; CHAIN_ID=200202 ;;
            esac
            ;;
        spc)
            case $env in
                mainnet) NETWORK_ID=36911; CHAIN_ID=36911 ;;
                testnet) NETWORK_ID=36912; CHAIN_ID=36912 ;;
                local) NETWORK_ID=36913; CHAIN_ID=36913 ;;
            esac
            ;;
        hanzo)
            case $env in
                mainnet) NETWORK_ID=36963; CHAIN_ID=36963 ;;
                testnet) NETWORK_ID=36962; CHAIN_ID=36962 ;;
                local) NETWORK_ID=36964; CHAIN_ID=36964 ;;
            esac
            ;;
        *)
            error "Unknown network: $network"
            ;;
    esac
    
    GENESIS_FILE="$CONFIGS_DIR/$env/$network/genesis.json"
    if [[ ! -f "$GENESIS_FILE" ]]; then
        error "Genesis file not found: $GENESIS_FILE"
    fi
}

# Start primary network
start_lux_network() {
    local env=$1
    get_network_config $env lux
    
    log "Starting LUX $env network (ID: $NETWORK_ID)..."
    
    # Kill any existing instance
    pkill -f "luxd.*network-id=$NETWORK_ID" || true
    sleep 2
    
    # Prepare data directory
    DATA_DIR="$HOME/.luxd-$env"
    mkdir -p "$DATA_DIR"
    
    # Launch luxd
    nohup "$LUXD_PATH" \
        --network-id=$NETWORK_ID \
        --genesis-file="$GENESIS_FILE" \
        --data-dir="$DATA_DIR" \
        --http-host=0.0.0.0 \
        --http-port=9630 \
        --staking-ephemeral-cert-enabled \
        --public-ip=127.0.0.1 \
        > "$DATA_DIR/node.log" 2>&1 &
    
    # Wait for startup
    log "Waiting for node to start..."
    sleep 10
    
    # Check if running
    if curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"info.isBootstrapped","params":{"chain":"P"}}' \
        http://localhost:9630/ext/info > /dev/null 2>&1; then
        log "✅ LUX $env network started successfully!"
        log "RPC endpoint: http://localhost:9630"
        log "Logs: $DATA_DIR/node.log"
    else
        error "Failed to start LUX network. Check logs at $DATA_DIR/node.log"
    fi
}

# Start L2 network
start_l2_network() {
    local env=$1
    local network=$2
    get_network_config $env $network
    
    log "Creating $network L2 on $env..."
    
    # Create L2 using lux-cli
    "$LUX_CLI_PATH" blockchain create $network-$env \
        --evm \
        --genesis-file="$GENESIS_FILE" \
        --chain-id=$CHAIN_ID \
        --force
    
    # Deploy L2
    log "Deploying $network L2..."
    "$LUX_CLI_PATH" blockchain deploy $network-$env \
        --local \
        --avalanchego-version latest
    
    log "✅ $network L2 deployed successfully!"
}

# Stop network
stop_network() {
    local env=$1
    local network=$2
    
    if [[ "$network" == "lux" ]]; then
        get_network_config $env lux
        log "Stopping LUX $env network..."
        pkill -f "luxd.*network-id=$NETWORK_ID" || true
        log "✅ Network stopped"
    else
        warn "Use lux-cli to stop L2 networks"
    fi
}

# Show network info
show_network_info() {
    local env=$1
    local network=$2
    
    echo ""
    echo "Network Information"
    echo "==================="
    echo "Environment: $env"
    echo "Network: $network"
    
    if [[ "$network" == "lux" ]]; then
        get_network_config $env lux
        echo "Network ID: $NETWORK_ID"
        echo "Genesis: $GENESIS_FILE"
        echo "RPC: http://localhost:9630"
        echo ""
        
        # Try to get blockchain IDs
        if curl -s http://localhost:9630/ext/info > /dev/null 2>&1; then
            echo "Blockchain IDs:"
            curl -s -X POST -H "Content-Type: application/json" \
                -d '{"jsonrpc":"2.0","id":1,"method":"info.getBlockchainID","params":{"alias":"C"}}' \
                http://localhost:9630/ext/info 2>/dev/null | jq -r '.result.blockchainID' | xargs echo "  C-Chain:"
            curl -s -X POST -H "Content-Type: application/json" \
                -d '{"jsonrpc":"2.0","id":1,"method":"info.getBlockchainID","params":{"alias":"P"}}' \
                http://localhost:9630/ext/info 2>/dev/null | jq -r '.result.blockchainID' | xargs echo "  P-Chain:"
            curl -s -X POST -H "Content-Type: application/json" \
                -d '{"jsonrpc":"2.0","id":1,"method":"info.getBlockchainID","params":{"alias":"X"}}' \
                http://localhost:9630/ext/info 2>/dev/null | jq -r '.result.blockchainID' | xargs echo "  X-Chain:"
        else
            warn "Node not running or not accessible"
        fi
    else
        get_network_config $env $network
        echo "Chain ID: $CHAIN_ID"
        echo "Genesis: $GENESIS_FILE"
        
        # Try to get L2 info
        if command -v "$LUX_CLI_PATH" > /dev/null; then
            "$LUX_CLI_PATH" blockchain describe $network-$env 2>/dev/null || warn "L2 not deployed"
        fi
    fi
    echo ""
}

# Main execution
main() {
    if [[ $# -lt 2 ]]; then
        usage
        exit 1
    fi
    
    ENV=$1
    NETWORK=$2
    ACTION=${3:-start}
    
    # Validate environment
    case $ENV in
        mainnet|testnet|local) ;;
        *) error "Invalid environment: $ENV" ;;
    esac
    
    # Check prerequisites
    check_prerequisites
    
    # Handle 'all' network option
    if [[ "$NETWORK" == "all" ]]; then
        # Start primary network first
        case $ACTION in
            start)
                start_lux_network $ENV
                sleep 5
                # Then start all L2s
                for l2 in zoo spc hanzo; do
                    log "Starting $l2 L2..."
                    start_l2_network $ENV $l2
                done
                ;;
            info)
                for net in lux zoo spc hanzo; do
                    show_network_info $ENV $net
                done
                ;;
            stop)
                stop_network $ENV lux
                warn "Use lux-cli to stop L2 networks"
                ;;
        esac
    else
        # Handle specific network
        case $ACTION in
            start)
                if [[ "$NETWORK" == "lux" ]]; then
                    start_lux_network $ENV
                else
                    # Check if primary network is running
                    if ! curl -s http://localhost:9630/ext/info > /dev/null 2>&1; then
                        warn "Primary network not running. Starting it first..."
                        start_lux_network $ENV
                        sleep 5
                    fi
                    start_l2_network $ENV $NETWORK
                fi
                ;;
            stop)
                stop_network $ENV $NETWORK
                ;;
            info)
                show_network_info $ENV $NETWORK
                ;;
            *)
                error "Unknown action: $ACTION"
                ;;
        esac
    fi
}

# Run main
main "$@"