# Lux Network Operator Runbook

## Chain Data Import Process

This runbook covers the complete process of importing chain data into a Lux node, including subnet-to-C-Chain migration with canonical mapping fixes.

## Quick Start: Production Migration

For subnet-to-C-Chain migration, use the automated script:

```bash
# Complete migration with all fixes
./migrate_with_rebuild.sh /subnet96369/pebbledb /data/cchain-full

# Launch node
luxd --db-dir /data/cchain-full --network-id 96369 --staking-enabled=false

# Verify with RPC
./tools/rpc_verify.sh
```

## Prerequisites

- Lux node binary (`luxd`) compiled with import support
- Source chain data (PebbleDB or LevelDB format)
- Sufficient disk space (at least 2x the size of chain data)
- System with at least 16GB RAM

## Step-by-Step Process

### 1. Prepare Import Environment

```bash
# Set environment variables
export LUXD_PATH=$HOME/work/lux/node/build/luxd
export DATA_DIR=$HOME/.luxd-import
export NETWORK_ID=96369
export LOG_DIR=./logs

# Create directories
mkdir -p $LOG_DIR
mkdir -p $DATA_DIR
```

### 2. Start Import Process

Use the provided import script:

```bash
./scripts/import-chain-data.sh /path/to/source/chaindata
```

This script will:
- Stop any existing node processes
- Start node with `--import-chain-data` flag
- Monitor the import progress
- Automatically restart in normal mode when complete

### 3. Monitor Import Progress

During import, monitor the logs:

```bash
# Watch import progress
tail -f logs/import-*.log

# Look for key indicators:
# - "Rebuilding state snapshot" - Import has started
# - "Generated state snapshot" - Import completed
# - Any ERROR messages
```

Expected import times:
- Small chain (<10GB): 30 minutes - 1 hour
- Medium chain (10-50GB): 1-3 hours  
- Large chain (>50GB): 3-6 hours

### 4. Post-Import Verification

Once import completes and node restarts:

```bash
# Check node is running
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.isBootstrapped","params":{"chain":"C"}}' \
  http://localhost:9630/ext/info

# Check block height
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc
```

### 5. Backup Database

After successful import:

```bash
# Use the generated backup script
./logs/backup-database.sh

# Or manually:
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
tar -czf backups/luxd-import-$TIMESTAMP.tar.gz -C $DATA_DIR .
```

### 6. Monitor for 48 Hours

Start the monitoring script:

```bash
# Run monitoring in background
nohup ./scripts/monitor-node.sh > monitoring.out 2>&1 &

# Check monitoring status
tail -f monitoring.log
```

The monitor will:
- Check node health every 60 seconds
- Log block height and peer count
- Alert after 5 consecutive failures
- Notify when 48-hour milestone is reached

### 7. Enable Indexing (After 48h)

Once node is stable for 48 hours:

```bash
# Stop node
pkill luxd

# Restart with indexing enabled
$LUXD_PATH \
  --network-id=$NETWORK_ID \
  --data-dir=$DATA_DIR \
  --http-host=0.0.0.0 \
  --http-port=9630 \
  --index-enabled \
  --pruning-enabled \
  --state-sync-enabled=false
```

### 8. Deploy Additional Validators

For each additional validator:

1. Copy the verified database:
```bash
rsync -av $DATA_DIR/ validator2:/path/to/data/
```

2. Generate unique node ID:
```bash
$LUXD_PATH --data-dir=/new/validator/dir --generate-staking-cert
```

3. Start validator with unique staking port:
```bash
$LUXD_PATH \
  --network-id=$NETWORK_ID \
  --data-dir=/new/validator/dir \
  --staking-port=9651 \
  --http-port=9652
```

## Troubleshooting

### Import Fails

1. Check disk space:
```bash
df -h $DATA_DIR
```

2. Verify source data integrity:
```bash
# For PebbleDB
ls -la /source/chaindata/CURRENT

# For LevelDB  
ls -la /source/chaindata/CURRENT
```

3. Check import logs for specific errors:
```bash
grep -i error logs/import-*.log
```

### Node Won't Sync After Import

1. Check peers:
```bash
curl -X POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.peers","params":[]}' \
  http://localhost:9630/ext/info
```

2. Add bootstrap nodes:
```bash
--bootstrap-ips=<ip1>,<ip2> \
--bootstrap-ids=<id1>,<id2>
```

3. Check firewall rules:
```bash
# Ensure ports are open
sudo ufw allow 9630/tcp  # HTTP API
sudo ufw allow 9651/tcp  # Staking
```

### High Memory Usage

1. Adjust cache sizes:
```bash
--db-cache=512  # Reduce from default
```

2. Enable memory profiling:
```bash
--profile-dir=./profiles \
--profile-continuous-enabled
```

## Performance Tuning

### Recommended Settings

For production nodes:
```bash
$LUXD_PATH \
  --network-id=$NETWORK_ID \
  --data-dir=$DATA_DIR \
  --db-cache=1024 \
  --pruning-enabled \
  --state-sync-enabled=false \
  --api-max-duration=0 \
  --api-max-blocks-per-request=0 \
  --continuous-profiler-frequency=900000000000
```

### System Requirements

Minimum:
- CPU: 8 cores
- RAM: 16GB
- Disk: 500GB SSD
- Network: 100Mbps

Recommended:
- CPU: 16 cores
- RAM: 32GB
- Disk: 1TB NVMe SSD
- Network: 1Gbps

## Monitoring Metrics

Key metrics to track:
- Block height progression
- Peer count (should be >5)
- Memory usage (<80% of system RAM)
- Disk I/O (watch for saturation)
- Network bandwidth usage

## Emergency Procedures

### Node Crash

1. Check system resources:
```bash
free -h
df -h
top
```

2. Check node logs:
```bash
tail -1000 logs/normal-*.log | grep -i error
```

3. Restart from backup if corrupted:
```bash
# Stop node
pkill luxd

# Restore from backup
rm -rf $DATA_DIR/*
tar -xzf backups/latest-backup.tar.gz -C $DATA_DIR

# Restart
./scripts/import-chain-data.sh
```

### Rollback Procedure

If issues arise after deployment:

1. Stop all validators
2. Restore from pre-deployment backup
3. Restart with previous version
4. Investigate issues before retry

## Version Management

Before deploying to production:

1. Tag the build:
```bash
git tag -a v1.0.0-import -m "Post-import stable version"
git push origin v1.0.0-import
```

2. Document build parameters:
```bash
go version > build-info.txt
git rev-parse HEAD >> build-info.txt
date >> build-info.txt
```

3. Create release notes documenting:
- Import source and date
- Node configuration used
- Any custom patches applied
- Known issues or limitations

## Contact Information

For emergencies:
- Primary: [Your contact]
- Secondary: [Backup contact]
- Escalation: [Team lead]

## Appendix: Quick Commands

```bash
# Check node status
curl -sX POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"health.health","params":[]}' \
  http://localhost:9630/ext/health

# Get network info
curl -sX POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkName","params":[]}' \
  http://localhost:9630/ext/info

# Check C-Chain sync status
curl -sX POST -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_syncing","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc
```