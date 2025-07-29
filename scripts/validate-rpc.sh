#!/bin/bash
# Validate RPC endpoints after migration

set -e

RPC_URL="${RPC_URL:-http://localhost:9650/ext/bc/C/rpc}"
TREASURY="0x9011e888251ab053b7bd1cdb598db4f9ded94714"

echo "🔍 Validating RPC endpoints at $RPC_URL"
echo ""

# Check if RPC is accessible
echo -n "✓ Checking RPC availability... "
if curl -s -f -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
    "$RPC_URL" > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    echo "❌ RPC endpoint not accessible at $RPC_URL"
    exit 1
fi

# Get block number
echo -n "✓ Getting block number... "
BLOCK_HEX=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result')
BLOCK_DEC=$((16#${BLOCK_HEX#0x}))
echo "$BLOCK_DEC (hex: $BLOCK_HEX)"

# Get chain ID
echo -n "✓ Getting chain ID... "
CHAIN_HEX=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    "$RPC_URL" | jq -r '.result')
CHAIN_DEC=$((16#${CHAIN_HEX#0x}))
echo "$CHAIN_DEC"

if [ "$CHAIN_DEC" != "96369" ]; then
    echo "⚠️  Warning: Chain ID is $CHAIN_DEC, expected 96369"
fi

# Get genesis block
echo -n "✓ Getting genesis block... "
GENESIS=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",false],"id":1}' \
    "$RPC_URL" | jq -r '.result.hash')
echo "$GENESIS"

# Get latest block
echo -n "✓ Getting latest block... "
LATEST=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["latest",false],"id":1}' \
    "$RPC_URL" | jq -r '.result.hash')
echo "$LATEST"

# Check treasury balance
echo -n "✓ Checking treasury balance... "
BALANCE_HEX=$(curl -s -X POST -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$TREASURY\",\"latest\"],\"id\":1}" \
    "$RPC_URL" | jq -r '.result')

# Convert balance to decimal (wei)
BALANCE_WEI=$(echo "ibase=16; ${BALANCE_HEX#0x}" | bc 2>/dev/null || echo "0")

# Convert to LUX (divide by 10^18)
if command -v python3 > /dev/null; then
    BALANCE_LUX=$(python3 -c "print(f'{int('$BALANCE_WEI') / 10**18:,.0f}')")
    echo "$BALANCE_LUX LUX"
    
    # Check if > 1.9T
    if python3 -c "exit(0 if int('$BALANCE_WEI') >= 19 * 10**29 else 1)"; then
        echo "  ✅ Balance is >= 1.9T LUX as expected"
    else
        echo "  ⚠️  Balance is less than 1.9T LUX"
    fi
else
    echo "$BALANCE_HEX"
fi

# Test eth_getLogs
echo -n "✓ Testing eth_getLogs... "
LOGS=$(curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_getLogs","params":[{"fromBlock":"latest","toBlock":"latest"}],"id":1}' \
    "$RPC_URL" | jq -r '.result')
if [ "$LOGS" != "null" ]; then
    LOG_COUNT=$(echo "$LOGS" | jq 'length')
    echo "OK (found $LOG_COUNT logs)"
else
    echo "FAILED"
fi

echo ""
echo "✅ RPC validation complete!"
echo ""
echo "Summary:"
echo "  Chain ID: $CHAIN_DEC"
echo "  Block Height: $BLOCK_DEC"
echo "  Treasury Balance: $BALANCE_LUX LUX"
echo ""
echo "Your C-Chain is ready at $RPC_URL"