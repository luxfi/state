# Genesis Migration with Canonical Key Patches

This repository provides a complete Docker-based solution for migrating old subnet EVM chaindata to the new Lux C-Chain format with 9-byte canonical keys.

## Overview

The migration process:
1. Imports subnet chaindata with namespace stripping
2. Removes the 0x6e suffix from canonical keys (10-byte → 9-byte)
3. Rebuilds canonical hash mappings
4. Launches luxd with the migrated data at height 1,082,780

## Quick Start

### Using Docker Compose

```bash
# Build and run
docker compose up -d

# Check logs
docker compose logs -f

# Test RPC
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc
```

### Using Docker Directly

```bash
# Build image
docker build -f docker/Dockerfile -t luxfi/genesis-migration .

# Run with chaindata
docker run -d \
  --name lux-genesis \
  -p 9630:9630 \
  -v /path/to/chaindata:/app/chaindata:ro \
  -v /path/to/runtime:/app/runtime \
  luxfi/genesis-migration
```

### Local Testing

```bash
# Run the test script
./test-local.sh
```

## Directory Structure

```
genesis/
├── docker/
│   ├── Dockerfile          # Single Dockerfile for complete workflow
│   └── entrypoint.sh       # Migration and launch script
├── chaindata/              # Mount old subnet data here
├── runtime/                # Migrated data stored here
├── compose.yml            # Easy local deployment
└── test-local.sh          # Local testing script
```

## Environment Variables

- `NETWORK_ID` - Network ID (default: 96369)
- `CHAIN_ID` - Chain ID (default: X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3)
- `VM_ID` - VM ID (default: mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6)
- `TIP_HEIGHT` - Expected blockchain height (default: 1082780)
- `TIP_HASH` - Expected tip hash

## Key Changes

### Canonical Key Format
- **Old**: `0x68` + block_number(8) + `0x6e` = 10 bytes
- **New**: `0x68` + block_number(8) = 9 bytes
- The `0x6e` suffix has been removed

### Patched Files
The Dockerfile automatically patches luxd during build:
- `vms/cchainvm/database.go` - Added `canonicalKey()` function
- `geth/core/rawdb/schema.go` - Removed suffix from `headerHashKey()`
- Related accessor and database files

## CI/CD

GitHub Actions workflow (`.github/workflows/build-and-test.yml`):
- Builds Docker image on push
- Tags as `canonical-9byte`
- Pushes to GitHub Container Registry
- Runs integration tests

## Verification

Expected output after migration:
```
[cchainvm] Found canonical hash for height 1082780
[boot] chain 96369 fully bootstrapped (⟙ 1,082,780)
```

RPC verification:
```bash
# Should return 0x10859c (1082780 in hex)
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc
```

## Troubleshooting

### Container won't start
Check logs: `docker logs lux-genesis`

### Migration fails
Ensure chaindata is mounted correctly and contains PebbleDB data

### Wrong block height
Verify the canonical key patches were applied during build

## Architecture

The solution uses a single Dockerfile that:
1. Builds the genesis migration tool
2. Installs/builds luxd with canonical key patches
3. Runs the migration pipeline on startup
4. Launches luxd with the migrated data

This ensures a reproducible, containerized workflow for booting the old subnet EVM as C-Chain data.