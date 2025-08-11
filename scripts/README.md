# SubnetEVM Database Analysis Scripts

These scripts analyze the SubnetEVM database structure to understand how blocks and state are stored.

## Key Findings

The SubnetEVM database contains:

- **1,082,781 blocks** (0 to 1,082,780)  
- All blocks have headers, bodies, and receipts
- All data is namespaced with a 32-byte prefix
- State nodes are stored as 64-byte keys (32 namespace + 32 hash)

### Namespace
0x337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1

### Key Patterns Found

- 73 bytes: namespace(32) + h/b/r(1) + num(8) + hash(32) = headers/bodies/receipts
- 41 bytes: namespace(32) + H(1) + num(8) = Canonical chain  
- 65 bytes: namespace(32) + H(1) + hash(32) = Hash to number mapping
- 64 bytes: namespace(32) + stateHash(32) = State trie nodes

### Important Blocks

- Genesis (0): 0x3f4fa2a0b0ce089f52bf0ae9199c75ffdd76ecafc987794050cb0d286f1ec61e
- Target (1,082,780): 0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0

## Usage

go run analyze-blocks.go
go run analyze-75byte-keys.go
