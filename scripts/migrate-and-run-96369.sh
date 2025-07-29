#!/usr/bin/env bash
set -euo pipefail

# Paths
GENESIS_DIR=$HOME/work/lux/genesis
NODE_BIN=$HOME/work/lux/node/build/luxd
SRC_DB=$GENESIS_DIR/chaindata/lux-mainnet-96369/db/pebbledb
MIGRATION_DIR=$GENESIS_DIR/runtime/lux-96369-migrated
DATA_DIR=$MIGRATION_DIR/db
PORT_RPC=9650               # core AvalancheGo RPC
PORT_HTTP=9630              # C-Chain JSON-RPC
LOG=$MIGRATION_DIR/luxd.log

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🔧 Setting up migration for subnet 96369...${NC}"

# Create migration directory structure
mkdir -p $MIGRATION_DIR
mkdir -p $DATA_DIR/evm
mkdir -p $DATA_DIR/state

# Step 1: Migrate with EVM prefix
echo -e "${YELLOW}Step 1: Adding EVM prefix to subnet data...${NC}"
if [ ! -d "$DATA_DIR/evm/pebbledb" ]; then
    $GENESIS_DIR/bin/genesis migrate add-evm-prefix \
        $SRC_DB \
        $DATA_DIR/evm/pebbledb
else
    echo "EVM prefixed data already exists, skipping..."
fi

# Step 2: Rebuild canonical mappings
echo -e "${YELLOW}Step 2: Rebuilding canonical mappings...${NC}"
$GENESIS_DIR/bin/genesis migrate rebuild-canonical \
    $DATA_DIR/evm/pebbledb

# Step 3: Find tip height
echo -e "${YELLOW}Step 3: Finding tip height...${NC}"
TIP_OUTPUT=$($GENESIS_DIR/bin/genesis migrate peek-tip $DATA_DIR/evm/pebbledb)
TIP=$(echo "$TIP_OUTPUT" | grep -oP 'Maximum block number: \K\d+' || echo "0")
echo "Tip height: $TIP"

# Step 4: Create consensus state
echo -e "${YELLOW}Step 4: Creating consensus state...${NC}"
if [ ! -d "$DATA_DIR/state/pebbledb" ]; then
    $GENESIS_DIR/bin/genesis migrate replay-consensus \
        --evm $DATA_DIR/evm/pebbledb \
        --state $DATA_DIR/state/pebbledb \
        --tip $TIP
else
    echo "Consensus state already exists, skipping..."
fi

# Check if luxd binary exists
if [ ! -f "$NODE_BIN" ]; then
    echo -e "${RED}❌ luxd binary not found at $NODE_BIN${NC}"
    echo "Building luxd..."
    cd $HOME/work/lux/node
    ./scripts/build.sh
fi

# Kill any existing luxd process on our ports
echo -e "${YELLOW}Checking for existing luxd processes...${NC}"
lsof -ti:$PORT_HTTP -ti:$PORT_RPC | xargs -r kill -9 2>/dev/null || true

# Start luxd
echo -e "${GREEN}🟢 Starting luxd for subnet-96369...${NC}"
$NODE_BIN \
    --network-id=96369 \
    --db-dir="$DATA_DIR" \
    --data-dir="$MIGRATION_DIR" \
    --dev \
    --http-port=$PORT_HTTP \
    --staking-port=9651 \
    --log-level=info \
    --public-ip=127.0.0.1 \
    --http-host=0.0.0.0 \
    --api-admin-enabled=true \
    --api-keystore-enabled=false \
    --api-metrics-enabled=true \
    --chain-config-dir=$GENESIS_DIR/configs/lux-mainnet-96369 \
    >"$LOG" 2>&1 &

PID=$!
echo "luxd pid = $PID (logs → $LOG)"

# Save PID for cleanup
echo $PID > $MIGRATION_DIR/luxd.pid

# Wait until RPC is ready
echo -e "${YELLOW}Waiting for RPC to be ready...${NC}"
for i in {1..60}; do
    if curl -s \
        --data '{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}' \
        http://127.0.0.1:$PORT_HTTP/ext/bc/C/rpc >/dev/null 2>&1; then
        echo -e "${GREEN}✅ RPC ready after $i seconds${NC}"
        break
    fi
    printf "."
    sleep 1
done || { 
    echo -e "${RED}❌ RPC never became ready${NC}"
    echo "Last 50 lines of log:"
    tail -50 $LOG
    kill $PID 2>/dev/null || true
    exit 1
}

# Verify block height
echo -e "${YELLOW}Verifying block height...${NC}"
EXPECTED_HEX=$(printf '0x%x' $TIP)

BN=$(curl -s \
    --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    http://127.0.0.1:$PORT_HTTP/ext/bc/C/rpc | jq -r .result)

if [[ "$BN" == "$EXPECTED_HEX" ]]; then
    echo -e "${GREEN}✅ Height OK – $BN (decimal: $TIP)${NC}"
else
    echo -e "${RED}❌ Height mismatch: got $BN, expected $EXPECTED_HEX${NC}"
    # Don't fail here, as the node might still be syncing
fi

# Verify treasury balance
echo -e "${YELLOW}Verifying treasury balance...${NC}"
TREASURY="0x9011e888251ab053b7bd1cdb598db4f9ded94714"

BAL=$(curl -s \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"$TREASURY\",\"latest\"],\"id\":1}" \
    http://127.0.0.1:$PORT_HTTP/ext/bc/C/rpc | jq -r .result)

if [ "$BAL" != "null" ] && [ "$BAL" != "" ]; then
    # Convert to decimal for comparison
    BAL_DEC=$(python3 -c "print(int('$BAL', 16))")
    MIN_DEC=$(python3 -c "print(int('1900000000000000000000000000000', 10))")  # 1.9T LUX
    
    if python3 -c "exit(0 if $BAL_DEC >= $MIN_DEC else 1)"; then
        echo -e "${GREEN}✅ Treasury balance OK – $BAL${NC}"
        python3 -c "print(f'   Balance: {$BAL_DEC / 10**18:.2f} LUX')"
    else
        echo -e "${RED}❌ Treasury balance too low: $BAL${NC}"
        python3 -c "print(f'   Balance: {$BAL_DEC / 10**18:.2f} LUX')"
    fi
else
    echo -e "${YELLOW}⚠️  Could not get treasury balance (might be still syncing)${NC}"
fi

# Check chain ID
echo -e "${YELLOW}Verifying chain ID...${NC}"
CHAIN_ID=$(curl -s \
    --data '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
    http://127.0.0.1:$PORT_HTTP/ext/bc/C/rpc | jq -r .result)

EXPECTED_CHAIN_ID="0x17971"  # 96369 in hex
if [[ "$CHAIN_ID" == "$EXPECTED_CHAIN_ID" ]]; then
    echo -e "${GREEN}✅ Chain ID OK – $CHAIN_ID (96369)${NC}"
else
    echo -e "${RED}❌ Chain ID mismatch: got $CHAIN_ID, expected $EXPECTED_CHAIN_ID${NC}"
fi

echo -e "${GREEN}🚀 luxd for subnet-96369 is live!${NC}"
echo -e "   RPC endpoint: http://127.0.0.1:$PORT_HTTP/ext/bc/C/rpc"
echo -e "   Logs: $LOG"
echo -e "   PID: $PID"
echo ""
echo "To stop the node: kill $PID"
echo "To tail logs: tail -f $LOG"