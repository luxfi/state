# EVM Archaeology

A comprehensive tool for extracting, analyzing, and migrating historical blockchain data from various EVM chains.

## Features

- **Extract**: Remove namespace prefixes and extract clean blockchain data from LevelDB/PebbleDB
- **Scan**: Find NFTs and tokens on external chains (Ethereum, BSC, etc.)
- **Genesis**: Generate X-Chain or P-Chain genesis files with complete historical data
- **Analyze**: Examine blockchain databases without extraction
- **List**: View available configurations and supported chains

## Installation

```bash
# From the genesis directory
make build-archaeology

# Or build directly
cd cmd/evmarchaeology
go build -o ../../bin/evmarchaeology .
```

## Usage

### Extract Blockchain Data

Extract data from namespaced PebbleDB to clean format:

```bash
# Extract Lux mainnet data
evmarchaeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet

# Extract with state data
evmarchaeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet \
  -all

# Extract specific accounts only
evmarchaeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet \
  -state \
  -addresses 0x123...,0x456...
```

### Scan External Assets

Find NFTs and tokens on other EVM chains:

```bash
# Scan Lux NFTs on Ethereum
evmarchaeology scan \
  --chain ethereum \
  --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
  --project lux \
  --type nft

# Scan Zoo tokens on BSC (auto-detect type)
evmarchaeology scan \
  --chain bsc \
  --contract 0xYOUR_CONTRACT_ADDRESS \
  --project zoo \
  --type auto

# Scan with custom RPC
evmarchaeology scan \
  --rpc https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY \
  --contract 0xADDRESS \
  --project lux
```

### Generate Genesis Files

Create complete genesis files with all historical assets:

```bash
# Generate X-Chain genesis with all assets
evmarchaeology genesis \
  --nft-csv exports/lux-nfts-ethereum.csv \
  --token-csv exports/zoo-tokens-bsc.csv \
  --accounts-csv exports/7777-accounts.csv \
  --output configs/xchain-genesis-complete.json

# Generate with only NFTs for validator staking
evmarchaeology genesis \
  --nft-csv exports/lux-nfts-ethereum.csv \
  --chain x-chain \
  --output configs/xchain-genesis.json
```

### Analyze Database

Examine database contents without extraction:

```bash
# Analyze database structure
evmarchaeology analyze --db /path/to/db --show-types

# Find specific account
evmarchaeology analyze \
  --db /path/to/db \
  --account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714

# Get database statistics
evmarchaeology analyze --db /path/to/db --show-stats
```

### List Configurations

View available chains, projects, and RPC endpoints:

```bash
# List known blockchain configurations
evmarchaeology list chains

# List project configurations (NFT/token contracts)
evmarchaeology list projects

# List default RPC endpoints
evmarchaeology list rpcs
```

## Supported Chains

### For Extraction
- lux-mainnet (96369)
- lux-testnet (96368)
- zoo-mainnet (200200)
- spc-mainnet (36911)

### For Scanning
- ethereum
- bsc (Binance Smart Chain)
- polygon
- arbitrum
- optimism
- avalanche

## Project Configurations

### Lux
- **NFTs**: Ethereum contract `0x31e0f919c67cedd2bc3e294340dc900735810311`
- **Staking Powers**:
  - Validator NFT: 1M LUX
  - Card NFT: 500K LUX
  - Coin NFT: 100K LUX

### Zoo
- **Tokens**: BSC contract (add address)
- **NFTs**: BSC contracts (if any)
- **Staking Powers**:
  - Animal NFT: 1M ZOO
  - Habitat NFT: 750K ZOO
  - Item NFT: 250K ZOO

### SPC (Sparkle Pony Club)
- **Staking Powers**:
  - Pony NFT: 1M SPC
  - Accessory NFT: 500K SPC

### Hanzo
- **Staking Powers**:
  - AI NFT: 1M AI
  - Algorithm NFT: 750K AI
  - Data NFT: 500K AI

## Complete Workflow Example

```bash
# 1. Build the tool
make build-archaeology

# 2. Export 7777 accounts
make export-7777-accounts

# 3. Scan external NFTs
make scan-ethereum-nfts

# 4. Scan external tokens (add contract first)
# make scan-bsc-tokens

# 5. Generate complete X-Chain genesis
make generate-xchain-complete
```

## Output Formats

### NFT/Token CSV
```csv
address,asset_type,collection_type,balance_or_count,staking_power_wei,staking_power_token,chain_name,contract_address,project_name,last_activity_block,received_on_chain,token_ids
```

### X-Chain Genesis
```json
{
  "allocations": [
    {
      "assetAlias": "LUX_Validator_NFT",
      "assetID": "...",
      "initialState": {
        "nftMintOutput": [...]
      },
      "memo": "NFT Collection: lux Validator from 0x31e..."
    }
  ],
  "startTime": 1234567890,
  "message": "LUX Network X-Chain Genesis - Complete Historical Asset Integration"
}
```

## Development

### Adding New Chains

1. Add to `pkg/scanner/types.go`:
```go
var chainRPCs = map[string]string{
    "newchain": "https://rpc.newchain.com",
}
```

2. Add to `cmd/evmarchaeology/evmarchaeology/types.go`:
```go
"newchain-mainnet": {
    NetworkID:    "12345",
    ChainID:      12345,
    Name:         "New Chain",
    TokenSymbol:  "NEW",
},
```

### Adding New Projects

Update `pkg/scanner/types.go`:
```go
"newproject": {
    NFTContracts: map[string]string{
        "ethereum": "0x...",
    },
    StakingPowers: map[string]*big.Int{
        "Type1": new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
    },
},
```

## Notes

- Always use your own RPC endpoints for production scanning
- Cross-reference functionality requires access to existing chain data
- NFT staking enables validator participation without direct token holding
- Genesis generation creates time-locked vesting for large holders