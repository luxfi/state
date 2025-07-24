# Genesis Tools Guide

This guide provides comprehensive documentation for all genesis generation and management tools in the Lux Network ecosystem.

## Table of Contents

1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [Core Tools](#core-tools)
4. [Common Workflows](#common-workflows)
5. [Configuration Files](#configuration-files)
6. [Troubleshooting](#troubleshooting)

## Overview

The Lux Network genesis tools are designed to:
- Generate genesis files for P-Chain, C-Chain, and X-Chain
- Import and process historical blockchain data
- Manage cross-chain token migrations
- Validate genesis configurations

The primary tool is `genesis`, which provides:
- Simple genesis generation for all chains with one command
- Proper directory structure matching Lux node expectations
- Integration with other specialized tools
- A `tools` command to discover all available utilities

## Directory Structure

### Standard Genesis Organization

The tools follow the Lux node's expected directory structure:

```
configs/
├── mainnet/           # Mainnet configurations
│   ├── P/            # P-Chain (Platform)
│   │   └── genesis.json
│   ├── C/            # C-Chain (Contract/EVM)
│   │   └── genesis.json
│   └── X/            # X-Chain (Exchange)
│       └── genesis.json
└── testnet/           # Testnet configurations
    ├── P/
    ├── C/
    └── X/
```

### Data Organization

```
chaindata/             # Raw blockchain data
├── lux-mainnet-96369/ # LUX mainnet data
├── zoo-mainnet-200200/# ZOO network data
├── spc-mainnet-36911/ # SPC network data
└── eth-mainnet/       # Ethereum crosschain data
```

## Core Tools

### Tool Selection Guide

The `genesis` tool is now the unified interface for all genesis-related operations:

- **Need to generate all genesis files?** → Use `genesis generate`
- **Managing validators?** → Use `genesis validators`
- **Extracting blockchain data?** → Use `genesis extract state`
- **Analyzing extracted data?** → Use `genesis analyze`
- **Cross-chain migrations?** → Use `genesis migrate`
- **Not sure which command?** → Run `genesis tools`

### 1. genesis

The unified tool for all genesis-related operations. It combines functionality from all specialized tools into a single interface.

```bash
# Generate genesis files (all chains)
./bin/genesis generate

# Generate for testnet
./bin/genesis generate --network testnet

# Generate with custom output directory
./bin/genesis generate --output /path/to/custom/dir

# Generate without standard directory structure
./bin/genesis generate --standard-dirs=false

# List all available commands
./bin/genesis tools

# Get help for any command
./bin/genesis <command> --help
```

**Main Commands:**
- `generate`: Generate genesis files for all chains
- `validators`: Manage validators (list, add, remove, generate)
- `extract`: Extract blockchain data from various sources
- `import`: Import blockchain data and allocations
- `analyze`: Analyze extracted blockchain data
- `scan`: Scan external blockchains for assets
- `migrate`: Migrate cross-chain assets
- `process`: Process historical blockchain data
- `validate`: Validate genesis configuration
- `tools`: List all available commands

**Generate Options:**
- `--network`: Network type (mainnet/testnet)
- `--output`: Output directory (default: configs/{network})
- `--standard-dirs`: Use P/, C/, X/ subdirectories (default: true)
- `--p-chain`: Generate P-Chain genesis (default: true)
- `--c-chain`: Generate C-Chain genesis (default: true)
- `--x-chain`: Generate X-Chain genesis (default: true)
- `--include-treasury`: Include treasury allocation (default: true)
- `--validators`: Path to validators JSON file

### Example Subcommands

#### Extract State (formerly denamespace)

```bash
# Extract with state data
./bin/genesis extract state /path/to/pebbledb /path/to/output \
    --network 96369 \
    --state

# Extract without state (faster, headers only)
./bin/genesis extract state /path/to/pebbledb /path/to/output \
    --network 96369
```

#### Extract Genesis from Blockchain

```bash
# Extract complete genesis configuration from blockchain database
./bin/genesis extract genesis /path/to/chaindata \
    --type auto \
    --output genesis.json \
    --alloc \
    --pretty

# Extract genesis without allocations (smaller file)
./bin/genesis extract genesis /path/to/chaindata \
    --alloc=false \
    --output genesis-config-only.json

# Extract genesis and export allocations to CSV
./bin/genesis extract genesis /path/to/chaindata \
    --output genesis.json \
    --csv allocations.csv

# Auto-detect database type (PebbleDB or LevelDB)
./bin/genesis extract genesis ~/.luxd/data/C/db \
    --type auto \
    --output c-chain-genesis.json
```

#### Validators Management

```bash
# List validators
./bin/genesis validators list

# Add a validator
./bin/genesis validators add \
    --node-id NodeID-ABC123... \
    --eth-address 0x... \
    --weight 100000000000000

# Generate validators from mnemonic
./bin/genesis validators generate \
    --mnemonic "your twelve word mnemonic phrase here" \
    --offsets 0,1,2,3,4
```

#### Analyze Blockchain Data

```bash
# Analyze extracted data
./bin/genesis analyze \
    -db /path/to/extracted/data \
    -network lux-mainnet

# Find specific account balance
./bin/genesis analyze \
    -db /path/to/extracted/data \
    -account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

#### Import Operations

```bash
# Import from original genesis block
./bin/genesis import genesis /path/to/genesis.json \
    --chain C \
    --allocations-only \
    --output allocations.json

# Import allocations from CSV
./bin/genesis import allocations allocations.csv \
    --format csv \
    --merge

# Import state from specific block (requires RPC)
./bin/genesis import block 1000000 \
    --rpc http://localhost:9650/ext/bc/C/rpc \
    --output block-1M-state.csv

# Import C-Chain data from extracted blockchain
./bin/genesis import cchain /path/to/extracted-data
```

#### Cross-Chain Operations

```bash
# Migrate ZOO tokens from BSC
./bin/genesis migrate zoo-migrate

# Scan NFT holders on Ethereum
./bin/genesis scan nft-holders \
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311

# Scan token burns
./bin/genesis scan token-burns \
    --chain bsc \
    --token 0x0a6045b79151d0a54dbd5227082445750a023af2
```

## Common Workflows

### 1. Complete Genesis Generation

```bash
# Step 1: Extract blockchain data (if needed)
./bin/genesis extract state ~/.luxd/db/pebbledb ./extracted-data \
    --network 96369 \
    --state

# Step 2: Generate genesis
./bin/genesis generate \
    --network mainnet \
    --validators configs/mainnet-validators.json

# Step 3: Verify the output
ls -la configs/mainnet/*/genesis.json

# Step 4: Validate the configuration
./bin/genesis validate --network mainnet
```

### 2. Importing Historical Data

```bash
# Extract 7777 network data
./bin/genesis extract state /archived/lux-7777/pebbledb ./extracted-7777 \
    --network 7777 \
    --state

# Analyze the data
./bin/genesis analyze \
    -db ./extracted-7777 \
    -network lux-7777
```

### 3. Importing from Original Genesis

```bash
# Step 1: Import allocations from original genesis
./bin/genesis import genesis /path/to/original/genesis.json \
    --chain C \
    --allocations-only \
    --output original-allocations.json

# Step 2: Merge with new allocations
./bin/genesis import allocations new-allocations.csv \
    --format csv \
    --merge

# Step 3: Generate new genesis with imported data
./bin/genesis generate \
    --network mainnet \
    --import-allocations original-allocations.json
```

### 4. Cross-Chain Token Migration

```bash
# Step 1: Scan BSC for ZOO burns
./bin/genesis scan token-burns \
    --chain bsc \
    --token 0x0a6045b79151d0a54dbd5227082445750a023af2

# Step 2: Scan egg NFT holders
./bin/genesis scan egg-holders

# Step 3: Generate migration allocations
./bin/genesis migrate zoo-migrate \
    --output exports/zoo-migration.csv
```

## Configuration Files

### Validator Configuration (validators.json)

```json
[
  {
    "nodeID": "NodeID-...",
    "ethAddress": "0x...",
    "publicKey": "0x...",
    "proofOfPossession": "0x...",
    "weight": 100000000000000,
    "delegationFee": 20000
  }
]
```

### Network Configuration

Located in `chaindata/configs/{network}/`:
- `chain.json`: Chain-specific parameters
- `node-config.json`: Node configuration
- `subnet.json`: Subnet configuration (for L2s)

## Troubleshooting

### Common Issues

1. **"Failed to open database"**
   - Ensure the PebbleDB path is correct
   - Check permissions on the directory
   - Verify the database wasn't corrupted

2. **"Invalid chain ID"**
   - Double-check the network parameter
   - Use the correct chain ID (96369 for mainnet, 96368 for testnet)

3. **"Missing allocations file"**
   - Ensure all CSV files are in the expected locations
   - Check the paths in the tool's help output

### Debug Commands

```bash
# Verify PebbleDB structure
ls -la /path/to/pebbledb/

# Check extracted data
find ./extracted-data -name "*.sst" | wc -l

# Validate genesis JSON
jq . configs/mainnet/C/genesis.json > /dev/null
```

## Best Practices

1. **Always backup data** before running extraction or migration tools
2. **Test on small datasets** first using the `-limit` flag
3. **Verify outputs** using the archeology tools
4. **Keep logs** of all operations for auditing
5. **Use standard directory structure** for compatibility with luxd

## Advanced Usage

### Custom Genesis Generation

```bash
# Generate only specific chains
./bin/genesis \
    --p-chain=false \
    --c-chain=true \
    --x-chain=false

# Use custom allocations
./bin/genesis \
    --lux7777 custom-allocations.csv \
    --zoo custom-zoo-allocations.csv
```

### Batch Processing

```bash
# Process multiple networks
for network in mainnet testnet; do
    ./bin/genesis --network $network
done

# Extract multiple chain data
for chainid in 96369 200200 36911; do
    ./bin/denamespace \
        -src /data/$chainid/pebbledb \
        -dst ./extracted/$chainid \
        -network $chainid \
        -state
done
```

## Tool Development

To add new features or modify existing tools:

1. Tools are located in `cmd/` directory
2. Shared packages are in `pkg/`
3. Build all tools: `make build`
4. Run tests: `make test`

## Support

For issues or questions:
1. Check the logs in the output directory
2. Review the tool's help output (`--help`)
3. Consult the main README.md
4. Check GitHub issues for similar problems