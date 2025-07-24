# Lux Network Summary

## Primary Network (LUX)

### Mainnet
- **Network ID**: 96369
- **Chain ID**: 96369 (C-Chain)
- **Validators**: 11 nodes
- **Treasury**: 2T LUX tokens
- **Start Time**: 2020-01-01 00:00:00 UTC
- **Consensus**: Snowball (PoS)

### Testnet
- **Network ID**: 96368
- **Chain ID**: 96368 (C-Chain)
- **Validators**: 11 nodes (same set as mainnet)
- **Treasury**: 2T LUX tokens
- **Start Time**: 2025-07-24
- **Purpose**: Public testing environment

### Local
- **Network ID**: 12345
- **Chain ID**: 12345 (C-Chain)
- **Validators**: 5 nodes
- **Treasury**: 1000 LUX tokens
- **Purpose**: Local development

## L2 Networks

### ZOO Network
**Token**: ZOO
**Type**: EVM-compatible L2

| Environment | Chain ID | Treasury Balance | Migration Status |
|-------------|----------|------------------|------------------|
| Mainnet | 200200 | 2T ZOO | BSC migration data included |
| Testnet | 200201 | 2T ZOO | Fresh deployment |
| Local | 200202 | 1000 ZOO | Development only |

### SPC Network
**Token**: SPC (Sparkle Pony Coin)
**Type**: EVM-compatible L2

| Environment | Chain ID | Treasury Balance | Special Address |
|-------------|----------|------------------|-----------------|
| Mainnet | 36911 | 2T LUX | 0x12c6EE1d...e8597e (10M SPC) |
| Testnet | 36912 | 2T LUX | Treasury only |
| Local | 36913 | 1000 LUX + 10K SPC | Development allocations |

### Hanzo Network
**Token**: AI
**Type**: EVM-compatible L2
**Status**: Fresh deployment (no historical data)

| Environment | Chain ID | Treasury Balance | Notes |
|-------------|----------|------------------|-------|
| Mainnet | 36963 | 2T LUX | AI token deployment pending |
| Testnet | 36962 | 2T LUX | Testing environment |
| Local | 36964 | 1000 LUX | Development only |

## Key Addresses

### Treasury
- **Address**: `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
- **Purpose**: Network treasury and initial token distribution
- **Holdings**: 2T tokens on each mainnet/testnet, 1000 on local

### Development Accounts (Local Only)
- **Account 1**: `0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC`
  - 1M LUX on local networks
  - Test account for development

## Validator Summary

### Mainnet/Testnet Validators (11 nodes)
All validators have:
- Equal staking weight: 1,000,000
- Delegation fee: 2% (20000)
- BLS signatures enabled
- Full validator configurations in `validators.json`

Example validator:
```
NodeID: NodeID-3fyVgFRSVC4Ma7twsmghQatodnzGPvXgN
Reward Address: 0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC
```

### Local Validators (5 nodes)
Simplified set for development:
- Equal staking weight
- Ephemeral keys (regenerated each time)
- No delegation fees

## Network Features

### All Networks Include:
- ✅ EVM compatibility
- ✅ Dynamic fee configuration
- ✅ All Ethereum hard forks activated
- ✅ Subnet EVM features
- ✅ 2-second target block time

### Mainnet/Testnet Additional Features:
- ✅ Warp messaging
- ✅ Full validator set
- ✅ Production fee configuration

### Local Network Features:
- ✅ Low minimum base fee (1 gwei)
- ✅ Fast block production
- ✅ Pre-funded development accounts

## Migration Notes

1. **ZOO Network**: Mainnet includes BSC token migration allocations
2. **SPC Network**: Has existing chain data to import
3. **Hanzo Network**: Fresh deployment, no migration needed
4. **LUX Primary**: Can import from existing 96369 chain data

## Quick Reference

### RPC Endpoints (when running)
- Primary Network: `http://localhost:9630/ext/bc/C/rpc`
- L2 Networks: `http://localhost:9630/ext/bc/{blockchainID}/rpc`

### Chain Aliases
- C-Chain: Contract Chain (EVM)
- P-Chain: Platform Chain (Staking)
- X-Chain: Exchange Chain (DAG)