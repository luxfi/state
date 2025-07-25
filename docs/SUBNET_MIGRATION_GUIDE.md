# Subnet Migration Guide

This guide explains how to migrate subnet EVM blockchain data to either C-Chain or L2 formats.

## Overview

When migrating subnet data, you have two options:

1. **C-Chain Migration**: Converts subnet data to C-Chain format with blockchain ID prefixes
2. **L2 Migration**: Preserves subnet data format for L2 deployment

## Key Differences

### C-Chain Format
- All keys are prefixed with 32-byte blockchain ID
- Used for primary C-Chain in luxd node
- Example: `2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC` + original key

### L2 Format  
- Keys remain in original format (no blockchain ID prefix)
- Used for L2/subnet deployments
- Direct copy of extracted subnet data

## Migration Process

### Step 1: Extract Subnet Data

First, extract the subnet data using namespace tool to remove the 33-byte namespace prefixes:

```bash
# Extract any subnet (example: ZOO mainnet)
./bin/genesis extract state \
    /path/to/archived/chaindata/2024-200200 \
    ./extracted-zoo-200200 \
    --network 200200 \
    --state
```

### Step 2: Choose Migration Type

#### Option A: Migrate to C-Chain Format

For primary network C-Chain (e.g., migrating 96369 subnet to become the new C-Chain):

```bash
./bin/genesis migrate subnet-to-cchain \
    ./extracted-subnet-96369 \
    /home/z/.luxd/chainData/[blockchain-id]/db/pebbledb \
    --blockchain-id "2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC" \
    --clear-dest
```

#### Option B: Migrate to L2 Format

For L2/subnet deployments (e.g., ZOO, SPC):

```bash
./bin/genesis migrate subnet-to-l2 \
    ./extracted-zoo-200200 \
    ./l2-data/zoo-200200 \
    --chain-id 200200 \
    --clear-dest \
    --verify
```

## Network Reference

| Network | Chain ID | Type | Migration Target |
|---------|----------|------|------------------|
| LUX (historic) | 96369 | Subnet → C-Chain | C-Chain format |
| ZOO | 200200 | Subnet → L2 | L2 format |
| SPC | 36911 | Subnet → L2 | L2 format |
| Hanzo | 36963 | New L2 | Fresh deployment |

## Blockchain IDs

- **C-Chain**: `2S76s9v5CCCpFkvsvnVcGiTHZ8oTnek99Pp9sTJkGKGzD1inzC`
- **ZOO**: `bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM`
- **SPC**: `QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1`

## Verification

After migration, verify the data:

### For C-Chain
```bash
# Check via RPC (after starting node)
curl -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
    http://localhost:9630/ext/bc/C/rpc
```

### For L2s
```bash
# The migrate command includes verification
# Look for output like:
# ✓ Found block 14552 with hash: 0x1234...
# ✓ Found 14553 headers in destination
```

## Deployment

### C-Chain Deployment
1. Stop luxd node
2. Replace the pebbledb directory with migrated data
3. Start luxd node
4. Verify it continues from the last block

### L2 Deployment
1. Create L2 with lux-cli:
   ```bash
   lux-cli subnet create zoo --evm --chainId=200200
   ```
2. Copy migrated data to L2's chain directory
3. Deploy and start the L2

## Important Notes

- Always backup original data before migration
- C-Chain format requires blockchain ID prefix on ALL keys
- L2 format preserves original key structure
- Both formats maintain complete blockchain state and history
- Chain continuity markers ensure proper block progression

## Troubleshooting

### "Could not determine blockchain ID"
- For C-Chain: Specify with `--blockchain-id` flag
- The ID should match your destination path structure

### "Database already exists"  
- Use `--clear-dest` flag to overwrite
- Or choose a different destination path

### Block number mismatch
- Verify source data was extracted correctly
- Check that namespace tool included `--state` flag
- Ensure chain ID matches the original subnet