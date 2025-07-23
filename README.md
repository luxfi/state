# Lux Network Genesis Management System

A unified, DRY (Don't Repeat Yourself) system for managing Lux Network genesis configurations, validators, and network launches.

## Quick Start

```bash
# Install dependencies
make install

# Generate 11 validators
MNEMONIC="your twelve word mnemonic phrase" make validators-generate

# Generate genesis configuration
make genesis-generate

# Launch mainnet
make launch-mainnet

# Or use the unified launcher
./scripts/launch.sh full
```

## Features

### Unified CLI Tool
- **Single Command Interface**: All genesis operations through `genesis`
- **Validator Management**: Generate, add, remove, and list validators
- **Genesis Generation**: Create genesis for mainnet, testnet, or local networks
- **Import Support**: Import C-Chain data and CSV allocations
- **Validation**: Comprehensive genesis validation

### Simplified Operations
- **DRY Makefile**: No redundant targets, clear and concise
- **Unified Launch Script**: Single script for all launch modes
- **Human-Readable Amounts**: Support for 2T, 1.5B, 500M notation
- **Comprehensive Testing**: Full test coverage with `./test-all.sh`

## Usage

### Command Line Interface

```bash
# Generate genesis with validators
./bin/genesis generate \
  --network mainnet \
  --validators configs/mainnet/validators.json \
  --treasury-amount 2T

# Manage validators
./bin/genesis validators list
./bin/genesis validators generate --mnemonic "..." --offsets "0,1,2,3,4,5"
./bin/genesis validators add --node-id NodeID-xxx --eth-address 0x...
./bin/genesis validators remove --index 5

# Validate genesis
./bin/genesis validate --network mainnet

# List all available commands
./bin/genesis tools
```

### Makefile Commands

```bash
make help                 # Show all available commands
make validators-generate  # Generate validators
make genesis-generate     # Generate genesis file
make launch-dev          # Single node dev mode
make launch-mainnet      # Full 11-node network
make test                # Run all tests
```

## Network Configuration

| Network | Chain ID | Type | Status |
|---------|----------|------|--------|
| Lux Mainnet | 96369 | L1 Primary | Active |
| Lux Testnet | 96368 | L1 Primary | Active |
| Zoo Mainnet | 200200 | L2 Subnet | Active |
| Zoo Testnet | 200201 | L2 Subnet | Active |
| SPC Mainnet | 36911 | L2 Subnet | Active |

## Documentation

- **[Genesis Tools Guide](docs/GENESIS_TOOLS_GUIDE.md)** - Comprehensive guide for all genesis generation and management tools
- **Primary Genesis Tool**: Run `./bin/genesis` to generate all chain genesis files
- **[General Documentation](docs/README.md)** - Detailed guides and reference documentation for the Lux Network genesis transition

## Project Structure

```
genesis/
├── bin/                    # Built binaries
│   ├── genesis-cli        # Unified CLI tool
│   ├── luxd              # Lux node binary
│   └── lux-cli           # Lux CLI
├── cmd/                   # Command source code
│   └── genesis-cli/       # Genesis CLI implementation
├── configs/               # Configuration files
│   └── *-validators.json  # Validator configurations
├── pkg/                   # Go packages
│   └── genesis/          # Core genesis logic
│       ├── allocation/   # Allocation management
│       ├── config/       # Network configurations
│       └── validator/    # Validator key generation
├── scripts/              # Shell scripts
│   └── launch.sh         # Unified launch script
├── validator-keys/       # Generated validator keys
├── Makefile             # Simplified, DRY Makefile
└── test-all.sh          # Comprehensive test suite
```

## Advanced Usage

### Import Operations

```bash
# Import existing C-Chain genesis
IMPORT_CCHAIN=path/to/cchain-genesis.json make genesis-generate

# Import allocations from CSV
IMPORT_ALLOCS=path/to/allocations.csv make genesis-generate
```

### Custom Treasury

```bash
# Set custom treasury amount
TREASURY_AMOUNT=1.5B make genesis-generate

# Set custom treasury address and amount
TREASURY=0x... TREASURY_AMOUNT=2T make genesis-generate
```

### Launch Modes

```bash
# Dev mode (single node, minimal consensus)
./scripts/launch.sh dev

# POA mode (single node, auto-mining)
./scripts/launch.sh poa

# Full network (11 nodes by default)
./scripts/launch.sh full

# Custom number of nodes
NODES=5 ./scripts/launch.sh full
```

## Testing

The system includes comprehensive test coverage:

```bash
# Run all tests
make test

# Run comprehensive test suite
./test-all.sh

# Run specific test packages
go test ./pkg/genesis/allocation/... -v
go test ./pkg/genesis/validator/... -v
go test ./pkg/genesis/... -v
```

## Troubleshooting

### Genesis Generation Issues
- Ensure dependencies are installed: `make install`
- Check validator file exists: `ls configs/*-validators.json`
- Validate network parameter: mainnet, testnet, or local

### Network Launch Issues
- Check port availability: `netstat -tlnp | grep 9630`
- Ensure genesis file exists: `ls genesis_*.json`
- Verify luxd is built: `ls bin/luxd`

### Validator Problems
- For deterministic generation, ensure `MNEMONIC` is set
- Check validator keys: `ls validator-keys/`
- Validate BLS key format (48 bytes hex)

## Security

- Validator keys generated deterministically from mnemonic
- Never commit real keys or mnemonics
- Use separate mnemonics for different networks
- Keys stored in git-ignored directories

## License

MIT License - see LICENSE file
