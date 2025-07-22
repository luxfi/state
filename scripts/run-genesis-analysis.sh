#!/bin/bash
# Comprehensive Genesis Analysis - Optimized Version
# Uses efficient current state scanning and caching

set -e

echo "=== Comprehensive Genesis Analysis (Optimized) ==="
echo "Starting at: $(date)"
echo ""

# Configuration
OUTPUT_DIR="exports/genesis-analysis-$(date +%Y%m%d-%H%M%S)"
CACHE_DIR="cache/genesis-data"
mkdir -p "$OUTPUT_DIR" "$CACHE_DIR"

# RPC Endpoints
BSC_RPCS=(
    "https://bsc-dataseed.bnbchain.org"
    "https://bsc-dataseed.nariox.org"
    "https://bsc-dataseed.defibit.io"
    "https://bsc-dataseed.ninicoin.io"
    "https://bsc-dataseed1.binance.org"
    "https://bsc-dataseed2.binance.org"
)

ETH_RPCS=(
    "https://eth.llamarpc.com"
    "https://rpc.ankr.com/eth"
    "https://ethereum.publicnode.com"
)

# Contract Addresses
ZOO_TOKEN_BSC="0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
EGG_NFT_BSC="0x5bb68cf06289d54efde25155c88003be685356a8"
LUX_NFT_ETH="0x31e0f919c67cedd2bc3e294340dc900735810311"
DEAD_ADDRESS="0x000000000000000000000000000000000000dEaD"

echo "Building archeology tool..."
make build-archeology

echo ""
echo "=== Step 1: Scan Current EGG NFT Holders on BSC ==="
RPC_ARGS=""
for rpc in "${BSC_RPCS[@]}"; do
    RPC_ARGS="$RPC_ARGS --rpc $rpc"
done

./bin/archeology scan-current-holders \
    $RPC_ARGS \
    --contract "$EGG_NFT_BSC" \
    --project zoo \
    --concurrent 20 \
    --output "$OUTPUT_DIR/egg_nft_holders.csv" \
    --output-json "$OUTPUT_DIR/egg_nft_holders.json" || echo "Note: Some RPCs may have failed"

echo ""
echo "=== Step 2: Scan ZOO Token Burns to Dead Address (Cached) ==="
./bin/archeology scan-burns-cached \
    $RPC_ARGS \
    --token "$ZOO_TOKEN_BSC" \
    --burn-address "$DEAD_ADDRESS" \
    --from-block 14000000 \
    --cache-dir "$CACHE_DIR/zoo-burns" \
    --concurrent 10 \
    --batch-size 2000 \
    --output "$OUTPUT_DIR/zoo_burns.csv" \
    --output-json "$OUTPUT_DIR/zoo_burns.json" || echo "Note: Scan may be partial due to RPC limits"

echo ""
echo "=== Step 3: Scan Current LUX NFT Holders on Ethereum ==="
if [ -n "$INFURA_KEY" ]; then
    ETH_RPCS=("https://mainnet.infura.io/v3/$INFURA_KEY" "${ETH_RPCS[@]}")
fi

RPC_ARGS=""
for rpc in "${ETH_RPCS[@]}"; do
    RPC_ARGS="$RPC_ARGS --rpc $rpc"
done

./bin/archeology scan-current-holders \
    $RPC_ARGS \
    --contract "$LUX_NFT_ETH" \
    --project lux \
    --concurrent 10 \
    --output "$OUTPUT_DIR/lux_nft_holders.csv" \
    --output-json "$OUTPUT_DIR/lux_nft_holders.json" || echo "Note: Ethereum scan may require API key"

echo ""
echo "=== Step 4: Extract Local Chain Data ==="

# Extract LUX 7777 if not already done
if [ ! -d "data/extracted/lux-genesis-7777" ]; then
    echo "Extracting LUX 7777 data..."
    ./bin/archeology extract \
        --src chaindata/lux-genesis-7777/db/pebbledb \
        --dst data/extracted/lux-genesis-7777 \
        --chain-id 7777 \
        --include-state || echo "7777 extraction failed"
fi

