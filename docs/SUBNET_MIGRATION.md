# Subnet to C-Chain Migration Guide

This document describes the process of migrating Subnet EVM data to the C-Chain format for Lux Network.

## Overview

Subnet EVM chains store their data with a 32-byte namespace prefix to allow multiple chains to coexist in the same database. When migrating to C-Chain, we need to:

1. Extract the namespaced data (removing the prefix)
2. Copy all blockchain data to the new location
3. Set proper chain continuity markers

## Key Discoveries

### Database Structure
- **Subnet databases**: Use 33-byte keys (32-byte namespace + 1-byte prefix)
- **Namespace format**: Fixed 32-byte prefix for all keys in a subnet
- **Chain ID encoding**: The namespace is derived from the chain ID

### Migration Process

The unified genesis tool handles the entire migration automatically:

```bash
./bin/genesis import subnet <source-db> <destination-db>
```

This command will:
1. Detect if the source has namespace prefixes
2. Extract the data if needed (removes prefixes)
3. Copy all data to the destination
4. Set chain continuity markers

### Manual Steps (if needed)

1. **Extract namespaced data**:
```bash
./bin/genesis extract state <source-db> <extracted-db> --network <chain-id>
```

2. **Import to C-Chain**:
```bash
./bin/genesis import subnet <extracted-db> <cchain-db>
```

3. **Launch with imported data**:
```bash
./bin/genesis launch L1
```

## Chain-Specific Settings

### LUX Mainnet (96369)
- **Source**: `chaindata/lux-mainnet-96369/db/pebbledb`
- **Blockchain ID**: `dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ`
- **Size**: ~7.2GB
- **Keys**: ~4M (mostly state data)

### ZOO Mainnet (200200)
- **Source**: `chaindata/zoo-mainnet-200200/db/pebbledb`
- **Blockchain ID**: `bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM`
- **Size**: ~3.7MB
- **Keys**: ~70K

### SPC Mainnet (36911)
- **Source**: `chaindata/spc-mainnet-36911/db/pebbledb`
- **Blockchain ID**: `QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1`
- **Size**: ~48KB
- **Keys**: ~1K

## Verification

After migration, verify the chain:

```bash
# Check database structure
./bin/genesis inspect keys <db-path> --limit 10

# Check prefixes (should show EVM prefixes like 0x26, 0x68, etc.)
./bin/genesis inspect prefixes <db-path>

# Launch and verify via RPC
./bin/genesis launch verify
```

## Troubleshooting

### "db contains invalid genesis hash" error
This occurs when the chain configuration doesn't match the migrated data. Ensure you're using the correct genesis configuration for the network.

### Extraction takes too long
The extraction process can be slow for large databases. For LUX mainnet with 4M+ keys:
- Expected time: 10-30 minutes
- Monitor progress in the output
- The process can be safely interrupted and resumed

### No EVM data after extraction
If you see unusual key prefixes (not 0x26, 0x68, etc.), the source may not contain EVM data. Check:
- Is this the correct database path?
- Does it contain consensus data instead of EVM data?
- Try looking for a separate EVM database

## Next Steps

After successful migration:
1. Launch luxd with the imported data
2. Verify chain height and treasury balance
3. Test account queries and transaction submission
4. Enable indexing services
5. Set up additional validator nodes