# Genesis Migration Guide

## Overview

This guide explains how to migrate blockchain data from Subnet EVM to C-Chain format for the Lux Network.

## Key Concepts

### Database Structure
- **Subnet EVM**: Uses namespaced PebbleDB with 33-byte prefixes
- **C-Chain**: Uses standard Ethereum/Geth database format
- **Consensus Layer**: Snowman++ consensus engine with separate metadata

### Migration Components
1. **State Data**: Account balances, contract storage (Merkle Patricia Trie)
2. **Blockchain Data**: Headers, bodies, receipts, canonical mappings
3. **Consensus Data**: Snowman block IDs, acceptance status, height mappings

## Migration Process

### Step 1: Analyze Source Database
```bash
# Check what data exists in the subnet database
./bin/analyze-subnet-blocks /path/to/subnet/db/pebbledb

# Find the highest block
./bin/find-highest-block /path/to/subnet/db/pebbledb

# Analyze key structure
./bin/analyze-key-structure /path/to/subnet/db/pebbledb
```

### Step 2: Add EVM Prefix (if needed)
```bash
# Add "evm" prefix to all keys for C-Chain compatibility
./bin/add-evm-prefix-to-blocks \
  -src /path/to/subnet/db/pebbledb \
  -dst /tmp/migrated-with-prefix
```

### Step 3: Generate Consensus State
Since subnet data lacks Snowman consensus metadata, we need to generate it:

```bash
# Replay blocks through consensus engine
./bin/replay-consensus-pebble \
  --evm /tmp/migrated-with-prefix \
  --state /tmp/consensus-state \
  --tip 1082780 \
  --batch 10000
```

### Step 4: Merge Databases
```bash
# Combine EVM and consensus databases
# This step requires custom tooling based on your setup
```

### Step 5: Launch Node
```bash
# Start luxd with migrated data
./launch-mainnet-with-chaindata.sh
```

## Common Issues

### No Canonical Hash Mappings
**Problem**: Migrated database only contains state trie nodes, not blockchain data.

**Solution**: 
1. Ensure you're migrating from the correct source (needs both state AND blockchain data)
2. Use subnet-to-cchain-replayer to reconstruct missing data

### Block Height Remains 0
**Problem**: Node starts but shows block 0 despite having data.

**Solution**:
1. Consensus metadata is missing
2. Run replay-consensus-pebble to generate proper Snowman state
3. Ensure versiondb metadata is present

### Import Path Conflicts
**Problem**: Build fails with ethereum/go-ethereum vs luxfi/geth conflicts.

**Solution**:
1. Run `./fix-imports.sh` to update all imports
2. Use local rawdb constants package for unexported values

## Database Key Mappings

### Subnet Format
```
[33-byte namespace prefix][actual key]
```

### C-Chain Format
```
evm[single-byte prefix][key data]
```

### Common Prefixes
- `h` (0x68): Block headers
- `b` (0x62): Block bodies  
- `H` (0x48): Hash to number
- `n` (0x6e): Number to hash
- `r` (0x72): Receipts
- `l` (0x6c): Transaction lookups

## Tools Reference

### Analysis Tools
- `analyze-subnet-blocks`: Comprehensive subnet database analysis
- `find-highest-block`: Find the tip of the chain
- `check-canonical-keys`: Verify canonical hash mappings exist

### Migration Tools
- `add-evm-prefix-to-blocks`: Add C-Chain compatible prefixes
- `migrate-subnet-tool`: Full subnet to C-Chain migration
- `replay-consensus-pebble`: Generate Snowman consensus state

### Verification Tools
- `check-head-pointers`: Verify head block pointers
- `verify-final-migration`: Comprehensive migration verification

## Best Practices

1. **Always backup** source data before migration
2. **Test on small dataset** first (use --max-keys flag)
3. **Verify each step** before proceeding
4. **Monitor logs** during node startup
5. **Check RPC responses** after migration

## Architecture Notes

### VersionDB Layer
- All Avalanche databases use versiondb MVCC wrapper
- Adds metadata and revision suffixes to keys
- Direct database writes bypass this layer (causing issues)

### Consensus Requirements
- Snowman needs block relationships and acceptance status
- Cannot initialize from raw Ethereum data alone
- Requires proper replay through consensus engine

### State vs Blockchain Data
- State: Accounts, storage (can start fresh)
- Blockchain: Headers, receipts (needed for history)
- Both required for full node operation