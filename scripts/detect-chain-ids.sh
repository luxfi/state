#!/bin/bash
# Detect Chain ID and VM ID for a given network

NETWORK_ID=${1:-96369}
DATA_DIR=${2:-runtime}

echo "🔍 Detecting Chain ID and VM ID for network $NETWORK_ID..."

# First, try to get from running node if available
if curl -s http://localhost:9630/ext/info >/dev/null 2>&1; then
    echo "📡 Querying running node..."
    
    # Get blockchain info
    BLOCKCHAINS=$(curl -s -X POST --data '{
        "jsonrpc":"2.0",
        "id"     :1,
        "method" :"platform.getBlockchains",
        "params" :{}
    }' -H 'content-type:application/json' http://localhost:9630/ext/bc/P)
    
    # Extract C-Chain info
    C_CHAIN_INFO=$(echo "$BLOCKCHAINS" | jq -r '.result.blockchains[] | select(.vmID == "mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6" or .name == "C-Chain")')
    
    if [ ! -z "$C_CHAIN_INFO" ]; then
        export CHAIN_ID=$(echo "$C_CHAIN_INFO" | jq -r '.id')
        export VM_ID=$(echo "$C_CHAIN_INFO" | jq -r '.vmID')
        echo "✅ Found from running node:"
        echo "   Chain ID: $CHAIN_ID"
        echo "   VM ID: $VM_ID"
        return 0
    fi
fi

# For network 96369, use known values
if [ "$NETWORK_ID" = "96369" ]; then
    export CHAIN_ID="2vd59DPuN4Y9kQmmsbz8TGgJhJg5kVo8TCCYVBByTTWpSda3R1"
    export VM_ID="rXnv1kBRV9v14hJ6Ny94Gj9WZtpQ7wYZZH68aDbqiteS5RGiP"
    echo "✅ Using known values for network 96369:"
    echo "   Chain ID: $CHAIN_ID"
    echo "   VM ID: $VM_ID"
    return 0
fi

# Try to detect from genesis hash in logs
if [ -f "$DATA_DIR/logs/main.log" ]; then
    GENESIS_HASH=$(grep "initializing database" "$DATA_DIR/logs/main.log" | tail -1 | grep -oP 'genesisHash":\s*"\K[^"]+')
    if [ ! -z "$GENESIS_HASH" ]; then
        # Map genesis hash to chain ID (this would need a lookup table)
        case "$GENESIS_HASH" in
            "2vd59DPuN4Y9kQmmsbz8TGgJhJg5kVo8TCCYVBByTTWpSda3R1")
                export CHAIN_ID="2vd59DPuN4Y9kQmmsbz8TGgJhJg5kVo8TCCYVBByTTWpSda3R1"
                export VM_ID="rXnv1kBRV9v14hJ6Ny94Gj9WZtpQ7wYZZH68aDbqiteS5RGiP"
                ;;
        esac
    fi
fi

# Check if we found values
if [ -z "$CHAIN_ID" ] || [ -z "$VM_ID" ]; then
    echo "❌ Could not detect Chain ID and VM ID"
    echo "   Please ensure node is running or provide known values"
    return 1
fi

# Export for use in other scripts
echo ""
echo "📝 Export these values:"
echo "export CHAIN_ID=$CHAIN_ID"
echo "export VM_ID=$VM_ID"