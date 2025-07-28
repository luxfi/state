#!/bin/bash

# Launch luxd with migrated block data

# Create directories if they don't exist
mkdir -p runtime/mainnet/db/C

# Copy the migrated block data to C-Chain location
echo "Copying migrated block data to C-Chain location..."
cp -r output/mainnet/C/chaindata-complete-evm/. runtime/mainnet/db/C/

# Copy the genesis to the expected location
mkdir -p runtime/mainnet/configs/C
cp configs/mainnet/C/genesis.json runtime/mainnet/configs/C/

# Launch luxd with the migrated data
echo "Launching luxd with migrated block data..."
../node/build/luxd \
    --network-id=96369 \
    --data-dir=runtime/mainnet \
    --dev \
    --enable-automining \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --public-ip=127.0.0.1 \
    --consensus-sample-size=1 \
    --consensus-quorum-size=1 \
    --consensus-shutdown-timeout=1s \
    --log-level=debug