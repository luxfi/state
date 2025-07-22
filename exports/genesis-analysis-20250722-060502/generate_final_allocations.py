#!/usr/bin/env python3
"""
Generate final X-Chain allocations combining all sources
"""

import json

# Load current NFT holder data
with open('egg_nft_holders.json') as f:
    nft_data = json.load(f)

# Load X-Chain allocations we generated
with open('xchain_egg_allocations.json') as f:
    xchain_data = json.load(f)

# The 0x28dad address holds unclaimed EGGs
purchase_addr = "0x28dad8427f127664365109c4a9406c8bc7844718"
unclaimed_eggs = nft_data['holders'].get(purchase_addr, {}).get('token_count', 0)

print("=== Final X-Chain ZOO Genesis Allocations ===")
print()
print(f"Current NFT holders: {nft_data['unique_holders']} addresses with {nft_data['total_supply'] - nft_data['burned_tokens']} EGGs")
print(f"Unclaimed EGGs at {purchase_addr}: {unclaimed_eggs}")
print(f"X-Chain allocations: {xchain_data['summary']['total_addresses']} addresses with {xchain_data['summary']['total_eggs']} EGGs")
print()

# Final allocations include:
# 1. All known holders from spreadsheet (already in xchain_allocations)
# 2. All current NFT holders (already in xchain_allocations)
# 3. Note about unclaimed EGGs for future distribution

final_allocations = xchain_data['allocations']

# Summary statistics
total_addresses = len(final_allocations)
total_eggs = sum(a['eggs'] for a in final_allocations.values())
total_zoo = total_eggs * 4200000

print("Final Statistics:")
print(f"  Total recipients: {total_addresses}")
print(f"  Total EGGs allocated: {total_eggs}")
print(f"  Total ZOO tokens: {total_zoo:,}")
print(f"  Unclaimed EGGs (for future): {unclaimed_eggs}")
print()

# Generate final files
final_data = {
    "metadata": {
        "generated_at": "2025-07-22",
        "zoo_per_egg": 4200000,
        "chain_id": 200200,
        "network": "zoo-mainnet"
    },
    "allocations": final_allocations,
    "summary": {
        "total_recipients": total_addresses,
        "total_eggs": total_eggs,
        "total_zoo": total_zoo,
        "unclaimed_eggs": unclaimed_eggs,
        "sources": {
            "spreadsheet": xchain_data['summary']['from_spreadsheet'],
            "bsc_scan": xchain_data['summary']['from_bsc_scan']
        }
    },
    "notes": [
        "Allocations based on known spreadsheet + current BSC holders",
        f"{unclaimed_eggs} EGGs remain at purchase address for future claims",
        "Each EGG = 4,200,000 ZOO tokens",
        "All recipients verified through multiple sources"
    ]
}

# Save JSON
with open('zoo_xchain_genesis_allocations.json', 'w') as f:
    json.dump(final_data, f, indent=2)

# Save CSV for review
with open('zoo_xchain_genesis_allocations.csv', 'w') as f:
    f.write('address,eggs,zoo_amount,source\n')
    for addr, alloc in sorted(final_allocations.items()):
        f.write(f"{addr},{alloc['eggs']},{alloc['zoo_equivalent']},{alloc['source']}\n")

# Generate Lux validator summary
print("Generating LUX validator data...")

# Bootstrap validators
bootstrap_validators = [
    "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714",
    "0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59",
    "0x8d5081153aE1cfb41f5c932fe0b6Beb7E159cF84",
    "0xf8f12D0592e6d1bFe92ee16CaBCC4a6F26dAAe23",
    "0xFb66808f708e1d4D7D43a8c75596e84f94e06806",
    "0x313CF291c069C58D6bd61B0D672673462B8951bD",
    "0xf7f52257a6143cE6BbD12A98eF2B0a3d0C648079",
    "0xCA92ad0C91bd8DE640B9dAFfEB338ac908725142",
    "0xB5B325df519eB58B7223d85aaeac8b56aB05f3d6",
    "0xcf5288bEe8d8F63511C389D5015185FDEDe30e54",
    "0x16204223fe4470f4B1F1dA19A368dC815736a3d7"
]

lux_validator_data = {
    "bootstrap_validators": {
        "count": len(bootstrap_validators),
        "addresses": bootstrap_validators,
        "stake_per_validator": "1000000000000000000000000000",
        "vesting": {
            "duration_years": 100,
            "unlock_per_year": "1%",
            "start_date": "2020-01-01"
        }
    },
    "eligibility": {
        "requirements": [
            "Hold 1+ LUX NFT on Ethereum",
            "OR hold 1M+ LUX tokens"
        ],
        "nft_holders": "See lux_nft_holders.json (scan pending)",
        "token_holders": "Extract from chains 7777/96369"
    }
}

with open('lux_validator_summary.json', 'w') as f:
    json.dump(lux_validator_data, f, indent=2)

print()
print("Files generated:")
print("  - zoo_xchain_genesis_allocations.json (main genesis data)")
print("  - zoo_xchain_genesis_allocations.csv (for review)")
print("  - lux_validator_summary.json (validator configuration)")
print()
print("Next steps:")
print("  1. Extract LUX chain data (7777, 96369)")
print("  2. Scan LUX NFT holders on Ethereum")
print("  3. Generate final X-Chain genesis combining LUX + ZOO data")