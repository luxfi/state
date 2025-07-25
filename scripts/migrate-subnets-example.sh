#!/bin/bash

# Example script showing how to migrate subnet data to both C-Chain and L2 formats

set -e

echo "=== Subnet Migration Example ==="
echo ""

# Configuration
ARCHIVE_BASE="/home/z/archived/spelunking/chaindata"
NAMESPACE_DIR="/home/z/work/lux/genesis"
GENESIS_BIN="$NAMESPACE_DIR/bin/genesis"

# Ensure we have the latest binary
cd $NAMESPACE_DIR
echo "Building genesis tool..."
go build -o bin/genesis ./cmd/genesis

echo ""
echo "=== 1. Extract Subnet Data (Remove Namespace Prefixes) ==="

# Example: Extract ZOO mainnet (200200)
if [ ! -d "./extracted-zoo-200200" ]; then
    echo "Extracting ZOO mainnet subnet data..."
    $GENESIS_BIN extract state \
        "$ARCHIVE_BASE/2024-200200" \
        "./extracted-zoo-200200" \
        --network 200200 \
        --state
else
    echo "ZOO data already extracted"
fi

# Example: Extract SPC mainnet (36911)  
if [ ! -d "./extracted-spc-36911" ]; then
    echo "Extracting SPC mainnet subnet data..."
    $GENESIS_BIN extract state \
        "$ARCHIVE_BASE/2024-36911" \
        "./extracted-spc-36911" \
        --network 36911 \
        --state
else
    echo "SPC data already extracted"
fi

echo ""
echo "=== 2. Migrate to Different Formats ==="

# A. Migrate 96369 subnet to C-Chain (with blockchain ID prefix)
echo ""
echo "A. Migrating 96369 to C-Chain format..."
if [ ! -d "/home/z/.luxd/chainData/2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC/db/migrated-cchain" ]; then
    $GENESIS_BIN migrate subnet-to-cchain \
        "./extracted-subnet-96369" \
        "/home/z/.luxd/chainData/2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC/db/migrated-cchain" \
        --blockchain-id "2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC" \
        --clear-dest
else
    echo "C-Chain migration already complete"
fi

# B. Migrate ZOO to L2 format (no blockchain ID prefix)
echo ""
echo "B. Migrating ZOO (200200) to L2 format..."
if [ ! -d "./l2-data/zoo-200200" ]; then
    mkdir -p ./l2-data
    $GENESIS_BIN migrate subnet-to-l2 \
        "./extracted-zoo-200200" \
        "./l2-data/zoo-200200" \
        --chain-id 200200 \
        --clear-dest \
        --verify
else
    echo "ZOO L2 migration already complete"
fi

# C. Migrate SPC to L2 format
echo ""
echo "C. Migrating SPC (36911) to L2 format..."
if [ ! -d "./l2-data/spc-36911" ]; then
    mkdir -p ./l2-data
    $GENESIS_BIN migrate subnet-to-l2 \
        "./extracted-spc-36911" \
        "./l2-data/spc-36911" \
        --chain-id 36911 \
        --clear-dest \
        --verify
else
    echo "SPC L2 migration already complete"
fi

echo ""
echo "=== Migration Summary ==="
echo ""
echo "C-Chain Migration (96369):"
echo "  - Format: Keys prefixed with blockchain ID"
echo "  - Location: /home/z/.luxd/chainData/2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC/db/migrated-cchain"
echo "  - Usage: Direct replacement for luxd C-Chain"
echo ""
echo "L2 Migrations:"
echo "  - ZOO (200200): ./l2-data/zoo-200200"
echo "  - SPC (36911): ./l2-data/spc-36911"
echo "  - Format: Original key format (no blockchain ID prefix)"
echo "  - Usage: For subnet/L2 deployment"
echo ""
echo "Next Steps:"
echo "1. For C-Chain: Stop luxd, replace pebbledb directory, restart"
echo "2. For L2s: Use with lux-cli to deploy as L2 networks"
echo "   Example: lux-cli subnet create zoo --evm --chainId=200200"
echo "   Then copy the migrated data to the L2's chain directory"