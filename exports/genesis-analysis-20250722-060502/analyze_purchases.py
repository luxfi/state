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
