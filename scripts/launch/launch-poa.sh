#!/bin/bash
set -e

echo "Starting luxd in POA mode..."

# Set paths
LUXD="../node/build/luxd"
GENESIS="genesis_mainnet_96369.json"

# Launch luxd with POA configuration
$LUXD \
    --network-id=96369 \
    --genesis-file=$GENESIS \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-enabled=false \
    --sybil-protection-enabled=false \
    --snow-sample-size=1 \
    --snow-quorum-size=1 \
    --snow-concurrent-repolls=1 \
    --snow-optimal-processing=1 \
    --consensus-shutdown-timeout=1s \
    --log-level=info