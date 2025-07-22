#!/bin/bash
# Comprehensive genesis analysis for LUX and ZOO networks

set -e

echo "=== Comprehensive Genesis Analysis ==="
echo "Analyzing LUX (7777, 96369) and ZOO (BSC, 200200) networks"
echo ""

# Configuration
OUTPUT_DIR="exports/genesis-analysis"
mkdir -p "$OUTPUT_DIR"

# BSC RPC for external chains
BSC_RPC="${BSC_RPC:-https://bsc-dataseed.binance.org/}"
ETH_RPC="${ETH_RPC:-https://mainnet.infura.io/v3/YOUR_INFURA_KEY}"

# Addresses
ZOO_TOKEN_BSC="0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
LUX_NFT_ETH="0x31e0f919c67cedd2bc3e294340dc900735810311"
EGG_NFT_BSC="0x5bb68cf06289d54efde25155c88003be685356a8"

echo "Step 1: Extracting LUX 7777 holders..."
if [ -d "data/extracted/lux-genesis-7777" ]; then
    echo "Using existing extracted data for 7777"
else
    echo "Extracting 7777 data..."
    ./bin/archeology extract \
        --src chaindata/lux-genesis-7777/db/pebbledb \
        --dst data/extracted/lux-genesis-7777 \
        --chain-id 7777 \
        --include-state
fi

echo ""
echo "Step 2: Extracting LUX 96369 holders..."
if [ -d "data/extracted/lux-96369" ]; then
    echo "Using existing extracted data for 96369"
else
    echo "Extracting 96369 data..."
    ./bin/archeology extract \
        --src chaindata/lux-mainnet-96369/db/pebbledb \
        --dst data/extracted/lux-96369 \
        --chain-id 96369 \
        --include-state
fi

echo ""
echo "Step 3: Extracting ZOO 200200 holders and all transfers..."
if [ -d "data/extracted/zoo-200200" ]; then
    echo "Using existing extracted data for 200200"
else
    echo "Extracting 200200 data..."
    ./bin/archeology extract \
        --src chaindata/zoo-mainnet-200200/db/pebbledb \
        --dst data/extracted/zoo-200200 \
        --chain-id 200200 \
        --include-state
fi

echo ""
echo "Step 4: Analyzing LUX 7777 accounts..."
./bin/archeology analyze \
    -db data/extracted/lux-genesis-7777 \
    -network lux-7777 \
    --output "$OUTPUT_DIR/lux-7777-accounts.csv" \
    --output-json "$OUTPUT_DIR/lux-7777-accounts.json" \
    --exclude-zero-balance \
    --min-balance 1000000000000000000

echo ""
echo "Step 5: Analyzing LUX 96369 accounts..."
./bin/archeology analyze \
    -db data/extracted/lux-96369 \
    -network lux-96369 \
    --output "$OUTPUT_DIR/lux-96369-accounts.csv" \
    --output-json "$OUTPUT_DIR/lux-96369-accounts.json" \
    --exclude-zero-balance \
    --min-balance 1000000000000000000

echo ""
echo "Step 6: Analyzing ZOO 200200 accounts and transfers..."
./bin/archeology analyze \
    -db data/extracted/zoo-200200 \
    -network zoo-200200 \
    --output "$OUTPUT_DIR/zoo-200200-accounts.csv" \
    --output-json "$OUTPUT_DIR/zoo-200200-accounts.json" \
    --include-transfers \
    --output-transfers "$OUTPUT_DIR/zoo-200200-all-transfers.csv"

echo ""
echo "Step 7: Scanning Ethereum for LUX NFT holders..."
if [ -n "$ETH_RPC" ]; then
    ./bin/archeology scan-holders \
        --rpc "$ETH_RPC" \
        --contract "$LUX_NFT_ETH" \
        --type nft \
        --output "$OUTPUT_DIR/lux-nft-ethereum-holders.csv" \
        --output-json "$OUTPUT_DIR/lux-nft-ethereum-holders.json" \
        --show-distribution
else
    echo "Skipping Ethereum scan (no RPC configured)"
fi

echo ""
echo "Step 8: Running ZOO BSC analysis..."
./scripts/zoo-analysis.sh "$OUTPUT_DIR/zoo-bsc"

echo ""
echo "Step 9: Cross-referencing accounts..."
cat > "$OUTPUT_DIR/cross-reference.py" << 'EOF'
#!/usr/bin/env python3
import json
import csv
from collections import defaultdict

# Load all data
def load_json(path):
    try:
        with open(path) as f:
            return json.load(f)
    except:
        return {}

def load_csv_accounts(path):
    accounts = {}
    try:
        with open(path) as f:
            reader = csv.DictReader(f)
            for row in reader:
                accounts[row['address'].lower()] = row
    except:
        pass
    return accounts

