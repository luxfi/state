# Lux Network Genesis Tools

This repository contains tools for creating and managing genesis data for the Lux Network ecosystem, including complete historical asset integration from external chains.

## Overview

The Lux Network Genesis project provides:
- **EVM Archaeology**: Extract and analyze blockchain data from various EVM chains
- **External Asset Scanning**: Find NFTs and tokens on Ethereum, BSC, and other chains
- **Genesis Generation**: Create complete X-Chain and P-Chain genesis files
- **Cross-chain Migration**: Integrate historical assets into the new network

## Key Features

### 🔍 Blockchain Archaeology Tool
A comprehensive CLI for blockchain data extraction and analysis:
- Extract data from LevelDB/PebbleDB databases
- Scan external chains for NFTs and tokens
- Generate genesis files with complete historical data
- Analyze blockchain databases

### 🌉 External Asset Integration
Complete integration of assets from other chains:
- **Lux Genesis NFTs** from Ethereum (0x31e0f919c67cedd2bc3e294340dc900735810311)
- **Historic ZOO tokens** from BSC
- **Validator NFT staking** - NFTs that enable validator participation
- Cross-reference with existing chain data to avoid duplicates

### 📊 Network Support
- **Primary**: Lux Network (96369/96368)
- **L2s**: ZOO (200200/200201), SPC (36911/36912), Hanzo (36963/36962)
- **External**: Ethereum, BSC, Polygon, Arbitrum, Optimism

## Quick Start

### Build Tools
```bash
# Build all tools
make build

# Build specific tools
make build-archaeology  # EVM archaeology CLI
make build-tools       # Extraction utilities
```

### Scan External Assets
```bash
# Scan Ethereum for Lux NFTs
make scan-ethereum-nfts

# Scan BSC for Zoo tokens (add contract address first)
# make scan-bsc-tokens
```

### Generate Complete Genesis
```bash
# Export 7777 accounts
make export-7777-accounts

# Generate complete X-Chain genesis with all assets
make generate-xchain-complete
```

## Project Structure

```
genesis/
├── cmd/
│   ├── archeology/         # Main CLI tool
│   │   ├── commands/       # Subcommands (extract, scan, genesis, etc.)
│   │   └── archeology/     # Core functionality
│   ├── denamespace*/       # Database extraction tools
│   └── ...
├── pkg/
│   ├── scanner/            # External asset scanning
│   └── genesis/            # Genesis file generation
├── scripts/                # Utility scripts
├── configs/                # Network configurations
├── exports/                # Scanned asset data (CSV)
└── chaindata/             # Blockchain data
```

## Blockchain Archaeology Usage

### Extract Blockchain Data
```bash
archeology extract \
  -src /path/to/source/db \
  -dst /path/to/clean/db \
  -network lux-mainnet
```

### Import NFTs from Any Chain
```bash
# Import from Ethereum
archeology import-nft \
  --network ethereum \
  --chain-id 1 \
  --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
  --project lux

# Import from BSC
archeology import-nft \
  --network bsc \
  --chain-id 56 \
  --contract 0xYOUR_CONTRACT_ADDRESS \
  --project zoo

# Import with custom RPC
archeology import-nft \
  --rpc https://your-rpc-endpoint.com \
  --chain-id 137 \
  --contract 0xYOUR_CONTRACT_ADDRESS \
  --project custom
```

### Import ERC20 Tokens from Any Chain
```bash
# Import Zoo tokens from BSC
archeology import-token \
  --network bsc \
  --chain-id 56 \
  --contract 0xYOUR_ZOO_TOKEN_ADDRESS \
  --project zoo \
  --symbol ZOO

# Import tokens from local 7777 chain
archeology import-token \
  --rpc http://localhost:9650/ext/bc/C/rpc \
  --chain-id 7777 \
  --contract 0xTOKEN_ADDRESS \
  --project lux \
  --symbol LUX

# Import USDC from Ethereum
archeology import-token \
  --network ethereum \
  --chain-id 1 \
  --contract 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
  --project usdc \
  --symbol USDC \
  --decimals 6
```

### Scan External NFTs/Tokens (Legacy)
```bash
archeology scan \
  --chain ethereum \
  --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
  --project lux \
  --type nft
```

### Generate Genesis
```bash
archeology genesis \
  --nft-csv exports/lux-nfts-ethereum.csv \
  --accounts-csv exports/7777-accounts.csv \
  --output configs/xchain-genesis-complete.json
```

## NFT Validator Staking

NFTs from external chains can be used as validators in the Lux Network:

| NFT Type | Staking Power | Project |
|----------|---------------|---------|
| Lux Validator | 1M LUX | Lux |
| Lux Card | 500K LUX | Lux |
| Lux Coin | 100K LUX | Lux |
| Zoo Animal | 1M ZOO | Zoo |
| Zoo Habitat | 750K ZOO | Zoo |
| SPC Pony | 1M SPC | SPC |
| Hanzo AI | 1M AI | Hanzo |

## Project History

- **2020**: Work to launch Lux Network began
- **2021-2022**: Private network launch and testing
- **2023**: Public beta launch with chain ID 7777
- **2024**: Mainnet upgrade to chain ID 96369
- **2025**: Public staking enabled, full decentralization

This genesis represents the culmination of years of development, preserving all historical data while enabling a new era in decentralized finance.

## Development

### Adding New Chains
1. Add RPC endpoint to `pkg/scanner/types.go`
2. Add chain config to `cmd/evmarchaeology/evmarchaeology/types.go`
3. Update scanner to support new chain specifics

### Adding New Projects
Update `pkg/scanner/types.go` with:
- Contract addresses
- NFT types and staking powers
- Type identifiers for classification

## Testing

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Test full workflow
make test-full-integration
```

## Contributing

1. Fork the repository
2. Create your feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

[License information]

## Support

For questions and support:
- GitHub Issues: [Create an issue](https://github.com/luxfi/genesis/issues)
- Documentation: See `/docs` directory