# Lux Network Deployments

This directory contains all genesis configurations for the Lux Network ecosystem, organized by environment and network.

## Directory Structure

```
deployments/
└── configs/
    ├── mainnet/      # Production mainnet configurations
    ├── testnet/      # Public testnet configurations
    └── local/        # Local development configurations
```

## Networks

### Primary Network (LUX)

The Lux primary network hosts the P-Chain, C-Chain, and X-Chain.

| Environment | Network ID | Chain ID | Validators |
|-------------|------------|----------|------------|
| Mainnet     | 96369      | 96369    | 11         |
| Testnet     | 96368      | 96368    | 11         |
| Local       | 12345      | 12345    | 5          |

### L2 Networks

#### ZOO Network
| Environment | Network ID | Chain ID | Token |
|-------------|------------|----------|-------|
| Mainnet     | 200200     | 200200   | ZOO   |
| Testnet     | 200201     | 200201   | ZOO   |
| Local       | 200202     | 200202   | ZOO   |

#### SPC Network
| Environment | Network ID | Chain ID | Token |
|-------------|------------|----------|-------|
| Mainnet     | 36911      | 36911    | SPC   |
| Testnet     | 36912      | 36912    | SPC   |
| Local       | 36913      | 36913    | SPC   |

#### Hanzo Network
| Environment | Network ID | Chain ID | Token |
|-------------|------------|----------|-------|
| Mainnet     | 36963      | 36963    | AI    |
| Testnet     | 36962      | 36962    | AI    |
| Local       | 36964      | 36964    | AI    |

## Configuration Files

Each network directory contains:
- `genesis.json` - The genesis configuration file
- `validators.json` - Validator configurations (for primary networks)

## Treasury Address

All networks include the treasury address with initial allocation:
- Address: `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
- Mainnet/Testnet: 2,000,000,000,000 tokens
- Local: 1,000 tokens

## Usage

### Launching with luxd

```bash
# Mainnet
luxd --network-id=96369 --genesis-file=deployments/configs/mainnet/lux/genesis.json

# Testnet
luxd --network-id=96368 --genesis-file=deployments/configs/testnet/lux/genesis.json

# Local
luxd --network-id=12345 --genesis-file=deployments/configs/local/lux/genesis.json
```

### Creating L2s with lux-cli

```bash
# ZOO L2 on mainnet
lux blockchain create zoo \
  --evm \
  --genesis-file deployments/configs/mainnet/zoo/genesis.json

# Deploy to network
lux blockchain deploy zoo --avalanchego-version latest
```

## Validator Information

### Mainnet Validators (11 nodes)
See `deployments/configs/mainnet/lux/validators.json` for full details including:
- Node IDs
- ETH addresses for rewards
- BLS public keys and proofs of possession
- Staking weights

### Testnet Validators
Uses the same validator set as mainnet for consistency.

### Local Validators (5 nodes)
Simplified validator set for local development. See `deployments/configs/local/lux/validators.json`.

## Network Features

All networks include:
- EVM compatibility (Berlin, London hard forks activated)
- Subnet EVM features
- Dynamic fees configuration
- Warp messaging support (mainnet/testnet)

## Important Notes

1. **Chain Data Import**: For ZOO and SPC networks on mainnet/testnet, you should import existing chain data to preserve historical state.

2. **Hanzo Network**: Fresh deployment on all environments (no historical data to import).

3. **Local Development**: Uses different chain IDs to avoid conflicts with public networks.

4. **Validator Keys**: Store validator private keys securely. The public configurations in `validators.json` files do not contain private keys.

## Archival

This structure is designed for long-term archival:
- Clear separation between environments
- Consistent naming conventions
- All configurations in one place
- Version control friendly
- Easy to backup and restore