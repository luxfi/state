# Lux Chain Import Quick Start

## Overview

This guide provides quick commands to import chain data into a Lux node.

## Prerequisites

- Built `luxd` binary at `$HOME/work/lux/node/build/luxd`
- Source chain data (PebbleDB format)
- At least 500GB free disk space

## Quick Commands

### 1. Import Chain Data

```bash
# Simple import command
./scripts/lux-import.sh import /path/to/source/chaindata

# Or with custom environment
NETWORK_ID=96369 DATA_DIR=~/.luxd-mainnet \
  ./scripts/import-chain-data.sh /path/to/chaindata
```

### 2. Check Status

```bash
# Quick status check
./scripts/lux-import.sh status

# Detailed monitoring
./scripts/lux-import.sh monitor
```

### 3. Backup Database

```bash
# Create backup after successful import
./scripts/lux-import.sh backup
```

## Import Timeline

1. **Start Import** (0h)
   - Run import script
   - Monitor logs for "Rebuilding state snapshot"

2. **Import Complete** (1-6h depending on size)
   - Script automatically restarts node
   - Verify with status command

3. **Initial Monitoring** (0-48h)
   - Keep monitor script running
   - Check for stable block progression

4. **Enable Features** (48h+)
   - Enable indexing
   - Deploy additional validators
   - Switch to production config

## Common Import Paths

### For Mainnet (96369)
```bash
# From existing node
./scripts/lux-import.sh import ~/.luxd/chains/C/

# From backup
./scripts/lux-import.sh import /backups/luxd-mainnet/
```

### For L2s
```bash
# ZOO L2 (200200)
./scripts/lux-import.sh import /data/zoo-mainnet/

# SPC L2 (36911)  
./scripts/lux-import.sh import /data/spc-mainnet/
```

## Troubleshooting

### Import Hangs
```bash
# Check logs
tail -f logs/import-*.log

# Look for errors
grep -i error logs/import-*.log
```

### Node Won't Start After Import
```bash
# Check last 100 lines of log
tail -100 logs/normal-*.log

# Verify data directory
ls -la ~/.luxd-import/chains/
```

### Out of Disk Space
```bash
# Check disk usage
df -h

# Clean old logs
rm -f logs/import-*.log
find ~/.luxd-import/logs -mtime +7 -delete
```

## Next Steps

After successful import:

1. Review [OPERATOR_RUNBOOK.md](./OPERATOR_RUNBOOK.md) for detailed procedures
2. Set up automated monitoring
3. Plan validator deployment
4. Configure production settings

## Support

For issues:
1. Check logs in `./logs/` directory
2. Verify disk space and permissions
3. Ensure source data is not corrupted
4. Review full runbook for advanced troubleshooting