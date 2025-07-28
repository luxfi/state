# LLM.md - AI Assistant Guidance

This file provides context and guidance for AI assistants (Claude, GPT, etc.) working with the Lux Network Genesis repo.

## Quick Reference

- **Primary Tool**: `genesis` - Unified CLI for all genesis operations
- **Output Structure**: `configs/{network}/{P,C,X}/genesis.json`
- **Key Chain IDs**: LUX=96369, ZOO=200200, SPC=36911
- **Treasury**: `0x9011e888251ab053b7bd1cdb598db4f9ded94714`
- **Migration Data**: `$HOME/work/lux/genesis/runtime/lux-96369-migrated/`

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
$HOME/work/lux/genesis/data/unified-genesis/
├── lux-mainnet/96369/    # 7.2GB - Primary C-Chain
├── lux-testnet-96368/    # 1.1MB - REAL testnet (not 142MB one)
├── zoo-mainnet/200200/   # 3.7MB
├── zoo-testnet-200201/   # 292KB
├── spc-mainnet/36911/    # 48KB
├── lux-genesis-7777/             # Historical
└── configs/              # All configurations
```

### 3. Important Account
`0x9011E888251AB053B7bD1cdB598Db4f9DEd94714` - Treasury account used for verification:
- Started with 2T in each network
- Mainnet shows ~1.995T (real usage)
- Testnet should show <1.9T (real usage)

## Tools You Should Know

### 1. genesis (Primary Tool - ONLY USE THIS!)
```bash
# Generate all genesis files with standard directory structure
./bin/genesis generate

# Output structure:
# configs/mainnet/P/genesis.json
# configs/mainnet/C/genesis.json  
# configs/mainnet/X/genesis.json

# Custom options
./bin/genesis generate --network testnet --output /custom/path

# Import commands (PRIMARY METHOD FOR SUBNET MIGRATION)
./bin/genesis import subnet <src> <dst>               # Import L2 as C-Chain with continuity
./bin/genesis import chain-data <src>                 # Import chain data
./bin/genesis import cchain <src>                     # Import C-Chain state
./bin/genesis import monitor                          # Monitor import progress
./bin/genesis import status                           # Check import status

# Analyze commands
./bin/genesis analyze                                 # Analyze blockchain data
./bin/genesis analyze keys                            # Analyze database keys
./bin/genesis analyze blocks                          # Analyze block structure
./bin/genesis analyze subnet                          # Analyze subnet data
./bin/genesis analyze structure                       # Analyze data structure
./bin/genesis analyze balance <address>               # Check account balance

# Inspect commands
./bin/genesis inspect                                 # Inspect database contents
./bin/genesis inspect keys                            # Inspect database keys
./bin/genesis inspect blocks                          # Inspect block data
./bin/genesis inspect headers                         # Inspect block headers
./bin/genesis inspect snowman                         # Inspect snowman consensus
./bin/genesis inspect prefixes                        # Inspect key prefixes
./bin/genesis inspect tip                             # Find chain tip

# Launch commands
./bin/genesis launch L1                               # Load chaindata into C-Chain
./bin/genesis launch L2                               # Load as L2 with lux primary network
./bin/genesis launch verify                           # Verify launched node
./bin/genesis launch clean                            # Clean launch (no data)
./bin/genesis launch mainnet                          # Launch mainnet
./bin/genesis launch testnet                          # Launch testnet

# Migration commands (ADVANCED - use import instead)
./bin/genesis migrate                                 # Full migration pipeline
./bin/genesis migrate --destination <path>            # Specify destination

# Other commands
./bin/genesis validators list                         # List validators
./bin/genesis validators add                          # Add validator
./bin/genesis extract state <src> <dst>               # Extract blockchain data
./bin/genesis tools                                   # List all commands
./bin/genesis validate                                # Validate genesis config
```
- **CRITICAL**: This is the ONLY tool you should use - NO shell scripts!
- Unified tool for ALL genesis operations
- Combines functionality from 100+ old scripts and tools
- Uses standard P/, C/, X/ directory structure by default
- Handles L2 to C-Chain migration with proper chain continuity

### 2. namespace
```bash
# Extract blockchain data
./bin/namespace -src <pebbledb> -dst <output> -network <chain-id> -state
```
- Use this for any network extraction
- Always include `-state` for account balances
- Supports all chain IDs

### 3. evmarchaeology
```bash
# Analyze blockchain data
./bin/evmarchaeology analyze -db <path> -network <name>
```
- Good for finding accounts and balances
- Can trace historical changes

### 4. Helper Scripts
- `extract_all_networks.sh` - Extracts all 6 networks
- `copy_all_networks.sh` - Copies to unified directory
- `collect_all_chain_configs.sh` - Gathers configurations

## Common Tasks

### Extracting a Network
```bash
# Example: Extract ZOO mainnet state (removes namespace prefix)
./bin/genesis extract state \
    chaindata/zoo-mainnet-200200/db/pebbledb \
    ./extracted-zoo \
    --network 200200
