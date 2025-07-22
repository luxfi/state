# Lux Network Genesis Tools

A comprehensive suite of tools for creating and managing genesis data for the Lux Network ecosystem, including complete historical asset integration from external chains.

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Tools Overview](#tools-overview)
  - [EVM Archaeology](#evm-archaeology)
  - [Genesis Builder](#genesis-builder)
  - [Scanner Package](#scanner-package)
  - [Lux CLI](#lux-cli)
- [Network Configuration](#network-configuration)
- [Complete Workflows](#complete-workflows)
- [Production Launch](#production-launch)
- [Development](#development)
- [Testing](#testing)
- [Contributing](#contributing)

## Overview

The Lux Network Genesis project provides a complete toolkit for:
- **Blockchain Data Extraction**: Extract and analyze data from various EVM chains
- **External Asset Scanning**: Find NFTs and tokens on Ethereum, BSC, and other chains
- **Genesis Generation**: Create complete X-Chain, P-Chain, and C-Chain genesis files
- **Cross-chain Migration**: Integrate historical assets into the new network
- **Network Launch**: Deploy L1 primary network and L2 subnets with full configurations

### Project History

- **2020**: Work to launch Lux Network began
- **2021-2022**: Private network launch and testing
- **2023**: Public beta launch with chain ID 7777
- **2024**: Mainnet upgrade to chain ID 96369
- **2025**: Public staking enabled, full decentralization

## Key Features

### 🔍 Blockchain Archaeology
- Extract data from LevelDB/PebbleDB databases
- Remove namespace prefixes from raw blockchain data
- Analyze blockchain databases without extraction
- Support for all Lux networks and subnets

### 🌉 External Asset Integration
Complete integration of assets from other chains:
- **Lux Genesis NFTs** from Ethereum (0x31e0f919c67cedd2bc3e294340dc900735810311)
- **Historic ZOO tokens** from BSC
- **Validator NFT staking** - NFTs that enable validator participation
- Cross-reference with existing chain data to avoid duplicates

### 📊 Multi-Network Support
- **L1 Primary**: Lux Network (96369/96368)
- **L2 Subnets**: ZOO (200200/200201), SPC (36911/36912), Hanzo (36963/36962)
- **External**: Ethereum, BSC, Polygon, Arbitrum, Optimism
- **Historical**: Chain ID 7777 preservation

### 🚀 Complete Launch System
- Generate genesis with validator configurations
- Import existing C-Chain data for continuity
- Deploy L1 and L2 networks with lux-cli
- Bootstrap node configuration management

## Quick Start

### Prerequisites

```bash
# Install Go 1.21.12 or later
wget https://go.dev/dl/go1.21.12.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.12.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Build core components (from /home/z/work/lux)
cd /home/z/work/lux
make build-node    # Build Lux Node (luxd)
make build-cli     # Build Lux CLI  (lux)
```

### Build Genesis Tools

```bash
# Build all tools
cd genesis
make build

# Or build individually
make build-archaeology      # EVM archaeology CLI
make build-genesis          # Genesis builder package
make build-tools            # Extraction utilities
```

### Quick Launch

```bash
# Launch mainnet with all L2s
make launch-mainnet

# Launch testnet
make launch-testnet

# Launch local development
make launch-local
```

## Architecture

### Directory Structure

```
genesis/
├── cmd/
│   ├── archeology/         # Main CLI tool for data extraction
│   │   ├── commands/       # Subcommands (extract, scan, genesis, etc.)
│   │   └── archeology/     # Core functionality
│   ├── genesis-builder/    # Genesis file generation tool
│   ├── denamespace/        # Database extraction tools
├── pkg/
│   ├── scanner/            # External asset scanning utilities
│   ├── genesis/            # Genesis generation package
│   │   ├── config/         # Network configurations
│   │   ├── address/        # Address conversion utilities
│   │   ├── allocation/     # Token allocation management
│   │   ├── cchain/         # C-Chain genesis builder
│   │   └── validator/      # Validator key management
│   └── blockchain/         # Blockchain interaction utilities
├── data/
│   ├── unified-genesis/    # Extracted network data
│   └── configs/           # Configuration files
├── scripts/               # Utility and launch scripts
├── configs/              # Network and validator configurations
├── exports/              # Scanned asset data (CSV)
└── docs/                # Documentation
```

### Network Architecture

#### L1 Primary Network
- **Network ID**: 96369 (mainnet), 96368 (testnet)
- **C-Chain**: Imported from existing 96369 data
- **Validators**: 11 bootstrap nodes
- **Consensus**: Snowman++ with production parameters

#### L2 Subnets
- **Zoo**: Chain ID 200200 (mainnet), 200201 (testnet)
- **SPC**: Chain ID 36911 (mainnet), 36912 (testnet)
- **Hanzo**: Chain ID 36963 (mainnet), 36962 (testnet) - prepared but not deployed

## Tools Overview

### EVM Archaeology

A comprehensive CLI for blockchain data extraction and analysis.

#### Installation
```bash
make build-archaeology
# Binary will be at ./bin/archaeology
```

#### Key Commands

##### Extract Blockchain Data
```bash
# Extract with state data
archaeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet \
  -all

# Extract specific accounts only
archaeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet \
  -state \
  -addresses 0x123...,0x456...
```

##### Import External Assets
```bash
# Import NFTs from Ethereum
archaeology import-nft \
  --network ethereum \
  --chain-id 1 \
  --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
  --project lux

# Import tokens from BSC
archaeology import-token \
  --network bsc \
  --chain-id 56 \
  --contract 0xYOUR_ZOO_TOKEN_ADDRESS \
  --project zoo \
  --symbol ZOO
```

##### Generate Genesis
```bash
archaeology genesis \
  --nft-csv exports/lux-nfts-ethereum.csv \
  --accounts-csv exports/7777-accounts.csv \
  --output configs/xchain-genesis-complete.json
```

### Genesis Builder

Modular Go package for generating genesis configurations.

#### Installation
```bash
make build-genesis-pkg
# Binary will be at ./cmd/genesis-builder/genesis-builder
```

#### Usage

##### As a Library
```go
import "github.com/luxfi/genesis/pkg/genesis"

// Create builder for mainnet
builder, err := genesis.NewBuilder("mainnet")

// Add allocations
builder.AddAllocation("0x...", big.NewInt(1000000000000))

// Import existing C-Chain data
builder.ImportCChainGenesis("/path/to/cchain-genesis.json")

// Build and save genesis
g, err := builder.Build()
builder.SaveToFile(g, "genesis.json")
```

##### Command Line
```bash
# Generate mainnet genesis with C-Chain import
./genesis-builder \
    --network mainnet \
    --import-cchain data/unified-genesis/lux-mainnet-96369/genesis.json \
    --import-allocations data/unified-genesis/lux-mainnet-96369/allocations_combined.json \
    --validators configs/mainnet-validators.json \
    --output genesis_mainnet.json

# List all networks
./genesis-builder -list-networks
```

### Scanner Package

Reusable blockchain scanning utilities for analyzing tokens, NFTs, and cross-chain balances.

#### Key Components

##### Token Burn Scanner
```go
config := &scanner.TokenBurnScanConfig{
    RPC:          "https://bsc-rpc-endpoint",
    TokenAddress: "0x0a6045...",
    BurnAddress:  scanner.DeadAddress,
}

scanner, err := scanner.NewTokenBurnScanner(config)
burns, err := scanner.ScanBurns()
```

##### NFT Holder Scanner
```go
config := &scanner.NFTHolderScanConfig{
    RPC:             "https://eth-rpc-endpoint",
    ContractAddress: "0x31e0f9...",
    IncludeTokenIDs: true,
}

scanner, err := scanner.NewNFTHolderScanner(config)
holders, err := scanner.ScanHolders()
```

### Lux CLI

Command line tool for subnet management and network deployment.

#### Installation
```bash
# From the cli directory
cd /home/z/work/lux/cli
go build -o bin/lux

# Or use the pre-built binary
./bin/lux --version
```

#### Key Features
- Creation of Lux EVM and custom VM subnet configurations
- Local deployment of subnets for development
- Mainnet and testnet deployment
- Validator management with flexible wallet configurations
- Network snapshot management

#### Basic Usage
```bash
# Create and deploy a subnet
lux subnet create mysubnet
lux subnet deploy mysubnet

# Network management
lux network start
lux network stop
lux network status

# Validator management
lux node validator add --name mainnet-0 --seed "your seed phrase"
lux node validator start --name mainnet-0
```

## Network Configuration

### Network IDs and Chain IDs

| Network | Chain ID | Network ID | Blockchain ID |
|---------|----------|------------|---------------|
| LUX Mainnet | 96369 | 96369 | dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ |
| LUX Testnet | 96368 | 96368 | 2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG |
| ZOO Mainnet | 200200 | 96369 | bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM |
| ZOO Testnet | 200201 | 96368 | 2usKC5aApgWQWwanB4LL6QPoqxR1bWWjPCtemBYbZvxkNfcnbj |
| SPC Mainnet | 36911 | 96369 | QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1 |
| Hanzo Mainnet | 36963 | 96369 | (To be deployed) |
| Hanzo Testnet | 36962 | 96368 | (To be deployed) |

### NFT Validator Staking

NFTs from external chains can be used as validators:

| NFT Type | Staking Power | Project |
|----------|---------------|---------|
| Lux Validator | 1M LUX | Lux |
| Lux Card | 500K LUX | Lux |
| Lux Coin | 100K LUX | Lux |
| Zoo Animal | 1M ZOO | Zoo |
| Zoo Habitat | 750K ZOO | Zoo |
| SPC Pony | 1M SPC | SPC |
| Hanzo AI | 1M AI | Hanzo |

### Important Addresses

- **Treasury**: `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
- **Test Account**: `0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC` (dev mode)

## Complete Workflows

### 1. Extract and Migrate 7777 Chain

```bash
# Extract 7777 accounts
make export-7777-accounts

# View migration documentation
cat docs/genesis-7777-migration/README.md
```

### 2. Scan External Assets

```bash
# Scan Ethereum for Lux NFTs
make scan-ethereum-nfts

# Scan BSC for Zoo tokens (add contract address first)
# make scan-bsc-tokens

# Generate complete X-Chain genesis
make generate-xchain-complete
```

### 3. Launch Complete Network

```bash
# Generate genesis with validators
cd genesis
make launch-mainnet

# This will:
# 1. Generate genesis with 11 validators
# 2. Import existing 96369 C-Chain data
# 3. Start primary network
# 4. Deploy Zoo L2 (200200)
# 5. Deploy SPC L2 (36911)
# 6. Prepare Hanzo L2 (36963)
```

### 4. Manual Network Launch

```bash
# Step 1: Generate genesis
cd genesis/cmd/genesis-builder
./genesis-builder \
    --network mainnet \
    --import-cchain ../data/unified-genesis/lux-mainnet-96369/genesis.json \
    --validators ../configs/mainnet-validators.json \
    --output ../genesis_mainnet.json

# Step 2: Launch with lux-cli
cd ../../../cli
./bin/lux network start \
    --luxgo-path ../node/build/luxd \
    --custom-network-genesis ../genesis/genesis_mainnet.json

# Step 3: Deploy L2 subnets
./bin/lux subnet create zoo --evm --chain-id 200200
./bin/lux subnet deploy zoo --mainnet
```

## Production Launch

### Bootstrap Nodes

The network uses 11 bootstrap validator nodes configured in:
- `configs/mainnet-validators.json` - Validator configurations
- `scripts/node-config-mainnet.json` - Node parameters

### RPC Endpoints

After launch:
- **C-Chain**: `http://localhost:9650/ext/bc/C/rpc`
- **Zoo L2**: `http://localhost:9650/ext/bc/{BLOCKCHAIN_ID}/rpc`
- **SPC L2**: `http://localhost:9650/ext/bc/{BLOCKCHAIN_ID}/rpc`

Get blockchain IDs:
```bash
./bin/lux subnet describe zoo --json | jq -r '.blockchainID'
./bin/lux subnet describe spc --json | jq -r '.blockchainID'
```

### Launch Checklist

See [docs/MAINNET_LAUNCH_CHECKLIST.md](docs/MAINNET_LAUNCH_CHECKLIST.md) for complete production launch procedures.

## Development

### Adding New Chains

1. Add RPC endpoint to `pkg/scanner/types.go`:
```go
var chainRPCs = map[string]string{
    "newchain": "https://rpc.newchain.com",
}
```

2. Add chain config to `cmd/archaeology/archaeology/types.go`:
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

### Database Key Prefixes

When working with raw blockchain data:
- `0x68`: headers
- `0x48`: hash->number mappings
- `0x62`: bodies
- `0x72`: receipts
- `0x26`: accounts
- `0xa3`: storage
- `0x73`: state

## Testing

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Test full workflow
make test-full-integration

# Test specific components
cd pkg/genesis && go test ./...
cd cmd/archaeology && go test ./...
```

See [docs/TESTING.md](docs/TESTING.md) for comprehensive testing guide.

## Troubleshooting

### Genesis Not Found
```bash
cd genesis
make build-genesis-pkg
./cmd/genesis/builder --network mainnet
```

### Network Won't Start
1. Check if ports 9650/9651 are free: `netstat -tulpn | grep 965`
2. Clean previous runs: `./bin/lux network clean`
3. Check logs: `./bin/lux network logs`

### L2 Deployment Fails
1. Ensure primary network is running: `./bin/lux network status`
2. Check validator set has enough stake
3. Verify genesis files exist in `data/unified-genesis/`

### Database Connection Errors
1. Ensure you have the correct database path
2. Check file permissions
3. Verify database isn't corrupted

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Ensure all tests pass (`make test`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Support

For questions and support:
- GitHub Issues: [Create an issue](https://github.com/luxfi/genesis/issues)
- Documentation: See `/docs` directory

## License

[License information]
