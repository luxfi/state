#!/bin/bash

# Test the migrated database with luxd

echo "=== Testing Migrated Database ==="
echo "Migrated DB: /tmp/migrated-chaindata"
echo ""

# Create test directory structure
TEST_DIR="/tmp/test-lux-migration"
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/C/db"

# Copy migrated database to test location
echo "Copying migrated database to test location..."
cp -r /tmp/migrated-chaindata/* "$TEST_DIR/C/db/"

# Create minimal config for testing
cat > "$TEST_DIR/config.json" <<EOF
{
  "network-id": "96369",
  "staking-enabled": false,
  "health-check-frequency": "30s",
  "log-level": "info",
  "api-admin-enabled": true,
  "api-ipcs-enabled": true,
  "api-keystore-enabled": false,
  "api-metrics-enabled": true,
  "http-host": "0.0.0.0",
  "http-port": 9651,
  "http-allowed-origins": "*",
  "data-dir": "$TEST_DIR",
  "chain-config-dir": "/home/z/work/lux/node/configs/chains",
  "snow-sample-size": 1,
  "snow-quorum-size": 1,
  "snow-mixed-query-num-push-vdr": 1,
  "staking-tls-cert-file": "/home/z/work/lux/node/staking/local/staker1.crt",
  "staking-tls-key-file": "/home/z/work/lux/node/staking/local/staker1.key"
}
EOF

# Start luxd with migrated database
echo "Starting luxd with migrated database..."
cd /home/z/work/lux/node

# Run in background and capture output
./build/luxd --config-file="$TEST_DIR/config.json" > "$TEST_DIR/luxd.log" 2>&1 &
LUXD_PID=$!

echo "Luxd started with PID: $LUXD_PID"
echo "Waiting for node to initialize..."
sleep 10

# Test RPC calls
echo ""
echo "=== Testing RPC Calls ==="

# Test 1: Get block number
echo "1. Testing eth_blockNumber:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9651/ext/bc/C/rpc | jq

# Test 2: Get block 0
echo ""
echo "2. Testing eth_getBlockByNumber for block 0:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0", false],"id":1}' \
  http://localhost:9651/ext/bc/C/rpc | jq

# Test 3: Get block 100
echo ""
echo "3. Testing eth_getBlockByNumber for block 100:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x64", false],"id":1}' \
  http://localhost:9651/ext/bc/C/rpc | jq

# Test 4: Get latest block
echo ""
echo "4. Testing eth_getBlockByNumber for latest block:"
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest", false],"id":1}' \
  http://localhost:9651/ext/bc/C/rpc | jq

# Test 5: Get balance of an address
echo ""
echo "5. Testing eth_getBalance for a known address:"
# Use the test address that should have funds
curl -s -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC", "latest"],"id":1}' \
  http://localhost:9651/ext/bc/C/rpc | jq

# Check logs for errors
echo ""
echo "=== Checking Logs ==="
echo "Last 20 lines of luxd.log:"
tail -20 "$TEST_DIR/luxd.log"

# Kill luxd
echo ""
echo "Stopping luxd..."
kill $LUXD_PID 2>/dev/null
wait $LUXD_PID 2>/dev/null

echo ""
echo "=== Test Complete ==="
echo "Full logs available at: $TEST_DIR/luxd.log"