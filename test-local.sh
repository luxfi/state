#!/bin/bash
# Local test script for genesis migration

set -e

echo "=== Testing Genesis Migration Locally ==="
echo

# Check if chaindata exists
if [ ! -d "chaindata/lux-mainnet-96369/db/pebbledb" ] && [ ! -d "chaindata/db/pebbledb" ]; then
    echo "❌ Error: No chaindata found in ./chaindata/"
    echo "Please ensure you have the subnet chaindata in:"
    echo "  ./chaindata/lux-mainnet-96369/db/pebbledb"
    echo "  or"
    echo "  ./chaindata/db/pebbledb"
    exit 1
fi

# Clean up any previous runtime data
echo "1. Cleaning previous runtime data..."
rm -rf runtime/node-data
mkdir -p runtime

# Build the Docker image
echo "2. Building Docker image..."
docker build -f docker/Dockerfile -t luxfi/genesis-migration:test .

# Run the container
echo "3. Starting genesis migration container..."
docker run -d \
    --name lux-genesis-test \
    -p 9630:9630 \
    -v "$(pwd)/chaindata:/app/chaindata:ro" \
    -v "$(pwd)/runtime:/app/runtime" \
    -e NETWORK_ID=96369 \
    -e TIP_HEIGHT=1082780 \
    -e TIP_HASH=0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0 \
    luxfi/genesis-migration:test

# Wait for startup
echo "4. Waiting for luxd to start (60 seconds)..."
sleep 10

# Check if container is running
if ! docker ps | grep -q lux-genesis-test; then
    echo "❌ Container failed to start. Checking logs:"
    docker logs lux-genesis-test
    exit 1
fi

# Follow logs for a bit
echo "5. Container logs:"
docker logs -f lux-genesis-test &
LOG_PID=$!

# Wait for RPC to be available
echo "6. Waiting for RPC to be available..."
for i in {1..30}; do
    if curl -s -X POST -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
        http://localhost:9630/ext/bc/C/rpc > /dev/null 2>&1; then
        echo "✅ RPC is available!"
        break
    fi
    echo -n "."
    sleep 2
done
echo

# Kill log following
kill $LOG_PID 2>/dev/null || true

# Test the RPC endpoint
echo "7. Testing RPC endpoint..."
RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc)

echo "Response: $RESPONSE"

# Extract block number
BLOCK_HEX=$(echo "$RESPONSE" | grep -o '"result":"0x[0-9a-fA-F]*"' | cut -d'"' -f4)
if [ -n "$BLOCK_HEX" ]; then
    BLOCK_DEC=$((BLOCK_HEX))
    echo "✅ Current block height: $BLOCK_DEC"
    
    if [ "$BLOCK_DEC" -eq 1082780 ]; then
        echo "✅ SUCCESS! Blockchain is at expected height 1,082,780"
    else
        echo "⚠️  Warning: Block height is $BLOCK_DEC, expected 1,082,780"
    fi
else
    echo "❌ Failed to get block number"
fi

echo
echo "8. Cleanup commands:"
echo "   Stop container:  docker stop lux-genesis-test"
echo "   Remove container: docker rm lux-genesis-test"
echo "   View logs:       docker logs lux-genesis-test"
echo
echo "=== Test Complete ==="