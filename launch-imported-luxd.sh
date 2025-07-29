#!/bin/bash

# Set environment variables for imported blockchain
export LUX_IMPORTED_BLOCK_ID="646572b42a6210ac8efea0ab0df2a028acde2297c3ae07bc8dd1fc3e120b802a"
export LUX_IMPORTED_HEIGHT="1082780"
export LUX_IMPORTED_TIMESTAMP="1717148410"

echo "Starting luxd with imported blockchain data:"
echo "  Block ID: $LUX_IMPORTED_BLOCK_ID"
echo "  Height: $LUX_IMPORTED_HEIGHT"
echo "  Timestamp: $LUX_IMPORTED_TIMESTAMP ($(date -d @$LUX_IMPORTED_TIMESTAMP))"

# Launch luxd with the migrated data
../node/build/luxd \
    --dev \
    --network-id=96369 \
    --db-dir=runtime/luxd-final/db \
    --chain-config-dir=runtime/luxd-final/configs \
    --config-file=runtime/config.json \
    --plugin-dir=runtime/plugins \
    --log-level=info