# Subnet to C-Chain Migration

This document describes the process of migrating a Subnet EVM blockchain (like the original chain 96369) to become the C-Chain of Lux Network.

## Background

The Lux Network originally ran chain 96369 as a subnet. To make it the primary C-Chain, we need to:
1. Migrate the blockchain data from subnet format to C-Chain format
2. Generate the missing Snowman consensus metadata
3. Configure the node to recognize the migrated data

## Current Status

- ✅ Tools created for analyzing and migrating subnet data
- ✅ Import path conflicts resolved (ethereum/go-ethereum → luxfi/geth)
- ✅ Database key structure understood and documented
- 🚧 Consensus replay implementation in progress
- ⏳ Full integration with luxd pending

## Quick Start

```bash
# 1. Build the tools
make -C /home/z/work/lux/genesis all

# 2. Analyze your subnet database
./bin/analyze-subnet-blocks /path/to/subnet/db

# 3. Migrate the data
./bin/migrate-subnet-tool \
  --source /path/to/subnet/db \
  --target /tmp/migrated-cchain \
  --verbose

# 4. Generate consensus state
./bin/replay-consensus-pebble \
  --evm /tmp/migrated-cchain \
  --state /tmp/consensus-state \
  --tip <highest-block-number>

# 5. Launch the node
cd /home/z/work/lux/node
./build/luxd --data-dir=/tmp/migrated-data ...
```

## Technical Details

### Subnet Database Structure
- Uses PebbleDB with 33-byte namespace prefixes
- Contains state trie nodes (accounts, storage)
- May or may not include blockchain headers/bodies

### C-Chain Requirements
- Standard Ethereum database format with "evm" prefix
- Snowman consensus metadata (block IDs, acceptance status)
- VersionDB wrapper for MVCC support

### Key Challenges
1. **Missing Blockchain Data**: Some subnet databases only have state, not headers/bodies
2. **Consensus Metadata**: Snowman engine needs its own block tracking
3. **VersionDB Layer**: Direct database writes don't include required metadata

## Tools Overview

| Tool | Purpose |
|------|---------|
| `analyze-subnet-blocks` | Analyze subnet database structure |
| `add-evm-prefix-to-blocks` | Add C-Chain compatible prefixes |
| `replay-consensus-pebble` | Generate Snowman consensus state |
| `check-head-pointers` | Verify migration success |
| `migrate-subnet-tool` | Complete migration pipeline |

## Next Steps

1. Complete consensus replay implementation
2. Test with real subnet data
3. Integrate with luxd initialization
4. Document production migration process

## Contributing

To work on this migration:

1. Ensure you have Go 1.24.5 installed
2. Clone both `/home/z/work/lux/genesis` and `/home/z/work/lux/node`
3. Run tests: `go test ./...`
4. Submit improvements via PR

## References

- [Migration Guide](docs/MIGRATION_GUIDE.md)
- [Lux Node Documentation](../node/README.md)
- [Genesis Tools Guide](docs/GENESIS_TOOLS_GUIDE.md)