# AGENTS.md CLAUDE.md LLM.md - AI Assistant Guidance

This file provides context and guidance for AI assistants (Claude, GPT, etc.) working with the Lux Network Genesis repo.

## Project Context

You are working with the Lux Network's 2025 genesis data, which represents a major transition:
- Network 7777, launched in 2023 is fully retired, with account balances
  migrating to 96369 in the final, decentralized genesis of Lux Network (2025 edition).
- Network 96369, launched in 2024 becomes the final C-Chain (due to chain ID
  update)
- Pre-existing subnets can now elect to become sovereign L1s or remain as based
  L2 validators of Lux L1.
- Complete historical preservation for transparency

## Key Understanding Points

### 1. Network Evolution
- **7777**: Launched in 2023, Original network chain ID, now retired.
- **96369**: Launched in 2024, New primary network, NOT inheriting from 7777
- **Lux L2s**: ZOO, SPC, Hanzo - can upgrade to L1s at their discretion

### 2. Data Locations
```
/home/z/work/lux/genesis/data/unified-genesis/
├── lux-mainnet-96369/    # 7.2GB - Primary C-Chain
├── lux-testnet-96368/    # 1.1MB - REAL testnet (not 142MB one)
├── zoo-mainnet-200200/   # 3.7MB
├── zoo-testnet-200201/   # 292KB
├── spc-mainnet-36911/    # 48KB
├── lux-genesis-7777/             # Historical
└── configs/              # All configurations
```

### 3. Important Account
`0x9011E888251AB053B7bD1cdB598Db4f9DEd94714` - Treasury account used for verification:
- Started with 2T in each network
- Mainnet shows ~1.995T (real usage)
- Testnet should show <1.9T (real usage)

## Tools You Should Know

### 1. denamespace
```bash
# Extract blockchain data
./bin/denamespace -src <pebbledb> -dst <output> -network <chain-id> -state
```
- Use this for any network extraction
- Always include `-state` for account balances
- Supports all chain IDs

### 2. evmarchaeology
```bash
# Analyze blockchain data
./bin/evmarchaeology analyze -db <path> -network <name>
```
- Good for finding accounts and balances
- Can trace historical changes

### 3. Helper Scripts
- `extract_all_networks.sh` - Extracts all 6 networks
- `copy_all_networks.sh` - Copies to unified directory
- `collect_all_chain_configs.sh` - Gathers configurations

## Common Tasks

### Extracting a Network
```bash
# Example: Extract ZOO mainnet
./bin/denamespace \
    -src /path/to/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/db/pebbledb \
    -dst ./extracted-zoo \
    -network 200200 \
    -state
```

### Finding Account Balances
```bash
# In extracted data
./bin/evmarchaeology analyze \
    -db ./extracted-zoo \
    -account 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
```

### Running a Network
```bash
# Start with luxd
luxd --network-id=96369 --data-dir=./data/unified-genesis/lux-mainnet-96369/db

# Or with lux-cli
./lux-cli network start --genesis-path=./configs/lux-mainnet-96369/genesis.json
```

## Important Technical Details

### 1. PebbleDB Namespacing
- Raw data has 33-byte namespace prefixes
- Chain ID encoded as hex in prefix
- Tools strip these for standard access

### 2. Network IDs
| Network | Chain ID | Blockchain ID |
|---------|----------|---------------|
| LUX Mainnet | 96369 | dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ |
| LUX Testnet | 96368 | 2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG |
| ZOO Mainnet | 200200 | bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM |
| ZOO Testnet | 200201 | 2usKC5aApgWQWwanB4LL6QPoqxR1bWWjPCtemBYbZvxkNfcnbj |
| SPC Mainnet | 36911 | QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1 |

### 3. Key Prefixes in Database
- `0x68`: headers
- `0x48`: hash->number mappings
- `0x62`: bodies
- `0x72`: receipts
- `0x26`: accounts
- `0xa3`: storage
- `0x73`: state

## Decision Rationale

### Why 96369 as C-Chain?
- Fresh architecture without 7777's technical debt
- Better performance and features as we migrated from leveldb to pebbledb
- Clean start with proper airdrops

### Why Keep 7777?
- Transparency for airdrop verification
- Historical record (like Ethereum Classic)
- Some may prefer to keep running it

### Why L2s Can Become L1s?
- Sovereignty and decentralization
- Custom consensus rules
- Independent validator sets
- Maintained interoperability

## Helpful Context for Responses

When asked about:
- **"Why transition?"** - Evolution from single to multi-chain, better architecture
- **"What's the real testnet?"** - 1.1MB one with actual usage, not 142MB
- **"Can I verify?"** - Yes, all data and tools provided for transparency
- **"What about 7777?"** - Preserved for historical records, and to ensure
  proper mainnet genesis including all users of the original genesis network.

## Testing Commands

Always test extraction before full operations:
```bash
# Test extraction (small dataset)
./bin/denamespace \
    -src /path/to/data \
    -dst /tmp/test \
    -network <id> \
    -state \
    -limit 1000

# Verify it worked
ls -la /tmp/test/
```

## Final Notes

- This is a transparent, verifiable transition
- All historical data is preserved
- Anyone can run these tools to verify
- The goal is a decentralized multi-chain future

Remember: When in doubt, the raw blockchain data is the source of truth!
