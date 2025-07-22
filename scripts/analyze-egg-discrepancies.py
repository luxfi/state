#!/usr/bin/env python3
"""
Analyze EGG NFT discrepancies between payments and actual holdings
"""

import json
import csv

# Known EGG holder data from spreadsheet
KNOWN_HOLDERS = {
    "0x51c29390c8e24baaa92fea06ce95d90d0877ca9e": {"eggs": 5, "tokenIds": ["140", "141", "142", "143", "144"]},
    "0x9dd1ed97a2aa965de37b087fa95e6ddb7c5e4d5f": {"eggs": 1, "tokenIds": ["122"]},
    "0xcc9c4d6a8502c38f2dcda03b0e9fb5e5e6d6b88e": {"eggs": 1, "tokenIds": ["107"]},
    "0xca92ad0c91bd8de640b9daffeb338ac908725142": {"eggs": 12, "tokenIds": ["34", "35", "36", "37", "38", "39", "40", "41", "42", "43", "44", "52"]},
    "0x67be46cc9fc7c4d98da9e95a0f1e31c0ad17c2b7": {"eggs": 1, "tokenIds": ["115"]},
    "0xdb4c9e33b93df5da19821c4ad0c1b1ea4e007a75": {"eggs": 3, "tokenIds": ["109", "110", "111"]},
    "0x0bde1f18b1e866cf4c4f3a017a90d1fc5a1db76d": {"eggs": 1, "tokenIds": ["95"]},
    "0x8a96797f29c16d6e49b83a6c4f68bb8de92bb6d7": {"eggs": 2, "tokenIds": ["105", "106"]},
    "0x47fa1419ceeb9bd3e40e0c7e30dd75c67da7e9ab": {"eggs": 6, "tokenIds": ["125", "126", "127", "128", "129", "130"]},
    "0x08f4c12c05e5cf92c3b10c81e4ad7fb5ba97ea73": {"eggs": 4, "tokenIds": ["132", "133", "134", "135"]},
    "0xa69bb47e3a8eb01dd982b43a0d91b9096f907fad": {"eggs": 1, "tokenIds": ["76"]},
    "0x95d988f89b978d44e87bb2bea7b825322a0e6c58": {"eggs": 1, "tokenIds": ["131"]},
    "0x9011e888251ab053b7bd1cdb598db4f9ded94714": {"eggs": 1, "tokenIds": ["33"]},
    "0xf7fb4fc5d5c2a59f8eb88ed6c0f09f8cf9c7c33f": {"eggs": 2, "tokenIds": ["123", "124"]},
    "0x0771bb1a8e42cf1328a3f0b9fb1b35d9e903b23d": {"eggs": 4, "tokenIds": ["117", "118", "119", "120"]},
    "0xd93fa39b8b1c2c7c01e892c29e8ab10f4cf77c1f": {"eggs": 2, "tokenIds": ["108", "121"]},
    "0x7de6b6b6c2da8fa8c96ba49b2e17b87dcfe9e026": {"eggs": 8, "tokenIds": ["45", "46", "47", "48", "49", "50", "51", "137"]},
    "0xd7e2ea6c4d40a90b1c8f4bb43bf63c96770a5078": {"eggs": 1, "tokenIds": ["53"]},
    "0x2e387a97c6795dc73d4e8a3e456c4b9848fb5c73": {"eggs": 5, "tokenIds": ["112", "113", "114", "116", "136"]},
    "0xa64b3eb2c92c9987d7c6e97f088b05e31e93cf77": {"eggs": 8, "tokenIds": ["54", "55", "56", "57", "58", "59", "60", "61"]},
    "0x5b5bc5c57ae3bb0e7ab99e57b0a2e88bbf956c15": {"eggs": 3, "tokenIds": ["96", "97", "98"]},
    "0x57d9e829e1cd43c12f959e088b12b670b3ae5eac": {"eggs": 10, "tokenIds": ["86", "87", "88", "89", "90", "91", "92", "93", "94", "139"]},
    "0xfbdacc3db0c6a16c51f4e09c4e97a18bb16c3cc4": {"eggs": 1, "tokenIds": ["138"]},
    "0x07e40e03bb00e13c03c0e017ffd7cbb029e32730": {"eggs": 9, "tokenIds": ["77", "78", "79", "80", "81", "82", "83", "84", "85"]},
    "0xb59ba2c05c93a9cc1fb18c993e5bb079dc17ab77": {"eggs": 1, "tokenIds": ["145"]},
    "0x18c21cbaab25d24a96e7c18b7c6e3e968b77df9f": {"eggs": 10, "tokenIds": ["99", "100", "101", "102", "103", "104", "62", "63", "64", "75"]},
    "0x1708e387c0b1b0aae7c1beac79b12c4ca837c2c9": {"eggs": 12, "tokenIds": ["65", "66", "67", "68", "69", "70", "71", "72", "73", "74", "1", "2"]}
}

