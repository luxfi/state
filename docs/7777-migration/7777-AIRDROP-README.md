# 7777 Chain Airdrop Data for 96369 Mainnet

This directory contains the extracted account balances from the original 7777 chain for potential airdrop or migration to the new 96369 mainnet.

## Files

### 1. `genesis-7777-evm.json`
- Complete genesis file for chain 7777
- Contains 151 accounts with total supply of 2 trillion LUX
- Genesis block height: 888,834 (0xD9002)
- Original timestamp: 2023-12-05 (0x656F5D00)

### 2. `7777-airdrop-96369-mainnet.csv`
- Full CSV export of all 151 accounts
- Columns: rank, address, balance_lux, balance_wei, balance_hex, percentage
- Sorted by balance (descending)
- Includes treasury account (99.74% of supply)

### 3. `7777-airdrop-96369-mainnet-no-treasury.csv`
- CSV export excluding the main treasury account
- Contains 150 accounts with 5,260,094,602.72 LUX
- Same format as full CSV
- Useful for community airdrops

### 4. `7777-airdrop-summary.json`
- JSON summary of the airdrop data
- Includes distribution statistics
- Top 10 accounts breakdown
- File generation metadata

## Key Statistics

- **Total Supply**: 2,000,000,000,000 LUX (2 trillion)
- **Total Accounts**: 151
- **Treasury Account**: 0x9011e888251ab053b7bd1cdb598db4f9ded94714
- **Treasury Balance**: 1,994,739,905,397.28 LUX (99.74%)
- **Community Balance**: 5,260,094,602.72 LUX (0.26%)

## Distribution Breakdown (Excluding Treasury)

| Range | Count | Total LUX | % of Remaining |
|-------|-------|-----------|----------------|
| 1B+ LUX | 4 | 4,001,181,074.79 | 76.07% |
| 100M-1B LUX | 4 | 1,056,806,142.78 | 20.09% |
| 10M-100M LUX | 6 | 112,434,557.93 | 2.14% |
| 1M-10M LUX | 24 | 83,476,404.63 | 1.59% |
| < 1M LUX | 112 | 6,196,423.59 | 0.11% |

## Usage

### For Airdrop Implementation
```python
import csv

# Read the airdrop data
with open('7777-airdrop-96369-mainnet-no-treasury.csv', 'r') as f:
    reader = csv.DictReader(f)
    # Skip comment lines
    for row in reader:
        if row['rank'].startswith('#'):
            continue
        address = row['address']
        balance_wei = row['balance_wei']
        # Process airdrop...
```

### For Analysis
```python
import json

# Load summary
with open('7777-airdrop-summary.json', 'r') as f:
    summary = json.load(f)
    
print(f"Total accounts: {summary['total_accounts']}")
print(f"Treasury percentage: {summary['treasury_account']['percentage']}%")
```

## Migration Notes

1. The 7777 chain was originally deployed as a subnet with chain ID 7777
2. All balances have been preserved from block height 888,834
3. The new mainnet uses chain ID 96369 but maintains the same token (LUX)
4. Account addresses remain the same (EVM compatible)

## Scripts

- `analyze_top_accounts.py` - Analyze top account holders
- `analyze_distribution.py` - Analyze token distribution
- `generate_7777_airdrop_csv.py` - Generate these CSV files

Generated: 2025-07-21