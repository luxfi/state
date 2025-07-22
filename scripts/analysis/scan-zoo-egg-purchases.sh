#!/bin/bash
# Scan ZOO transfers to EGG purchase address

set -e

echo "=== Scanning ZOO Payments for EGG Purchases ==="
echo ""

OUTPUT_DIR="exports/genesis-analysis-20250722-060502"
CACHE_DIR="cache/zoo-egg-purchases"
mkdir -p "$CACHE_DIR"

# Configuration
ZOO_TOKEN="0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
EGG_PURCHASE_ADDR="0x28dad8427f127664365109c4a9406c8bc7844718"

# Build if needed
make build-archeology

echo "Scanning ZOO transfers to EGG purchase address..."
echo "Each 4.2M ZOO = 1 EGG NFT"
echo ""

# Use scan-transfers to find all payments to 0x28dad
./bin/archeology scan-transfers \
    --rpc https://bsc-dataseed.bnbchain.org \
    --rpc https://bsc-dataseed1.binance.org \
    --rpc https://bsc-dataseed2.binance.org \
    --token "$ZOO_TOKEN" \
    --target "$EGG_PURCHASE_ADDR" \
    --direction to \
    --from-block 16000000 \
    --to-block 25000000 \
    --batch-size 5000 \
    --output "$OUTPUT_DIR/zoo_egg_purchases.csv" \
    --output-json "$OUTPUT_DIR/zoo_egg_purchases.json" \
    --show-balances || echo "Note: Partial scan due to RPC limits"

echo ""
echo "Analyzing purchase data..."

# Create analysis script
cat > "$OUTPUT_DIR/analyze_purchases.py" << 'EOF'
#!/usr/bin/env python3
import json
import csv
from decimal import Decimal

# Load purchase data
try:
    with open('zoo_egg_purchases.json') as f:
        purchase_data = json.load(f)
except:
    print("No purchase data found yet. Run with better RPC.")
    exit(1)

# Load NFT holder data
with open('egg_nft_holders.json') as f:
    nft_data = json.load(f)

print("=== ZOO EGG Purchase Analysis ===")
print()

ZOO_PER_EGG = Decimal('4200000') * Decimal('10') ** 18  # 4.2M with 18 decimals

# Process transfers
purchasers = {}
if 'transfers' in purchase_data:
    for transfer in purchase_data['transfers']:
        buyer = transfer['from'].lower()
        amount = Decimal(transfer['amount'])
        
        if buyer not in purchasers:
            purchasers[buyer] = Decimal('0')
        purchasers[buyer] += amount

# Calculate eggs purchased
eggs_purchased = {}
for buyer, zoo_amount in purchasers.items():
    eggs = int(zoo_amount / ZOO_PER_EGG)
    if eggs > 0:
        eggs_purchased[buyer] = {
            'zoo_paid': str(zoo_amount),
            'eggs_purchased': eggs,
            'zoo_per_egg': float(zoo_amount / eggs / (10**18)) if eggs > 0 else 0
        }

print(f"Total purchasers: {len(eggs_purchased)}")
print(f"Total EGGs purchased: {sum(e['eggs_purchased'] for e in eggs_purchased.values())}")
print()

# Compare with actual holders
nft_holders = {addr.lower(): data for addr, data in nft_data['holders'].items()}

print("Purchase vs Holdings Comparison:")
print()

discrepancies = []
for buyer, purchase in eggs_purchased.items():
    if buyer in nft_holders:
        held = nft_holders[buyer]['token_count']
        purchased = purchase['eggs_purchased']
        if held != purchased:
            discrepancies.append({
                'address': buyer,
                'purchased': purchased,
                'held': held,
                'difference': purchased - held
            })
            print(f"{buyer}: Purchased {purchased}, Holds {held} (diff: {purchased - held})")
    else:
        discrepancies.append({
            'address': buyer,
            'purchased': purchase['eggs_purchased'],
            'held': 0,
            'difference': purchase['eggs_purchased']
        })
        print(f"{buyer}: Purchased {purchase['eggs_purchased']}, Holds 0 (NOT DELIVERED)")

