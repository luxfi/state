# Lux Network Genesis Pipeline

This document describes the complete pipeline for building genesis data for Lux Network's decentralized L1 and L2 chains, including migration of external assets.

## Overview

The genesis pipeline consists of four main phases:

1. **Data Extraction** - Extract blockchain data from existing chains
2. **Asset Migration** - Import external assets from other blockchains
3. **Genesis Generation** - Combine all data into genesis files
4. **Network Launch** - Deploy networks with complete historical data

## Tools

### `archeology` - Blockchain Data Extraction
Extracts and analyzes blockchain data from PebbleDB/LevelDB databases.

### `teleport` - External Asset Migration
Scans external blockchains (Ethereum, BSC, etc.) and prepares assets for migration to Lux Network as L1, L2, or L3.

### `genesis` - Genesis Generation & Launch
Generates genesis files and launches networks with complete configurations.

## Complete Pipeline Workflow

### Phase 1: Extract Existing Blockchain Data

Extract data from existing Lux chains:

```bash
# Extract LUX mainnet (96369)
archeology extract \
  --source /path/to/lux-96369/db/pebbledb \
  --destination ./data/extracted/lux-96369 \
  --chain-id 96369 \
  --include-state

# Extract ZOO mainnet (200200)
lux-archeology extract \
  --source /path/to/zoo-200200/db/pebbledb \
  --destination ./data/extracted/zoo-200200 \
  --network zoo-mainnet \
  --include-state

# Extract SPC mainnet (36911)
archeology extract \
  --source /path/to/spc-36911/db/pebbledb \
  --destination ./data/extracted/spc-36911 \
  --chain-id 36911 \
  --include-state
```

### Phase 2: Analyze Extracted Data

Verify extraction and analyze contents:

```bash
# Analyze LUX data
archeology analyze \
  --db ./data/extracted/lux-96369 \
  --output ./reports/lux-analysis.json

# Check specific accounts (e.g., treasury)
archeology analyze \
  --db ./data/extracted/lux-96369 \
  --account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

### Phase 3: Migrate External Assets

#### Scan NFTs from Ethereum

```bash
# Scan Lux Genesis NFTs from Ethereum
teleport scan-nft \
  --chain ethereum \
  --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
  --project lux \
  --output ./data/external/lux-nfts-ethereum.json
```

#### Scan Tokens from BSC

```bash
# Scan historic ZOO tokens from BSC
teleport scan-token \
  --chain bsc \
  --contract 0xZOO_TOKEN_ADDRESS \
  --project zoo \
  --output ./data/external/zoo-tokens-bsc.json
```

#### Migrate Any Project to Lux

```bash
# Migrate any ERC20 token to a new L2 subnet
teleport migrate \
  --source-chain ethereum \
  --contract 0xYOUR_TOKEN \
  --token-type erc20 \
  --target-layer L2 \
  --target-name your-subnet \
  --target-chain-id 100001
```

### Phase 4: Generate Genesis Files

Combine all data sources into genesis files:

```bash
# Generate LUX mainnet genesis
genesis generate \
  --network lux-mainnet \
  --chain-id 96369 \
  --data ./data/extracted/lux-96369 \
  --external ./data/external/ \
  --output ./genesis/lux-mainnet-96369.json

# Generate ZOO mainnet genesis
genesis generate \
  --network zoo-mainnet \
  --chain-id 200200 \
  --data ./data/extracted/zoo-200200 \
  --external ./data/external/ \
  --output ./genesis/zoo-mainnet-200200.json
```

### Phase 5: Launch Networks

#### Development Mode (Single Node)

```bash
# Launch LUX with automining
genesis launch \
  --network lux-mainnet \
  --genesis ./genesis/lux-mainnet-96369.json \
  --dev-mode \
  --automining \
  --rpc-port 9650
```

#### Production Mode (Multi-Validator)

```bash
# Launch with 5 validators
genesis launch \
  --network lux-mainnet \
  --genesis ./genesis/lux-mainnet-96369.json \
  --validators 5 \
  --detached
```

### Phase 6: Deploy Subnets

```bash
# Deploy L2 subnets
genesis deploy \
  --subnet zoo \
  --genesis ./genesis/zoo-mainnet-200200.json \
  --validators 3

