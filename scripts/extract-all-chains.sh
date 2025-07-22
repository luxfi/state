#!/bin/bash
# Extract all chain data for genesis generation

set -e

echo "=== Extracting All Chain Data ==="
echo "Starting at: $(date)"
echo ""

# Create extraction directory
mkdir -p data/extracted

# Build tools
echo "Building tools..."
make build-archeology
make build-genesis

# Extract LUX 7777
echo ""
echo "1. Extracting LUX Genesis 7777..."
if [ -d "chaindata/lux-genesis-7777/db/pebbledb" ]; then
    ./bin/archeology extract \
        --src chaindata/lux-genesis-7777/db/pebbledb \
        --dst data/extracted/lux-genesis-7777 \
        --chain-id 7777 \
        --include-state || echo "Warning: 7777 extraction had issues"
else
    echo "Warning: No 7777 chaindata found"
fi

# Extract LUX 96369
echo ""
echo "2. Extracting LUX Mainnet 96369..."
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    ./bin/archeology extract \
        --src chaindata/lux-mainnet-96369/db/pebbledb \
        --dst data/extracted/lux-mainnet-96369 \
        --chain-id 96369 \
        --include-state || echo "Warning: 96369 extraction had issues"
else
    echo "Warning: No 96369 chaindata found"
fi

# Extract ZOO 200200
echo ""
echo "3. Extracting ZOO Mainnet 200200..."
if [ -d "chaindata/zoo-mainnet-200200/db/pebbledb" ]; then
    ./bin/archeology extract \
        --src chaindata/zoo-mainnet-200200/db/pebbledb \
        --dst data/extracted/zoo-mainnet-200200 \
        --chain-id 200200 \
        --include-state || echo "Warning: 200200 extraction had issues"
else
    echo "Warning: No 200200 chaindata found"
fi

# Analyze extracted data
echo ""
echo "4. Analyzing extracted data..."

for chain in "lux-genesis-7777" "lux-mainnet-96369" "zoo-mainnet-200200"; do
    if [ -d "data/extracted/$chain" ]; then
        echo "   Analyzing $chain..."
        ./bin/archeology analyze \
            -db "data/extracted/$chain" \
            -network "$chain" \
            --output "data/extracted/$chain/accounts.csv" \
            --output-json "data/extracted/$chain/accounts.json" \
            --exclude-zero-balance || echo "   Warning: Analysis had issues"
    fi
done

echo ""
echo "=== Extraction Complete ==="
echo "Extracted data in: data/extracted/"
ls -la data/extracted/
echo ""
echo "Completed at: $(date)"