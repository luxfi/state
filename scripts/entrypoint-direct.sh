#!/bin/bash

# Entrypoint script for direct luxd execution

# Default values
NETWORK_ID="${NETWORK_ID:-96369}"
HTTP_PORT="${HTTP_PORT:-9630}"
STAKING_PORT="${STAKING_PORT:-9631}"
LOG_LEVEL="${LOG_LEVEL:-info}"

# Always run in dev mode
echo "Starting luxd in dev mode for network $NETWORK_ID..."
echo "HTTP port: $HTTP_PORT"
echo "Staking port: $STAKING_PORT"

# Execute luxd with dev mode and all arguments
exec luxd --dev \
    --network-id="$NETWORK_ID" \
    --http-port="$HTTP_PORT" \
    --staking-port="$STAKING_PORT" \
    --log-level="$LOG_LEVEL" \
    --data-dir=/data \
    "$@"