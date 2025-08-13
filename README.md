# Lux State Database Documentation

## Overview

This directory contains blockchain state data for various networks. The primary database is SubnetEVM format stored in PebbleDB, which can be migrated to Coreth format for use with the Lux node.

## Directory Structure

```
/Users/z/work/lux/state/
├── chaindata/
│   ├── lux-mainnet-96369/       # Lux mainnet (chain ID: 96369)
│   │   ├── db/
│   │   │   └── pebbledb/        # SubnetEVM database (7.1GB)
│   │   ├── bootnodes.json       # Network bootstrap nodes
│   │   └── metadata.json        # Chain metadata
│   ├── lux-testnet-96368/       # Lux testnet
│   ├── zoo-mainnet-200200/      # Zoo mainnet
│   ├── zoo-testnet-200201/      # Zoo testnet
│   ├── spc-mainnet-36911/       # SPC mainnet
│   ├── eth-mainnet/             # Ethereum mainnet
│   └── bsc-mainnet/             # BSC mainnet
```

## Database Format: SubnetEVM

### Key Structure

SubnetEVM uses a **namespaced plain key** format in PebbleDB:

```
Key = namespace(32 bytes) + suffix
```

**Namespace for Lux mainnet**: `337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1`

### Key Types

#### 1. Block Headers
```
Key:   namespace + 'h' + be8(blockNumber) + blockHash
Value: RLP-encoded header (603 bytes)
```

#### 2. Block Bodies
```
Key:   namespace + 'b' + be8(blockNumber) + blockHash
Value: RLP-encoded body (transactions)
```

#### 3. Receipts
```
Key:   namespace + 'r' + be8(blockNumber) + blockHash
Value: RLP-encoded receipts array
```

#### 4. Hash to Number Mapping
```
Key:   namespace + 'H' + blockHash
Value: be8(blockNumber) - 8-byte big-endian
```

#### 5. Metadata Keys
```
Key:   namespace + "AcceptorTipKey"
Value: 32-byte hash of latest accepted block

Key:   namespace + "AcceptorTipHeightKey"
Value: be8(blockHeight) of latest accepted block
```

### Header Format

SubnetEVM headers are RLP-encoded with **17 fields**:

```go
type SubnetEVMHeader struct {
    ParentHash    [32]byte     // Field 0: Parent block hash
    UncleHash     [32]byte     // Field 1: Uncle hash (always empty hash)
    Coinbase      [20]byte     // Field 2: Beneficiary address
    Root          [32]byte     // Field 3: State root
    TxHash        [32]byte     // Field 4: Transactions root
    ReceiptHash   [32]byte     // Field 5: Receipts root
    Bloom         [256]byte    // Field 6: Logs bloom filter
    Difficulty    *big.Int     // Field 7: Difficulty (always 1)
    Number        *big.Int     // Field 8: Block number
    GasLimit      uint64       // Field 9: Gas limit
    GasUsed       uint64       // Field 10: Gas used
    Time          uint64       // Field 11: Timestamp
    Extra         []byte       // Field 12: Extra data
    MixDigest     [32]byte     // Field 13: Mix digest
    Nonce         [8]byte      // Field 14: Nonce
    BaseFee       *big.Int     // Field 15: EIP-1559 base fee
    ExtDataHash   [32]byte     // Field 16: Extension data hash (SubnetEVM specific)
}
```

**Important**: SubnetEVM headers do NOT include Cancun fields (WithdrawalsHash, BlobGasUsed, ExcessBlobGas, ParentBeaconRoot).

### Example Data

**Current tip (block 1,082,780)**:
- Hash: `0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0`
- Height: 1,082,780 (0x10859c)
- Header size: 603 bytes

**Genesis block**:
- Hash: `0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e`
- Height: 0

## Migration to Coreth

### Target Format

Coreth uses BadgerDB with a simpler key structure:

```
Headers:   'h' + be8(number) + hash
Bodies:    'b' + be8(number) + hash
Receipts:  'r' + be8(number) + hash
TD:        't' + be8(number) + hash
Canonical: 'h' + be8(number) → hash
Number:    'H' + hash → RLP(number)
```

### Migration Tool

The migration tool (`/Users/z/work/lux/genesis/cmd/migrate_final.go`) performs:

1. **Read tip**: Get latest block from `AcceptorTipKey`
2. **Walk chain**: Traverse backwards from tip to genesis
3. **Copy data**: Transfer raw RLP without modification
4. **Write metadata**: Set VM state (lastAccepted, height, initialized)
5. **Verify**: Check invariants after migration

