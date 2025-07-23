# Usage Guide

The Makefile supports dynamic network targeting using the `NETWORK` environment variable.

## Quick Examples

### Run complete pipeline for any network:
```bash
make pipeline NETWORK=zoo    # ZOO with BSC migration
make pipeline NETWORK=lux    # LUX with Ethereum NFTs
make pipeline NETWORK=spc    # SPC bootstrap
```

### Individual commands with NETWORK:
```bash
# Extract blockchain data
make extract NETWORK=zoo

# Scan external chains
make scan NETWORK=zoo     # Scans BSC for burns + eggs
make scan NETWORK=lux     # Scans ETH for NFTs
make scan NETWORK=spc     # No external scan needed

# Analyze token distribution
make analyze NETWORK=zoo

# Build genesis
make genesis NETWORK=zoo

# Deploy network
make deploy NETWORK=zoo
```

## Network-Specific Details

### LUX Network
- **Migration**: Ethereum NFT holders get validator rights
- **NFT Contract**: `0x31e0f919c67cedd2bc3e294340dc900735810311`
- **Pipeline steps**:
  1. Extract LUX chain data
  2. Scan Ethereum for NFT holders
  3. Build genesis with NFT allocations

### ZOO Network
- **Migration**: BSC token burns + egg NFTs
- **Token**: `0x0a6045b79151d0a54dbd5227082445750a023af2`
- **Egg NFT**: `0x5bb68cf06289d54efde25155c88003be685356a8`
- **Rules**: Burns = 1:1 credit, Each egg = 4.2M ZOO
- **Pipeline steps**:
  1. Extract ZOO chain data
  2. Scan BSC for burns and egg holders
  3. Analyze distribution
  4. Build genesis with migration data

### SPC Network
- **No migration** - purely Lux-native
- **Bootstrap**: 10M initial supply
- **Pipeline steps**:
  1. Extract SPC chain data (if exists)
  2. Analyze distribution
  3. Build bootstrap genesis

## Advanced Usage

### Custom RPC endpoints:
```bash
BSC_RPC=https://my-bsc-node.com make scan NETWORK=zoo
ETH_RPC=https://my-eth-node.com make scan NETWORK=lux
```

### Custom output directory:
```bash
OUTPUT_DIR=/tmp/genesis make pipeline NETWORK=zoo
```

### Verbose output:
```bash
VERBOSE=1 make pipeline NETWORK=zoo
```

### Specific chain ID override:
```bash
CHAIN_ID=12345 make genesis NETWORK=zoo
```

## Complete Workflow Example

```bash
# 1. Clean previous outputs
make clean

# 2. Build tools if needed
make build-tools

# 3. Run full ZOO migration
make pipeline NETWORK=zoo

# 4. Validate the genesis
make validate

# 5. Deploy the network
make deploy NETWORK=zoo
```

## Parallel Processing

Run multiple networks in parallel:
```bash
make pipeline NETWORK=lux &
make pipeline NETWORK=zoo &
make pipeline NETWORK=spc &
wait
```

## Troubleshooting

### If BSC scan times out:
```bash
# Use a dedicated RPC endpoint
BSC_RPC=https://bsc-dataseed1.defibit.io/ make scan NETWORK=zoo

# Or scan with specific block range
make scan-burns CHAIN=bsc TOKEN=0x0a6... FROM_BLOCK=20000000 TO_BLOCK=21000000
```

### If extraction fails:
```bash
# Check data directory exists
ls -la chaindata/zoo-mainnet/200200/

# Try with explicit path
DATA_DIR=/path/to/chaindata make extract NETWORK=zoo
```

### View all available targets:

```bash
make help
```

## Auditing & Verification

For auditors and community members, see the [Verification Guide](VERIFICATION.md) for step-by-step instructions on verifying treasury balances and allocations.
