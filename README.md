# Lux Network Genesis Tools

Comprehensive toolset for generating genesis files and managing blockchain data for Lux Network L1 and L2 chains.

## Quick Start

```bash
# Install and build
make build
make install-plugin

# Deploy networks
make deploy          # Local 11-node network
make deploy-mainnet  # Mainnet with historical data
make deploy-testnet  # Testnet network
```

## Features

- **Genesis Generation**: Create complete genesis files for all networks
- **Blockchain Archaeology**: Extract and analyze historical blockchain data
- **CLI Plugins**: Extends lux-cli with genesis and archaeology functionality  
- **Multi-Chain Support**: Lux (96369), Zoo (200200), SPC (36911), and future L2s
- **Historical Import**: Prepare existing blockchain state for migration

## CLI Usage

After installing plugins with `make install-plugin`:

### Genesis Commands
```bash
lux-cli genesis generate --network mainnet
lux-cli genesis import historic --chain-data ./chaindata --network-id 96369
```

### Archaeology Commands  
```bash
lux-cli archaeology extract --source ./pebbledb --chain-id 96369
lux-cli archaeology scan-holders --contract 0x... --rpc https://...
lux-cli archaeology scan-burns --token 0x... --burn-address 0x000...dead
```

## Network Configuration

| Network | Chain ID | Type | Status |
|---------|----------|------|--------|
| Lux Mainnet | 96369 | L1 Primary | Active |
| Lux Testnet | 96368 | L1 Primary | Active |
| Zoo Mainnet | 200200 | L2 Subnet | Active |
| Zoo Testnet | 200201 | L2 Subnet | Active |
| SPC Mainnet | 36911 | L2 Subnet | Active |

## Project Structure

```
├── cmd/            # Command-line tools
├── pkg/            # Core packages
├── scripts/        # Deployment scripts
├── chaindata/      # Blockchain data (git-ignored)
├── configs/        # Network configurations
├── output/         # Generated genesis files
└── validator-keys/ # Validator keys (git-ignored)
```

## Key Tools

### genesis
Main tool for genesis file generation with modular architecture supporting P-Chain, C-Chain, and X-Chain.

### archaeology  
Blockchain data extraction and analysis tool for historical chain data migration.

### denamespace
Removes namespace prefixes from PebbleDB data for C-Chain compatibility.

## Development

```bash
# Build specific tools
make build-genesis
make build-archeology

# Run tests
make test-all

# Clean build artifacts
make clean
```

## Architecture

- **Plugin System**: Clean extension of lux-cli functionality
- **Modular Design**: Separate packages for genesis, archaeology, and utilities
- **Chain-Specific**: Dedicated modules for P/C/X chain handling
- **External Integration**: Import assets from Ethereum, BSC, and other chains

## Security

- Validator keys generated deterministically from mnemonic
- Never commit real keys or mnemonics
- Use separate mnemonics for different networks
- Keys stored in git-ignored directories

## License

MIT License - see LICENSE file