#\!/bin/bash
set -e

# Launch luxd directly with proper genesis
GENESIS_DIR="/home/z/work/lux/genesis"
LUXD_PATH="/home/z/work/lux/node/build/luxd"
DATA_DIR="$HOME/.luxd"

echo "Starting luxd with mainnet genesis..."

# Clean existing data
rm -rf "$DATA_DIR"
mkdir -p "$DATA_DIR/staking"

# Use first validator keys
cp validator-keys/validator-1/staking/staker.crt "$DATA_DIR/staking/"
cp validator-keys/validator-1/staking/staker.key "$DATA_DIR/staking/"
cp validator-keys/validator-1/bls.key "$DATA_DIR/staking/signer.key"

# Launch luxd
$LUXD_PATH \
    --network-id=96369 \
    --data-dir="$DATA_DIR" \
    --genesis-file="$GENESIS_DIR/genesis_mainnet_96369.json" \
    --http-host=0.0.0.0 \
    --http-port=9630 \
    --staking-enabled=false \
    --snow-sample-size=1 \
    --snow-quorum-size=1 \
    --log-level=info
