# Lux Network Genesis Pipeline

Complete pipeline for building genesis data for Lux Network's L1 and L2 chains.

## Overview

The genesis pipeline consists of three main phases:

1. **Data Extraction** - Extract blockchain data from existing chains
2. **Genesis Generation** - Create genesis files with all allocations
3. **Network Deployment** - Launch networks with historical data

## Tools

### CLI Plugins

Both tools are available as lux-cli plugins after running `make install-plugin`:

#### `lux-cli genesis`
- Generate genesis files for all networks
- Import historical blockchain data
- Validate genesis configurations

#### `lux-cli archaeology`
- Extract data from PebbleDB/LevelDB databases
- Scan external blockchains for assets
- Analyze token holders and burns

### Standalone Tools

#### `denamespace`
Removes namespace prefixes from PebbleDB data for C-Chain compatibility.

#### `extract-state`
Extracts full state from blockchain databases.

## Workflows

### 1. Generate Genesis Files

```bash
# Using make commands
make generate-all-genesis

# Using CLI plugin
lux-cli genesis generate --network mainnet --validators-file configs/mainnet-validators.json
```

### 2. Extract Blockchain Data

```bash
# Extract mainnet data
lux-cli archaeology extract \
  --source chaindata/lux-mainnet-96369/db/pebbledb \
  --destination data/extracted/lux-96369 \
  --chain-id 96369 \
  --include-state
```

### 3. Deploy Networks

```bash
# Local test network
make deploy

# Mainnet with historical data
make deploy-mainnet

# Testnet
make deploy-testnet
```

## Network Configuration

| Network | Chain ID | Genesis Status |
|---------|----------|----------------|
| Lux Mainnet | 96369 | ✅ Complete |
| Lux Testnet | 96368 | ✅ Complete |
| Zoo Mainnet | 200200 | ✅ Complete |
| Zoo Testnet | 200201 | ✅ Complete |
| SPC Mainnet | 36911 | ✅ Complete |

## Genesis Components

### P-Chain (Platform)
- Validator set configuration
- Staking parameters
- Network topology

### C-Chain (Contract)
- Account balances from CSV airdrop
- Treasury allocation (2T LUX)
- Contract states

### X-Chain (Exchange)
- UTXO allocations
- Asset definitions
- NFT configurations

## Key Files

- `configs/mainnet-validators.json` - Validator configurations
- `output/mainnet/genesis-mainnet-96369.json` - Complete mainnet genesis
- `chaindata/*/7777-airdrop-*.csv` - Airdrop allocations

## Security Notes

- Validator keys are generated deterministically from mnemonic
- Never commit real validator keys or mnemonics
- Use separate mnemonics for different networks
- All sensitive files are gitignored