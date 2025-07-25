# Archeology Read-Genesis Command

The `archeology read-genesis` command extracts genesis configuration from historic blockchain data.

## Overview

This command can read genesis data from various blockchain databases (PebbleDB, LevelDB) used by avalanchego/luxd. It attempts multiple approaches to find and extract the genesis:

1. Direct genesis key lookup
2. Block 0 header extraction  
3. Config key scanning
4. Minimal genesis creation if none found

## Usage

```bash
# Basic usage
./bin/archeology read-genesis [chaindata-path]

# Save to file
./bin/archeology read-genesis [chaindata-path] -o genesis.json

# Output raw genesis bytes
./bin/archeology read-genesis [chaindata-path] -r -o genesis.blob

# Different output formats
./bin/archeology read-genesis [chaindata-path] -f hex
./bin/archeology read-genesis [chaindata-path] -f base64
```

## Options

- `-o, --output string`: Output file path (default: stdout)
- `-p, --pretty`: Pretty print JSON output (default: true)
- `-i, --show-id`: Show derived blockchain ID (default: true)
- `-r, --raw`: Output raw genesis bytes
- `-f, --format string`: Output format: json, hex, base64 (default: "json")

## Examples

### Extract from LUX mainnet data
```bash
./bin/archeology read-genesis /home/z/work/lux/genesis/chaindata/lux-mainnet-96369
```

### Extract from archived blockchain data
```bash
./bin/archeology read-genesis /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ
```

### Save genesis and show blockchain ID
```bash
./bin/archeology read-genesis /path/to/chaindata -o genesis.json -i
```

## Implementation Details

The command searches for genesis in the following locations:
- Direct "genesis" key in the database
- Block 0 header and associated data
- Config-related keys that might contain genesis
- Falls back to creating a minimal genesis if none found

The blockchain ID is derived by taking the SHA256 hash of the genesis bytes and converting it to an Avalanche ID format.

## Notes

- Works with both PebbleDB and LevelDB databases
- Automatically detects database type
- Handles namespaced and non-namespaced keys
- Compatible with avalanchego/luxd database formats