# Save results
results = {
    'purchases': eggs_purchased,
    'discrepancies': discrepancies,
    'summary': {
        'total_purchasers': len(eggs_purchased),
        'total_eggs_purchased': sum(e['eggs_purchased'] for e in eggs_purchased.values()),
        'total_delivered': sum(h['token_count'] for h in nft_holders.values()),
        'total_undelivered': sum(d['difference'] for d in discrepancies if d['difference'] > 0)
    }
}

with open('egg_purchase_analysis.json', 'w') as f:
    json.dump(results, f, indent=2)

print(f"\nTotal undelivered EGGs: {results['summary']['total_undelivered']}")
print("Results saved to: egg_purchase_analysis.json")
EOF

cd "$OUTPUT_DIR" && python3 analyze_purchases.py || echo "Analysis pending..."

echo ""
echo "=== Generating Final X-Chain Allocations ==="

cat > "$OUTPUT_DIR/generate_xchain_allocations.py" << 'EOF'
#!/usr/bin/env python3
import json

# Load all data
try:
    with open('egg_purchase_analysis.json') as f:
        purchase_analysis = json.load(f)
    purchases = purchase_analysis.get('purchases', {})
except:
    purchases = {}

with open('egg_nft_holders.json') as f:
    nft_data = json.load(f)

# Known holders from spreadsheet
KNOWN_HOLDERS = {
    "0x51c29390c8e24baaa92fea06ce95d90d0877ca9e": 5,
    "0x9dd1ed97a2aa965de37b087fa95e6ddb7c5e4d5f": 1,
    "0xcc9c4d6a8502c38f2dcda03b0e9fb5e5e6d6b88e": 1,
    "0xca92ad0c91bd8de640b9daffeb338ac908725142": 12,
    "0x67be46cc9fc7c4d98da9e95a0f1e31c0ad17c2b7": 1,
    "0xdb4c9e33b93df5da19821c4ad0c1b1ea4e007a75": 3,
    "0x0bde1f18b1e866cf4c4f3a017a90d1fc5a1db76d": 1,
    "0x8a96797f29c16d6e49b83a6c4f68bb8de92bb6d7": 2,
    "0x47fa1419ceeb9bd3e40e0c7e30dd75c67da7e9ab": 6,
    "0x08f4c12c05e5cf92c3b10c81e4ad7fb5ba97ea73": 4,
    "0xa69bb47e3a8eb01dd982b43a0d91b9096f907fad": 1,
    "0x95d988f89b978d44e87bb2bea7b825322a0e6c58": 1,
    "0x9011e888251ab053b7bd1cdb598db4f9ded94714": 1,
    "0xf7fb4fc5d5c2a59f8eb88ed6c0f09f8cf9c7c33f": 2,
    "0x0771bb1a8e42cf1328a3f0b9fb1b35d9e903b23d": 4,
    "0xd93fa39b8b1c2c7c01e892c29e8ab10f4cf77c1f": 2,
    "0x7de6b6b6c2da8fa8c96ba49b2e17b87dcfe9e026": 8,
    "0xd7e2ea6c4d40a90b1c8f4bb43bf63c96770a5078": 1,
    "0x2e387a97c6795dc73d4e8a3e456c4b9848fb5c73": 5,
    "0xa64b3eb2c92c9987d7c6e97f088b05e31e93cf77": 8,
    "0x5b5bc5c57ae3bb0e7ab99e57b0a2e88bbf956c15": 3,
    "0x57d9e829e1cd43c12f959e088b12b670b3ae5eac": 10,
    "0xfbdacc3db0c6a16c51f4e09c4e97a18bb16c3cc4": 1,
    "0x07e40e03bb00e13c03c0e017ffd7cbb029e32730": 9,
    "0xb59ba2c05c93a9cc1fb18c993e5bb079dc17ab77": 1,
    "0x18c21cbaab25d24a96e7c18b7c6e3e968b77df9f": 10,
    "0x1708e387c0b1b0aae7c1beac79b12c4ca837c2c9": 12
}

