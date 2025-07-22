#!/bin/bash
# Extract chain data and generate P, X genesis files

set -e

echo "=== Genesis Generation Pipeline ==="
echo "Starting at: $(date)"
echo ""

# Ensure tools are built
make build-tools
make build-genesis

# Create directories
mkdir -p data/extracted
mkdir -p genesis/P-Chain
mkdir -p genesis/X-Chain
mkdir -p genesis/C-Chain

# Extract LUX 96369 for accounts
echo "1. Extracting LUX 96369 accounts..."
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    ./bin/denamespace \
        -src chaindata/lux-mainnet-96369/db/pebbledb \
        -dst data/extracted/lux-96369 \
        -network 96369 \
        -state || echo "Warning: Extraction had issues"
    echo "   Extraction complete"
else
    echo "   Error: No 96369 chaindata found"
fi

# Use existing C-Chain genesis (we're continuing with it)
echo ""
echo "2. Copying existing C-Chain genesis..."
if [ -f "chaindata/lux-mainnet-96369/config/genesis.json" ]; then
    cp chaindata/lux-mainnet-96369/config/genesis.json genesis/C-Chain/
    echo "   C-Chain genesis copied (continuing existing chain)"
else
    echo "   Warning: No existing C-Chain genesis found"
fi

# Generate P-Chain genesis with bootstrap validators
echo ""
echo "3. Generating P-Chain genesis..."
cat > genesis/P-Chain/genesis.json << 'EOF'
{
  "networkID": 1,
  "allocations": [],
  "startTime": 1577836800,
  "initialStakeDuration": 31536000,
  "initialStakeDurationOffset": 5400,
  "initialStakers": [
    {
      "nodeID": "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
      "rewardAddress": "P-lux1d4wfwrfgu4dkkyq7dlhx0lt69y2hjkjeejnhca",
      "delegationFee": 1000000
    },
    {
      "nodeID": "NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ",
      "rewardAddress": "P-lux1g65uqn6t77p656w64023nh8nd9updzmxwd59gh",
      "delegationFee": 500000
    },
    {
      "nodeID": "NodeID-NFBbbJ4qCmNaCzeW7sxErhvWqvEQMnYcN",
      "rewardAddress": "P-lux1zg69v7yszg5xws6a28jg5dj9mm8qnm4s7eu8wr",
      "delegationFee": 250000
    },
    {
      "nodeID": "NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu",
      "rewardAddress": "P-lux1d59dmasw7wgw78jvqpwz3n378t3uu0ygp8xptz",
      "delegationFee": 125000
    },
    {
      "nodeID": "NodeID-P7oB2McjBGgW2NXXWVYjV8JEDFoW9xDE5",
      "rewardAddress": "P-lux1hqqr7jkaysrfrfywn4wqdqmqpwmj2u5fzeqhcc",
      "delegationFee": 62500
    }
  ],
  "chains": [
    {
      "genesisData": {
        "buildTimestamp": 1599696000,
        "codecVersion": 0
      },
      "vmID": "jvYyfQTxGMJLuGWa55kdP2p2zSUYsQ5Raupu4TW34ZAUBAbtq",
      "fxIDs": ["spdxUxVJQbX85MGxMHbKw1sHxMnSqJ3QBzDyDYEP3h6TLuxqQ"],
      "name": "X-Chain",
      "subnetID": "11111111111111111111111111111111LpoYY"
    }
  ],
  "initialStakedFunds": [
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
  ],
  "message": "Lux Network 2025 Genesis"
}
EOF
echo "   P-Chain genesis generated"

# Generate X-Chain genesis with allocations
echo ""
echo "4. Generating X-Chain genesis..."

# Create X-Chain genesis with all allocations
cat > genesis/X-Chain/genesis_template.json << 'EOF'
{
  "config": {
    "version": 1,
    "networkID": 1,
    "codecVersion": 0,
    "feeConfig": {
      "txFee": 1000000,
      "createAssetTxFee": 10000000,
      "createSubnetTxFee": 1000000000,
      "createBlockchainTxFee": 1000000000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x5ff07b00",
  "initialSupply": "720000000000000000",
  "allocations": []
}
EOF

# Generate allocations from our data
echo "   Generating X-Chain allocations..."
cat > genesis/X-Chain/generate_allocations.py << 'EOF'
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
EOF

python3 genesis/X-Chain/generate_allocations.py

# Summary
echo ""
echo "=== Genesis Files Generated ==="
echo ""
echo "P-Chain: genesis/P-Chain/genesis.json"
echo "C-Chain: genesis/C-Chain/genesis.json (existing)"
echo "X-Chain: genesis/X-Chain/genesis.json"
echo ""
ls -la genesis/*/genesis.json

echo ""
echo "Bootstrap validators configured:"
cat chaindata/lux-mainnet-96369/bootnodes.json | jq -r '.bootstrapNodes[]'

echo ""
echo "Completed at: $(date)"