### Migration Process

```bash
# Build migration tool
cd /Users/z/work/lux/genesis
go build -o bin/migrate_final cmd/migrate_final.go

# Run migration (takes ~75 seconds for 1M blocks)
./bin/migrate_final

# Output location
# Coreth DB: ~/.luxd/network-96369/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/ethdb
# VM metadata: ~/.luxd/network-96369/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm
```

### Migration Statistics

- **Total blocks**: 1,082,781 (genesis to 1,082,780)
- **Database size**: 7.1GB source → ~8GB destination
- **Migration time**: ~75 seconds
- **Processing rate**: ~14,430 blocks/second

## Key Encoding Functions

### Big-Endian 8-byte encoding
```go
func be8(n uint64) []byte {
    var b [8]byte
    binary.BigEndian.PutUint64(b[:], n)
    return b[:]
}
```

### Plain key construction
```go
func plainKey(namespace []byte, prefix byte, number uint64, hash common.Hash) []byte {
    key := make([]byte, 0, 73)
    key = append(key, namespace...)
    key = append(key, prefix)
    key = append(key, be8(number)...)
    key = append(key, hash.Bytes()...)
    return key
}
```

## Network Configuration

### Lux Mainnet (96369)
- **Chain ID**: 96369
- **Consensus**: Lux consensus (formerly Avalanche)
- **VM**: Coreth (Ethereum-compatible)
- **Block time**: ~2 seconds
- **Finality**: Sub-second

### Chain Configuration
```go
ChainConfig{
    ChainID:             96369,
    HomesteadBlock:      0,
    EIP150Block:         0,
    EIP155Block:         0,
    EIP158Block:         0,
    ByzantiumBlock:      0,
    ConstantinopleBlock: 0,
    PetersburgBlock:     0,
    IstanbulBlock:       0,
    MuirGlacierBlock:    0,
    BerlinBlock:         0,
    LondonBlock:         0,  // EIP-1559 active
    // Cancun not activated
}
```

## Querying the Database

### Using PebbleDB directly
```go
import "github.com/cockroachdb/pebble"

db, _ := pebble.Open("/path/to/pebbledb", &pebble.Options{ReadOnly: true})
defer db.Close()

// Read header at block 1082780
namespace, _ := hex.DecodeString("337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1")
hash, _ := hex.DecodeString("32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0")
key := plainKey(namespace, 'h', 1082780, common.BytesToHash(hash))

value, closer, err := db.Get(key)
if err == nil {
    defer closer.Close()
    // value contains RLP-encoded header
}
```

### After migration (via RPC)
```bash
# Query balance at specific block
curl -X POST -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","0x10859c"],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc
```

## Important Addresses

### Known accounts to verify
- **luxdefi.eth**: `0x9011E888251AB053B7bD1cdB598Db4f9DEd94714`
  - Expected balance at block 1,082,780: ~1.9T LUX
- **Additional address**: `0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59`

## Quick Migration Commands

```bash
# Full migration test
make

# Or step by step:
make import  # Import subnet data to C-Chain
make node    # Run luxd with imported data
make test    # Verify via RPC
```

## Troubleshooting

### Common Issues

1. **Missing headers**: Ensure namespace is correct (337fb73f...)
2. **RLP decode errors**: Headers have 17 fields, not Cancun-compatible
3. **Database locks**: Only one process can access PebbleDB at a time
4. **VM metadata**: Must be written to correct path for node to start

### Verification Commands

```bash
# Check database size
du -sh /Users/z/work/lux/state/chaindata/lux-mainnet-96369/db/pebbledb/

# Verify migration output exists
ls -la ~/.luxd/network-96369/chains/*/ethdb/

# Check VM metadata
ls -la ~/.luxd/network-96369/chains/*/vm/

# Query block number after node starts
curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
  http://localhost:9650/ext/bc/C/rpc | jq .result
```

## References

- [SubnetEVM Repository](https://github.com/ava-labs/subnet-evm)
- [Coreth Repository](https://github.com/ava-labs/coreth)
- [Lux Node Repository](https://github.com/luxfi/node)
- [Migration Guide](/Users/z/work/lux/genesis/MIGRATION_GUIDE.md)
- [Migration Tool](/Users/z/work/lux/genesis/cmd/migrate_final.go)
- [LLM Documentation](/Users/z/work/lux/genesis/LLM.md)