# Load scanned data
with open('exports/genesis-analysis-20250722-060502/egg_nft_holders.json') as f:
    scanned_data = json.load(f)

print("=== EGG NFT Discrepancy Analysis ===")
print()

# Compare known vs scanned
known_total = sum(h['eggs'] for h in KNOWN_HOLDERS.values())
known_addresses = set(KNOWN_HOLDERS.keys())

scanned_holders = scanned_data['holders']
scanned_addresses = set(addr.lower() for addr in scanned_holders.keys())

print(f"Known holders: {len(KNOWN_HOLDERS)} with {known_total} EGGs")
print(f"Scanned holders: {len(scanned_holders)} with {scanned_data['total_supply'] - scanned_data['burned_tokens']} EGGs")
print()

# Check 0x28dad
purchase_addr = "0x28dad8427f127664365109c4a9406c8bc7844718"
if purchase_addr in scanned_holders:
    print(f"Purchase address {purchase_addr}:")
    print(f"  Holds {scanned_holders[purchase_addr]['token_count']} EGGs")
    print(f"  This likely represents unclaimed/undelivered EGGs")
    print()

# Find missing from known list
print("Known holders missing from scan:")
missing_count = 0
for addr in known_addresses:
    if addr not in scanned_addresses:
        holder = KNOWN_HOLDERS[addr]
        print(f"  {addr}: {holder['eggs']} EGGs (IDs: {', '.join(holder['tokenIds'])})")
        missing_count += holder['eggs']

if missing_count == 0:
    print("  None - all known holders found!")
else:
    print(f"\nTotal missing: {missing_count} EGGs")

print()

# Find extra in scan (not in known list)
print("Addresses in scan but not in known list:")
extra_eggs = 0
for addr in scanned_addresses:
    if addr not in known_addresses and addr != purchase_addr:
        holder = scanned_holders[addr]
        if holder['token_count'] > 0:
            print(f"  {addr}: {holder['token_count']} EGGs")
            extra_eggs += holder['token_count']

print(f"\nTotal extra (excluding purchase address): {extra_eggs} EGGs")

# Generate final X-Chain allocation
print("\n=== X-Chain EGG Allocation Plan ===")
print()

xchain_allocations = {}

# 1. Add all known holders (from spreadsheet)
print("1. Known holders from spreadsheet:")
for addr, data in KNOWN_HOLDERS.items():
    xchain_allocations[addr] = {
        "eggs": data['eggs'],
        "zoo_equivalent": data['eggs'] * 4200000,
        "source": "spreadsheet",
        "token_ids": data['tokenIds']
    }
    print(f"   {addr}: {data['eggs']} EGGs")

# 2. Add any additional holders from scan (not in spreadsheet)
print("\n2. Additional holders from scan:")
added = 0
for addr, holder in scanned_holders.items():
    if addr.lower() not in known_addresses and addr != purchase_addr:
        xchain_allocations[addr] = {
            "eggs": holder['token_count'],
            "zoo_equivalent": holder['token_count'] * 4200000,
            "source": "bsc_scan",
            "token_ids": [str(id) for id in holder['token_ids']]
        }
        print(f"   {addr}: {holder['token_count']} EGGs")
        added += holder['token_count']

print(f"\nTotal X-Chain EGG allocations: {len(xchain_allocations)} addresses")
print(f"Total EGGs: {sum(a['eggs'] for a in xchain_allocations.values())}")
print(f"Total ZOO equivalent: {sum(a['zoo_equivalent'] for a in xchain_allocations.values()):,}")

# Save allocations
with open('exports/genesis-analysis-20250722-060502/xchain_egg_allocations.json', 'w') as f:
    json.dump({
        "allocations": xchain_allocations,
        "summary": {
            "total_addresses": len(xchain_allocations),
            "total_eggs": sum(a['eggs'] for a in xchain_allocations.values()),
            "total_zoo_equivalent": sum(a['zoo_equivalent'] for a in xchain_allocations.values()),
            "from_spreadsheet": len(KNOWN_HOLDERS),
            "from_bsc_scan": added
        },
        "notes": [
            f"Purchase address {purchase_addr} holds {scanned_holders.get(purchase_addr, {}).get('token_count', 0)} unclaimed EGGs",
            "X-Chain allocations based on known spreadsheet data + BSC scan",
            "Each EGG = 4,200,000 ZOO tokens"
        ]
    }, f, indent=2)

print("\nSaved X-Chain allocations to: xchain_egg_allocations.json")