#!/bin/bash
set -e

echo "Starting single Lux node..."

# Set paths
LUXD="../node/build/luxd"
GENESIS="genesis_mainnet_96369.json"

# Clean previous runs
echo "Cleaning previous data..."
pkill -f "luxd.*network-id=96369" || true
sleep 2
rm -rf ~/.luxd

# Create directories
mkdir -p ~/.luxd/staking

# Copy first validator's keys (without BLS for now)
cp validator-keys/validator-1/staking/staker.crt ~/.luxd/staking/
cp validator-keys/validator-1/staking/staker.key ~/.luxd/staking/

# Start node without BLS signing (staking disabled)
echo "Starting luxd..."
$LUXD \
    --network-id=96369 \
    --genesis-file=$GENESIS \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-enabled=false \
    --sybil-protection-enabled=false \
    --snow-sample-size=1 \
    --snow-quorum-size=1 \
    --log-level=info

echo "Node started!"