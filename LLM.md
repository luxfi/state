# LLM Documentation - Lux State Database

## Executive Summary

This document provides comprehensive technical documentation for understanding and working with the Lux blockchain state database, focusing on the SubnetEVM format and its migration to Coreth.

## Database Architecture

### SubnetEVM Database Format

**Location**: `/Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb`

**Database Engine**: PebbleDB (CockroachDB's LSM-based storage)

**Size**: 7.1GB containing 1,082,781 blocks

### Key Space Design

SubnetEVM uses a **namespaced plain key** architecture:

```
Key Structure: namespace(32 bytes) || suffix(variable)
```

**Namespace for Lux Mainnet**: 
```
337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1
```

### Key Types and Formats

#### 1. Block Headers
```
Key:   namespace || 'h' || be8(blockNumber) || blockHash(32)
Value: RLP(Header) - typically 603 bytes
Total: 73 bytes key
```

#### 2. Block Bodies (Transactions)
```
Key:   namespace || 'b' || be8(blockNumber) || blockHash(32)
Value: RLP(Body) - variable size
Total: 73 bytes key
```

#### 3. Receipts
```
Key:   namespace || 'r' || be8(blockNumber) || blockHash(32)
Value: RLP([]Receipt) - variable size
Total: 73 bytes key
```

#### 4. Hash to Number Mapping
```
Key:   namespace || 'H' || blockHash(32)
Value: be8(blockNumber) - exactly 8 bytes
Total: 65 bytes key
```

#### 5. Metadata Keys
```
AcceptorTipKey:       namespace || "AcceptorTipKey" (ASCII)
                      → 32-byte hash of latest block

AcceptorTipHeightKey: namespace || "AcceptorTipHeightKey" (ASCII)
                      → 8-byte BE of latest height
```

## Data Structures

### SubnetEVM Header Format

**RLP-encoded with exactly 17 fields**:

| Field | Name | Type | Size | Description |
|-------|------|------|------|-------------|
| 0 | ParentHash | Hash | 32 bytes | Previous block hash |
| 1 | UncleHash | Hash | 32 bytes | Always empty hash |
| 2 | Coinbase | Address | 20 bytes | Block beneficiary |
| 3 | Root | Hash | 32 bytes | State trie root |
| 4 | TxHash | Hash | 32 bytes | Transaction trie root |
| 5 | ReceiptHash | Hash | 32 bytes | Receipt trie root |
| 6 | Bloom | Bloom | 256 bytes | Logs bloom filter |
| 7 | Difficulty | BigInt | Variable | Always 1 |
| 8 | Number | BigInt | Variable | Block number |
| 9 | GasLimit | uint64 | Variable | Gas limit |
| 10 | GasUsed | uint64 | Variable | Gas used |
| 11 | Time | uint64 | Variable | Unix timestamp |
| 12 | Extra | bytes | Variable | Extra data |
| 13 | MixDigest | Hash | 32 bytes | Mix digest |
| 14 | Nonce | uint64 | 8 bytes | Block nonce |
| 15 | BaseFee | BigInt | Variable | EIP-1559 base fee |
| 16 | ExtDataHash | Hash | 32 bytes | SubnetEVM extension |

**Important**: Does NOT include Cancun fields (WithdrawalsHash, BlobGasUsed, ExcessBlobGas, ParentBeaconRoot)

### Example Block Data

**Genesis Block (Height 0)**:
```
Hash:   0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
Number: 0
```

**Current Tip (Height 1,082,780)**:
```
Hash:   0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0
Number: 1,082,780 (0x10859c)
Size:   603 bytes (header)
```

## Encoding Functions

### Big-Endian 8-byte Encoding
```go
func be8(n uint64) []byte {
    var b [8]byte
    binary.BigEndian.PutUint64(b[:], n)
    return b[:]
}
```

### Key Construction
```go
func constructKey(namespace []byte, prefix byte, number uint64, hash []byte) []byte {
    key := make([]byte, 0, 73)
    key = append(key, namespace...)  // 32 bytes
    key = append(key, prefix)         // 1 byte ('h', 'b', 'r')
    key = append(key, be8(number)...) // 8 bytes
    key = append(key, hash...)        // 32 bytes
    return key
}
```

## Query Patterns

### Reading a Block Header
```go
namespace := hex.Decode("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")
blockNum := uint64(1082780)
blockHash := hex.Decode("32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0")

key := constructKey(namespace, 'h', blockNum, blockHash)
headerRLP, _ := db.Get(key)
```

### Finding Block Number from Hash
```go
key := append(namespace, 'H')
key = append(key, blockHash...)
numberBytes, _ := db.Get(key)
blockNumber := binary.BigEndian.Uint64(numberBytes)
```

### Getting Latest Block
```go
tipKey := append(namespace, []byte("AcceptorTipKey")...)
tipHash, _ := db.Get(tipKey)

heightKey := append(namespace, []byte("AcceptorTipHeightKey")...)
heightBytes, _ := db.Get(heightKey)
tipHeight := binary.BigEndian.Uint64(heightBytes)
```

## Migration to Coreth

### Target Database Format

**Location**: `~/.luxd/network-96369/chains/*/ethdb` (BadgerDB)

**Key Differences**:
- No namespace prefix
- Simpler key structure
- Additional metadata in vm/ directory

### Key Mappings

| SubnetEVM | Coreth |
|-----------|--------|
| ns + 'h' + num + hash | 'h' + num + hash |
| ns + 'b' + num + hash | 'b' + num + hash |
| ns + 'r' + num + hash | 'r' + num + hash |
| ns + 'H' + hash | 'H' + hash → RLP(num) |
| ns + "AcceptorTipKey" | vm/lastAccepted |

### Migration Process

1. **Read tip from SubnetEVM**
   - Get hash from AcceptorTipKey
   - Get height from H mapping

2. **Walk chain backwards**
   - Read header, body, receipts
   - Extract parent hash
   - Write to Coreth format
   - Continue to genesis

3. **Write metadata**
   - Head pointers in ethdb/
   - VM state in vm/
   - Chain config under genesis

### Performance Metrics

- **Blocks**: 1,082,781
- **Migration Time**: 75 seconds
- **Rate**: 14,430 blocks/second
- **Source Size**: 7.1GB
- **Target Size**: ~8GB

## Important Addresses

### Treasury Account
```
Address: 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
ENS:     luxdefi.eth
Balance: ~1.9T LUX at block 1,082,780
```

### Additional Account
```
Address: 0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59
```

## Network Parameters

### Chain Configuration
```javascript
{
  chainId: 96369,
  networkId: 96369,
  consensus: "Lux Consensus",
  blockTime: "~2 seconds",
  finality: "sub-second",
  eips: {
    homestead: 0,
    byzantium: 0,
    constantinople: 0,
    petersburg: 0,
    istanbul: 0,
    berlin: 0,
    london: 0,        // EIP-1559 active
    shanghai: 0,      // Some features
    cancun: null      // Not activated
  }
}
```

## Troubleshooting Guide

### Issue: Cannot find headers
**Symptom**: "Missing header at block X"
**Cause**: Wrong namespace or key format
**Solution**: 
- Verify namespace is `337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1`
- Ensure keys are plain, not hashed
- Check block exists in range 0-1,082,780

### Issue: RLP decode errors
**Symptom**: "rlp: input string too short for common.Hash"
**Cause**: Trying to decode SubnetEVM header into Coreth struct
**Solution**: Copy raw RLP without decoding, or use legacy struct

### Issue: Database locked
**Symptom**: "resource temporarily unavailable"
**Cause**: Multiple processes accessing PebbleDB
**Solution**: Ensure only one process reads database

### Issue: Wrong genesis
**Symptom**: "Genesis mismatch"
**Cause**: Chain config not under correct genesis hash
**Solution**: Write config under hash of block 0

## Query Examples

### Using PebbleDB CLI
```bash
# List keys with prefix
pebbledb scan --db=/path/to/db --prefix=337fb73f

# Get specific key
pebbledb get --db=/path/to/db --key=<hex>
```

### Using Go Code
```go
import "github.com/cockroachdb/pebble"

db, _ := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
defer db.Close()

// Iterate keys
it := db.NewIter(&pebble.IterOptions{})
for it.First(); it.Valid(); it.Next() {
    key := it.Key()
    val := it.Value()
    // Process...
}
```

### Using RPC (after migration)
```bash
# Get block by number
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x10859c",false],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc

# Get balance at specific block
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","0x10859c"],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc
```

## BadgerDB Best Practices

### Batch Writes for Efficiency
```go
batch := db.NewBatch()
defer batch.Reset()

const batchSize = 1000
for i := 0; i < totalBlocks; i++ {
    batch.Put(key, value)
    if i % batchSize == 0 {
        batch.Write()  // Flush batch
        batch.Reset()
    }
}
batch.Write() // Final flush
```

### Single Writer Pattern
- BadgerDB requires **single writer** for transactions
- Multiple readers are fine and encouraged
- Use sync.Mutex if multiple goroutines need to write

### Transaction Best Practices
```go
// Single writer transaction
txn := db.NewTransaction(true) // true = read-write
defer txn.Discard()

txn.Set(key1, value1)
txn.Set(key2, value2)

if err := txn.Commit(); err != nil {
    return err
}
```

## References and Resources

### Repositories
- [SubnetEVM](https://github.com/ava-labs/subnet-evm) - Source format
- [Coreth](https://github.com/ava-labs/coreth) - Target format
- [Lux Node](https://github.com/luxfi/node) - Node implementation
- [PebbleDB](https://github.com/cockroachdb/pebble) - Source database
- [BadgerDB](https://github.com/dgraph-io/badger) - Target database

### Documentation
- [State README](/Users/z/work/lux/state/README.md) - High-level overview
- [Migration Guide](/Users/z/work/lux/genesis/MIGRATION_GUIDE.md)
- [Migration Tool](/Users/z/work/lux/genesis/cmd/migrate_final.go)
- [Verification Tool](/Users/z/work/lux/genesis/cmd/verify_migration.go)

### Key Files
```
/Users/z/work/lux/state/
├── README.md           # High-level documentation
├── LLM.md             # This detailed technical guide
└── chaindata/
    └── lux-mainnet-96369/
        ├── db/
        │   └── pebbledb/  # Source database
        ├── bootnodes.json
        └── metadata.json
```

## Version History

- **v1.0** (2024-08-12): Initial documentation
- Database contains blocks 0 through 1,082,780
- Migration tool supports full chain migration
- Verified with Lux node version from genesis branch
- Added BadgerDB batch write optimizations