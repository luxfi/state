# Lux Network Deployment Guide

## Overview

This guide explains how to deploy the Lux Network with historic chain data preserved.

## Chain Configuration

### P-Chain (Platform Chain)
- **Validators**: 11 bootstrap validators
- **Status**: Ready with BLS keys configured

### C-Chain (Contract Chain)
- **Chain ID**: 96369
- **Mode**: Historic blockchain data (no genesis needed)
- **Database**: 7.2G PebbleDB from Lux 96369 mainnet
- **Features**:
  - All existing accounts preserved
  - All deployed contracts intact
  - Complete transaction history
  - Continues from last known block

### X-Chain (Exchange Chain)
- **Assets**: Multi-asset support (LUX and ZOO)
- **LUX holders**: 189 (Lux 7777 delta + Ethereum NFT holders)
- **ZOO holders**: 109 (Historic burn participants with Egg NFTs)

### Subnet L2s
- **Zoo Network**: Chain ID 200200, 6,896 accounts
- **SPC Network**: Chain ID 36911, 49 accounts

## Deployment Steps

### 1. Setup C-Chain with Historic Data

```bash
# This copies the 7.2G historic blockchain data
./scripts/setup-cchain-with-historic-data.sh
```

### 2. Launch the Network

```bash
# Launch with historic C-Chain data
./launch-mainnet/with-historic-cchain.sh
```

### 3. Deploy Subnets (after network is running)

```bash
# Deploy Zoo and SPC L2s
./deploy-subnets-to-network.sh
```

## Important Notes

1. **C-Chain Data**: The C-Chain uses the complete historic blockchain data from Lux 96369 mainnet. This means:
   - All existing accounts and balances are preserved
   - All smart contracts continue to function
   - Transaction history is maintained

   **Note on C-Chain Genesis**: While the C-Chain state is preserved from historical data, a `genesis.json` file is still used to configure essential parameters like the chain ID and to define the network's initial settings. The genesis file does not contain any account allocations, as those are already present in the historic chain data.

2. **X-Chain**: Uses a new genesis with multi-asset support for both LUX and ZOO tokens

3. **Subnets**: Will be deployed as L2s after the main network is running

## Verification

After deployment, verify the network:

```bash
# Check validators
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"platform.getCurrentValidators","params":[]}' \
  http://localhost:9630/ext/bc/P

# Check C-Chain
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc

# Check X-Chain assets
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"avm.getUTXOs","params":{"addresses":["X-local1...]"}}' \
  http://localhost:9630/ext/bc/X
```

## Network Endpoints

- **RPC URL**: http://localhost:9630
- **C-Chain RPC**: http://localhost:9630/ext/bc/C/rpc
- **X-Chain RPC**: http://localhost:9630/ext/bc/X
- **P-Chain RPC**: http://localhost:9630/ext/bc/P
