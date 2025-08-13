# SubnetEVM to C-Chain Migration Notes

## Updated: 2025-08-11

### Critical Discovery: Bootstrap Requirements

**Problem**: Single node cannot complete bootstrap phase
- Bootstrap requires receiving `accepted-frontier` from peers
- With only one node, there's no peer to complete handshake
- Node stays in "subnets not bootstrapped" state indefinitely

**Solution**: Two-node bootstrap setup
```bash
# Node A starts first
./luxd --network-id=96369 --http-port=9650 --staking-port=9651

# Get Node A's ID
NODE_A_ID=$(curl -s http://localhost:9650/ext/info \
  -X POST -H 'content-type: application/json' \
  -d '{"jsonrpc":"2.0","id":1,"method":"info.getNodeID"}' | jq -r '.result.nodeID')

# Node B bootstraps from Node A
./luxd --network-id=96369 --http-port=9652 --staking-port=9653 \
  --bootstrap-ips=127.0.0.1:9651 \
  --bootstrap-ids=$NODE_A_ID
```

Both nodes will complete bootstrap within seconds, then you can stop Node B.

### Database Migration Issues

1. **Namespace Prefix**: SubnetEVM uses 32-byte prefix `337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1`
   - Must be stripped during migration
   - Keys without namespace are metadata (keep as-is)

2. **Missing Canonical Mappings**: Direct PebbleDB->BadgerDB migration loses canonical chain
   - No `h+num+n` keys after block ~100
   - Must rebuild using rawdb.WriteCanonicalHash

3. **Fork Configuration**: C-Chain requires cancun fork settings
   ```json
   {
     "cancunTime": 0,
     "blobSchedule": {
       "cancun": {
         "targetBlobGas": 393216,
         "blobGasPriceMinimum": 1,
         "blobGasPriceExponential": 2,
         "blobGasPriceUpdateFraction": 3338477
       }
     }
   }
   ```

### Verified Working Process

1. **Migrate with namespace removal**:
   ```bash
   ./bin/migrate_subnet \
     --src /Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb \
     --dst /tmp/cchain-db \
     --strip-namespace
   ```

2. **Verify canonical chain**:
   ```bash
   ./bin/scan_canonical /tmp/cchain-db
   # If broken, run:
   ./bin/fix_canonical /tmp/cchain-db
   ```

3. **Launch with proper config**:
   - Place migrated DB at: `~/.luxd/chainData/C/db/`
   - Add config at: `~/.luxd/configs/chains/C/config.json`
   - Use two-node bootstrap method

### Expected Results

- Block height: 1,082,781 (0x10859d)
- Treasury balance: ~1.995T LUX at `0x9011e888251ab053b7bd1cdb598db4f9ded94714`
- Genesis hash: Matches original subnet genesis
- All historical transactions accessible

### Tools Created

- `scan_canonical`: Verifies canonical chain integrity
- `fix_canonical`: Rebuilds missing canonical mappings
- `inspect_db`: Shows database key structure
- `migrate_subnet_to_cchain`: Handles namespace removal

### Next Steps

1. Complete canonical chain fix
2. Launch two-node bootstrap
3. Verify treasury balance
4. Test transaction replay
5. Document final process