# LLM.md - AI Assistant Guidance

This file provides context and guidance for AI assistants working with the Lux Network Genesis migration tool.

## Quick Start

```bash
# Run complete migration test
make

# Step by step
make deps      # Install luxd
make build     # Build genesis tool
make import    # Import subnet data
make node      # Launch node
```

## Migration Flow

The tool migrates subnet 96369 to C-Chain through these steps:

1. **De-namespace**: Strip 33-byte prefix from subnet keys
2. **Add EVM prefix**: Convert keys to C-Chain format (evmh, evmb, evmr, evmn)
3. **Rebuild canonical**: Fix truncated evmn mappings
4. **Replay consensus**: Build state database with Snowman consensus

## Key Information

- **Chain ID**: 96369 (Lux Mainnet)
- **Source**: `chaindata/lux-mainnet-96369/db/pebbledb`
- **Runtime**: `runtime/` (local to repo)
- **Treasury**: `0x9011e888251ab053b7bd1cdb598db4f9ded94714`
- **Expected Balance**: ~1.995T LUX (from 2T initial)

## Tool Commands

### Primary Tool: genesis

```bash
# Migration commands
./bin/genesis migrate add-evm-prefix --src <src> --dst <dst>
./bin/genesis migrate rebuild-canonical --db <db>
./bin/genesis migrate peek-tip --db <db>
./bin/genesis migrate replay-consensus --evm <evm> --state <state> --tip <tip>

# Import shortcuts
./bin/genesis import subnet <src> <dst>    # Full subnet import

# Analysis
./bin/genesis analyze                      # Analyze blockchain data
./bin/genesis inspect tip                  # Find chain tip

# Launch
./bin/genesis launch L1                    # Launch as C-Chain
```

## Makefile Targets

- `make` - Full test: deps → build → import → launch → verify
- `make deps` - Install luxd from genesis branch
- `make build` - Build genesis tool
- `make import` - Import subnet 96369 as C-Chain
- `make node` - Launch luxd with imported data
- `make test` - Run unit tests
- `make quality` - Run code quality checks

## Directory Structure

```
genesis/
├── chaindata/         # Source blockchain data
│   └── lux-mainnet-96369/db/pebbledb/
├── configs/           # Network configurations
├── runtime/           # Imported/processed data
│   ├── evm/          # Migrated EVM data
│   └── state/        # Consensus state
├── bin/              # Built binaries
│   ├── genesis       # Migration tool
│   └── luxd          # Node binary
└── Makefile          # Build system
```

## RPC Verification

After launch, verify with:

```bash
# Check block height
curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc | jq .result

# Check treasury balance
curl -s --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011e888251ab053b7bd1cdb598db4f9ded94714","latest"],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc | jq .result
```

## Production Deployment

1. **Import**: `make import` (may take time for 4M+ keys)
2. **Deploy**: Copy `runtime/` to each validator
3. **Launch**: Start all validators with same data
4. **Monitor**: Watch for 2/3+ stake online

## Key Technical Details

- Subnet keys have 33-byte namespace prefix
- C-Chain uses "evm" prefix for all keys
- evmn keys must be 12-byte format (8-byte height)
- State DB requires versiondb wrapping

## Troubleshooting

- **Port conflict**: Use `--http-port 9630`
- **Genesis mismatch**: Delete ChainConfigKey
- **Import fails**: Check disk space (need ~50GB)
- **RPC down**: Wait for bootstrapping to complete