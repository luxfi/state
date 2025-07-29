#!/usr/bin/env bash
set -euo pipefail

echo "▶ Preparing runtime directory"
TARGET_EVM="$RUNTIME_DIR/db/chains/$CHAIN_ID/vm/$VM_ID/evm"

if [ ! -d "$TARGET_EVM" ] || [ -z "$(ls -A "$TARGET_EVM" 2>/dev/null)" ]; then
  mkdir -p "$(dirname "$TARGET_EVM")"
  
  echo "▶ Importing subnet data with namespace stripping"
  genesis import subnet \
    "$SRC_CHAINDATA_DIR/lux-mainnet-96369/db/pebbledb" \
    "$TARGET_EVM"

  echo "▶ Cleaning 10-byte canonical keys"
  genesis repair delete-suffix "$TARGET_EVM" 6e --prefix 68

  echo "▶ Rebuilding canonical mappings with 9-byte keys"
  genesis rebuild-canonical "$TARGET_EVM"

  echo "▶ Writing consensus markers (height $TIP_HEIGHT)"
  write_markers \
      --db "$RUNTIME_DIR/db/chains/$CHAIN_ID" \
      --tip "$TIP_HASH" \
      --height "$TIP_HEIGHT"
else
  echo "▶ Using existing migrated data"
fi

echo "▶ Launching luxd on port 9630"
exec luxd \
  --network-id "$NETWORK_ID" \
  --data-dir   "$RUNTIME_DIR" \
  --db-type    pebbledb \
  --http-host  0.0.0.0 \
  --http-port  9630 \
  --dev \
  --poa-single-node-mode \
  "$@"