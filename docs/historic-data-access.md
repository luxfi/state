# Historic Data Access Guide

This guide explains how to access and analyze historic blockchain data from 2023 (chain 7777) and 2024 (chain 96369).

## Available Historic Data

### 2023 - LUX Chain 7777
- **Location**: `data/2023-7777/`
- **PebbleDB Archive**: `pebble-clean-7777.tar.gz` (340MB)
- **Raw PebbleDB**: `pebble-clean/` directory (441MB)
- **CSV Exports**: 
  - `7777-airdrop-96369-mainnet.csv` - All 151 accounts
  - `7777-airdrop-96369-mainnet-no-treasury.csv` - 150 accounts
- **Genesis**: `genesis-7777-evm.json`
- **Summary**: `7777-airdrop-summary.json`

### 2024 - LUX Chain 96369
- **Location**: `data/2024-96369/`
- **Database Archive**: `lux-db-clean.tar.gz` (1.9GB)
- **Genesis**: `genesis-96369-with-7777-accounts.json`

## Loading Data

### Extract PebbleDB from Archive
```bash
# For 2023 data
cd data/2023-7777
tar -xzf pebble-clean-7777.tar.gz

# For 2024 data
cd data/2024-96369
tar -xzf lux-db-clean.tar.gz
```

### Load CSV for Analysis
```python
import pandas as pd

# Load 7777 airdrop data
df = pd.read_csv('data/2023-7777/7777-airdrop-96369-mainnet.csv', comment='#')

# View top accounts
print(df.head(10))

# Total supply verification
total_supply = df['balance_wei'].astype(int).sum()
print(f"Total Supply: {total_supply / 10**18:.2f} LUX")
```

### Import into Modern Luxd
Use the import scripts in `scripts/`:
```bash
# Prepare and import 7777 data
./scripts/import-7777-historic.sh

# For direct PebbleDB access
luxd --db-dir=data/2023-7777/pebble-clean ...
```

## Key Statistics

### Chain 7777 (December 2023)
- **Total Accounts**: 151
- **Total Supply**: 2 trillion LUX
- **Treasury**: 99.74% (0x9011e888251ab053b7bd1cdb598db4f9ded94714)
- **Community**: 0.26% (150 accounts)
- **Blocks**: 888,834

### Chain 96369 (2024)
- **Status**: Current mainnet
- **Migration**: Includes all 7777 account balances

## Git LFS Usage

Large data files are stored using Git LFS:
```bash
# Pull LFS files
git lfs pull

# Track new archives
git lfs track "*.tar.gz"
git add .gitattributes
git add data/**/*.tar.gz
git commit -m "Add historic data archives"
```

## Analysis Scripts

- `scripts/generate_7777_airdrop_csv.py` - Generate CSV from genesis
- `scripts/2023-7777/convert-7777-specific.go` - Convert DB formats
- `scripts/import-7777-historic.sh` - Full import pipeline