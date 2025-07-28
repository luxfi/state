# Lux Network Launch Guide

This guide explains how to launch the full Lux Network with historic data using the Makefile targets.

## Quick Start

Launch the full network (LUX primary network + all L2s):
```bash
make launch
```

## Network Configuration

### Primary Network (LUX)
- **Chain ID**: 1 (presented to apps) + 96369 (network ID)
- **Type**: C-Chain with POA automining
- **Data**: Imported from existing chain data
- **RPC**: http://localhost:9630/ext/bc/C/rpc

### L2 Networks

#### ZOO L2
- **Chain ID**: 200200
- **Token**: ZOO
- **Data**: Imported from existing chain data + BSC migration
- **Status**: Deployed with historic data

#### SPC L2
- **Chain ID**: 36911
- **Token**: SPC
- **Data**: Imported from existing chain data
- **Status**: Deployed with historic data

#### Hanzo L2
- **Chain ID**: 36963
- **Token**: AI
- **Data**: Fresh deployment (no historic data)
- **Status**: New deployment

## Available Commands

### Full Network Operations
```bash
make launch              # Launch full network (primary + L2s)
make launch-full         # Same as 'make launch'
make launch-primary      # Launch only LUX primary network
make launch-test         # Launch test configuration
make kill-node           # Stop all running nodes
make network-info        # Show network information
```

### Individual Steps
```bash
make prepare-import      # Prepare genesis data for import
make launch-lux          # Launch LUX primary network
make create-l2s          # Create L2 configurations
make deploy-l2s          # Deploy L2s to local network
```

## Prerequisites

1. Build the node and CLI tools:
```bash
cd $HOME/work/lux/node && ./scripts/build.sh
cd $HOME/work/lux/cli && go build -o bin/lux cmd/main.go
```

2. Ensure you have the chaindata available:
- Located in `chaindata/` directory
- Or in `~/.luxd/` directory

## Docker-based Deployment (Recommended)

For a more isolated and repeatable setup, you can use the provided Docker Compose configuration. This is the recommended way to launch a full, production-like network.

### Quick Start with Docker

1.  **Build the Docker image:**
    ```bash
    # This command is run from the genesis directory
    docker-compose -f docker/compose.yml build
    ```

2.  **Launch the network:**
    ```bash
    # This will start the primary network and deploy all subnets
    docker-compose -f docker/compose.yml up

    # To run in detached mode:
    docker-compose -f docker/compose.yml up -d
    ```

### Services

The `docker-compose.yml` file defines the following services:
- `lux-primary`: The main Lux network node.
- `subnet-deployer`: A service that waits for the primary node to be healthy and then deploys the ZOO, SPC, and Hanzo subnets.
- `lux-genesis-7777`: An optional service to run the historic 7777 network.
- `monitor`: An optional Prometheus service for monitoring.

### Using Profiles

You can launch optional services using profiles:

```bash
# Launch with the historic 7777 network
docker-compose -f docker/compose.yml up --profile historic

# Launch with the monitoring stack
docker-compose -f docker/compose.yml up --profile monitoring

# Launch with all services
docker-compose -f docker/compose.yml up --profile "*"
```

### Stopping the Network

```bash
docker-compose -f docker/compose.yml down
```

## Troubleshooting

### Node won't start
```bash
make kill-node           # Kill any existing processes
make launch              # Try again
```

### Check if network is running
```bash
curl -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  -H 'content-type:application/json;' http://localhost:9630/ext/bc/C/rpc
```

### View logs
```bash
tail -f output/lux-mainnet.log
```

### Get L2 blockchain IDs
```bash
make network-info
```

## Advanced Configuration

### Environment Variables
```bash
NODE_DIR=/path/to/node make launch      # Custom node directory
CLI_DIR=/path/to/cli make launch        # Custom CLI directory
DATA_DIR=/path/to/data make launch      # Custom data directory
```

### Launch with specific chain data
```bash
# First prepare the import
make prepare-import

# Then launch with the prepared data
make launch-lux
make create-l2s
make deploy-l2s
```

## Architecture Notes

- The LUX primary network runs with chain ID 1 for Ethereum compatibility
- The actual network ID is 96369 for POA consensus
- L2s maintain their original chain IDs (200200, 36911, 36963)
- All networks run on a single node for development
- POA automining is enabled for instant transactions