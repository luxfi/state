#!/bin/bash
# Main deployment script - delegates to specific deployment scripts

case "$1" in
    mainnet)
        exec scripts/deploy/deploy-mainnet.sh
        ;;
    testnet)
        exec scripts/deploy/deploy-testnet.sh
        ;;
    local|"")
        exec scripts/launch/launch-11-nodes.sh
        ;;
    *)
        echo "Usage: $0 [mainnet|testnet|local]"
        echo "  mainnet - Deploy mainnet with historical data"
        echo "  testnet - Deploy testnet"
        echo "  local   - Deploy local 11-node network (default)"
        exit 1
        ;;
esac