genesis deploy \
  --subnet spc \
  --genesis ./genesis/spc-mainnet-36911.json \
  --validators 3
```

## Migration Scenarios

### Scenario 1: Migrate ERC20 Token to L2 Subnet

```bash
# 1. Scan token holders
teleport scan-token \
  --chain ethereum \
  --contract 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
  --project usdc

# 2. Migrate to L2
teleport migrate \
  --source-chain ethereum \
  --contract 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
  --target-layer L2 \
  --target-name usdc-subnet

# 3. Generate genesis
genesis generate \
  --network usdc-subnet \
  --external ./migrations/usdc-subnet-L2/

# 4. Deploy subnet
genesis deploy \
  --subnet usdc-subnet \
  --genesis ./genesis/usdc-subnet.json
```

### Scenario 2: Create Sovereign L1 from NFT Collection

```bash
# 1. Scan NFT collection
teleport scan-nft \
  --chain ethereum \
  --contract 0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D \
  --project bayc \
  --include-metadata

# 2. Migrate to sovereign L1
teleport migrate \
  --source-chain ethereum \
  --contract 0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D \
  --token-type erc721 \
  --target-layer L1 \
  --target-name ape-chain

# 3. Launch L1
genesis launch \
  --network ape-chain \
  --genesis ./migrations/ape-chain-L1/genesis.json \
  --validators 5
```

### Scenario 3: Application-Specific L3

```bash
# Migrate game token to L3
teleport migrate \
  --source-chain polygon \
  --contract 0xGAME_TOKEN \
  --target-layer L3 \
  --target-name game-chain \
  --genesis-template ./templates/gaming-l3.json
```

## Directory Structure

```
./data/
├── extracted/          # Blockchain data from lux-archeology
│   ├── lux-96369/
│   ├── zoo-200200/
│   └── spc-36911/
├── external/           # External assets from teleport
│   ├── lux-nfts-ethereum.json
│   └── zoo-tokens-bsc.json
└── genesis/            # Generated genesis files
    ├── lux-mainnet-96369.json
    ├── zoo-mainnet-200200.json
    └── spc-mainnet-36911.json

./migrations/           # Migration artifacts
├── usdc-subnet-L2/
├── ape-chain-L1/
└── game-chain-L3/
```

## Validation Checklist

Before launching production networks:

1. **Data Validation**
   ```bash
   lux-archeology validate --db ./data/extracted/lux-96369
   ```

2. **Genesis Validation**
   ```bash
   genesis validate --genesis ./genesis/lux-mainnet-96369.json
   ```

3. **Cross-Reference Check**
   ```bash
   teleport verify \
     --genesis ./genesis/lux-mainnet-96369.json \
     --external ./data/external/
   ```

4. **Treasury Balance Verification**
   - Verify treasury account has expected balance
   - Check total supply matches expectations

## Network Endpoints

After deployment:

### L1 Networks
- RPC: `https://api.<network>.network`
- Explorer: `https://explorer.<network>.network`

### L2/L3 Subnets
- RPC: `https://api.lux.network/ext/bc/<subnet-id>/rpc`
- Explorer: `https://subnets.lux.network/<subnet-name>`

## Troubleshooting

### Common Issues

1. **Namespace Errors**
   ```bash
   # Use denamespace command
   lux-archeology denamespace \
     --source /path/to/raw/db \
     --destination ./denamespaced \
     --chain-id 96369
   ```

2. **Missing Cross-Chain Assets**
   ```bash
   # Re-scan with broader block range
   teleport scan-token \
     --from-block 0 \
     --to-block latest
   ```

3. **Genesis Validation Failures**
   ```bash
   # Check with verbose output
   genesis validate \
     --genesis ./genesis/network.json \
     --verbose
   ```

## Best Practices

1. **Always validate data** after extraction
2. **Take snapshots** before migrations
3. **Test in development** before production
4. **Keep backups** of all genesis files
5. **Document custom configurations**
6. **Use version control** for genesis files

## Support

For issues or questions:
- GitHub: https://github.com/luxfi/genesis
- Discord: https://discord.gg/lux-network
- Docs: https://docs.lux.network