# Extract LUX 96369 if not already done
if [ ! -d "data/extracted/lux-96369" ]; then
    echo "Extracting LUX 96369 data..."
    ./bin/archeology extract \
        --src chaindata/lux-mainnet-96369/db/pebbledb \
        --dst data/extracted/lux-96369 \
        --chain-id 96369 \
        --include-state || echo "96369 extraction failed"
fi

# Extract ZOO 200200 if not already done
if [ ! -d "data/extracted/zoo-200200" ]; then
    echo "Extracting ZOO 200200 data..."
    ./bin/archeology extract \
        --src chaindata/zoo-mainnet-200200/db/pebbledb \
        --dst data/extracted/zoo-200200 \
        --chain-id 200200 \
        --include-state || echo "200200 extraction failed"
fi

echo ""
echo "=== Step 5: Analyze Extracted Chain Data ==="

# Analyze each chain
for chain in "lux-genesis-7777" "lux-96369" "zoo-200200"; do
    if [ -d "data/extracted/$chain" ]; then
        echo "Analyzing $chain..."
        ./bin/archeology analyze \
            -db "data/extracted/$chain" \
            -network "$chain" \
            --output "$OUTPUT_DIR/${chain}_accounts.csv" \
            --output-json "$OUTPUT_DIR/${chain}_accounts.json" \
            --exclude-zero-balance || echo "$chain analysis failed"
    fi
done

echo ""
echo "=== Step 6: Generate Cross-Reference Report ==="
cat > "$OUTPUT_DIR/cross_reference.py" << 'EOF'
#!/usr/bin/env python3
import json
import csv
from collections import defaultdict

def load_json(path):
    try:
        with open(path) as f:
            return json.load(f)
    except:
        print(f"Warning: Could not load {path}")
        return {}

# Load all data
print("Loading data files...")

# NFT holders
egg_holders = load_json('egg_nft_holders.json')
lux_nft_holders = load_json('lux_nft_holders.json')

# ZOO burns
zoo_burns = load_json('zoo_burns.json')

# Chain accounts
lux_7777 = load_json('lux-genesis-7777_accounts.json')
lux_96369 = load_json('lux-96369_accounts.json')
zoo_200200 = load_json('zoo-200200_accounts.json')

# Process data
xchain_eligible = {
    'lux': {
        'from_7777_not_in_96369': {},
        'nft_holders': {},
        'validator_eligible': set()
    },
    'zoo': {
        'bsc_burners': {},
        'egg_holders': {},
        'undelivered_burns': {}
    }
}

# LUX: Find 7777 holders not in 96369
print("\nProcessing LUX holders...")
if lux_7777 and lux_96369:
    for addr in lux_7777.get('accounts', {}):
        if addr.lower() not in [a.lower() for a in lux_96369.get('accounts', {})]:
            xchain_eligible['lux']['from_7777_not_in_96369'][addr] = lux_7777['accounts'][addr]

# LUX: Add NFT holders (all are validator eligible)
if lux_nft_holders.get('holders'):
    for addr, info in lux_nft_holders['holders'].items():
        xchain_eligible['lux']['nft_holders'][addr] = info
        xchain_eligible['lux']['validator_eligible'].add(addr)

# ZOO: Process burners
print("\nProcessing ZOO burns...")
if zoo_burns.get('burns_by_address'):
    for addr, amount in zoo_burns['burns_by_address'].items():
        # Check if they're in 200200
        in_200200 = addr.lower() in [a.lower() for a in zoo_200200.get('accounts', {})]
        
        if not in_200200:
            xchain_eligible['zoo']['undelivered_burns'][addr] = {
                'burned_amount': amount,
                'in_200200': False
            }

# ZOO: Add EGG holders
if egg_holders.get('holders'):
    for addr, info in egg_holders['holders'].items():
        xchain_eligible['zoo']['egg_holders'][addr] = {
            'eggs': info.get('token_count', 0),
            'zoo_equivalent': info.get('zoo_equivalent', 0)
        }

