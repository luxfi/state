#!/bin/bash
set -e

echo "Starting luxd in dev mode for network 7777..."

# Set paths
LUXD="../node/build/luxd"

# Clean data directory
rm -rf ~/.luxd-dev
mkdir -p ~/.luxd-dev

# Launch luxd in dev mode
$LUXD \
    --dev \
    --network-id=7777 \
    --data-dir=~/.luxd-dev \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --log-level=info

echo "✅ Dev node launched on network 7777!"
echo "RPC: http://localhost:9630"