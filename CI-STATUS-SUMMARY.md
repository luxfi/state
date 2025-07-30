# CI Status Summary for Genesis Migration

## What We've Built

A complete Docker-based solution that:
1. Builds the genesis migration tool
2. Installs/builds luxd with 9-byte canonical key patches
3. Runs the migration pipeline automatically
4. Launches luxd on port 9630 with migrated data

## CI Workflow

The GitHub Actions workflow (`build-genesis-migration.yml`) will:
- Build and test the genesis tool
- Build the Docker image with all patches
- Push to GitHub Container Registry as `ghcr.io/luxfi/genesis:canonical-9byte`
- Run basic integration tests

## Check CI Status

### Option 1: Web Browser
Visit: https://github.com/luxfi/genesis/actions/workflows/build-genesis-migration.yml

### Option 2: GitHub CLI
```bash
./check-ci-status.sh
```

### Option 3: Direct Link
Latest runs: https://github.com/luxfi/genesis/actions

## Expected CI Results

✅ **Green CI means:**
- Genesis tool builds successfully
- Docker image builds with patches
- Container starts without errors
- Image pushed to registry

## Using the Built Image

Once CI is green, pull and run:

```bash
# Pull the image
docker pull ghcr.io/luxfi/genesis:canonical-9byte

# Run with your chaindata
docker run -d \
  --name lux-genesis \
  -p 9630:9630 \
  -v /path/to/chaindata:/app/chaindata:ro \
  -v /path/to/runtime:/app/runtime \
  ghcr.io/luxfi/genesis:canonical-9byte
```

## Local Testing

```bash
# Use compose for easy local testing
docker compose up -d

# Check logs
docker compose logs -f genesis-migration

# Test RPC
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc
```

## Key Files Changed

1. `docker/Dockerfile` - Single Dockerfile with genesis + luxd
2. `docker/entrypoint.sh` - Migration pipeline and launch script
3. `compose.yml` - Docker Compose configuration
4. `.github/workflows/build-genesis-migration.yml` - CI workflow
5. Port changed from 9650 → 9630 for Lux

## Canonical Key Fix

The patches ensure:
- Old: `0x68 + block_number + 0x6e` (10 bytes)
- New: `0x68 + block_number` (9 bytes)
- Blockchain starts at height 1,082,780