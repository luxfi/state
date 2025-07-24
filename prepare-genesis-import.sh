#!/bin/bash
# Prepare genesis data for import into C-Chain and L2s

set -e

echo "🔧 Preparing Genesis Data for Import"
echo "===================================="

GENESIS_DIR="/home/z/work/lux/genesis"
OUTPUT_DIR="$GENESIS_DIR/output"
IMPORT_DIR="$OUTPUT_DIR/import-ready"

# Create directories
mkdir -p "$IMPORT_DIR"/{lux,zoo,spc}/{C,L2}

# Function to prepare chain data for import
prepare_chain_data() {
    local network=$1
    local chain_id=$2
    local source_dir=$3
    local target_dir=$4
    
    echo "Preparing $network (Chain ID: $chain_id)..."
    
    # Check if source PebbleDB exists
    if [ -d "$source_dir/db/pebbledb" ]; then
        echo "  ✅ Found PebbleDB at $source_dir/db/pebbledb"
        
        # Copy PebbleDB for import
        echo "  📦 Copying PebbleDB for import..."
        cp -r "$source_dir/db/pebbledb" "$target_dir/chaindata"
        
        # Copy genesis.json
        if [ -f "$source_dir/../configs/$network/genesis.json" ]; then
            cp "$source_dir/../configs/$network/genesis.json" "$target_dir/genesis.json"
        elif [ -f "$GENESIS_DIR/chaindata/configs/$network/genesis.json" ]; then
            cp "$GENESIS_DIR/chaindata/configs/$network/genesis.json" "$target_dir/genesis.json"
        fi
        
        # Create import metadata
        cat > "$target_dir/import-info.json" <<EOF
{
    "network": "$network",
    "chainId": $chain_id,
    "dataSource": "$source_dir",
    "importType": "pebbledb",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
        echo "  ✅ $network data prepared for import"
    else
        echo "  ⚠️  No PebbleDB found for $network, will use genesis only"
        
        # Just copy genesis.json
        if [ -f "$GENESIS_DIR/chaindata/configs/$network/genesis.json" ]; then
            cp "$GENESIS_DIR/chaindata/configs/$network/genesis.json" "$target_dir/genesis.json"
            echo "  ✅ Genesis file copied"
        fi
    fi
}

# Prepare LUX mainnet data
prepare_chain_data "lux-mainnet-96369" 96369 \
    "$GENESIS_DIR/chaindata/lux-mainnet-96369" \
    "$IMPORT_DIR/lux/C"

# Prepare ZOO L2 data
prepare_chain_data "zoo-mainnet-200200" 200200 \
    "$GENESIS_DIR/chaindata/zoo-mainnet-200200" \
    "$IMPORT_DIR/zoo/L2"

# Prepare SPC L2 data
prepare_chain_data "spc-mainnet-36911" 36911 \
    "$GENESIS_DIR/chaindata/spc-mainnet-36911" \
    "$IMPORT_DIR/spc/L2"

# Create combined genesis with BSC migration data for ZOO
echo ""
echo "🔄 Merging BSC migration data for ZOO..."
if [ -f "$GENESIS_DIR/exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.json" ]; then
    python3 <<EOF
import json

# Load base genesis
with open('$IMPORT_DIR/zoo/L2/genesis.json', 'r') as f:
    genesis = json.load(f)

# Load egg allocations
with open('$GENESIS_DIR/exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.json', 'r') as f:
    egg_data = json.load(f)

# Merge allocations
if 'alloc' not in genesis:
    genesis['alloc'] = {}

for address, data in egg_data.get('allocations', {}).items():
    # Each egg = 4.2M ZOO
    balance_wei = str(int(data['zoo_equivalent']) * 10**18)
    genesis['alloc'][address.lower()] = {"balance": balance_wei}

# Save merged genesis
with open('$IMPORT_DIR/zoo/L2/genesis-with-migration.json', 'w') as f:
    json.dump(genesis, f, indent=2)

print(f"✅ Added {len(egg_data.get('allocations', {}))} egg holder allocations to ZOO genesis")
EOF
fi

# Create launch configuration
echo ""
echo "📝 Creating launch configuration..."
cat > "$IMPORT_DIR/launch-config.json" <<EOF
{
    "networks": {
        "lux": {
            "type": "primary",
            "chainId": 96369,
            "dataPath": "$IMPORT_DIR/lux/C",
            "rpcPort": 9650,
            "consensusParameters": {
                "snow-sample-size": 1,
                "snow-quorum-size": 1,
                "snow-virtuous-commit-threshold": 1,
                "snow-rogue-commit-threshold": 1
            }
        },
        "zoo": {
            "type": "L2",
            "chainId": 200200,
            "dataPath": "$IMPORT_DIR/zoo/L2",
            "genesisFile": "$IMPORT_DIR/zoo/L2/genesis-with-migration.json"
        },
        "spc": {
            "type": "L2",
            "chainId": 36911,
            "dataPath": "$IMPORT_DIR/spc/L2",
            "genesisFile": "$IMPORT_DIR/spc/L2/genesis.json"
        }
    }
}
EOF

# Create import instructions
cat > "$IMPORT_DIR/IMPORT-INSTRUCTIONS.md" <<EOF
# Import Instructions

## LUX Mainnet (C-Chain)

To import the LUX mainnet data:

\`\`\`bash
# Copy chaindata to luxd data directory
cp -r $IMPORT_DIR/lux/C/chaindata ~/.luxd/chains/C/

# Launch luxd with the imported data
luxd --network-id=96369 \\
     --chain-data-dir=$IMPORT_DIR/lux/C/chaindata \\
     --http-host=0.0.0.0 \\
     --staking-enabled=false \\
     --sybil-protection-enabled=false \\
     --snow-sample-size=1 \\
     --snow-quorum-size=1
\`\`\`

## ZOO L2

To create and deploy ZOO L2 with migration data:

\`\`\`bash
lux blockchain create zoo \\
    --evm \\
    --genesis-file $IMPORT_DIR/zoo/L2/genesis-with-migration.json \\
    --chain-id 200200

lux blockchain deploy zoo --local
\`\`\`

## SPC L2

To create and deploy SPC L2:

\`\`\`bash
lux blockchain create spc \\
    --evm \\
    --genesis-file $IMPORT_DIR/spc/L2/genesis.json \\
    --chain-id 36911

lux blockchain deploy spc --local
\`\`\`
EOF

echo ""
echo "✅ Genesis data prepared for import!"
echo ""
echo "Import-ready data location: $IMPORT_DIR"
echo ""
echo "Contents:"
find "$IMPORT_DIR" -type f -name "*.json" -o -name "genesis.json" | sort
echo ""
echo "To launch the full network, run:"
echo "  ./launch-full-network.sh"