#!/usr/bin/env python3
import json

# Load ZOO allocations
zoo_allocations = {}
try:
    with open('exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.json') as f:
        zoo_data = json.load(f)
        for addr, alloc in zoo_data['allocations'].items():
            zoo_allocations[addr] = alloc['zoo_equivalent'] * 10**9  # Convert to nLUX
except:
    print("Warning: No ZOO allocations found")

# Load LUX 7777 accounts (if extracted)
lux_allocations = {}
try:
    # This would come from extracted data
    pass
except:
    print("Warning: No LUX 7777 data found")

# Combine allocations
all_allocations = []

# Add ZOO allocations
for addr, amount in zoo_allocations.items():
    all_allocations.append({
        "ethAddr": addr,
        "avaxAddr": f"X-lux1{addr[2:]}",  # Simplified conversion
        "initialAmount": amount,
        "unlockSchedule": []
    })

# Add bootstrap validators with initial funds
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

for addr in bootstrap_validators:
    if addr.lower() not in [a['ethAddr'].lower() for a in all_allocations]:
        all_allocations.append({
            "ethAddr": addr,
            "avaxAddr": f"X-lux1{addr[2:]}",
            "initialAmount": 100000000000000000,  # 100M nLUX
            "unlockSchedule": []
        })

# Load template
with open('genesis/X-Chain/genesis_template.json') as f:
    genesis = json.load(f)

genesis['allocations'] = all_allocations

# Save final genesis
with open('genesis/X-Chain/genesis.json', 'w') as f:
    json.dump(genesis, f, indent=2)

print(f"Generated X-Chain genesis with {len(all_allocations)} allocations")