# Generate summary
print("\n=== X-Chain Genesis Summary ===")
print(f"\nLUX:")
print(f"  - From 7777 (not in 96369): {len(xchain_eligible['lux']['from_7777_not_in_96369'])}")
print(f"  - NFT holders (Ethereum): {len(xchain_eligible['lux']['nft_holders'])}")
print(f"  - Validator eligible: {len(xchain_eligible['lux']['validator_eligible'])}")

print(f"\nZOO:")
print(f"  - EGG NFT holders: {len(xchain_eligible['zoo']['egg_holders'])}")
print(f"  - Undelivered burns: {len(xchain_eligible['zoo']['undelivered_burns'])}")

# Save results
with open('xchain_eligible.json', 'w') as f:
    json.dump(xchain_eligible, f, indent=2, default=str)

# Generate validator list
with open('lux_validators.txt', 'w') as f:
    f.write("# LUX Validator Eligible Addresses\n")
    f.write("# NFT Holders + 1M+ LUX holders\n\n")
    
    # Add NFT holders
    f.write("## NFT Holders (All Eligible)\n")
    for addr in sorted(xchain_eligible['lux']['validator_eligible']):
        f.write(f"{addr}\n")
    
    # Check for 1M+ LUX holders
    f.write("\n## 1M+ LUX Holders from 7777\n")
    for addr, data in xchain_eligible['lux']['from_7777_not_in_96369'].items():
        try:
            balance = int(data.get('balance', '0'))
            if balance >= 1000000 * 10**18:  # 1M LUX
                f.write(f"{addr} # {balance / 10**18:.2f} LUX\n")
        except:
            pass

print("\nFiles generated:")
print("  - xchain_eligible.json: All X-Chain eligible addresses")
print("  - lux_validators.txt: Validator eligible addresses")
EOF

cd "$OUTPUT_DIR" && python3 cross_reference.py

echo ""
echo "=== Step 7: Generate Final Report ==="
cat > "$OUTPUT_DIR/GENESIS_REPORT.md" << EOF
# Genesis Analysis Report

Generated: $(date)
Output Directory: $OUTPUT_DIR

## Data Sources Analyzed

### External Chains
- **BSC**: EGG NFT holders, ZOO token burns
- **Ethereum**: LUX NFT holders

### Local Chains
- **LUX 7777**: Historical holders
- **LUX 96369**: Current mainnet holders  
- **ZOO 200200**: Current Zoo chain holders

## Key Files Generated

### NFT Holders
- \`egg_nft_holders.json\`: Current EGG NFT holders on BSC
- \`lux_nft_holders.json\`: Current LUX NFT holders on Ethereum

### Token Burns
- \`zoo_burns.json\`: ZOO tokens burned to dead address on BSC
- Cache stored in: $CACHE_DIR/zoo-burns

### Chain Accounts
- \`lux-genesis-7777_accounts.json\`: All 7777 accounts
- \`lux-96369_accounts.json\`: All 96369 accounts
- \`zoo-200200_accounts.json\`: All Zoo mainnet accounts

### Cross-Reference
- \`xchain_eligible.json\`: All addresses eligible for X-Chain genesis
- \`lux_validators.txt\`: Addresses eligible to run validators

## Bootstrap Validators
Location: chaindata/lux-mainnet-96369/bootnodes.json
- 11 initial validators configured
- Each with 1B LUX staked
- 100-year vesting (1% per year)

## Next Steps
1. Review xchain_eligible.json for accuracy
2. Verify bootstrap validator addresses
3. Generate final X-Chain genesis using eligible addresses
4. Deploy network with initial validators
EOF

echo ""
echo "=== Analysis Complete ==="
echo "Results saved to: $OUTPUT_DIR"
echo ""
echo "Key files:"
echo "  - X-Chain eligible: $OUTPUT_DIR/xchain_eligible.json"
echo "  - Validator list: $OUTPUT_DIR/lux_validators.txt"
echo "  - Full report: $OUTPUT_DIR/GENESIS_REPORT.md"
echo "  - Bootstrap nodes: chaindata/lux-mainnet-96369/bootnodes.json"
echo ""
echo "Completed at: $(date)"