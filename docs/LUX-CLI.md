# Lux CLI Tool

This document explains how to properly launch the Lux Network (96369) and L2 subnets using lux-cli.

## Prerequisites

1. Build luxd and lux-cli:
```bash
cd $HOME/work/lux
make build-node    # Builds luxd
make build-cli     # Builds lux-cli
```

2. Build genesis tools:
```bash
cd genesis
make build-genesis-pkg
```

## Quick Start

### Launch Mainnet (96369) with L2s
```bash
cd genesis
make launch-mainnet
```

This will:
1. Generate genesis with 11 validators and bootstrap nodes
2. Import existing 96369 C-Chain data
3. Start primary network with lux-cli
4. Deploy Zoo L2 (200200)
5. Deploy SPC L2 (36911)
6. Prepare Hanzo L2 (36963) for future deployment

### Launch Testnet (96368)
```bash
make launch-testnet
```

### Launch Local Development Network
```bash
make launch-local
```

## Manual Steps

### 1. Generate Genesis
```bash
cd genesis/cmd/genesis

# For mainnet with C-Chain import
./genesis \
    --network mainnet \
    --import-cchain ../data/unified-genesis/lux-mainnet/96369/genesis.json \
    --import-allocations ../data/unified-genesis/lux-mainnet/96369/allocations_combined.json \
    --validators ../configs/mainnet/validators.json \
    --output ../genesis_mainnet.json
```

### 2. Launch with lux-cli
```bash
cd cli

# Start network
./bin/lux network start \
    --luxgo-path ../node/build/luxd \
    --custom-network-genesis ../genesis/genesis_mainnet.json

# Check status
./bin/lux network status
```

### 3. Deploy L2 Subnets
```bash
# Deploy Zoo L2
./bin/lux subnet create zoo \
    --evm \
    --chain-id 200200 \
    --custom-subnet-evm-genesis ../genesis/data/unified-genesis/zoo-mainnet/200200/genesis.json

./bin/lux subnet deploy zoo --mainnet

# Deploy SPC L2
./bin/lux subnet create spc \
    --evm \
    --chain-id 36911 \
    --custom-subnet-evm-genesis ../genesis/data/unified-genesis/spc-mainnet/36911/genesis.json

./bin/lux subnet deploy spc --mainnet
```

## Network Architecture

### Primary Network (L1)
- **Network ID**: 96369 (mainnet), 96368 (testnet)
- **C-Chain**: Imported from existing 96369 data
- **P-Chain**: Generate genesis from initial validator set / bootstrapping nodes
- **X-Chain**: Generate genesis based on genesis allocations required (LUX has NFTs to deliver, ZOO has ZOO on BSC + NFTs on X-Chain)
- **Validators**: 11 bootstrap nodes
- **Consensus**: Lux Consensus, probalastic metastable, 1s finality

### L2 Subnets
- **Zoo**: Chain ID 200200 (mainnet), 200201 (testnet)
- **SPC**: Chain ID 36911 (mainnet), 36912 (testnet)
- **Hanzo**: Chain ID 36963 (mainnet), 36962 (testnet) - new deployment

## Bootstrap Nodes

The 11 mainnet bootstrap nodes are configured in:
- `configs/mainnet/validators.json` - Validator configurations
- `configs/mainnet/node.json` - Node parameters

## RPC Endpoints

After launch:
- **C-Chain**: https://api.lux.network/ext/bc/C/rpc also alias to /
- **Zoo L2**: https://api.zoo.network/
- **SPC L2**: https://api.sparklepony.club/

Get blockchain IDs:
```bash
./bin/lux subnet describe zoo --json | jq -r '.blockchainID'
./bin/lux subnet describe spc --json | jq -r '.blockchainID'
```

## Run Lux Genesis Network (7777)

To run the historical 7777 network for reference:
```bash
make run-7777-historical
```

This runs in dev mode and is separate from the main network.

## Troubleshooting

### lux-cli Not Found
```bash
git clone ssh://github.com/luxfi/cli && cd ../cli && go build -o bin/lux
```

### Network Won't Start
1. Check if ports 9630/9631 are free
2. Clean previous runs: `./bin/lux network clean`
3. Check logs: `./bin/lux network logs`

### L2 Deployment Fails
1. Ensure primary network is running: `./bin/lux network status`
2. Check validator set has enough stake
3. Verify genesis files exist in configs/

## Commands Reference

```bash
# Network management
./bin/lux network start      # Start network
./bin/lux network stop       # Stop network
./bin/lux network status     # Check status
./bin/lux network clean      # Clean up

# Subnet management
./bin/lux subnet create      # Create subnet
./bin/lux subnet deploy      # Deploy subnet
./bin/lux subnet list        # List subnets
./bin/lux subnet describe    # Get subnet details

# Node management
./bin/lux node list          # List nodes
./bin/lux node status        # Node status
```
