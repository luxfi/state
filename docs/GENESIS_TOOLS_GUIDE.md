# Genesis Tools Guide

This guide provides comprehensive documentation for all genesis generation and management tools in the Lux Network ecosystem.

> **Quick Start**: To launch the full network with historic data, see [LAUNCH_GUIDE.md](../LAUNCH_GUIDE.md)

## Table of Contents

1. [Overview](#overview)
2. [Directory Structure](#directory-structure)
3. [Core Tools](#core-tools)
4. [Common Workflows](#common-workflows)
5. [Configuration Files](#configuration-files)
6. [Troubleshooting](#troubleshooting)

## Overview

The unified `genesis` CLI tool consolidates all genesis-related functionality into a single, well-organized command structure. It replaces dozens of individual scripts and tools with predictable, discoverable commands.

**Key Features:**
- **DRY Principle**: No duplicate functionality across different scripts
- **Predictable**: Consistent command structure and naming conventions
- **Discoverable**: Built-in help and command listing
- **Comprehensive**: All genesis operations in one tool
- **Well-Tested**: Integrated test suite for all commands

**Main Commands:**
- `generate` - Generate genesis files for all chains (P, C, X)
- `validators` - Manage validator configurations
- `extract` - Extract blockchain data from various sources
- `import` - Import blockchain data and cross-chain assets
- `analyze` - Analyze blockchain data and structures
- `inspect` - Inspect database contents in detail
- `scan` - Scan external blockchains for assets
- `migrate` - Migrate data between formats and chains
- `repair` - Fix and repair blockchain data issues
- `launch` - Launch Lux nodes with various configurations
- `export` - Export blockchain data in various formats
- `validate` - Validate configurations and data

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

### 1. genesis - Unified CLI Tool

The `genesis` tool is the single entry point for all genesis-related operations. It provides a consistent, hierarchical command structure.

```bash
# Generate genesis files (all chains)
./bin/genesis generate

# Get help for any command
./bin/genesis help
./bin/genesis <command> --help
./bin/genesis <command> <subcommand> --help

# See available subcommands
./bin/genesis <command>
```

**Complete Command Structure:**

```
genesis
├── generate        # Generate genesis files
│   ├── all        # Generate P, C, X chains (default)
│   ├── p-chain    # Generate P-Chain only
│   ├── c-chain    # Generate C-Chain only
│   └── x-chain    # Generate X-Chain only
│
├── validators      # Validator management
│   ├── list       # List validators
│   ├── add        # Add validator
│   ├── remove     # Remove validator
│   └── generate   # Generate validator keys
│
├── extract        # Extract blockchain data
│   ├── state      # Extract state data
│   ├── genesis    # Extract genesis config
│   ├── blocks     # Extract block data
│   └── allocations # Extract account allocations
│
├── import         # Import data
│   ├── chain-data # Import blockchain data
│   ├── genesis    # Import genesis config
│   ├── blocks     # Import block data
│   └── consensus  # Import consensus data
│
├── analyze        # Analysis operations
│   ├── keys       # Analyze database keys
│   ├── blocks     # Analyze block data
│   ├── subnet     # Analyze subnet data
│   ├── structure  # Analyze data structure
│   └── balance    # Analyze account balances
│
├── inspect        # Database inspection
│   ├── keys       # Inspect database keys
│   ├── blocks     # Inspect block details
│   ├── headers    # Inspect block headers
│   ├── snowman    # Inspect Snowman consensus DB
│   ├── prefixes   # Scan database prefixes
│   └── tip        # Find chain tip
│
├── scan           # Scan external blockchains
│   ├── tokens     # Scan for tokens
│   ├── nfts       # Scan for NFTs
│   ├── burns      # Scan burn events
│   └── holders    # Scan token holders
│
├── migrate        # Migration operations
│   ├── subnet     # Migrate subnet to C-Chain
│   ├── blocks     # Migrate blocks only
│   ├── evm        # Migrate EVM data
│   ├── full       # Full migration pipeline
│   ├── add-evm-prefix    # Add EVM prefix to keys
│   ├── rebuild-canonical # Fix evmn key format
│   ├── replay-consensus  # Create consensus state
│   └── zoo        # ZOO-specific migration
│
├── repair         # Fix/repair operations
│   ├── prefix     # Add EVM prefix to blocks
│   ├── snowman    # Fix Snowman consensus state
│   ├── canonical  # Rebuild canonical chain
│   ├── mappings   # Fix block mappings
│   └── pointers   # Set blockchain pointers
│
├── launch         # Launch Lux nodes
│   ├── clean      # Launch with clean state
│   ├── mainnet    # Launch mainnet config
│   ├── testnet    # Launch testnet config
│   ├── debug      # Launch with debug logging
│   └── migrated   # Launch with migrated data
│
├── export         # Export data
│   ├── state      # Export blockchain state
│   ├── genesis    # Export genesis config
│   ├── blocks     # Export block data
│   └── backup     # Create backup
│
├── validate       # Validate configurations
├── process        # Process historical data
└── help           # Show help information
```

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

#### Extract State (formerly namespace)

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
    --rpc http://localhost:9630/ext/bc/C/rpc \
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
# Complete end-to-end C-Chain import runbook
# This workflow takes your raw Subnet-EVM PebbleDB, rebuilds all canonical mappings,
# imports blocks and state, and launches a fully-bootstrapped C-Chain node with luxd.

# 0. Prerequisites
# - CLI tools built under bin/: migrate_evm, rebuild_canonical,
#   peek_tip_v2, replay-consensus-pebble, luxd
# - genesis JSON for chain-id 96369 in current directory
# - No other luxd running on ports 9630, 9650, etc.

# 1. Set up workspace
export TEST_ROOT=$HOME/.tmp/cchain-import
rm -rf $TEST_ROOT && mkdir -p \
  $TEST_ROOT/src/pebbledb \
  $TEST_ROOT/evm/pebbledb \
  $TEST_ROOT/state/pebbledb

# Sync your Subnet DB (full or partial for smoke test)
rsync -a /path/to/subnet/pebbledb/ $TEST_ROOT/src/pebbledb/

# 2. Migrate & de-namespace EVM keys
bin/migrate_evm \
  --src $TEST_ROOT/src/pebbledb \
  --dst $TEST_ROOT/evm/pebbledb

# Verify EVM prefix keys (h, b, r, n, H):
pebble lsm $TEST_ROOT/evm/pebbledb \
  | grep -E '^key\[0\]=0x65(766d68|766d62|766d72|766d6e|766d48)'

# 3. Rebuild canonical evmn mappings
bin/rebuild_canonical --db $TEST_ROOT/evm/pebbledb
# Pass 1: collect full hash→height; Pass 2: fix all evmn keys

# 4. Peek migrated tip
export TIP=$(bin/peek_tip_v2 --db $TEST_ROOT/evm/pebbledb)
echo "Final tip = $TIP"

# 5. Replay Snowman consensus into state DB
bin/replay-consensus-pebble \
  --evm   $TEST_ROOT/evm/pebbledb \
  --state $TEST_ROOT/state/pebbledb \
  --tip   $TIP

# 6. Drop old ChainConfigKey (force genesis reload)
pebble del \
  --db=$TEST_ROOT/evm/pebbledb \
  436861696e436f6e666967436f6e6669674b6579

# 7. Launch C-Chain node
luxd \
  --db-dir=$TEST_ROOT \
  --network-id=96369 \
  --staking-enabled=false \
  --http-port=9630 \
  --chain-configs.enable-indexing

# Look for bootstrap log lines:
# [C Chain] starting bootstrapper {"lastAcceptedHeight":<TIP>}
# [C Chain] bootstrapped

# 8. Smoke-test via RPC

# 8.1 Block number
curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc \
  | jq .result  # => "0x$(printf %x $TIP)"

# 8.2 Genesis hash
curl -s --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",false],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc \
  | jq .result.hash

# 8.3 Treasury balance
curl -s --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011e888251ab053b7bd1cdb598db4f9ded94714","latest"],"id":1}' \
  http://localhost:9630/ext/bc/C/rpc \
  | jq .result  # expect >= 0x1B1AE4D6E2EF50000
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

## Migration from Old Scripts/Tools

If you've been using the individual scripts and tools, here's how to migrate to the unified `genesis` command:

### Script to Command Mapping

| Old Script/Tool | New Command |
|----------------|-------------|
| `analyze-subnet-blocks.go` | `genesis analyze blocks --subnet` |
| `migrate-subnet-to-cchain.go` | `genesis migrate subnet` |
| `check-head-pointers.go` | `genesis inspect tip` |
| `fix-snowman-ids.go` | `genesis repair snowman` |
| `add-evm-prefix-to-blocks.go` | `genesis repair prefix` |
| `rebuild-canonical.go` | `genesis repair canonical` |
| `export-state-to-genesis.go` | `genesis export genesis` |
| `import-consensus.go` | `genesis import consensus` |
| `scan-db-prefixes.go` | `genesis inspect prefixes` |
| `launch-mainnet-automining.sh` | `genesis launch mainnet --automining` |
| `launch-clean-cchain.sh` | `genesis launch clean` |
| `analyze-keys-detailed.go` | `genesis analyze keys --detailed` |
| `find-highest-block.go` | `genesis inspect tip` |
| `check-block-format.go` | `genesis inspect blocks` |
| `migrate_evm.go` | `genesis migrate evm` |
| `namespace` | `genesis extract state` |
| `teleport` commands | `genesis scan` and `genesis migrate` |

### Example Migrations

**Old way:**
```bash
go run scripts/analyze-subnet-blocks.go
```

**New way:**
```bash
./bin/genesis analyze blocks --subnet
```

**Old way:**
```bash
./scripts/migrate-subnet-to-cchain.go /path/to/subnet/db /path/to/output
```

**New way:**
```bash
./bin/genesis migrate subnet /path/to/subnet/db /path/to/output
```

**Old way:**
```bash
./scripts/launch-mainnet-automining.sh
```

**New way:**
```bash
./bin/genesis launch mainnet --automining
```

### Benefits of Migration

1. **Single Tool**: No need to remember dozens of script names
2. **Consistent Flags**: All commands use the same flag conventions
3. **Better Help**: Built-in help for every command and subcommand
4. **Error Handling**: Consistent error messages and recovery
5. **Logging**: Unified logging across all operations
6. **Testing**: All commands have test coverage

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
3. **Verify outputs** using the archaeology tools
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
    ./bin/namespace \
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