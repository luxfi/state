# Lux Genesis - Subnet to C-Chain Migration

Unified tool for migrating Lux subnet data to C-Chain.

## Quick Start

```bash
# Clone and run full migration test
git clone https://github.com/luxfi/genesis
cd genesis
make
```

This will:
1. Install luxd from genesis branch
2. Build the genesis migration tool
3. Import subnet 96369 as C-Chain
4. Launch a test node
5. Verify via RPC

## Production Migration

### Step 1: Import Subnet Data
```bash
make import
```

### Step 2: Launch Node
```bash
make node
```

**Important**: To enable C-Chain indexing for historical logs and `eth_getLogs` queries, add the following flag when launching luxd:
```bash
--chain-configs.enable-indexing
```

### Step 3: Verify
```bash
# Check block height (should be 0x10859c = 1082780)
curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc | jq .result

# Or use the convenience command:
make check-height
```

## Quick Verification

After starting the container:

```bash
# Check current block height
make check-height

# Check president's balance  
make check-balance

# Monitor blockchain activity
make monitor

# Open interactive console
make console
```

## Available Commands

- `make` - Run full end-to-end test (default)
- `make deps` - Install dependencies (luxd)
- `make build` - Build genesis tool only
- `make import` - Import subnet data to C-Chain
- `make node` - Run luxd with imported data
- `make test` - Run unit tests
- `make clean` - Clean build artifacts

## Migration Details

The genesis tool performs these steps:

1. **De-namespace**: Remove 33-byte subnet prefix from keys
2. **Add EVM prefix**: Convert to C-Chain key format
3. **Rebuild mappings**: Fix canonical block mappings
4. **Replay consensus**: Build state database

## Network Information

- Chain ID: 96369
- Network: Lux Mainnet
- Import source: `chaindata/lux-mainnet-96369/db/pebbledb`
- Treasury: `0x9011e888251ab053b7bd1cdb598db4f9ded94714`

## Documentation

- [LLM.md](LLM.md) - Detailed technical guide for AI assistants
- [docs/](docs/) - Additional documentation

## License

[LICENSE](LICENSE)