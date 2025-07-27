# Lux Chain Data Import Guide

This guide covers importing existing chain data into a Lux node using the unified genesis CLI tool.

## Overview

The genesis tool provides integrated import functionality that replaces the previous shell scripts with a more robust Go implementation. All import operations are now handled through the `genesis import` command family.

## Prerequisites

- Built `luxd` binary at `/home/z/work/lux/node/build/luxd`
- Built `genesis` tool: `make build-genesis`
- Source chain data in PebbleDB or LevelDB format
- At least 500GB free disk space
- 16GB+ RAM recommended

## Import Workflow

### Step 1: Import Chain Data

Import existing blockchain data from another node or backup:

```bash
# Using the CLI directly
./bin/genesis import chain-data /path/to/source/chaindata \
  --data-dir=$HOME/.luxd-import \
  --network-id=96369 \
  --luxd-path=/home/z/work/lux/node/build/luxd

# Using Makefile
make import-chain-data SRC=/path/to/source/chaindata
```

Options:
- `--data-dir`: Target directory for imported data (default: ~/.luxd-import)
- `--network-id`: Network ID to use (default: 96369)
- `--luxd-path`: Path to luxd binary
- `--auto-restart`: Automatically restart in normal mode after import (default: true)

The import process will:
1. Kill any existing node processes
2. Start luxd with `--import-chain-data` flag
3. Monitor logs for "Generated state snapshot" completion marker
4. Automatically restart node in normal mode when complete

### Step 2: Monitor Node Health

Monitor the node for stability (recommended 48 hours):

```bash
# Using the CLI
./bin/genesis import monitor \
  --interval=60s \
  --duration=48h \
  --rpc-url=http://localhost:9650 \
  --failure-threshold=5

# Using Makefile
make import-monitor
```

Options:
- `--interval`: How often to check node health (default: 60s)
- `--duration`: Total monitoring duration (default: 48h)
- `--rpc-url`: Node RPC endpoint (default: http://localhost:9650)
- `--failure-threshold`: Consecutive failures before alert (default: 5)

The monitor will:
- Check node health every interval
- Track block height progression
- Alert after consecutive failures
- Notify when 48-hour milestone is reached
- Log all activity to `monitoring.log`

### Step 3: Check Status

Check the current node status at any time:

```bash
# Using the CLI
./bin/genesis import status --rpc-url=http://localhost:9650

# Using Makefile
make import-status
```

This shows:
- Node process status (running/not running)
- RPC accessibility
- Current block height
- Bootstrap status
- Connected peer count
- Disk usage

## Export Operations

### Backup Database

Create a backup of the imported database:

```bash
# Using the CLI
./bin/genesis export backup \
  --data-dir=$HOME/.luxd-import \
  --backup-dir=./backups \
  --compress=true

# Using Makefile
make export-backup
```

Options:
- `--data-dir`: Directory to backup (default: ~/.luxd-import)
- `--backup-dir`: Where to store backups (default: ./backups)
- `--compress`: Create tar.gz archive (default: true)

### Export State

Export blockchain state to CSV:

```bash
# Using the CLI
./bin/genesis export state output.csv \
  --rpc-url=http://localhost:9650/ext/bc/C/rpc \
  --block=0

# Using Makefile
make export-state OUTPUT=output.csv
```

### Export Genesis

Export current state as a new genesis file:

```bash
# Using the CLI
./bin/genesis export genesis genesis.json \
  --data-dir=$HOME/.luxd-import \
  --include-code=true

# Using Makefile
make export-genesis OUTPUT=genesis.json
```

## Common Import Scenarios

### Import from Another Node

```bash
# Copy from running node
make import-chain-data SRC=/path/to/other/node/.luxd/chains/C

# Monitor the import
make import-monitor
```

### Import from Backup

```bash
# Extract backup first
tar -xzf luxd-backup-20240127.tar.gz

# Import the data
make import-chain-data SRC=./luxd-backup/chains/C

# Check status
make import-status
```

### Import for L2 Migration

For ZOO and SPC L2s that need existing data:

```bash
# Import ZOO L2 data
make import-chain-data SRC=/archived/zoo-mainnet/chaindata NETWORK_ID=200200

# Import SPC L2 data  
make import-chain-data SRC=/archived/spc-mainnet/chaindata NETWORK_ID=36911
```

## Timeline

1. **Import Start** (0h)
   - Execute import command
   - Monitor starts automatically

2. **Import Complete** (1-6h)
   - Depends on data size
   - Node auto-restarts in normal mode

3. **Initial Sync** (0-24h)
   - Node catches up to network tip
   - Monitor block progression

4. **Stability Period** (24-48h)
   - Ensure stable operation
   - Monitor for any issues

5. **Production Ready** (48h+)
   - Enable indexing
   - Deploy validators
   - Switch to production config

## Troubleshooting

### Import Hangs

Check the import log:
```bash
tail -f logs/import-*.log
grep -i error logs/import-*.log
```

### Node Won't Start After Import

Check the normal mode log:
```bash
tail -100 logs/normal-*.log
ls -la ~/.luxd-import/chains/
```

### High Memory Usage

Adjust cache settings when restarting:
```bash
luxd --db-cache=512 ...
```

### No Peers

Add bootstrap nodes:
```bash
luxd --bootstrap-ips=<ip1>,<ip2> --bootstrap-ids=<id1>,<id2> ...
```

## Best Practices

1. **Always Create Backups**: Run `make export-backup` after successful import
2. **Monitor for 48 Hours**: Don't skip the monitoring period
3. **Check Logs Regularly**: Keep an eye on both import and normal logs
4. **Verify Data Integrity**: Use `make import-status` to check health
5. **Document Everything**: Keep notes on source data and import parameters

## Next Steps

After successful import and 48-hour monitoring:

1. Enable indexing for API queries
2. Deploy additional validator nodes
3. Configure production settings
4. Set up automated monitoring
5. Create disaster recovery plan

For more details, see:
- [Operator Runbook](./OPERATOR_RUNBOOK.md) - Detailed operational procedures
- [Import Quick Start](./IMPORT_QUICKSTART.md) - Quick reference commands