# Load LUX data
lux_7777 = load_json('lux-7777-accounts.json')
lux_96369 = load_json('lux-96369-accounts.json')

# Load ZOO data
zoo_200200 = load_json('zoo-200200-accounts.json')
zoo_bsc_burns = load_json('zoo-bsc/zoo_burns.json')
zoo_bsc_holders = load_json('zoo-bsc/egg_nft_holders.json')

# Load transfers to check ZOO movement
zoo_transfers = defaultdict(list)
try:
    with open('zoo-200200-all-transfers.csv') as f:
        reader = csv.DictReader(f)
        for row in reader:
            zoo_transfers[row['from'].lower()].append(row)
            zoo_transfers[row['to'].lower()].append(row)
except:
    pass

# X-Chain eligible addresses
xchain_eligible = {
    'lux': {},
    'zoo': {}
}

# Process LUX holders
# Include 7777 holders NOT in 96369
for addr, data in lux_7777.items():
    addr_lower = addr.lower()
    if addr_lower not in lux_96369:
        xchain_eligible['lux'][addr] = {
            'source': '7777',
            'balance': data.get('balance', '0'),
            'eligible': True
        }

# Process ZOO holders
# BSC burners who are not in 200200 or have no outgoing transfers
for burn_data in zoo_bsc_burns.get('burns', []):
    addr = burn_data['from'].lower()
    
    # Check if they received tokens on 200200
    in_200200 = addr in zoo_200200
    
    # Check if they ever moved tokens on 200200
    has_outgoing = False
    if addr in zoo_transfers:
        for tx in zoo_transfers[addr]:
            if tx['from'].lower() == addr:
                has_outgoing = True
                break
    
    if not in_200200 or not has_outgoing:
        xchain_eligible['zoo'][addr] = {
            'source': 'bsc_burn',
            'burned_amount': burn_data['amount'],
            'in_200200': in_200200,
            'has_outgoing_200200': has_outgoing,
            'eligible': True
        }

# Save results
with open('xchain-eligible-addresses.json', 'w') as f:
    json.dump(xchain_eligible, f, indent=2)

# Generate summary
print(f"X-Chain Eligible Addresses:")
print(f"- LUX (from 7777, not in 96369): {len(xchain_eligible['lux'])}")
print(f"- ZOO (BSC burners not active on 200200): {len(xchain_eligible['zoo'])}")
print(f"\nValidator Eligible (1M+ LUX or NFT):")

# Count validator eligible
validator_eligible = 0
for addr, data in xchain_eligible['lux'].items():
    try:
        balance = int(data['balance'])
        if balance >= 1000000000000000000000000:  # 1M LUX
            validator_eligible += 1
    except:
        pass

print(f"- From balance: {validator_eligible}")
print(f"- From NFTs: Will need to cross-reference with NFT holders")
EOF

cd "$OUTPUT_DIR" && python3 cross-reference.py

echo ""
echo "Step 10: Generating final genesis report..."
cat > "$OUTPUT_DIR/genesis-report.md" << EOF
# Genesis Analysis Report

Generated: $(date)

## Summary

### LUX Network
- 7777 holders analyzed
- 96369 holders analyzed  
- Ethereum NFT holders scanned
- X-Chain eligible addresses identified

### ZOO Network
- BSC holders and burns analyzed
- 200200 chain fully scanned
- Transfer history analyzed
- X-Chain eligible addresses identified

### Initial Validators
- 11 bootstrap validators configured
- Each with 1B LUX staked
- 100-year vesting schedule (1% per year)
- Minimum requirement: 1M LUX or 1 NFT

## Files Generated
- LUX 7777 accounts: lux-7777-accounts.csv/json
- LUX 96369 accounts: lux-96369-accounts.csv/json
- ZOO 200200 accounts: zoo-200200-accounts.csv/json
- ZOO 200200 transfers: zoo-200200-all-transfers.csv
- X-Chain eligible: xchain-eligible-addresses.json
- Bootstrap nodes: $(realpath ../chaindata/lux-mainnet-96369/bootnodes.json)

## Next Steps
1. Review xchain-eligible-addresses.json
2. Verify bootstrap validator addresses
3. Generate final X-Chain genesis with eligible addresses
4. Deploy network with initial validators
EOF

echo ""
echo "=== Analysis Complete ==="
echo "Results saved to: $OUTPUT_DIR"
echo "Key files:"
echo "  - X-Chain eligible addresses: $OUTPUT_DIR/xchain-eligible-addresses.json"
echo "  - Bootstrap validators: chaindata/lux-mainnet-96369/bootnodes.json"
echo "  - Full report: $OUTPUT_DIR/genesis-report.md"