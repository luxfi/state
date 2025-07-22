# Validator Key Generation Guide

This guide explains how to generate validator keys for the Lux Network using the genesis-builder tool.

## Overview

Each validator in the Lux Network requires:
1. **BLS Key Pair**: Used for consensus participation
   - Public Key (48 bytes)
   - Proof of Possession (96 bytes)
   - Private Key (32 bytes - keep secure!)

2. **TLS Certificate**: Used for node identity
   - Creates the NodeID
   - staker.crt (certificate)
   - staker.key (private key - keep secure!)

## Key Generation Methods

### Method 1: Generate Compatible Keys (Recommended for Production)

This method generates fully random keys compatible with luxd:

```bash
./bin/genesis-builder -generate-compatible \
    -account-count 11 \
    -save-keys validators.json \
    -save-keys-dir validator-keys/
```

This will:
- Generate 11 unique validators with random keys
- Save validator configurations to `validators.json`
- Save individual validator keys to `validator-keys/validator-N/`

### Method 2: Generate from Seed Phrase

This method derives keys from a seed phrase (useful for testing or recovery):

```bash
./bin/genesis-builder -generate-keys \
    -seed "your twelve word seed phrase here" \
    -account-start 0 \
    -account-count 5 \
    -save-keys validators-from-seed.json
```

**Note**: The seed-based generation uses a simple derivation method. For production use, implement proper BIP32/BIP44 derivation.

## Output Structure

Each validator gets a directory with:
```
validator-1/
├── validator.json       # Public validator info
├── bls.key             # BLS private key (mode 0600)
└── staking/
    ├── staker.crt      # TLS certificate (mode 0644)
    └── staker.key      # TLS private key (mode 0600)
```

## Using Generated Keys

### 1. For Genesis File

The `validators.json` file contains the public information needed for genesis:
```json
{
  "nodeID": "NodeID-...",
  "ethAddress": "0x...",
  "publicKey": "0x...",
  "proofOfPossession": "0x...",
  "weight": 1000000000000000000,
  "delegationFee": 20000
}
```

### 2. For Running a Validator

Copy the validator's directory to the node and configure luxd:
```bash
# Copy to remote node
scp -r validator-keys/validator-1 node1:/path/to/luxd/

# On the node, luxd will use:
luxd --staking-tls-cert-file=/path/to/staking/staker.crt \
     --staking-tls-key-file=/path/to/staking/staker.key
```

## Mainnet Launch Script

Use the provided script to generate all mainnet validators:
```bash
./scripts/generate-mainnet-validators.sh
```

This creates:
- 11 validator key sets
- Template configuration file
- Detailed instructions for deployment

## Security Best Practices

1. **Generate on Secure Machine**: Use an air-gapped machine for production keys
2. **Backup Keys Safely**: Store encrypted backups of private keys
3. **Never Share Private Keys**: Only share public keys and NodeIDs
4. **Use Unique Keys**: Never reuse keys across validators
5. **Secure Transport**: Use encrypted channels when copying keys to nodes

## Integration with Genesis Builder

After generating validators, use them in genesis:
```bash
./bin/genesis-builder \
    --network mainnet \
    --validators generated-validators.json \
    --output genesis_mainnet.json
```

## Troubleshooting

### "Failed to parse certificate"
The tool expects PEM-encoded certificates. This error usually means the certificate data is corrupted.

### "Failed to generate BLS signer"
Ensure you have sufficient entropy on the system. On Linux, check `/dev/random`.

### NodeID Format
NodeIDs are derived from the TLS certificate and look like: `NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg`

The tool also provides the X-Chain address format for convenience.