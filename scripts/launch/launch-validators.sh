#!/bin/bash
set -e

LUXD_PATH="../node/build/luxd"
GENESIS_FILE="genesis_mainnet_96369.json"

# Array to store node IDs
declare -a NODE_IDS

# First, get all the node IDs
echo "Getting node IDs..."
for i in {1..5}; do
    NODE_ID=$(cat configs/mainnet-validators.json | jq -r ".[$((i-1))].nodeID")
    NODE_IDS[$i]=$NODE_ID
    echo "Node$i: $NODE_ID"
done

# Build bootstrap IPs (all nodes except self)
echo ""
echo "Starting validators..."
for i in {1..5}; do
    echo "Starting node$i..."
    
    # Build bootstrap IPs - all nodes except self
    BOOTSTRAP_IPS=""
    for j in {1..5}; do
        if [ $i -ne $j ]; then
            if [ -n "$BOOTSTRAP_IPS" ]; then
                BOOTSTRAP_IPS="${BOOTSTRAP_IPS},"
            fi
            BOOTSTRAP_IPS="${BOOTSTRAP_IPS}${NODE_IDS[$j]}-127.0.0.1:$((9631 + j - 1))"
        fi
    done
    
    # Launch node
    $LUXD_PATH \
        --network-id=96369 \
        --data-dir=network-runner/node$i \
        --genesis=$GENESIS_FILE \
        --http-port=$((9630 + i - 1)) \
        --staking-port=$((9631 + i - 1)) \
        --bootstrap-ips="$BOOTSTRAP_IPS" \
        --bootstrap-ids="${NODE_IDS[1]},${NODE_IDS[2]},${NODE_IDS[3]},${NODE_IDS[4]},${NODE_IDS[5]}" \
        --staking-enabled=true \
        --snow-sample-size=3 \
        --snow-quorum-size=3 \
        --log-level=info \
        > network-runner/node$i/node.log 2>&1 &
    
    echo $! > network-runner/node$i/node.pid
    sleep 2
done

echo ""
echo "✅ Local validators launched!"
echo ""
echo "Node logs: network-runner/node*/node.log"
echo "RPC endpoints:"
for i in {1..5}; do
    echo "  Node$i: http://localhost:$((9630 + i - 1))"
done
echo ""
echo "To check network status:"
echo "  curl -X POST -H 'Content-Type: application/json' -d '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"info.getNetworkID\",\"params\":{}}' http://localhost:9630/ext/info"
echo ""
echo "To stop network:"
echo "  ./stop-validators.sh"