```

### Importing Subnet to C-Chain
```bash
# Import subnet data as C-Chain (handles extraction automatically)
./bin/genesis import subnet \
    chaindata/lux-mainnet-96369/db/pebbledb \
    runtime/lux-96369-cchain

# Launch with imported data
./bin/genesis launch L1
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
luxd --network-id=96369 --data-dir=./data/unified-genesis/lux-mainnet/96369/db

# Or with lux-cli
./lux-cli network start --genesis-path=./configs/lux-mainnet/96369/genesis.json
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
./bin/namespace \
    -src /path/to/data \
    -dst /tmp/test \
    -network <id> \
    -state \
    -limit 1000

# Verify it worked
ls -la /tmp/test/
```

## Subnet to C-Chain Migration Process

### Critical Finding: evmn Key Format Issue
Subnet databases store canonical mappings as `evmn<32-byte-hash>` but C-Chain expects `evmn<8-byte-number>`. 
This MUST be fixed using `rebuild-canonical` command.

### Complete Migration Pipeline
```bash
# Step 1: Add EVM prefix to all keys
./bin/genesis migrate add-evm-prefix \
    /path/to/subnet/db/pebbledb \
    /path/to/migrated/db/pebbledb

# Step 2: Fix evmn key format (CRITICAL!)
./bin/genesis migrate rebuild-canonical \
    /path/to/migrated/db/pebbledb

# Step 3: Create consensus state
./bin/genesis migrate replay-consensus \
    --evm /path/to/migrated/db/pebbledb \
    --state /path/to/consensus/db/pebbledb \
    --tip <highest-block-number>

# OR use the full pipeline:
./bin/genesis migrate full \
    /path/to/subnet/db/pebbledb \
    /path/to/output/root
```

### Launching luxd with Migrated Data

#### Using the genesis launch command (PREFERRED):
```bash
# Launch luxd with migrated subnet data
./bin/genesis launch L1

# Verify the chain is running correctly
./bin/genesis launch verify

# Expected output:
# ✅ Current block height: <number>
# ✅ Chain ID: 96369
# ✅ Treasury balance: <balance> wei
```

#### Manual launch (if needed):
```bash
~/work/lux/node/build/luxd \
    --network-id=96369 \
    --chain-config-dir=$HOME/work/lux/genesis/configs \
    --chain-data-dir=$HOME/work/lux/genesis/runtime/lux-96369-migrated \
    --dev  # Enables single-node mode with no sybil protection
    --http-host=0.0.0.0

# RPC endpoint: http://localhost:9650/ext/bc/C/rpc
```

#### Important Notes:
- The database contains its own genesis hash which must match
- Use `--dev` flag for single-node testing (replaces all the individual consensus flags)
- Migration creates keys with "evm" prefix as expected
- Genesis mismatch errors indicate the chain config doesn't match the data

### Testing with Ginkgo
```bash
# Run all migration tests
make test

# Run specific test steps
make test filter="Step 1"    # Test subnet data creation
make test filter="Step 2"    # Test EVM prefix migration  
make test filter="Step 3"    # Test synthetic blockchain
make test filter="Step 4"    # Test consensus generation
make test filter="Step 5"    # Test verification tools

# Run minilab migration test
cd test && ginkgo -v --focus "Mini-Lab Migration"
```

## Migration Status (As of 2025-07-28)

### Completed
- ✅ Tool consolidation into unified genesis CLI
- ✅ Namespace extraction tool (handles 32-byte prefixes)
- ✅ Import subnet command with auto-detection
- ✅ Launch commands for L1/L2
- ✅ All tests passing

### In Progress
- 🔄 Full extraction of LUX mainnet (4M+ keys)
- 🔄 Documentation updates
- 🔄 RPC verification scripts

### Pending
- ⏳ Enable indexing after 48h monitoring
- ⏳ Spin up additional validator nodes
- ⏳ Migrate ZOO and SPC networks

## Final Notes

- This is a transparent, verifiable transition
- All historical data is preserved
- Anyone can run these tools to verify
- The goal is a decentralized multi-chain future
- ALWAYS use the genesis CLI tool - NO shell scripts!
- See `docs/SUBNET_MIGRATION.md` for detailed migration guide

Remember: When in doubt, the raw blockchain data is the source of truth!
