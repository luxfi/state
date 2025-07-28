#!/bin/bash
set -e

echo "=== RPC Verification Test Script ==="
echo

# Set test root
export TEST_ROOT=$HOME/.tmp/cchain-mini

# Check if database exists
if [ ! -d "$TEST_ROOT/evm/pebbledb" ]; then
    echo "❌ Test database not found at $TEST_ROOT/evm/pebbledb"
    exit 1
fi

echo "✅ Found test database at $TEST_ROOT/evm/pebbledb"

# Step 1: Check current tip before rebuild
echo
echo "Step 1: Checking current tip..."
TIP=$(./bin/peek_tip_v2 --db $TEST_ROOT/evm/pebbledb 2>/dev/null | grep "maxHeight" | awk '{print $3}')
echo "Current tip = $TIP"

# Step 2: Rebuild canonical mappings
echo
echo "Step 2: Rebuilding canonical mappings..."
echo "Note: This database has namespace prefixes, so standard rebuild won't work"
echo "The tip shows we have $TIP blocks but only 10977 canonical mappings"

# Step 3: Try to launch node
echo
echo "Step 3: Attempting to launch node..."
echo "Command would be:"
echo "luxd \\"
echo "  --db-dir $TEST_ROOT \\"
echo "  --network-id 96369 \\"
echo "  --staking-enabled=false \\"
echo "  --http-port 9630 \\"
echo "  --chain-configs.enable-indexing"
echo
echo "Note: Node launch requires luxd binary"

# Step 4: Verify RPC would work
echo
echo "Step 4: RPC verification commands (requires running node):"
echo
echo "Block height check:"
echo 'curl -s --data '\''{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'\'' \'
echo '  http://localhost:9630/ext/bc/C/rpc | jq .result'
echo
echo "Expected result: 0x$(printf %x $TIP)"
echo
echo "Treasury balance check:"
echo 'curl -s --data '\''{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011e888251ab053b7bd1cdb598db4f9ded94714","latest"],"id":1}'\'' \'
echo '  http://localhost:9630/ext/bc/C/rpc | jq .result'
echo
echo "Expected result: >= 0x1B1AE4D6E2EF500000 (1.9T LUX)"

# Step 5: Summary
echo
echo "=== Summary ==="
echo "- Database tip: $TIP blocks"
echo "- Canonical mappings: 10977 (missing $(expr $TIP - 10977))"
echo "- Database format: Namespaced (33-byte prefix)"
echo "- Need: Custom rebuild tool for namespaced format"
echo "- RPC tests: Ready to run once node is launched"