print("=== Final X-Chain EGG/ZOO Allocations ===")
print()

# Combine all sources
xchain_allocations = {}

# 1. Add known holders
for addr, eggs in KNOWN_HOLDERS.items():
    xchain_allocations[addr.lower()] = {
        'eggs': eggs,
        'zoo_equivalent': eggs * 4200000,
        'source': 'spreadsheet'
    }

# 2. Add/update from purchases (if we have the data)
for addr, purchase in purchases.items():
    eggs = purchase['eggs_purchased']
    if addr in xchain_allocations:
        # Take the max of spreadsheet vs purchased
        if eggs > xchain_allocations[addr]['eggs']:
            xchain_allocations[addr]['eggs'] = eggs
            xchain_allocations[addr]['zoo_equivalent'] = eggs * 4200000
            xchain_allocations[addr]['source'] = 'purchase_data'
    else:
        xchain_allocations[addr] = {
            'eggs': eggs,
            'zoo_equivalent': eggs * 4200000,
            'source': 'purchase_only'
        }

# 3. Add current NFT holders (safety net)
for addr, holder in nft_data['holders'].items():
    addr_lower = addr.lower()
    if addr_lower != "0x28dad8427f127664365109c4a9406c8bc7844718":  # Skip purchase address
        eggs = holder['token_count']
        if addr_lower not in xchain_allocations:
            xchain_allocations[addr_lower] = {
                'eggs': eggs,
                'zoo_equivalent': eggs * 4200000,
                'source': 'current_holder'
            }
        elif eggs > xchain_allocations[addr_lower]['eggs']:
            # Update if they hold more than allocated
            xchain_allocations[addr_lower]['eggs'] = eggs
            xchain_allocations[addr_lower]['zoo_equivalent'] = eggs * 4200000
            xchain_allocations[addr_lower]['source'] = 'current_holder_updated'

# Summary
total_eggs = sum(a['eggs'] for a in xchain_allocations.values())
total_zoo = sum(a['zoo_equivalent'] for a in xchain_allocations.values())

print(f"Total X-Chain Recipients: {len(xchain_allocations)}")
print(f"Total EGGs: {total_eggs}")
print(f"Total ZOO: {total_zoo:,}")
print()

# Source breakdown
sources = {}
for alloc in xchain_allocations.values():
    source = alloc['source']
    sources[source] = sources.get(source, 0) + 1

print("Allocation sources:")
for source, count in sources.items():
    print(f"  {source}: {count} addresses")

# Save final allocations
final_data = {
    'allocations': xchain_allocations,
    'summary': {
        'total_recipients': len(xchain_allocations),
        'total_eggs': total_eggs,
        'total_zoo': total_zoo,
        'sources': sources
    },
    'metadata': {
        'zoo_per_egg': 4200000,
        'generated_at': '2025-07-22',
        'notes': [
            'Combined from: spreadsheet + purchases + current holders',
            'Each recipient gets max(spreadsheet, purchased, held)',
            'Purchase address 0x28dad excluded from allocations'
        ]
    }
}

with open('xchain_zoo_final_allocations.json', 'w') as f:
    json.dump(final_data, f, indent=2)

# Generate CSV for easy review
with open('xchain_zoo_final_allocations.csv', 'w') as f:
    f.write('address,eggs,zoo_amount,source\n')
    for addr, alloc in sorted(xchain_allocations.items()):
        f.write(f"{addr},{alloc['eggs']},{alloc['zoo_equivalent']},{alloc['source']}\n")

print("\nFiles generated:")
print("  - xchain_zoo_final_allocations.json")
print("  - xchain_zoo_final_allocations.csv")
EOF

cd "$OUTPUT_DIR" && python3 generate_xchain_allocations.py

echo ""
echo "=== Analysis Complete ==="
echo "Check exports/genesis-analysis-20250722-060502/ for:"
echo "  - zoo_egg_purchases.json (if scan completed)"
echo "  - egg_purchase_analysis.json (if purchases found)"
echo "  - xchain_zoo_final_allocations.json (final allocations)"
echo "  - xchain_zoo_final_allocations.csv (for review)"