# Migration Verification Guide

This guide walks through verifying the successful migration of subnet data to C-Chain.

## Prerequisites

- Docker and Docker Compose installed
- Genesis repository cloned and built
- Migration completed successfully

## 1. Initial Container Launch

Start the container and monitor logs:

```bash
# Build and start the container
make docker-run

# Watch logs for successful startup
docker compose logs -f
```

### Expected Log Output

Look for these key indicators:

```
<C Chain> cchainvm/vm.go:296 C-Chain VM starting from imported state
<C Chain> ... lastAcceptedHeight=1082780 lastAcceptedID=0x32dede...
...
node bootstrapped ✔
```

## 2. RPC Verification

### Check Block Height

Through the Nginx proxy (port 8080):

```bash
make check-height
# Expected: Height: 1082780 (0x10859c)
```

Direct to node (port 9630):

```bash
curl -s -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
  http://localhost:9630/ext/bc/C/rpc | jq
```

### Verify President's Balance

Check the known address balance:

```bash
make check-balance
# Should return the balance in wei
```

### Inspect Specific Block

```bash
docker compose exec lux luxd inspect block 1082780
```

## 3. Interactive Verification

### Open JavaScript Console

```bash
make console
```

In the console, run:

```javascript
// Check current block
eth.blockNumber

// Get block details
eth.getBlock(1082780)

// Check an account balance
eth.getBalance("0x9011E888251AB053B7bD1cdB598Db4f9DEd94714")
```

### Monitor Live Blocks

```bash
make monitor
```

This shows real-time block production (if any).

## 4. Database Verification

### Create a Snapshot

```bash
make snapshot
# Creates: snapshot-YYYYMMDD-HHMMSS.tgz
```

### Verify Key Format

Check that canonical keys are 9-byte format:

```bash
docker compose exec lux /opt/genesis/bin/genesis inspect canonical /opt/lux/runtime/db/pebbledb
```

Expected output should show 9-byte canonical keys for blocks.

## 5. Network Connectivity Tests

### Test eth_getLogs

```bash
curl -s -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_getLogs","params":[{"fromBlock":"0x0","toBlock":"latest"}]}' \
  http://localhost:8080/ | jq
```

Note: This requires `--chain-configs.enable-indexing` flag.

### Test eth_getTransactionByHash

Use a known transaction hash from the migrated data:

```bash
curl -s -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"eth_getTransactionByHash","params":["0xYOUR_TX_HASH"]}' \
  http://localhost:8080/ | jq
```

## 6. Common Issues and Solutions

### Issue: Node shows height 0

**Cause**: Using old 10-byte canonical key format
**Solution**: Apply the canonical key patches and rebuild

### Issue: eth_getLogs returns empty

**Cause**: Indexing not enabled
**Solution**: Add `--chain-configs.enable-indexing` to launch command

### Issue: Cannot connect to RPC

**Check**:
- Container is running: `docker ps`
- Ports are mapped: `docker compose ps`
- Nginx is forwarding: `docker compose logs nginx`

## 7. Production Deployment Checklist

- [ ] Container builds successfully
- [ ] Node starts at correct height (1082780)
- [ ] RPC responds on port 8080
- [ ] President's balance matches expected
- [ ] eth_blockNumber returns correct height
- [ ] No "height 0" fallback in logs
- [ ] Database uses 9-byte canonical keys
- [ ] Snapshot created for backup

## Next Steps

1. Push image to registry:
   ```bash
   docker tag lux-migrated:latest registry.example.com/lux:1.0.0
   docker push registry.example.com/lux:1.0.0
   ```

2. Deploy to validators:
   ```bash
   docker pull registry.example.com/lux:1.0.0
   docker compose up -d
   ```

3. Update load balancer to include new node

4. Monitor logs and metrics