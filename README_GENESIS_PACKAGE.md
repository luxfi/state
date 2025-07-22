# Lux Genesis Package

A modular Go package for generating genesis configurations for Lux Network and its L2 subnets.

## Features

- **Multi-network support**: Generate genesis for mainnet, testnet, and all L2 subnets
- **C-Chain data import**: Import existing C-Chain genesis and allocations 
- **Address conversion**: Convert between Ethereum and Lux address formats
- **Allocation management**: Handle simple and vested token allocations
- **Validator configuration**: Configure initial validator sets
- **L1/L2 hierarchy**: Proper parent-child relationships for subnets

## Package Structure

```
pkg/genesis/
├── config/       # Network configurations
├── address/      # Address conversion utilities  
├── allocation/   # Token allocation management
├── cchain/       # C-Chain genesis builder
├── types.go      # Core types
└── builder.go    # Main genesis builder
```

## Usage

### As a Library

```go
import "github.com/luxfi/genesis/pkg/genesis"

// Create builder for mainnet
builder, err := genesis.NewBuilder("mainnet")

// Add allocations
builder.AddAllocation("0x...", big.NewInt(1000000000000))

// Import existing C-Chain data
builder.ImportCChainGenesis("/path/to/cchain-genesis.json")

// Build genesis
g, err := builder.Build()

// Save to file
builder.SaveToFile(g, "genesis.json")
```

### Command Line Tool

```bash
# Build the tool
cd cmd/genesis-builder
go build

# Generate mainnet genesis
./genesis-builder -network mainnet

# Import existing C-Chain data
./genesis-builder -network mainnet \
  -import-cchain /path/to/96369-genesis.json \
  -import-allocations /path/to/allocations.json

# Generate L2 subnet genesis
./genesis-builder -network zoo-mainnet

# List all networks
./genesis-builder -list-networks
```

## Network Configuration

### L1 Networks (Primary)
- **mainnet**: Lux Mainnet (Chain ID: 96369)
- **testnet**: Lux Testnet (Chain ID: 96368)
- **local**: Local development (Chain ID: 12345)

### L2 Networks (Subnets)
- **zoo-mainnet**: Zoo L2 on mainnet (Chain ID: 200200)
- **zoo-testnet**: Zoo L2 on testnet (Chain ID: 200201)
- **spc-mainnet**: SPC L2 on mainnet (Chain ID: 36911)
- **hanzo-mainnet**: Hanzo L2 on mainnet (Chain ID: 36963)
- **hanzo-testnet**: Hanzo L2 on testnet (Chain ID: 36962)

## Genesis Structure

The generated genesis includes:
- Network ID and chain configuration
- X-Chain allocations (token distribution)
- P-Chain validators (initial staker set)
- C-Chain genesis (EVM configuration)

## Important Notes

1. **C-Chain Continuity**: For mainnet, import the existing 96369 C-Chain genesis to maintain chain data continuity
2. **Treasury Address**: Default treasury is `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
3. **Token Decimals**: LUX uses 9 decimals on X/P chains, 18 on C-Chain
4. **L2 Validators**: L2 subnet validators must also be validators on the parent L1

## Development

```bash
# Run tests
go test ./...

# Update dependencies
go mod tidy

# Build all
make build
```