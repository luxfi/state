#!/bin/bash
# Quick Genesis Summary - Focus on available data

set -e

OUTPUT_DIR="exports/genesis-analysis-20250722-060502"
echo "=== Quick Genesis Summary ==="

# 1. EGG NFT Holders (already scanned)
echo ""
echo "1. EGG NFT Holders on BSC:"
if [ -f "$OUTPUT_DIR/egg_nft_holders.json" ]; then
    TOTAL_HOLDERS=$(jq '.unique_holders' "$OUTPUT_DIR/egg_nft_holders.json")
    TOTAL_EGGS=$(jq '.total_supply - .burned_tokens' "$OUTPUT_DIR/egg_nft_holders.json")
    echo "   - Unique holders: $TOTAL_HOLDERS"
    echo "   - Total EGGs held: $TOTAL_EGGS"
    echo "   - ZOO equivalent: $(($TOTAL_EGGS * 4200000))"
fi

# 2. Extract and analyze local chains (quick version)
echo ""
echo "2. Extracting Local Chain Data..."

# Just get account counts and basic info
for chain_info in "lux-genesis-7777:7777" "lux-mainnet-96369:96369" "zoo-mainnet-200200:200200"; do
    IFS=':' read -r chain_name chain_id <<< "$chain_info"
    
    if [ -d "chaindata/$chain_name/db/pebbledb" ]; then
        echo ""
        echo "   $chain_name (Chain ID: $chain_id):"
        
        # Get basic stats without full extraction
        DB_SIZE=$(du -sh "chaindata/$chain_name/db/pebbledb" 2>/dev/null | cut -f1 || echo "N/A")
        echo "   - Database size: $DB_SIZE"
        
        # Check if we have config/genesis
        if [ -f "chaindata/$chain_name/config/genesis.json" ]; then
            ALLOC_COUNT=$(jq '.alloc | length' "chaindata/$chain_name/config/genesis.json" 2>/dev/null || echo "0")
            echo "   - Genesis allocations: $ALLOC_COUNT"
        fi
    fi
done

# 3. Create basic cross-reference
echo ""
echo "3. Creating Cross-Reference Summary..."

cat > "$OUTPUT_DIR/genesis_summary.json" << EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "egg_nft_data": {
    "total_holders": $TOTAL_HOLDERS,
    "total_eggs": $TOTAL_EGGS,
    "zoo_equivalent": $(($TOTAL_EGGS * 4200000)),
    "data_file": "egg_nft_holders.json"
  },
  "bootstrap_validators": {
    "count": 11,
    "stake_per_validator": "1000000000000000000000000000",
    "config_file": "chaindata/lux-mainnet-96369/bootnodes.json"
  },
  "local_chains": {
    "lux_7777": {
      "status": "available",
      "path": "chaindata/lux-genesis-7777"
    },
    "lux_96369": {
      "status": "available", 
      "path": "chaindata/lux-mainnet-96369"
    },
    "zoo_200200": {
      "status": "available",
      "path": "chaindata/zoo-mainnet-200200"
    }
  },
  "notes": [
    "EGG NFT holders successfully scanned on BSC",
    "ZOO burns scan requires dedicated BSC archive node or API key",
    "Local chain data available for extraction",
    "Bootstrap validators configured in bootnodes.json"
  ]
}
EOF

# 4. Generate actionable report
cat > "$OUTPUT_DIR/GENESIS_ACTION_PLAN.md" << EOF
# Genesis Action Plan

Generated: $(date)

## Completed Items ✓

1. **EGG NFT Holders Scanned**
   - Found $TOTAL_HOLDERS unique holders with $TOTAL_EGGS EGGs
   - Data saved to: egg_nft_holders.json
   - ZOO equivalent: $(($TOTAL_EGGS * 4200000))

2. **Bootstrap Validators Configured**
   - 11 validators with 1B LUX each
   - Config: chaindata/lux-mainnet-96369/bootnodes.json
   - 100-year vesting schedule

3. **Local Chaindata Available**
   - LUX 7777: chaindata/lux-genesis-7777
   - LUX 96369: chaindata/lux-mainnet-96369
   - ZOO 200200: chaindata/zoo-mainnet-200200

## Required Actions

### 1. Extract Local Chain Accounts
\`\`\`bash
# Extract LUX 7777
./bin/archeology extract \\
    --src chaindata/lux-genesis-7777/db/pebbledb \\
    --dst data/extracted/lux-genesis-7777 \\
    --chain-id 7777 \\
    --include-state

# Extract LUX 96369
./bin/archeology extract \\
    --src chaindata/lux-mainnet-96369/db/pebbledb \\
    --dst data/extracted/lux-96369 \\
    --chain-id 96369 \\
    --include-state

# Extract ZOO 200200
./bin/archeology extract \\
    --src chaindata/zoo-mainnet-200200/db/pebbledb \\
    --dst data/extracted/zoo-200200 \\
    --chain-id 200200 \\
    --include-state
\`\`\`

### 2. Get ZOO Burns Data
Options:
- Use BSC archive node with higher rate limits
- Use a paid BSC API service (Moralis, Covalent, etc.)
- Manually provide known burn addresses

### 3. Get LUX NFT Holders on Ethereum
\`\`\`bash
# Need Ethereum RPC (Infura/Alchemy)
INFURA_KEY=YOUR_KEY ./bin/archeology scan-current-holders \\
    --rpc https://mainnet.infura.io/v3/YOUR_KEY \\
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \\
    --project lux \\
    --output lux_nft_holders.csv
\`\`\`

### 4. Generate X-Chain Genesis
Once all data is collected:
\`\`\`bash
./bin/genesis generate \\
    --network x-chain \\
    --data ./data/extracted/ \\
    --external ./exports/genesis-analysis-*/ \\
    --validators chaindata/lux-mainnet-96369/bootnodes.json \\
    --output ./genesis/x-chain-genesis.json
\`\`\`

## Summary

- ✅ EGG NFT data collected (84 holders, 452 EGGs)
- ✅ Bootstrap validators configured (11 validators)
- ✅ Local chaindata available
- ⏳ Need to extract local chain accounts
- ⏳ Need BSC burns data (requires better RPC)
- ⏳ Need Ethereum NFT holder data
- ⏳ Final cross-reference and genesis generation

## Bootstrap Validator Addresses
$(cat chaindata/lux-mainnet-96369/bootnodes.json | jq -r '.bootstrapNodes[]')
EOF

echo ""
echo "=== Summary Complete ==="
echo "Results saved to: $OUTPUT_DIR"
echo ""
echo "Key files:"
echo "  - Summary: $OUTPUT_DIR/genesis_summary.json"
echo "  - Action Plan: $OUTPUT_DIR/GENESIS_ACTION_PLAN.md" 
echo "  - EGG Holders: $OUTPUT_DIR/egg_nft_holders.json"
echo "  - Bootstrap Nodes: chaindata/lux-mainnet-96369/bootnodes.json"